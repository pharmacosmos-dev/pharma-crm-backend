package services

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// Xodimlarning oylik ish haqi hisoboti (payroll).
//
// Yozish va o'qish qat'iy ajratilgan:
//
//	YOZISH — RecalculateDailyPayrolls cron'i har kecha 02:00 (Toshkent) da
//	         employee_payrolls'ni qayta hisoblab UPSERT qiladi.
//	O'QISH — GetEmployeePayrolls / GetMyPayroll / GetStorePayrolls faqat
//	         employee_payrolls'dan o'qiydi, hech qanday JOIN yo'q.
//
// Shu sababli xodim ismi, do'kon nomi, roli va kompaniyasi payroll qatoriga
// snapshot qilinadi: hisobot tez o'qiladi va xodim keyin boshqa do'konga
// ko'chirilsa ham eski oy hisoboti o'sha paytdagi holatni ko'rsatadi.
//
// KPI PROGRESSIV — har kuni o'zgaradi:
//
//	ish kunlari   = kalendar kun − yakshanba − holidays jadvalidagi sanalar
//	expected_plan = employee_targets.amount * (o'tgan ish kuni / oydagi ish kuni)
//	achievement   = individual_sales / expected_plan * 100
//	kpi_percent   = 0%    (achievement < 80)
//	                0.8%  (80 <= achievement < 90)
//	                1.0%  (90 <= achievement < 100)
//	                1.4%  (achievement >= 100)
//	kpi_amount    = individual_sales * kpi_percent / 100
//
// Xodim har kuni ertalab o'z foizini ko'radi va bugun qancha savdo qilsa KPI
// qaysi pog'onaga chiqishini biladi.
//
// Qolgan formulalar:
//
//	worked_hours         = SUM(employee_attendance_days.worked_minutes) / 60
//	individual_sales     = SUM(employee_attendance_days.sales_amount)
//	bonus_amount         = SUM(employee_bonus.bonus_amount)
//
// Savdo ataylab `sales` jadvalidan emas, employee_attendance_days'dan olinadi:
// u yerda kunlik savdo AggregateEmployeeAttendanceDays (00:30 Toshkent) orqali
// allaqachon yig'ilgan, jadval kichik va (employee_id, work_date) indeksi bor.
// `sales`da employee_id bo'yicha indeks yo'q — undan o'qish har kecha to'liq
// skanga olib kelardi.
//	actual_salary_amount = salary_rate * (worked_hours / avg_monthly_hours)
//	gross_salary_amount  = actual_salary + kpi_amount + bonus_amount
//	net_pay_amount       = gross − (avanslar + ushlab qolishlar)
//
// Avans va ushlab qolishlar qo'lda kiritiladi — cron ularga TEGMAYDI, faqat
// net_pay_amount'ni ular hisobga olingan holda qayta hisoblaydi.

// region Types

// storeRef — sahifalangan do'kon ro'yxati uchun minimal ma'lumot.
// EmployeeCount employees jadvalidan olinadi, payroll qatorlaridan emas —
// shuning uchun cron hali ishlamagan bo'lsa ham to'g'ri son ko'rsatiladi.
type storeRef struct {
	Id            string `gorm:"column:id"`
	Name          string `gorm:"column:name"`
	EmployeeCount int    `gorm:"column:employee_count"`
}

// employeePayrollPageRow — hisobot qatori + umumiy son. total_count har bir
// qatorda bir xil (COUNT(*) OVER ()) va faqat pagination uchun kerak, shuning
// uchun javobga chiqmaydi — domain.EmployeePayrollRow o'zgarishsiz qoladi.
type employeePayrollPageRow struct {
	domain.EmployeePayrollRow `gorm:"embedded"`

	TotalCount int64 `gorm:"column:total_count"`
}

// payrollReadFilter — o'qish so'rovining doirasi. Bo'sh maydon = filtrlamaslik.
type payrollReadFilter struct {
	EmployeeId string
	StoreId    string
	CompanyId  string
	Roles      []string
	Limit      int // 0 = cheklovsiz
	Offset     int
}

// payrollSalesRoles — savdo nuqtasida ishlaydigan rollar. Xodimlar hisoboti
// faqat shularni ko'rsatadi; o'z oyligini ko'rishda rol tekshirilmaydi, aks
// holda menejer o'z oyligini ko'ra olmasdi.
var payrollSalesRoles = []string{constants.RoleNameCashier, constants.RoleNameZavStore}

// nullIfEmpty — bo'sh satrni SQL NULL'ga aylantiradi: "filtr berilmagan" degani.
func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// region Get

// GetEmployeePayrolls — xodimlar hisoboti. employee_payrolls'dan to'g'ridan-to'g'ri
// o'qiydi: JOIN ham, hisob-kitob ham yo'q — hammasi cron tomonidan oldindan yozilgan.
//
// So'ralgan oy uchun cron hali ishlamagan bo'lsa natija bo'sh bo'ladi.
func (s *Services) GetEmployeePayrolls(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]domain.EmployeePayrollRow, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, 0, period, domain.BadRequestError
	}

	page, err := s.selectPayrolls(ctx, period, payrollReadFilter{
		CompanyId: params.CompanyId,
		StoreId:   params.StoreId,
		Roles:     payrollSalesRoles,
		Limit:     params.Limit,
		Offset:    params.Offset,
	})
	if err != nil {
		return nil, 0, period, err
	}

	// Umumiy son har bir qatorda takrorlanadi (COUNT(*) OVER ()), birinchisidan olinadi.
	var totalCount int64
	if len(page) > 0 {
		totalCount = page[0].TotalCount
	}

	return payrollRowsOf(page), totalCount, period, nil
}

// GetMyPayroll — token egasining o'z oyligi. Rol filtri yo'q: xodim qaysi
// lavozimda bo'lishidan qat'i nazar o'z oyligini ko'ra oladi.
func (s *Services) GetMyPayroll(
	ctx context.Context, employeeId string, year, month int,
) (*domain.MyPayrollResponse, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, domain.BadRequestError
	}

	page, err := s.selectPayrolls(ctx, period, payrollReadFilter{
		EmployeeId: employeeId,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(page) == 0 {
		return nil, domain.NotFoundError
	}

	return &domain.MyPayrollResponse{Period: period, Payroll: page[0].EmployeePayrollRow}, nil
}

// GetStorePayrolls — do'konlar kesimidagi yig'indilar, xodimlar ro'yxatisiz.
//
// Pagination do'konlarga qo'yiladi. Yig'indiga do'konning BARCHA xodimlari
// kiradi (rol filtri yo'q), shuning uchun do'kon summasi GetEmployeePayrolls
// qaytaradigan qatorlar yig'indisidan katta bo'lishi mumkin.
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

	var totals []domain.StorePayroll
	err = s.db.WithContext(ctx).Raw(storePayrollTotalsQuery, map[string]any{
		"year":      period.Year,
		"month":     period.Month,
		"store_ids": pq.StringArray(storeIdsOf(stores)),
	}).Scan(&totals).Error
	if err != nil {
		s.log.Errorf("payroll: could not get store totals: %v", err)
		return nil, 0, period, domain.InternalServerError
	}

	return mergeStoreTotals(stores, totals), totalCount, period, nil
}

// region Cron

// RecalculateDailyPayrolls — har kecha 02:00 (Toshkent) da chaqiriladi.
//
// Kechagi kun to'liq yakunlangani uchun hisob oy boshidan KECHAGI kungacha
// olib boriladi. Bitta INSERT ... ON CONFLICT DO UPDATE bilan:
//
//	qator yo'q bo'lsa → yaratiladi (oy boshi, yoki oy o'rtasida qo'shilgan yangi xodim)
//	qator bor bo'lsa  → qayta hisoblanib yangilanadi
//
// Oyning 1-kunida kecha o'tgan oyga tegishli bo'ladi: avval o'tgan oy yakuniy
// hisob bilan yopiladi, keyin yangi oy uchun qatorlar ochiladi.
func (s *Services) RecalculateDailyPayrolls() {
	const op = "cron RecalculateDailyPayrolls"

	ctx := context.Background()
	today := tashkentNow()
	yesterday := today.AddDate(0, 0, -1)

	// 1) Kecha qaysi oyga tegishli bo'lsa — o'sha oyni kechagi kungacha hisoblash.
	rows, err := s.recalculatePayrollMonth(ctx, yesterday.Year(), int(yesterday.Month()), yesterday)
	if err != nil {
		s.log.Errorf("%s: %d-%02d: %v", op, yesterday.Year(), int(yesterday.Month()), err)
		return
	}
	s.log.Infof("%s: %d-%02d recalculated through %s: %d rows",
		op, yesterday.Year(), int(yesterday.Month()), yesterday.Format(constants.TimeOnlyDateFormat), rows)

	// 2) Oy almashgan bo'lsa (bugun 1-kun) yangi oy uchun qatorlarni ochish.
	// elapsed_work_days = 0 bo'lgani uchun barcha summalar 0 dan boshlanadi.
	if today.Month() != yesterday.Month() {
		opened, err := s.recalculatePayrollMonth(ctx, today.Year(), int(today.Month()), today)
		if err != nil {
			s.log.Errorf("%s: new month %d-%02d: %v", op, today.Year(), int(today.Month()), err)
			return
		}
		s.log.Infof("%s: new month %d-%02d opened: %d rows", op, today.Year(), int(today.Month()), opened)
	}
}

// RecalculatePayroll — cron bajaradigan hisobni QO'LDA ishga tushiradi.
//
// Cron tunda ishlamay qolgan (server o'chgan, xato bo'lgan) hollar uchun: ertalab
// shu endpoint chaqirilsa ma'lumot darhol to'g'rilanadi. Eski oyni qayta hisoblash
// uchun ham ishlatiladi — masalan davomat qo'lda tuzatilgandan keyin.
//
// Xavfsiz: hisob har safar oy boshidan qayta quriladi, shuning uchun necha marta
// chaqirilishidan qat'i nazar natija bir xil (idempotent) va o'tkazib yuborilgan
// kunlarni alohida "o'ynatib chiqish" shart emas.
//
// year/month bo'sh bo'lsa joriy oy; dateStr bo'sh bo'lsa cron'dagi kabi kechagi kun.
func (s *Services) RecalculatePayroll(
	ctx context.Context, year, month int, dateStr string,
) (*domain.PayrollRecalcResult, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, domain.BadRequestError
	}

	calcDate, err := resolveCalcDate(period.Year, period.Month, dateStr)
	if err != nil {
		s.log.Errorf("payroll: invalid calc date: %v", err)
		return nil, domain.BadRequestError
	}

	startedAt := time.Now()
	rows, err := s.recalculatePayrollMonth(ctx, period.Year, period.Month, calcDate)
	if err != nil {
		s.log.Errorf("payroll: manual recalculation failed: %v", err)
		return nil, domain.InternalServerError
	}

	s.log.Infof("payroll: manual recalculation %d-%02d through %s: %d rows",
		period.Year, period.Month, calcDate.Format(constants.TimeOnlyDateFormat), rows)

	return &domain.PayrollRecalcResult{
		Year:         period.Year,
		Month:        period.Month,
		CalculatedTo: calcDate.Format(constants.TimeOnlyDateFormat),
		RowsAffected: rows,
		DurationMs:   time.Since(startedAt).Milliseconds(),
	}, nil
}

// recalculatePayrollMonth — bitta oyni oy boshidan calcDate'gacha qayta hisoblab
// employee_payrolls'ga UPSERT qiladi. Ta'sirlangan qatorlar sonini qaytaradi.
func (s *Services) recalculatePayrollMonth(
	ctx context.Context, year, month int, calcDate time.Time,
) (int64, error) {
	tx := s.db.WithContext(ctx).Exec(payrollUpsertQuery, map[string]any{
		"year":      year,
		"month":     month,
		"calc_date": calcDate.Format(constants.TimeOnlyDateFormat),
		"status":    constants.GeneralStatusActive,
		"draft":     domain.EmployeePayrollStatusDraft,
	})
	if tx.Error != nil {
		return 0, fmt.Errorf("upsert payrolls: %w", tx.Error)
	}
	return tx.RowsAffected, nil
}

// resolveCalcDate — hisob qaysi kungacha olib borilishini aniqlaydi.
//
// dateStr berilgan bo'lsa u tekshiriladi (shu oy ichida va kelajakda emas),
// bo'lmasa cron bilan bir xil qoida: kechagi kun, lekin oy chegarasidan chiqmagan
// holda. Oyning 1-kunida joriy oy uchun hali tugagan kun yo'q — oy boshi olinadi
// va barcha summalar 0 bo'ladi.
func resolveCalcDate(year, month int, dateStr string) (time.Time, error) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	if dateStr != "" {
		date, err := time.Parse(constants.TimeOnlyDateFormat, dateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("date must be YYYY-MM-DD: %w", err)
		}
		if date.Before(monthStart) || date.After(monthEnd) {
			return time.Time{}, fmt.Errorf("date %s is outside %d-%02d", dateStr, year, month)
		}
		if date.After(tashkentNow()) {
			return time.Time{}, fmt.Errorf("date %s is in the future", dateStr)
		}
		return date, nil
	}

	yesterday := tashkentNow().AddDate(0, 0, -1)
	switch {
	case yesterday.Before(monthStart):
		return monthStart, nil
	case yesterday.After(monthEnd):
		return monthEnd, nil
	default:
		return yesterday, nil
	}
}

// region Period

// tashkentNow — Toshkent devor-vaqti. Loyihada sana chegaralari hamma joyda
// UTC+5 bo'yicha olinadi (qarang: attendance service, report handler).
func tashkentNow() time.Time {
	return time.Now().UTC().Add(domain.TashkentTimeDif)
}

// resolvePayrollPeriod — so'ralgan yil/oyni tekshiradi. year yoki month 0 bo'lsa
// joriy yil/oy olinadi.
//
// IsLive faqat ma'lumot uchun: hisob doim employee_payrolls'dan o'qiladi, lekin
// joriy oy qatorlari har kecha o'zgarib turadi, o'tgan oylar esa yakuniy.
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

// region Helpers

// selectPayrolls — employee_payrolls'dan bir sahifa o'qiydi.
func (s *Services) selectPayrolls(
	ctx context.Context, period domain.PayrollPeriod, filter payrollReadFilter,
) ([]employeePayrollPageRow, error) {
	var page []employeePayrollPageRow

	err := s.db.WithContext(ctx).Raw(employeePayrollsSelectQuery, map[string]any{
		"year":        period.Year,
		"month":       period.Month,
		"employee_id": nullIfEmpty(filter.EmployeeId),
		"store_id":    nullIfEmpty(filter.StoreId),
		"company_id":  nullIfEmpty(filter.CompanyId),
		// Bo'sh massiv NULL bo'lib ketadi → o'sha shart tekshirilmaydi.
		"roles":  pq.StringArray(filter.Roles),
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}).Scan(&page).Error
	if err != nil {
		s.log.Errorf("payroll: could not read payrolls: %v", err)
		return nil, domain.InternalServerError
	}

	return page, nil
}

func payrollRowsOf(page []employeePayrollPageRow) []domain.EmployeePayrollRow {
	rows := make([]domain.EmployeePayrollRow, len(page))
	for i := range page {
		rows[i] = page[i].EmployeePayrollRow
	}
	return rows
}

// paginateStores — filtrga mos do'konlarning bir sahifasini va umumiy sonini qaytaradi.
func (s *Services) paginateStores(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]storeRef, int64, error) {
	// Count va Find uchun so'rov qaytadan quriladi: GORM'da finisher (Count)
	// chaqirilgandan keyin o'sha *gorm.DB'ni qayta ishlatish statement'ni ifloslantiradi.
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
		Select(`
			stores.id,
			stores.name,
			(SELECT COUNT(*)
			   FROM employees e
			  WHERE e.store_id = stores.id
			    AND e.is_active = TRUE
			    AND e.status = ?
			    AND e.deleted_at IS NULL) AS employee_count`,
			constants.GeneralStatusActive).
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

func storeIdsOf(stores []storeRef) []string {
	ids := make([]string, 0, len(stores))
	for _, store := range stores {
		ids = append(ids, store.Id)
	}
	return ids
}

// mergeStoreTotals — do'kon ro'yxati tartibini saqlab, yig'indilarni ularga
// biriktiradi. Payroll qatori yo'q do'kon ham javobda qoladi (nol summalar
// bilan), aks holda sahifadagi do'konlar soni _meta bilan mos kelmasdi.
func mergeStoreTotals(stores []storeRef, totals []domain.StorePayroll) []domain.StorePayroll {
	byStore := make(map[string]domain.StorePayroll, len(totals))
	for _, total := range totals {
		byStore[total.StoreId] = total
	}

	result := make([]domain.StorePayroll, 0, len(stores))
	for _, store := range stores {
		total, ok := byStore[store.Id]
		if !ok {
			total = domain.StorePayroll{StoreId: store.Id}
		}
		total.StoreName = store.Name
		// Xodimlar soni employees jadvalidan keladi, payroll yig'indisidan emas:
		// cron hali ishlamagan do'konda ham to'g'ri son ko'rinadi.
		total.EmployeeCount = store.EmployeeCount
		result = append(result, total)
	}
	return result
}

// region SQL

// payrollUpsertQuery — bitta oyni oy boshidan @calc_date'gacha hisoblab
// employee_payrolls'ga yozadi.
//
// Bosqichlar:
//
//	workdays — oydagi va o'tgan ish kunlari (yakshanba va holidays chiqarilgan)
//	base     — xodim ma'lumotlari + davomat, savdo, bonus, rejalar
//	calc     — actual_salary va kesilgan reja (expected_plan)
//	kpi      — bajarilish foizi, keyin undan KPI pog'onasi
//	INSERT   — UPSERT: qator yo'q bo'lsa yaratadi, bor bo'lsa yangilaydi
//
// ON CONFLICT'da avans/ushlab qolish, status, approved_by va completed_at
// YANGILANMAYDI — ular qo'lda kiritiladi. net_pay_amount esa mavjud qatordagi
// avans/ushlab qolishlar hisobga olingan holda qayta hisoblanadi.
const payrollUpsertQuery = `
WITH bounds AS (
    SELECT
        make_date(@year, @month, 1)                                      AS month_start,
        (make_date(@year, @month, 1) + INTERVAL '1 month - 1 day')::date AS month_end,
        CAST(@calc_date AS date)                                         AS calc_date
),
workdays AS (
    SELECT
        b.month_start,
        b.calc_date,
        -- Toshkent kun chegaralarining UTC timestamp ko'rinishi. employee_bonus'da
        -- created_at UTC'da saqlanadi; ustunning o'ziga funksiya qo'llash
        -- ((created_at + interval '5 hours')::date) indeksdan foydalanishga yo'l
        -- qo'ymaydi, shuning uchun chegara qiymatlari suriladi.
        b.month_start::timestamp - INTERVAL '5 hours'     AS from_ts,
        (b.calc_date + 1)::timestamp - INTERVAL '5 hours' AS to_ts,
        (SELECT COUNT(*)::int
           FROM generate_series(b.month_start, b.month_end, INTERVAL '1 day') d
          WHERE EXTRACT(DOW FROM d) <> 0
            AND NOT EXISTS (SELECT 1 FROM holidays h WHERE h.date = d::date)) AS month_work_days,
        (SELECT COUNT(*)::int
           FROM generate_series(b.month_start, b.calc_date, INTERVAL '1 day') d
          WHERE EXTRACT(DOW FROM d) <> 0
            AND NOT EXISTS (SELECT 1 FROM holidays h WHERE h.date = d::date)) AS elapsed_work_days
    FROM bounds b
),
-- Yig'indilar barcha xodimlar uchun BIR MARTA hisoblanadi.
-- Ilgari bular LATERAL orqali har bir xodimga alohida bajarilardi — xodimlar
-- soniga ko'paytirilgan skan bo'lib, so'rov daqiqalab cho'zilardi.
attendance_agg AS (
    SELECT d.employee_id,
           SUM(d.worked_minutes)::numeric / 60.0 AS worked_hours,
           SUM(d.sales_amount)                   AS individual_sales_amount
    FROM employee_attendance_days d, workdays w
    WHERE d.employee_id IS NOT NULL
      AND d.work_date BETWEEN w.month_start AND w.calc_date
    GROUP BY d.employee_id
),
bonus_agg AS (
    SELECT b2.employee_id, SUM(b2.bonus_amount) AS bonus_amount
    FROM employee_bonus b2, workdays w
    WHERE b2.employee_id IS NOT NULL
      AND b2.deleted_at IS NULL
      AND b2.created_at >= w.from_ts
      AND b2.created_at <  w.to_ts
    GROUP BY b2.employee_id
),
roles_agg AS (
    SELECT er.employee_id,
           string_agg(r.name, ', ' ORDER BY r.name) AS role,
           array_agg(r.name ORDER BY r.name)        AS role_names
    FROM employee_roles er
    JOIN roles r ON r.id = er.role_id
    GROUP BY er.employee_id
),
base AS (
    SELECT
        e.id                                    AS employee_id,
        e.company_id                            AS company_id,
        e.store_id                              AS store_id,
        COALESCE(s.name, '')                    AS store_name,
        COALESCE(e.first_name, '')              AS first_name,
        COALESCE(e.last_name, '')               AS last_name,
        COALESCE(e.full_name, '')               AS full_name,
        COALESCE(e.position, '')                AS position_snapshot,
        COALESCE(rl.role, '')                   AS role,
        COALESCE(rl.role_names, '{}')           AS role_names,
        e.experience_years                      AS experience_years,
        e.avg_monthly_hours                     AS avg_monthly_hours,
        e.salary                                AS salary_rate_amount,
        COALESCE(att.worked_hours, 0)            AS worked_hours,
        COALESCE(att.individual_sales_amount, 0) AS individual_sales_amount,
        COALESCE(bon.bonus_amount, 0)            AS bonus_amount,
        st.id                                   AS store_target_id,
        COALESCE(st.amount, 0)                  AS store_plan_amount,
        COALESCE(et.amount, 0)                  AS employee_plan_amount,
        w.month_work_days,
        w.elapsed_work_days
    FROM employees e
    CROSS JOIN workdays w
    LEFT JOIN stores s          ON s.id = e.store_id
    LEFT JOIN roles_agg rl      ON rl.employee_id = e.id
    LEFT JOIN attendance_agg att ON att.employee_id = e.id
    LEFT JOIN bonus_agg bon     ON bon.employee_id = e.id
    LEFT JOIN store_targets st
           ON st.store_id = e.store_id AND st.year = @year AND st.month = @month
    LEFT JOIN employee_targets et
           ON et.employee_id = e.id AND et.year = @year AND et.month = @month
    WHERE e.is_active
      AND e.status = @status
      AND e.deleted_at IS NULL
),
calc AS (
    SELECT b.*,
           -- avg_monthly_hours = 0 bo'lsa nolga bo'lish o'rniga 0 qaytadi
           COALESCE(ROUND(b.salary_rate_amount * (b.worked_hours / NULLIF(b.avg_monthly_hours, 0)), 2), 0) AS actual_salary_amount,
           -- reja o'tgan ish kunlariga proporsional kesiladi
           COALESCE(ROUND(b.employee_plan_amount * b.elapsed_work_days / NULLIF(b.month_work_days, 0), 2), 0) AS expected_plan_amount
    FROM base b
),
kpi AS (
    SELECT c.*,
           -- reja 0 bo'lsa (oy boshi yoki target yo'q) foiz ham 0
           COALESCE(ROUND(c.individual_sales_amount / NULLIF(c.expected_plan_amount, 0) * 100, 2), 0) AS plan_achievement_percent
    FROM calc c
),
final AS (
    SELECT k.*,
           CAST(CASE
               WHEN k.plan_achievement_percent >= 100 THEN 1.4
               WHEN k.plan_achievement_percent >= 90  THEN 1.0
               WHEN k.plan_achievement_percent >= 80  THEN 0.8
               ELSE 0
           END AS numeric) AS kpi_percent
    FROM kpi k
)
INSERT INTO employee_payrolls (
    id, employee_id, company_id, store_id, store_name,
    first_name, last_name, full_name, position_snapshot, role, role_names,
    experience_years, worked_hours, avg_monthly_hours,
    salary_rate_amount, actual_salary_amount, individual_sales_amount,
    store_target_id, store_plan_amount,
    employee_plan_amount, expected_plan_amount, plan_achievement_percent,
    month_work_days, elapsed_work_days,
    kpi_percent, kpi_amount, bonus_amount,
    gross_salary_amount, net_pay_amount,
    status, year, month, calculated_at
)
SELECT
    uuid_generate_v4(), f.employee_id, f.company_id, f.store_id, f.store_name,
    f.first_name, f.last_name, f.full_name, f.position_snapshot, f.role, f.role_names,
    f.experience_years, f.worked_hours, f.avg_monthly_hours,
    f.salary_rate_amount, f.actual_salary_amount, f.individual_sales_amount,
    f.store_target_id, f.store_plan_amount,
    f.employee_plan_amount, f.expected_plan_amount, f.plan_achievement_percent,
    f.month_work_days, f.elapsed_work_days,
    f.kpi_percent,
    ROUND(f.individual_sales_amount * f.kpi_percent / 100.0, 2),
    f.bonus_amount,
    f.actual_salary_amount + ROUND(f.individual_sales_amount * f.kpi_percent / 100.0, 2) + f.bonus_amount,
    f.actual_salary_amount + ROUND(f.individual_sales_amount * f.kpi_percent / 100.0, 2) + f.bonus_amount,
    @draft, @year, @month, NOW()
FROM final f
ON CONFLICT (employee_id, year, month) DO UPDATE SET
    company_id               = EXCLUDED.company_id,
    store_id                 = EXCLUDED.store_id,
    store_name               = EXCLUDED.store_name,
    first_name               = EXCLUDED.first_name,
    last_name                = EXCLUDED.last_name,
    full_name                = EXCLUDED.full_name,
    position_snapshot        = EXCLUDED.position_snapshot,
    role                     = EXCLUDED.role,
    role_names               = EXCLUDED.role_names,
    experience_years         = EXCLUDED.experience_years,
    worked_hours             = EXCLUDED.worked_hours,
    avg_monthly_hours        = EXCLUDED.avg_monthly_hours,
    salary_rate_amount       = EXCLUDED.salary_rate_amount,
    actual_salary_amount     = EXCLUDED.actual_salary_amount,
    individual_sales_amount  = EXCLUDED.individual_sales_amount,
    store_target_id          = EXCLUDED.store_target_id,
    store_plan_amount        = EXCLUDED.store_plan_amount,
    employee_plan_amount     = EXCLUDED.employee_plan_amount,
    expected_plan_amount     = EXCLUDED.expected_plan_amount,
    plan_achievement_percent = EXCLUDED.plan_achievement_percent,
    month_work_days          = EXCLUDED.month_work_days,
    elapsed_work_days        = EXCLUDED.elapsed_work_days,
    kpi_percent              = EXCLUDED.kpi_percent,
    kpi_amount               = EXCLUDED.kpi_amount,
    bonus_amount             = EXCLUDED.bonus_amount,
    gross_salary_amount      = EXCLUDED.gross_salary_amount,
    -- Avans va ushlab qolishlar qo'lda kiritilgan: EXCLUDED emas, MAVJUD qatordan olinadi
    net_pay_amount           = EXCLUDED.gross_salary_amount
                               - (employee_payrolls.advance_card_amount
                                + employee_payrolls.advance_cash_amount
                                + employee_payrolls.deduction_term_amount
                                + employee_payrolls.deduction_recount_amount
                                + employee_payrolls.deduction_fine_amount),
    calculated_at            = NOW(),
    updated_at               = NOW()`

// employeePayrollsSelectQuery — hisobotni employee_payrolls'dan o'qiydi.
// JOIN yo'q: kerakli hamma narsa qatorning o'zida snapshot qilingan.
// Filtrlar ixtiyoriy — NULL berilsa o'sha shart tekshirilmaydi.
const employeePayrollsSelectQuery = `
SELECT
    p.employee_id, p.store_id, p.store_name,
    p.first_name, p.last_name, p.full_name, p.position_snapshot, p.role,
    p.experience_years, p.worked_hours, p.avg_monthly_hours,
    p.salary_rate_amount, p.actual_salary_amount, p.individual_sales_amount,
    p.store_target_id, p.store_plan_amount,
    p.employee_plan_amount, p.expected_plan_amount, p.plan_achievement_percent,
    p.month_work_days, p.elapsed_work_days,
    p.kpi_percent, p.kpi_amount, p.bonus_amount, p.gross_salary_amount,
    p.advance_card_amount, p.advance_cash_amount,
    p.deduction_term_amount, p.deduction_recount_amount, p.deduction_fine_amount,
    p.net_pay_amount, p.status, p.year, p.month, p.completed_at, p.calculated_at,
    COUNT(*) OVER () AS total_count
FROM employee_payrolls p
WHERE p.year = @year
  AND p.month = @month
  AND (CAST(@employee_id AS uuid) IS NULL OR p.employee_id = CAST(@employee_id AS uuid))
  AND (CAST(@store_id AS uuid)    IS NULL OR p.store_id    = CAST(@store_id AS uuid))
  AND (CAST(@company_id AS uuid)  IS NULL OR p.company_id  = CAST(@company_id AS uuid))
  AND (CAST(@roles AS text[])     IS NULL OR p.role_names && CAST(@roles AS text[]))
ORDER BY p.store_name, p.full_name
LIMIT NULLIF(@limit, 0) OFFSET @offset`

// storePayrollTotalsQuery — do'kon kesimidagi yig'indilar. store_plan_amount
// har bir xodim qatorida takrorlanadi (bitta store_target'dan keladi), shuning
// uchun SUM emas, MAX olinadi.
const storePayrollTotalsQuery = `
SELECT
    p.store_id                     AS store_id,
    -- payroll qatorlari soni; do'kondagi xodimlar soni employees jadvalidan
    -- alohida olinadi (paginateStores), chunki cron hali qamramagan xodim
    -- bu yerda uchramaydi
    COUNT(*)::int                  AS payroll_count,
    SUM(p.worked_hours)            AS worked_hours,
    SUM(p.salary_rate_amount)      AS salary_rate_amount,
    SUM(p.actual_salary_amount)    AS actual_salary_amount,
    SUM(p.individual_sales_amount) AS individual_sales_amount,
    MAX(p.store_plan_amount)       AS store_plan_amount,
    SUM(p.kpi_amount)              AS kpi_amount,
    SUM(p.bonus_amount)            AS bonus_amount,
    SUM(p.gross_salary_amount)     AS gross_salary_amount,
    SUM(p.net_pay_amount)          AS net_pay_amount
FROM employee_payrolls p
WHERE p.year = @year
  AND p.month = @month
  AND p.store_id = ANY(CAST(@store_ids AS uuid[]))
GROUP BY p.store_id`
