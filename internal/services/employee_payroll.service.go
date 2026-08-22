package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Xodimlarning oylik ish haqi hisoboti (payroll).
//
// Ma'lumot ikkita manbadan keladi, tanlov so'ralgan davrga bog'liq:
//
//	Joriy oy  → manba jadvallardan JONLI hisoblanadi (oy boshidan bugungi kungacha)
//	O'tgan oy → employee_payrolls'dan O'QILADI (snapshot)
//
// Snapshot'ni AutoCreateMonthlyPayrolls cron'i har oyning 1-kuni yozadi — xuddi
// store_target'lardagi kabi. Ikkala yo'l ham bir xil domain.EmployeePayrollRow
// qaytaradi, shuning uchun handler uchun farqi yo'q; javobdagi period.is_live
// qaysi yo'l ishlaganini ko'rsatadi.
//
// Hisob-kitob formulalari (yagona manba — payrollCalcQuery, fayl oxirida):
//
//	worked_hours         = SUM(employee_attendance_days.worked_minutes) / 60
//	individual_sales     = SUM(employee_attendance_days.sales_amount)
//	bonus_amount         = SUM(employee_bonus.bonus_amount)
//	actual_salary_amount = salary_rate * (worked_hours / avg_monthly_hours)
//	kpi_amount           = individual_sales * kpi_percent / 100
//	gross_salary_amount  = actual_salary + kpi_amount + bonus_amount
//	net_pay_amount       = gross - (avanslar + ushlab qolishlar)
//
// Avans va ushlab qolishlar qo'lda kiritiladi, manba jadvali yo'q. Shu sababli
// jonli hisobda ular doim 0 (ya'ni net = gross), o'tgan oylarda esa
// employee_payrolls'dagi saqlangan qiymat ishlatiladi.

// region Types

// payrollScope — so'rovning WHERE qismi: hisobot qaysi xodimlarni qamraydi.
//
// Jonli hisob `employees e` jadvalidan, saqlangan hisob `employee_payrolls p`
// jadvalidan filtrlanadi — ustun nomlari boshqacha, shuning uchun ikkita alohida
// SQL bo'lagi kerak. Parametrlar ikkalasi uchun bir xil, shuning uchun bitta
// joyda turadi va tartibi adashmaydi.
//
// Bu bo'laklar faqat shu fayldagi doimiy satrlar — foydalanuvchi kiritmasi hech
// qachon SQL matniga qo'shilmaydi, u faqat args orqali o'tadi.
type payrollScope struct {
	employeesWhere string // `employees e` uchun
	payrollsWhere  string // `employee_payrolls p` uchun
	args           []any
}

// storeRef — sahifalangan do'kon ro'yxati uchun minimal ma'lumot.
type storeRef struct {
	Id   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// scopeAllEmployees — barcha faol xodimlar (cron uchun).
func scopeAllEmployees() payrollScope {
	return payrollScope{
		employeesWhere: "TRUE",
		payrollsWhere:  "TRUE",
	}
}

// scopeByStores — berilgan do'konlarning xodimlari.
func scopeByStores(storeIds []string) payrollScope {
	return payrollScope{
		employeesWhere: "e.store_id IN ?",
		payrollsWhere:  "p.store_id IN ?",
		args:           []any{storeIds},
	}
}

// scopeByCompany — bitta kompaniyaning faol do'konlaridagi xodimlar.
//
// Do'kon filtri paginateStores'dagi bilan bir xil (o'chirilmagan va faol), shu
// sababli do'konlar ro'yxati API'sida ko'rinmaydigan do'konning xodimi bu yerda
// ham chiqmaydi.
func scopeByCompany(companyId string) payrollScope {
	const storesOfCompany = `IN (
		SELECT id FROM stores
		WHERE company_id = ? AND deleted_at IS NULL AND is_active = TRUE
	)`

	return payrollScope{
		employeesWhere: "e.store_id " + storesOfCompany,
		payrollsWhere:  "p.store_id " + storesOfCompany,
		args:           []any{companyId},
	}
}

// scopeByEmployee — bitta xodim.
func scopeByEmployee(employeeId string) payrollScope {
	return payrollScope{
		employeesWhere: "e.id = ?",
		payrollsWhere:  "p.employee_id = ?",
		args:           []any{employeeId},
	}
}

// region Get

// GetStorePayrolls — do'konlar kesimidagi oylik hisobot: FAQAT do'kon yig'indilari,
// xodimlar ro'yxatisiz. Bitta do'konning xodimlarini olish uchun alohida
// GetEmployeePayrolls ishlatiladi (store_id bo'yicha).
//
// Pagination xodimlarga emas, DO'KONLARGA qo'yiladi: avval bir sahifa do'kon
// tanlanadi, keyin faqat o'sha do'konlarning xodimlari bitta so'rov bilan
// olinadi va do'kon yig'indisiga qo'shiladi.
func (s *Services) GetStorePayrolls(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]domain.StorePayroll, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, 0, period, domain.BadRequestError
	}

	stores, totalCount, err := s.paginateStores(ctx, params)
	if err != nil {
		return nil, 0, period, err
	}
	if len(stores) == 0 {
		return []domain.StorePayroll{}, totalCount, period, nil
	}

	rows, err := s.payrollRows(ctx, period, scopeByStores(storeIdsOf(stores)))
	if err != nil {
		return nil, 0, period, err
	}

	storePayrolls := buildStorePayrolls(stores, rows)

	// Yig'indilar allaqachon hisoblangan — javobda faqat do'kon qatorlari qoladi.
	// Xodimlar ro'yxati GetEmployeePayrolls orqali alohida so'raladi.
	for i := range storePayrolls {
		storePayrolls[i].Employees = nil
	}

	return storePayrolls, totalCount, period, nil
}

// GetEmployeePayrolls — bitta do'kon xodimlarining oylik hisoboti.
//
// Hisob-kitob GetStorePayrolls bilan AYNAN bir xil yo'ldan boradi: do'kon o'sha
// paginateStores filtri bilan tekshiriladi, so'ng o'sha payrollRows chaqiriladi
// (joriy oy → jonli hisob, o'tgan oy → employee_payrolls snapshot'i). Farqi
// faqat javob shaklida: do'kon yig'indisi emas, xodim qatorlari qaytadi.
//
// store_id ixtiyoriy: berilsa faqat o'sha do'kon xodimlari, berilmasa doira
// employeePayrollScope orqali aniqlanadi (kompaniya yoki barcha xodimlar).
//
// Pagination xodimlarga qo'yiladi va xotirada bajariladi — hisob so'rovi
// GetStorePayrolls bilan bitta bo'lib qolishi uchun unga LIMIT/OFFSET
// qo'shilmaydi.
func (s *Services) GetEmployeePayrolls(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]domain.EmployeePayrollRow, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, 0, period, domain.BadRequestError
	}

	scope, found, err := s.employeePayrollScope(ctx, params)
	if err != nil {
		return nil, 0, period, err
	}
	if !found {
		return []domain.EmployeePayrollRow{}, 0, period, nil
	}

	rows, err := s.payrollRows(ctx, period, scope)
	if err != nil {
		return nil, 0, period, err
	}

	return pagePayrollRows(rows, params.Limit, params.Offset), int64(len(rows)), period, nil
}

// GetMyPayroll — token egasining o'z oyligi.
func (s *Services) GetMyPayroll(
	ctx context.Context, employeeId string, year, month int,
) (*domain.MyPayrollResponse, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, domain.BadRequestError
	}

	rows, err := s.payrollRows(ctx, period, scopeByEmployee(employeeId))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.NotFoundError
	}

	return &domain.MyPayrollResponse{Period: period, Payroll: rows[0]}, nil
}

// region Cron

// AutoCreateMonthlyPayrolls — har oyning 1-kuni cron tomonidan chaqiriladi.
//
// Endigina tugagan oyni manba jadvallardan to'liq hisoblab, employee_payrolls'ga
// snapshot qilib yozadi. Shundan keyin o'sha oy uchun API saqlangan qiymatni
// qaytaradi — davomat yoki bonus keyinchalik o'zgarsa ham oylik "muzlatilgan"
// bo'lib qoladi.
//
// Takror ishga tushirish xavfsiz: UNIQUE(employee_id, year, month) bo'yicha
// konflikt bo'lsa yozuv jim o'tkazib yuboriladi, mavjud qiymat buzilmaydi.
func (s *Services) AutoCreateMonthlyPayrolls() {
	const op = "cron AutoCreateMonthlyPayrolls"

	ctx := context.Background()
	prevMonth := tashkentNow().AddDate(0, -1, 0)

	period, err := resolvePayrollPeriod(prevMonth.Year(), int(prevMonth.Month()))
	if err != nil {
		s.log.Errorf("%s: %v", op, err)
		return
	}

	// Ataylab payrollRows emas, to'g'ridan-to'g'ri calculatePayroll: maqsad —
	// o'tgan oyni manba jadvallardan hisoblab, employee_payrolls'ga BIRINCHI
	// marta yozish. payrollRows bo'lsa o'sha hali bo'sh jadvaldan o'qigan bo'lardi.
	rows, err := s.calculatePayroll(ctx, period, scopeAllEmployees())
	if err != nil {
		s.log.Errorf("%s: could not calculate: %v", op, err)
		return
	}

	created := 0
	for _, row := range rows {
		record := newPayrollRecord(row, period.Year, period.Month)

		err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&record).Error
		if err != nil {
			s.log.Errorf("%s: employee %s: %v", op, row.EmployeeId, err)
			continue
		}
		created++
	}

	s.log.Infof("%s: %d-%02d completed, %d records created", op, period.Year, period.Month, created)
}

// region Period

// tashkentNow — Toshkent devor-vaqti. Loyihada sana chegaralari hamma joyda
// UTC+5 bo'yicha olinadi (qarang: attendance service, report handler).
func tashkentNow() time.Time {
	return time.Now().UTC().Add(domain.TashkentTimeDif)
}

// resolvePayrollPeriod — so'ralgan yil/oyni tekshirib, davr chegaralarini qaytaradi.
// year yoki month 0 bo'lsa joriy yil/oy olinadi.
//
// Joriy oy uchun davr bugungi kunda tugaydi va IsLive=true bo'ladi (oy hali
// tugamagan, hisob jonli). O'tgan oylar uchun davr oyning oxirgi kunida tugaydi
// va IsLive=false — bunday oy uchun employee_payrolls'da tayyor snapshot bor.
func resolvePayrollPeriod(year, month int) (domain.PayrollPeriod, error) {
	now := tashkentNow()
	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}
	if month < 1 || month > 12 {
		return domain.PayrollPeriod{}, fmt.Errorf("invalid month: %d", month)
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	if firstDay.After(now) {
		return domain.PayrollPeriod{}, fmt.Errorf("period is in the future: %d-%02d", year, month)
	}

	isCurrentMonth := year == now.Year() && month == int(now.Month())

	lastDay := firstDay.AddDate(0, 1, -1)
	if isCurrentMonth {
		lastDay = now
	}

	return domain.PayrollPeriod{
		Year:   year,
		Month:  month,
		From:   firstDay.Format(constants.TimeOnlyDateFormat),
		To:     lastDay.Format(constants.TimeOnlyDateFormat),
		IsLive: isCurrentMonth,
	}, nil
}

// region Fetch

// payrollRows — davrga qarab to'g'ri manbani tanlaydi.
func (s *Services) payrollRows(
	ctx context.Context, period domain.PayrollPeriod, scope payrollScope,
) ([]domain.EmployeePayrollRow, error) {
	if period.IsLive {
		return s.calculatePayroll(ctx, period, scope)
	}
	return s.loadStoredPayroll(ctx, period, scope)
}

// calculatePayroll — manba jadvallardan (davomat, bonus, target) jonli hisoblaydi.
func (s *Services) calculatePayroll(
	ctx context.Context, period domain.PayrollPeriod, scope payrollScope,
) ([]domain.EmployeePayrollRow, error) {
	query := fmt.Sprintf(payrollCalcQuery, scope.employeesWhere)

	// Tartib payrollCalcQuery'dagi `?` belgilariga qat'iy mos kelishi shart —
	// SQL o'zgarsa shu ro'yxat ham o'zgaradi.
	args := []any{
		period.From, period.To, // att CTE — davomat davri
		period.From, period.To, // bon CTE — bonus davri
		period.Year, period.Month, // store_targets — shu oyning plani
		constants.GeneralStatusActive, // employees.status
	}
	args = append(args, scope.args...)

	rows, err := s.scanPayrollRows(ctx, query, args)
	if err != nil {
		return nil, err
	}

	// Bu uchta maydon hisoblangan qatorda SQL'dan kelmaydi — davrdan to'ldiramiz.
	// Hali tasdiqlanmagan hisob, shuning uchun status doim "draft".
	for i := range rows {
		rows[i].Status = domain.EmployeePayrollStatusDraft
		rows[i].Year = period.Year
		rows[i].Month = period.Month
	}
	return rows, nil
}

// loadStoredPayroll — employee_payrolls'dagi tayyor snapshot'ni o'qiydi.
// Cron o'sha oy uchun hali ishlamagan bo'lsa natija bo'sh bo'ladi.
func (s *Services) loadStoredPayroll(
	ctx context.Context, period domain.PayrollPeriod, scope payrollScope,
) ([]domain.EmployeePayrollRow, error) {
	query := fmt.Sprintf(payrollStoredQuery, scope.payrollsWhere)

	args := []any{period.Year, period.Month}
	args = append(args, scope.args...)

	return s.scanPayrollRows(ctx, query, args)
}

// scanPayrollRows — tayyor so'rovni bajarib, natijani qatorlarga o'giradi.
func (s *Services) scanPayrollRows(
	ctx context.Context, query string, args []any,
) ([]domain.EmployeePayrollRow, error) {
	var rows []domain.EmployeePayrollRow
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		s.log.Errorf("payroll: could not read rows: %v", err)
		return nil, domain.InternalServerError
	}
	return rows, nil
}

// region Helpers

// paginateStores — filtrga mos do'konlarning bir sahifasini va umumiy sonini qaytaradi.
func (s *Services) paginateStores(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]storeRef, int64, error) {
	// Count va Find uchun so'rov har safar qaytadan quriladi: GORM'da finisher
	// (Count) chaqirilgandan keyin o'sha *gorm.DB'ni qayta ishlatish statement'ni
	// ifloslantiradi va keyingi so'rovga Count'ning SELECT'i sizib o'tishi mumkin.
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).
			Table("stores").
			Where("deleted_at IS NULL").
			Where("is_active = TRUE")
		if params.CompanyId != "" {
			q = q.Where("company_id = ?", params.CompanyId)
		}
		if params.StoreId != "" {
			q = q.Where("id = ?", params.StoreId)
		}
		return q
	}

	var totalCount int64
	if err := newQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("payroll: could not count stores: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var stores []storeRef
	err := newQuery().
		Select("id, name").
		Order("name").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&stores).Error
	if err != nil {
		s.log.Errorf("payroll: could not get stores: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return stores, totalCount, nil
}

// employeePayrollScope — xodimlar hisobotining doirasini so'rov filtridan aniqlaydi.
//
// Ikkinchi qaytim qiymati false bo'lsa — so'ralgan do'kon topilmadi (o'chirilgan,
// nofaol yoki foydalanuvchining kompaniyasiga tegishli emas), bunda hisob umuman
// bajarilmaydi va bo'sh ro'yxat qaytadi.
//
//	store_id berilgan          → faqat o'sha do'kon (paginateStores bilan tekshiriladi)
//	store_id yo'q + company_id → o'sha kompaniyaning faol do'konlaridagi xodimlar
//	store_id yo'q + company_id yo'q (admin) → barcha faol xodimlar, shu jumladan
//	do'konga biriktirilmaganlar ham
func (s *Services) employeePayrollScope(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) (payrollScope, bool, error) {
	if params.StoreId == "" {
		if params.CompanyId != "" {
			return scopeByCompany(params.CompanyId), true, nil
		}
		return scopeAllEmployees(), true, nil
	}

	// Do'kon filtri do'konlar API'sidagi bilan bir xil bo'lishi uchun o'sha
	// paginateStores ishlatiladi; bu yerda u faqat bitta do'konni tekshiradi,
	// shuning uchun sahifa parametrlari almashtiriladi.
	storeParams := *params
	storeParams.Limit, storeParams.Offset = 1, 0

	stores, _, err := s.paginateStores(ctx, &storeParams)
	if err != nil {
		return payrollScope{}, false, err
	}
	if len(stores) == 0 {
		return payrollScope{}, false, nil
	}

	return scopeByStores(storeIdsOf(stores)), true, nil
}

// pagePayrollRows — tayyor xodim qatorlarini xotirada sahifalaydi.
//
// Tartib SQL'da qat'iy (ORDER BY store_name, full_name), shuning uchun sahifalar
// barqaror bo'ladi.
func pagePayrollRows(
	rows []domain.EmployeePayrollRow, limit, offset int,
) []domain.EmployeePayrollRow {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(rows) {
		return []domain.EmployeePayrollRow{}
	}

	rows = rows[offset:]
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}
	return rows
}

// storeIdsOf — do'konlar ro'yxatidan id'larni ajratib oladi.
func storeIdsOf(stores []storeRef) []string {
	ids := make([]string, 0, len(stores))
	for _, store := range stores {
		ids = append(ids, store.Id)
	}
	return ids
}

// buildStorePayrolls — xodim qatorlarini do'konlar bo'yicha guruhlaydi va har bir
// do'kon uchun yig'indilarni hisoblaydi.
//
// Do'kon ro'yxati tartibi saqlanadi va xodimi yo'q do'kon ham javobda qoladi
// (bo'sh employees massivi bilan) — aks holda sahifadagi do'konlar soni
// _meta.total_count bilan mos kelmay qolardi.
func buildStorePayrolls(stores []storeRef, rows []domain.EmployeePayrollRow) []domain.StorePayroll {
	rowsByStore := make(map[string][]domain.EmployeePayrollRow, len(stores))
	for _, row := range rows {
		if row.StoreId == nil {
			continue // do'konga biriktirilmagan xodim — hech qaysi guruhga tushmaydi
		}
		rowsByStore[*row.StoreId] = append(rowsByStore[*row.StoreId], row)
	}

	result := make([]domain.StorePayroll, 0, len(stores))
	for _, store := range stores {
		employees := rowsByStore[store.Id]
		if employees == nil {
			employees = []domain.EmployeePayrollRow{}
		}

		storePayroll := domain.StorePayroll{
			StoreId:       store.Id,
			StoreName:     store.Name,
			EmployeeCount: len(employees),
			Employees:     employees,
		}
		for _, employee := range employees {
			addEmployeeToStoreTotals(&storePayroll, employee)
		}
		// Do'kon plani har bir xodim qatorida takrorlanadi (bitta store_target'dan
		// keladi), shuning uchun qo'shilmaydi — bittasidan olinadi.
		if len(employees) > 0 {
			storePayroll.StorePlanAmount = employees[0].StorePlanAmount
		}

		result = append(result, storePayroll)
	}

	return result
}

// addEmployeeToStoreTotals — bitta xodimning summalarini do'kon yig'indisiga qo'shadi.
func addEmployeeToStoreTotals(store *domain.StorePayroll, employee domain.EmployeePayrollRow) {
	store.WorkedHours += employee.WorkedHours
	store.SalaryRateAmount += employee.SalaryRateAmount
	store.ActualSalaryAmount += employee.ActualSalaryAmount
	store.IndividualSalesAmount += employee.IndividualSalesAmount
	store.KpiAmount += employee.KpiAmount
	store.BonusAmount += employee.BonusAmount
	store.GrossSalaryAmount += employee.GrossSalaryAmount
	store.NetPayAmount += employee.NetPayAmount
}

// newPayrollRecord — hisoblangan qatordan employee_payrolls yozuvini yasaydi (cron uchun).
//
// Avans va ushlab qolishlar bu yerda to'ldirilmaydi: ular qo'lda kiritiladi va
// jadvalga DEFAULT 0 bilan tushadi. ApprovedBy/CompletedAt ham bo'sh qoladi —
// ular tasdiqlash bosqichida to'ldiriladi.
func newPayrollRecord(row domain.EmployeePayrollRow, year, month int) domain.EmployeePayroll {
	return domain.EmployeePayroll{
		Id:                    uuid.New().String(),
		EmployeeId:            row.EmployeeId,
		StoreId:               row.StoreId,
		PositionSnapshot:      row.PositionSnapshot,
		ExperienceYears:       row.ExperienceYears,
		WorkedHours:           row.WorkedHours,
		AvgMonthlyHours:       row.AvgMonthlyHours,
		SalaryRateAmount:      row.SalaryRateAmount,
		ActualSalaryAmount:    row.ActualSalaryAmount,
		IndividualSalesAmount: row.IndividualSalesAmount,
		StoreTargetId:         row.StoreTargetId,
		StorePlanAmount:       row.StorePlanAmount,
		KpiPercent:            row.KpiPercent,
		KpiAmount:             row.KpiAmount,
		BonusAmount:           row.BonusAmount,
		GrossSalaryAmount:     row.GrossSalaryAmount,
		NetPayAmount:          row.NetPayAmount,
		Status:                domain.EmployeePayrollStatusDraft,
		Year:                  year,
		Month:                 month,
	}
}

// region SQL

// payrollCalcQuery — oylikni manba jadvallardan hisoblaydi.
//
// %s — payrollScope.employeesWhere. Parametrlar tartibi calculatePayroll'da,
// izohlar bilan ko'rsatilgan; SQL'dagi `?` lar o'zgarsa u ro'yxat ham o'zgaradi.
//
// interval '5 hours' — employee_bonus.created_at UTC'da saqlanadi, sana esa
// Toshkent kuni bo'yicha kesiladi. employee_attendance_days.work_date allaqachon
// Toshkent sanasi, shuning uchun unga tuzatish kerak emas.
const payrollCalcQuery = `
WITH att AS (
    SELECT employee_id,
           SUM(worked_minutes)::numeric / 60.0 AS worked_hours,
           SUM(sales_amount)                   AS individual_sales_amount
    FROM employee_attendance_days
    WHERE work_date BETWEEN ? AND ?
    GROUP BY employee_id
),
bon AS (
    SELECT employee_id, SUM(bonus_amount) AS bonus_amount
    FROM employee_bonus
    WHERE deleted_at IS NULL
      AND (created_at + interval '5 hours')::date BETWEEN ? AND ?
    GROUP BY employee_id
),
base AS (
    SELECT
        e.id                                   AS employee_id,
        e.store_id                             AS store_id,
        COALESCE(s.name, '')                   AS store_name,
        e.first_name                           AS first_name,
        e.last_name                            AS last_name,
        e.full_name                            AS full_name,
        COALESCE(e.position, '')               AS position_snapshot,
        e.experience_years                     AS experience_years,
        e.avg_monthly_hours                    AS avg_monthly_hours,
        e.salary                               AS salary_rate_amount,
        e.kpi_percent                          AS kpi_percent,
        COALESCE(a.worked_hours, 0)            AS worked_hours,
        COALESCE(a.individual_sales_amount, 0) AS individual_sales_amount,
        COALESCE(b.bonus_amount, 0)            AS bonus_amount,
        st.id                                  AS store_target_id,
        COALESCE(st.amount, 0)                 AS store_plan_amount
    FROM employees e
    LEFT JOIN stores s         ON s.id = e.store_id
    LEFT JOIN att a            ON a.employee_id = e.id
    LEFT JOIN bon b            ON b.employee_id = e.id
    LEFT JOIN store_targets st ON st.store_id = e.store_id AND st.year = ? AND st.month = ?
    WHERE e.status = ?
      AND e.deleted_at IS NULL
      AND %s
),
calc AS (
    SELECT base.*,
           -- avg_monthly_hours = 0 bo'lsa nolga bo'lish o'rniga 0 qaytadi
           COALESCE(ROUND(salary_rate_amount * (worked_hours / NULLIF(avg_monthly_hours, 0)), 2), 0) AS actual_salary_amount,
           ROUND(individual_sales_amount * kpi_percent / 100.0, 2)                                   AS kpi_amount
    FROM base
)
SELECT calc.*,
       (actual_salary_amount + kpi_amount + bonus_amount) AS gross_salary_amount,
       -- avans va ushlab qolishlarning manba jadvali yo'q: jonli hisobda doim 0
       0::numeric                                         AS advance_card_amount,
       0::numeric                                         AS advance_cash_amount,
       0::numeric                                         AS deduction_term_amount,
       0::numeric                                         AS deduction_recount_amount,
       0::numeric                                         AS deduction_fine_amount,
       (actual_salary_amount + kpi_amount + bonus_amount) AS net_pay_amount
FROM calc
ORDER BY store_name, full_name`

// payrollStoredQuery — o'tgan oy uchun employee_payrolls'dagi snapshot'ni o'qiydi.
// Ustunlar ro'yxati payrollCalcQuery natijasi bilan bir xil bo'lishi shart —
// ikkalasi ham domain.EmployeePayrollRow'ga scan qilinadi.
//
// %s — payrollScope.payrollsWhere.
const payrollStoredQuery = `
SELECT
    p.employee_id, p.store_id, COALESCE(s.name, '') AS store_name,
    e.first_name, e.last_name, e.full_name,
    p.position_snapshot, p.experience_years, p.worked_hours, p.avg_monthly_hours,
    p.salary_rate_amount, p.actual_salary_amount, p.individual_sales_amount,
    p.store_target_id, p.store_plan_amount, p.kpi_percent, p.kpi_amount,
    p.bonus_amount, p.gross_salary_amount,
    p.advance_card_amount, p.advance_cash_amount,
    p.deduction_term_amount, p.deduction_recount_amount, p.deduction_fine_amount,
    p.net_pay_amount, p.status, p.month, p.year, p.completed_at
FROM employee_payrolls p
JOIN employees e      ON e.id = p.employee_id
LEFT JOIN stores s    ON s.id = p.store_id
WHERE p.year = ? AND p.month = ? AND %s
ORDER BY store_name, e.full_name`
