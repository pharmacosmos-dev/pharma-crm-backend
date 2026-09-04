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
// KPI DO'KON BO'YICHA va progressiv — har kuni o'zgaradi:
//
//	expected_plan = store_targets.amount * (o'tgan kun / oydagi kun)
//	achievement   = store_sales / expected_plan * 100
//	kpi_percent   = 0%    (achievement < 80)
//	                0.8%  (80 <= achievement < 90)
//	                1.0%  (90 <= achievement < 100)
//	                1.4%  (achievement >= 100)
//	kpi_amount    = KPI bazasi * kpi_percent / 100
//
// expected_plan KALENDAR kunlari bo'yicha kesiladi (oy 31 kun, kecha 30-sana →
// reja * 30/31): yakshanba va bayramlar chegirilmaydi, chunki do'kon o'sha
// kunlari ham savdo qiladi. Ish kunlari sanog'i (month_work_days) faqat
// avg_monthly_hours, ya'ni oylik norma soati uchun ishlatiladi.
//
// Reja bajarilishi DO'KON bo'yicha o'lchanadi — bitta do'konning barcha
// xodimlarida achievement bir xil. Foiz va baza esa xodimga qarab aniqlanadi:
//
//	FOIZ (kpi_percent):
//	  employees.kpi_percent > 0 → o'sha ishlatiladi (qo'lda kelishilgan foiz;
//	                              do'kon rejani bajarmasa ham amal qiladi)
//	  employees.kpi_percent = 0 → yuqoridagi reja pog'onasi
//
//	BAZA:
//	  roli "Заведующий"  → store_sales_amount (do'kon aylanmasi)
//	  qolganlar          → individual_sales_amount (o'z savdosi)
//	  ikkala rol ham bo'lsa — "Заведующий" ustun turadi
//
// employee_plan_amount (employee_targets) hisobga kirmaydi — u faqat ko'rsatish
// uchun saqlanadi.
//
// store_sales `sales` jadvalidan olinadi (stage = 9, sale_type = 'SALE',
// qaytarilmagan), oy boshidan calc_date'gacha — ya'ni do'konning haqiqiy
// aylanmasi, xodimlarga biriktirilgan-biriktirilmaganidan qat'i nazar.
//
// Qolgan formulalar:
//
//	worked_hours         = SUM(employee_attendance_days.worked_minutes) / 60
//	individual_sales     = SUM(sales.total_amount) WHERE employee_id = xodim
//	bonus_amount         = SUM(employee_bonus.bonus_amount)
//
// Savdo — do'konniki ham, xodimniki ham — bitta manbadan, `sales` jadvalidan
// olinadi. Ilgari xodim savdosi employee_attendance_days.sales_amount'dan
// olinardi; u faqat davomat qatori bor kunlarni qamrardi va qaytarilgan
// sotuvlarni chiqarib tashlamasdi. Endi ikkala summa ham bir xil filtrdan
// o'tadi, shuning uchun ular bir-biriga mos keladi.
//
// `sales` jadvali so'rovda BIR MARTA o'qiladi: sales_agg uni (store_id,
// employee_id) kesimida yig'adi, do'kon va xodim summalari o'shandan chiqadi.
//	avg_monthly_hours    = oydagi ish kuni * kunlik smena
//	actual_salary_amount = salary_rate * (worked_hours / avg_monthly_hours)
//	gross_salary_amount  = actual_salary + kpi_amount + bonus_amount
//	net_pay_amount       = gross − (avanslar + ushlab qolishlar)
//
// Kunlik smena employees.daily_work_hours'dan olinadi (4/7/8). Kiritilmagan bo'lsa
// (0) payrollWorkDayHours default'i ishlaydi — shunda yarim stavkada ishlaydigan
// xodimning normasi kichik bo'lib, to'liq ishlagani uchun butun stavkasini oladi.
//
// Ish kuni soni esa kalendardan olinadi, ya'ni fevral va mart uchun norma har xil.
//
// Avans va ushlab qolishlar qo'lda kiritiladi — cron ularga TEGMAYDI, faqat
// net_pay_amount'ni ular hisobga olingan holda qayta hisoblaydi.

// region Types

// payrollWorkDayHours — kunlik smenaning DEFAULT uzunligi. Xodimning
// employees.daily_work_hours'i 0 bo'lganda (kiritilmagan) shu ishlatiladi:
// avg_monthly_hours = oydagi ish kuni * COALESCE(daily_work_hours, 8).
const payrollWorkDayHours = 8

// storeRef — sahifalangan do'kon ro'yxati uchun minimal ma'lumot.
//
// Ikkala son ham employees/stores jadvallaridan keladi, payroll yig'indisidan
// emas — shuning uchun cron hali ishlamagan do'konda ham to'g'ri ko'rinadi.
type storeRef struct {
	Id                       string  `gorm:"column:id"`
	Name                     string  `gorm:"column:name"`
	EmployeeCount            float64 `gorm:"column:store_employee_count"`
	ActiveStoreEmployeeCount int     `gorm:"column:active_store_employee_count"`
	IsFranchise              bool    `gorm:"column:is_franchise"`
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
	CompanyId  string
	StoreId    string
	Search     string
	Roles      []string
	// OnlyWorked — faqat davomati bor xodimlar (worked_hours > 0). Cron barcha
	// faol xodimlarga qator yozadi, shu jumladan oy davomida umuman ishlamaganlarga
	// ham — ro'yxatda ular nol qatorlar bo'lib turmasligi uchun.
	OnlyWorked bool
	Limit      int // 0 = cheklovsiz
	Offset     int
}

// payrollSalesRoles — savdo nuqtasida ishlaydigan rollar. Xodimlar hisoboti
// faqat shularni ko'rsatadi; o'z oyligini ko'rishda rol tekshirilmaydi, aks
// holda menejer o'z oyligini ko'ra olmasdi.
var payrollSalesRoles = []string{constants.RoleNameCashier, constants.RoleNameZavStore}

// payrollNoLimit — 0 yoki manfiy limitni GORM'ning "cheklovsiz" qiymatiga (-1)
// aylantiradi. Limit(0) "LIMIT 0" bo'lib hech narsa qaytarmaydi.
func payrollNoLimit(limit int) int {
	if limit <= 0 {
		return -1
	}
	return limit
}

// nullIfEmpty — bo'sh satrni SQL NULL'ga aylantiradi: "filtr berilmagan" degani.
func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func payrollSearchPattern(value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%%%s%%", value)
}

// region Get

// GetEmployeePayrolls — xodimlar hisoboti. employee_payrolls'dan to'g'ridan-to'g'ri
// o'qiydi: JOIN ham, hisob-kitob ham yo'q — hammasi cron tomonidan oldindan yozilgan.
//
// Ro'yxatga faqat roli "Кассир"/"Заведующий" va davomati bor (worked_hours > 0)
// xodimlar kiradi. Cron barcha faol xodimlarga qator yozadi, shu jumladan oy
// davomida bironta smenaga chiqmaganlarga ham — ular hisobotni nol qatorlar
// bilan to'ldirib yubormasligi uchun bu yerda chiqarib tashlanadi.
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
		CompanyId:  params.CompanyId,
		StoreId:    params.StoreId,
		Search:     params.Search,
		Roles:      payrollSalesRoles,
		OnlyWorked: true,
		Limit:      params.Limit,
		Offset:     params.Offset,
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
// Pagination do'konlarga qo'yiladi. Yig'indi GetEmployeePayrolls bilan bir xil
// doiradan chiqadi — faqat roli "Кассир"/"Заведующий" va worked_hours > 0
// bo'lgan qatorlar — shuning uchun do'kon summasi o'sha do'konning xodimlar
// ro'yxatini qo'lda qo'shib chiqqandagi natijaga teng bo'ladi.
//
// Xodimlar soni esa bundan mustasno: employee_count va
// active_store_employee_count stores/employees jadvallaridan olinadi, ya'ni
// cron hali qamramagan xodimlarni ham sanaydi.
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
		"roles":     pq.StringArray(payrollSalesRoles),
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
		"year":           year,
		"month":          month,
		"calc_date":      calcDate.Format(constants.TimeOnlyDateFormat),
		"status":         constants.GeneralStatusActive,
		"draft":          domain.EmployeePayrollStatusDraft,
		"work_day_hours": payrollWorkDayHours,
		"zav_role":       constants.RoleNameZavStore,
	})
	if tx.Error != nil {
		return 0, fmt.Errorf("upsert payrolls: %w", tx.Error)
	}
	return tx.RowsAffected, nil
}

// UpdateEmployeePayrollAdvance — payroll qatoridagi qo'lda kiritiladigan
// maydonlarni yangilaydi: kpi_percent, salary va avanslar.
//
// Ikkita jadval tegiladi, ikkalasi ham BITTA transaksiyada:
//
//	employee_payrolls — kpi_percent, salary va avanslardan berilgani, ustiga
//	                    qayta hisoblangan summalar
//	employees         — kpi_percent / salary / daily_work_hours berilgan bo'lsa
//	                    (xodim kartochkasi, keyingi oylarga ham ta'sir qiladi)
//
// daily_work_hours FAQAT employees'ga yoziladi: payroll qatorida bunday ustun
// yo'q va oylik hisobiga ham kirmaydi.
//
// Summalar cron formulasi bilan AYNAN bir xil zanjir bo'yicha qayta hisoblanadi
// (qarang: payrollUpsertQuery'dagi calc / kpi_rate / final CTE'lari):
//
//	actual_salary = ROUND(salary * worked_hours / avg_monthly_hours, 2)
//	kpi_base      = zav bo'lsa store_sales, aks holda individual_sales
//	kpi_amount    = ROUND(kpi_base * kpi_percent / 100, 2)
//	gross         = actual_salary + kpi_amount + bonus
//	net           = gross − (avanslar + ushlab qolishlar)
//
// Kerakli hamma qiymat payroll qatorining o'zida snapshot qilingan, shuning
// uchun boshqa jadvaldan o'qilmaydi.
//
// NULL kelgan maydon o'zgarmaydi: COALESCE mavjud qiymatga tushadi.
func (s *Services) UpdateEmployeePayrollAdvance(
	ctx context.Context, id, updatedBy string, req *domain.EmployeePayrollAdvanceRequest,
) (*domain.EmployeePayroll, error) {
	// Yangi qiymatlar ichki SELECT'da BIR MARTA hisoblanadi, keyin SET ularga
	// murojaat qiladi — shunda uzun ifodalar takrorlanmaydi.
	const payrollQuery = `
		UPDATE employee_payrolls p
		SET employee_kpi_percent = n.kpi_percent,
			kpi_percent          = n.kpi_percent,
			salary_rate_amount   = n.salary,
			avg_monthly_hours    = n.avg_monthly_hours,
			actual_salary_amount = n.actual_salary,
			kpi_amount           = n.kpi_amount,
			gross_salary_amount  = n.gross,
			advance_card_amount  = n.card,
			advance_cash_amount  = n.cash,
			net_pay_amount       = n.gross - (n.card + n.cash
									+ p.deduction_term_amount
									+ p.deduction_recount_amount
									+ p.deduction_fine_amount),
			updated_at           = NOW()
		FROM (
			SELECT
				y.*,
				y.actual_salary + y.kpi_amount + y.bonus_amount AS gross
			FROM (
				SELECT
					x.*,
					-- actual_salary yangi normadan hisoblanadi
					COALESCE(ROUND(x.salary * (x.worked_hours
						/ NULLIF(x.avg_monthly_hours, 0)), 2), 0) AS actual_salary
				FROM (
					SELECT
						r.id,
						r.bonus_amount,
						r.worked_hours,
						COALESCE(CAST(@card AS numeric),   r.advance_card_amount) AS card,
						COALESCE(CAST(@cash AS numeric),   r.advance_cash_amount) AS cash,
						COALESCE(CAST(@kpi AS numeric),    r.kpi_percent)         AS kpi_percent,
						COALESCE(CAST(@salary AS numeric), r.salary_rate_amount)  AS salary,
						-- Kunlik soat berilgan bo'lsa oylik norma qaytadan chiqadi
						-- (ish kuni soni qatorda saqlangan), aks holda o'zgarmaydi.
						CASE WHEN CAST(@daily_hours AS numeric) IS NOT NULL
							 THEN r.month_work_days * CAST(@daily_hours AS numeric)
							 ELSE r.avg_monthly_hours
						END AS avg_monthly_hours,
						ROUND(CASE WHEN CAST(@zav_role AS text) = ANY(r.role_names)
								THEN r.store_sales_amount
								ELSE r.individual_sales_amount
							END * COALESCE(CAST(@kpi AS numeric), r.kpi_percent) / 100.0, 2) AS kpi_amount
					FROM employee_payrolls r
					WHERE r.id = CAST(@id AS uuid)
				) x
			) y
		) n
		WHERE p.id = n.id
		RETURNING p.*`

	const employeeQuery = `
		UPDATE employees
		SET kpi_percent      = COALESCE(CAST(@kpi AS numeric), kpi_percent),
			salary           = COALESCE(CAST(@salary AS numeric), salary),
			daily_work_hours = COALESCE(CAST(@daily_hours AS numeric), daily_work_hours),
			shift_type       = COALESCE(CAST(@shift_type AS varchar), shift_type),
			role_type        = COALESCE(CAST(@role_type AS varchar), role_type),
			updated_by       = CAST(@updated_by AS uuid),
			updated_at       = NOW()
		WHERE id = CAST(@employee_id AS uuid)`

	var res domain.EmployeePayroll

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Raw(payrollQuery, map[string]any{
			"id":          id,
			"card":        req.AdvanceCardAmount,
			"cash":        req.AdvanceCashAmount,
			"kpi":         req.KpiPercent,
			"salary":      req.Salary,
			"daily_hours": req.DailyWorkHours,
			"zav_role":    constants.RoleNameZavStore,
		}).Scan(&res)
		if result.Error != nil {
			s.log.Errorf("payroll: could not update payroll row: %v", result.Error)
			return domain.InternalServerError
		}
		if result.RowsAffected == 0 {
			return domain.NotFoundError
		}

		// Avanslar faqat shu oyga tegishli — xodim kartochkasiga tegmaymiz.
		if !req.TouchesEmployee() {
			return nil
		}

		if err := tx.Exec(employeeQuery, map[string]any{
			"employee_id": res.EmployeeId,
			"kpi":         req.KpiPercent,
			"salary":      req.Salary,
			"daily_hours": req.DailyWorkHours,
			"shift_type":  req.ShiftType,
			"role_type":   req.RoleType,
			"updated_by":  nullIfEmpty(updatedBy),
		}).Error; err != nil {
			s.log.Errorf("payroll: could not update employee card: %v", err)
			return domain.InternalServerError
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// payrollManagementFilterSQL — tahrirlash ro'yxatining WHERE bo'lagi.
//
// Ro'yxat ham, statistika ham AYNAN shu shartdan foydalanadi: aks holda ekrandagi
// qatorlar bilan yuqoridagi yig'ma raqamlar bir-biriga mos kelmay qolardi.
const payrollManagementFilterSQL = `
	WHERE p.year = @year
	  AND p.month = @month
	  AND (CAST(@store_id AS uuid)    IS NULL OR p.store_id    = CAST(@store_id AS uuid))
	  AND (CAST(@employee_id AS uuid) IS NULL OR p.employee_id = CAST(@employee_id AS uuid))
	  AND (CAST(@company_id AS uuid)  IS NULL OR p.company_id  = CAST(@company_id AS uuid))
	  -- role_type va shift_type payroll qatorida saqlanmaydi, xodim kartochkasidan
	  -- olinadi: ular tahrirlanganda filtr darhol yangi qiymatga qaraydi.
	  AND (CAST(@role_type AS text)   IS NULL OR e.role_type   = CAST(@role_type AS text))
	  AND (CAST(@shift_type AS text)  IS NULL OR e.shift_type  = CAST(@shift_type AS text))
	  AND (CAST(@search AS text)      IS NULL OR p.full_name ILIKE CAST(@search AS text)
											  OR e.phone     ILIKE CAST(@search AS text))
	  AND p.role_names && CAST(@roles AS text[])
	  AND COALESCE(e.status, '') <> CAST(@dismissed AS text)
	  AND e.deleted_at IS NULL
	  AND EXISTS (
		  SELECT 1
		  FROM stores st
		  JOIN companies co ON co.id = st.company_id
		  WHERE st.id = p.store_id
		    AND st.deleted_at IS NULL
		    AND st.is_active = TRUE
		    AND co.is_franchise = FALSE
	  )`

// payrollManagementArgs — ro'yxat va statistika uchun umumiy parametrlar.
// Limit/offset bu yerda yo'q: ular faqat ro'yxatga tegishli.
func payrollManagementArgs(
	period domain.PayrollPeriod, params *domain.EmployeePayrollAdvanceQueryParams,
) map[string]any {
	search := ""
	if params.Search != "" {
		search = fmt.Sprintf("%%%s%%", params.Search)
	}
	return map[string]any{
		"year":        period.Year,
		"month":       period.Month,
		"store_id":    nullIfEmpty(params.StoreId),
		"employee_id": nullIfEmpty(params.EmployeeId),
		"company_id":  nullIfEmpty(params.CompanyId),
		"role_type":   nullIfEmpty(params.RoleType),
		"shift_type":  nullIfEmpty(params.ShiftType),
		"search":      nullIfEmpty(search),
		"roles":       pq.StringArray(payrollSalesRoles),
		"dismissed":   constants.GeneralStatusDismissed,
	}
}

// GetPayrollManagementStatistics — tahrirlash ro'yxatining yig'ma ko'rsatkichlari.
//
// Sahifalanmaydi: filtrga tushgan BARCHA xodimlar bo'yicha hisoblanadi, shuning
// uchun limit/offset o'zgarganda raqamlar o'zgarmaydi.
func (s *Services) GetPayrollManagementStatistics(
	ctx context.Context, params *domain.EmployeePayrollAdvanceQueryParams,
) (*domain.PayrollManagementStatistics, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, period, domain.BadRequestError
	}

	const query = `
		SELECT
			(SELECT COUNT(*)
			   FROM stores st2
			   JOIN companies co2 ON co2.id = st2.company_id
			  WHERE st2.deleted_at IS NULL
			    AND st2.is_active = TRUE
			    AND co2.is_franchise = FALSE
			    AND (CAST(@store_id AS uuid)   IS NULL OR st2.id         = CAST(@store_id AS uuid))
			    AND (CAST(@company_id AS uuid) IS NULL OR st2.company_id = CAST(@company_id AS uuid))
			)::bigint                          AS total_stores,
			COUNT(*)::bigint                   AS total_employees,
			-- oylik fond stavkasi: xodim kartochkasidagi oklad, ishlagan soatdan mustaqil
			COALESCE(SUM(e.salary), 0)         AS total_salary,
			-- karta va naqd avanslar birga
			COALESCE(SUM(p.advance_card_amount + p.advance_cash_amount), 0) AS total_advance_amount
		FROM employee_payrolls p
		LEFT JOIN employees e ON e.id = p.employee_id` + payrollManagementFilterSQL

	// role_type bo'yicha sanoq alohida so'rov: asosiy so'rov bitta qator
	// qaytaradi, bu esa har bir rol uchun qator. Filtri aynan bir xil, shuning
	// uchun map qiymatlarining yig'indisi total_employees'ga teng bo'ladi.
	const roleTypeQuery = `
		SELECT COALESCE(e.role_type, '') AS role_type,
		       COUNT(*)::bigint          AS count
		FROM employee_payrolls p
		LEFT JOIN employees e ON e.id = p.employee_id` + payrollManagementFilterSQL + `
		GROUP BY COALESCE(e.role_type, '')`

	args := payrollManagementArgs(period, params)

	var stats domain.PayrollManagementStatistics
	if err := s.db.WithContext(ctx).Raw(query, args).Scan(&stats).Error; err != nil {
		s.log.Errorf("payroll: could not get management statistics: %v", err)
		return nil, period, domain.InternalServerError
	}

	var roleRows []struct {
		RoleType string `gorm:"column:role_type"`
		Count    int64  `gorm:"column:count"`
	}
	if err := s.db.WithContext(ctx).Raw(roleTypeQuery, args).Scan(&roleRows).Error; err != nil {
		s.log.Errorf("payroll: could not get role type counts: %v", err)
		return nil, period, domain.InternalServerError
	}

	// Bo'sh bo'lsa ham map yaratiladi: JSON'da null emas, {} chiqsin.
	stats.RoleTypeCounts = make(map[string]int64, len(roleRows))
	for _, r := range roleRows {
		stats.RoleTypeCounts[r.RoleType] = r.Count
	}

	return &stats, period, nil
}

// GetEmployeePayrollManagement — oylik tahrirlash ro'yxati: xodim kartochkasidagi
// qiymatlar (salary, daily_work_hours, shift_type, experience_years, phone,
// role_type) va so'ralgan oyning payroll qatoridan kpi_percent bilan avanslar.
//
// employee_payrolls asosiy jadval, employees unga LEFT JOIN qilinadi: shu sababli
// qaytgan har bir qatorda id bo'ladi va uni to'g'ridan-to'g'ri
// UpdateEmployeePayrollManagement'ga berish mumkin.
//
// Doirasi hisobotdan (GetEmployeePayrolls) KENGROQ: rol filtri bir xil
// ("Кассир"/"Заведующий"), lekin davomat sharti yo'q — oy davomida hali
// ishlamagan xodim ham ro'yxatda turadi, chunki unga avans yozish kerak bo'lishi
// mumkin. Shu sababli bu ro'yxat hisobotdagidan ko'proq qator qaytarishi normal.
func (s *Services) GetEmployeePayrollManagement(
	ctx context.Context, params *domain.EmployeePayrollAdvanceQueryParams,
) ([]domain.EmployeePayrollAdvanceRow, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, 0, period, domain.BadRequestError
	}

	const query = `
		SELECT
			p.id,
			p.employee_id,
			COALESCE(e.role_type, '')        AS role_type,
			COALESCE(e.first_name, '')       AS first_name,
			COALESCE(e.last_name, '')        AS last_name,
			COALESCE(e.phone, '')            AS phone,
			COALESCE(p.role_names, '{}')     AS roles,
			e.role_type,
			p.store_name,
			e.kpi_percent,
			COALESCE(e.salary, 0)            AS salary,
			COALESCE(e.daily_work_hours, 0)  AS daily_work_hours,
			e.shift_type,
			COALESCE(e.experience_years, 0)  AS experience_years,
			p.advance_card_amount,
			p.advance_cash_amount,
			COUNT(*) OVER () AS total_count
		FROM employee_payrolls p
		LEFT JOIN employees e ON e.id = p.employee_id` + payrollManagementFilterSQL + `
		ORDER BY p.store_name, p.full_name
		LIMIT NULLIF(@limit, 0) OFFSET @offset`

	var page []struct {
		domain.EmployeePayrollAdvanceRow `gorm:"embedded"`

		TotalCount int64 `gorm:"column:total_count"`
	}

	args := payrollManagementArgs(period, params)
	args["limit"] = params.Limit
	args["offset"] = params.Offset

	err = s.db.WithContext(ctx).Raw(query, args).Scan(&page).Error
	if err != nil {
		s.log.Errorf("payroll: could not get advance list: %v", err)
		return nil, 0, period, domain.InternalServerError
	}

	// Umumiy son har bir qatorda takrorlanadi (COUNT(*) OVER ()), birinchisidan olinadi.
	var totalCount int64
	if len(page) > 0 {
		totalCount = page[0].TotalCount
	}

	rows := make([]domain.EmployeePayrollAdvanceRow, 0, len(page))
	for _, r := range page {
		rows = append(rows, r.EmployeePayrollAdvanceRow)
	}

	return rows, totalCount, period, nil
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

	args := payrollSelectArgs(period, filter)
	args["limit"] = filter.Limit
	args["offset"] = filter.Offset

	err := s.db.WithContext(ctx).Raw(employeePayrollsSelectQuery, args).Scan(&page).Error
	if err != nil {
		s.log.Errorf("payroll: could not read payrolls: %v", err)
		return nil, domain.InternalServerError
	}

	return page, nil
}

// payrollSelectArgs — ro'yxat va statistika uchun umumiy parametrlar.
// Limit/offset bu yerda yo'q: ular faqat ro'yxatga tegishli.
func payrollSelectArgs(period domain.PayrollPeriod, filter payrollReadFilter) map[string]any {
	return map[string]any{
		"year":        period.Year,
		"month":       period.Month,
		"employee_id": nullIfEmpty(filter.EmployeeId),
		"store_id":    nullIfEmpty(filter.StoreId),
		"company_id":  nullIfEmpty(filter.CompanyId),
		"search":      nullIfEmpty(payrollSearchPattern(filter.Search)),
		// Bo'sh massiv NULL bo'lib ketadi → o'sha shart tekshirilmaydi.
		"roles":       pq.StringArray(filter.Roles),
		"only_worked": filter.OnlyWorked,
	}
}

// GetStorePayrollStatistics — GetStorePayrolls ro'yxatining yig'ma ko'rsatkichlari.
//
// Ro'yxat bilan bir xil filtrlardan o'tadi (davr, do'kon, kompaniya va rol +
// davomat doirasi), lekin sahifalanmaydi: limit/offset o'zgarganda raqamlar
// o'zgarmaydi, filtrga mos barcha do'kon hisobga olinadi.
func (s *Services) GetStorePayrollStatistics(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) (*domain.StorePayrollStatistics, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, period, domain.BadRequestError
	}

	var stats domain.StorePayrollStatistics
	err = s.db.WithContext(ctx).Raw(storePayrollStatisticsQuery, map[string]any{
		"year":       period.Year,
		"month":      period.Month,
		"company_id": nullIfEmpty(params.CompanyId),
		"store_id":   nullIfEmpty(params.StoreId),
		"status":     constants.GeneralStatusActive,
		"roles":      pq.StringArray(payrollSalesRoles),
	}).Scan(&stats).Error
	if err != nil {
		s.log.Errorf("payroll: could not get store statistics: %v", err)
		return nil, period, domain.InternalServerError
	}

	return &stats, period, nil
}

// GetPayrollStatistics — GetEmployeePayrolls ro'yxatining yig'ma ko'rsatkichlari.
//
// Ro'yxat bilan bir xil filtrlardan o'tadi (davr, do'kon, kompaniya, rol va
// "faqat ishlaganlar" doirasi), lekin sahifalanmaydi: limit/offset o'zgarganda
// raqamlar o'zgarmaydi, hamma mos xodim hisobga olinadi.
func (s *Services) GetPayrollStatistics(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) (*domain.PayrollStatistics, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, period, domain.BadRequestError
	}

	filter := payrollReadFilter{
		CompanyId:  params.CompanyId,
		StoreId:    params.StoreId,
		Search:     params.Search,
		Roles:      payrollSalesRoles,
		OnlyWorked: true,
	}

	var stats domain.PayrollStatistics
	err = s.db.WithContext(ctx).
		Raw(payrollStatisticsQuery, payrollSelectArgs(period, filter)).
		Scan(&stats).Error
	if err != nil {
		s.log.Errorf("payroll: could not get statistics: %v", err)
		return nil, period, domain.InternalServerError
	}

	return &stats, period, nil
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
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).
			Table("stores").
			Joins("LEFT JOIN companies c ON c.id = stores.company_id").
			Where("stores.deleted_at IS NULL").
			Where("stores.is_active = TRUE")
		if params.CompanyId != "" {
			q = q.Where("stores.company_id = ?", params.CompanyId)
		}
		if params.StoreId != "" {
			q = q.Where("stores.id = ?", params.StoreId)
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
			COALESCE(stores.employee_count, 0) AS store_employee_count,
			(SELECT COUNT(*)
			   FROM employees e
			  WHERE e.store_id = stores.id
			    AND e.is_active = TRUE
			    AND e.status = ?
			    AND e.deleted_at IS NULL) AS active_store_employee_count,
			COALESCE(c.is_franchise, false) AS is_franchise`,
			constants.GeneralStatusActive).
		Order("COALESCE(c.is_franchise, false) ASC").
		Order("stores.name ASC").
		// Limit <= 0 → cheklovsiz (Excel eksporti barcha do'konni oladi).
		// GORM'da Limit(0) "LIMIT 0" ga aylanadi va hech narsa qaytmaydi,
		// shuning uchun -1 beriladi: u LIMIT bandini butunlay tushiradi.
		Limit(payrollNoLimit(params.Limit)).
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
		total.EmployeeCount = store.EmployeeCount
		total.ActiveStoreEmployeeCount = store.ActiveStoreEmployeeCount
		total.IsFranchise = store.IsFranchise
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
//	*_agg    — davomat, bonus, do'kon savdosi, rejalar, rollar — har biri bir marta
//	base     — xodim ma'lumotlari + yuqoridagi yig'indilar
//	calc     — actual_salary va kesilgan DO'KON rejasi (expected_plan)
//	kpi      — do'kon bajarilish foizi
//	kpi_tier — foizdan reja pog'onasi (plan_kpi_percent)
//	kpi_rate — amaldagi kpi_percent: xodimniki bo'lsa o'sha, bo'lmasa pog'ona
//	final    — kpi_amount = (zav bo'lsa do'kon, aks holda o'z savdosi) * kpi_percent
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
            AND NOT EXISTS (SELECT 1 FROM holidays h WHERE h.date = d::date)) AS elapsed_work_days,
        EXTRACT(DAY FROM b.month_end)::int  AS month_days,
        EXTRACT(DAY FROM b.calc_date)::int  AS elapsed_days
    FROM bounds b
),
-- Yig'indilar barcha xodimlar uchun BIR MARTA hisoblanadi.
-- Ilgari bular LATERAL orqali har bir xodimga alohida bajarilardi — xodimlar
-- soniga ko'paytirilgan skan bo'lib, so'rov daqiqalab cho'zilardi.
attendance_agg AS (
    SELECT d.employee_id,
           SUM(d.worked_minutes)::numeric / 60.0 AS worked_hours
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
-- Davr savdosi — yagona manba sales jadvali. U BIR MARTA o'qiladi, natija
-- do'kon va xodim kesimida ikkiga bo'linadi: do'kon summasi KPI uchun, xodim
-- summasi ko'rsatish uchun.
--
-- Filtr UpdateStoreTargetSales'dagi bilan bir xil: yakunlangan (stage = 9),
-- qaytarish operatsiyasi bo'lmagan (sale_type = 'SALE'), qaytarilmagan sotuvlar.
sales_agg AS (
    SELECT sl.store_id, sl.employee_id, SUM(sl.total_amount) AS amount
    FROM sales sl, workdays w
    WHERE sl.stage = 9
      AND sl.sale_type = 'SALE'
      AND sl.is_returned IS NOT TRUE
      AND sl.created_at >= w.from_ts
      AND sl.created_at <  w.to_ts
    GROUP BY sl.store_id, sl.employee_id
),
-- Do'kon aylanmasi: xodimga biriktirilmagan sotuv ham kiradi.
store_sales_agg AS (
    SELECT sa.store_id, SUM(sa.amount) AS store_sales_amount
    FROM sales_agg sa
    WHERE sa.store_id IS NOT NULL
    GROUP BY sa.store_id
),
-- Xodimning shaxsiy savdosi. Do'kon bo'yicha cheklanmaydi: xodim boshqa
-- do'konda smenaga chiqib sotgan bo'lsa ham o'z hisobiga tushadi.
employee_sales_agg AS (
    SELECT sa.employee_id, SUM(sa.amount) AS individual_sales_amount
    FROM sales_agg sa
    WHERE sa.employee_id IS NOT NULL
    GROUP BY sa.employee_id
),
-- Xodimning oylik rejasi. employee_targets'ning unikal kaliti
-- (employee_id, store_id, year, month) — ya'ni ichida store_id bor, va xodim oy
-- davomida boshqa do'konga ko'chirilsa unda BIR NECHTA qator bo'ladi.
-- To'g'ridan-to'g'ri JOIN qilinsa xodim bir necha marta chiqib, INSERT bitta
-- (employee_id, year, month) ga ikki marta urinardi va Postgres
-- "ON CONFLICT DO UPDATE cannot affect row a second time" bilan yiqilardi.
-- Do'konlar bo'yicha yig'indi olinadi: xodimning oy uchun umumiy rejasi.
targets_agg AS (
    SELECT et.employee_id, SUM(et.amount) AS amount
    FROM employee_targets et
    WHERE et.year = @year AND et.month = @month
    GROUP BY et.employee_id
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
        -- Oylik norma = oydagi ish kunlari × xodimning kunlik smenasi. Kun soni
        -- kalendardan olinadi, shuning uchun fevral va mart uchun norma har xil
        -- bo'ladi va actual_salary to'g'ri proporsiyada chiqadi.
        --
        -- Kunlik smena employees.daily_work_hours'dan olinadi (4/7/8). Kiritilmagan
        -- bo'lsa (0) payrollWorkDayHours default'iga tushadi (eski xatti-harakat).
        -- DIQQAT: izohlarda parametr nomini yozmang. GORM ularni oddiy matn
        -- almashtirish bilan topadi va izoh ichidagisini ham bog'laydi; keyin
        -- Postgres "could not determine data type of parameter" bilan yiqiladi.
        w.month_work_days
            * COALESCE(NULLIF(e.daily_work_hours, 0), CAST(@work_day_hours AS numeric))
            AS avg_monthly_hours,
        e.salary                                AS salary_rate_amount,
        -- Xodim kartochkasidagi shaxsiy KPI foizi. 0 bo'lsa "kiritilmagan"
        -- degani va reja pog'onasi ishlatiladi (qarang: kpi_rate).
        COALESCE(e.kpi_percent, 0)              AS employee_kpi_percent,
        COALESCE(att.worked_hours, 0)            AS worked_hours,
        COALESCE(es.individual_sales_amount, 0)  AS individual_sales_amount,
        COALESCE(bon.bonus_amount, 0)            AS bonus_amount,
        st.id                                   AS store_target_id,
        COALESCE(st.amount, 0)                  AS store_plan_amount,
        COALESCE(ss.store_sales_amount, 0)      AS store_sales_amount,
        COALESCE(et.amount, 0)                  AS employee_plan_amount,
        w.month_work_days,
        w.elapsed_work_days,
        w.month_days,
        w.elapsed_days
    FROM employees e
    CROSS JOIN workdays w
    LEFT JOIN stores s          ON s.id = e.store_id
    LEFT JOIN roles_agg rl      ON rl.employee_id = e.id
    LEFT JOIN attendance_agg att ON att.employee_id = e.id
    LEFT JOIN employee_sales_agg es ON es.employee_id = e.id
    LEFT JOIN bonus_agg bon     ON bon.employee_id = e.id
    LEFT JOIN store_sales_agg ss ON ss.store_id = e.store_id
    LEFT JOIN store_targets st
           ON st.store_id = e.store_id AND st.year = @year AND st.month = @month
    LEFT JOIN targets_agg et ON et.employee_id = e.id
    WHERE e.is_active
      AND e.status = @status
      AND e.deleted_at IS NULL
),
calc AS (
    SELECT b.*,
           -- avg_monthly_hours = 0 bo'lsa nolga bo'lish o'rniga 0 qaytadi
           COALESCE(ROUND(b.salary_rate_amount * (b.worked_hours / NULLIF(b.avg_monthly_hours, 0)), 2), 0) AS actual_salary_amount,
           -- DO'KON rejasi o'tgan KALENDAR kunlariga proporsional kesiladi:
           -- oy 31 kun, kecha 30-sana bo'lsa → reja * 30 / 31.
           -- Ish kuni emas, ketma-ket kun — buxgalteriya shu tartibda hisoblaydi.
           COALESCE(ROUND(b.store_plan_amount * b.elapsed_days / NULLIF(b.month_days, 0), 2), 0) AS expected_plan_amount
    FROM base b
),
kpi AS (
    SELECT c.*,
           -- Do'kon savdosi kesilgan do'kon rejasiga nisbatan o'lchanadi.
           -- Reja 0 bo'lsa (oy boshi yoki target yo'q) foiz ham 0.
           COALESCE(ROUND(c.store_sales_amount / NULLIF(c.expected_plan_amount, 0) * 100, 2), 0) AS plan_achievement_percent
    FROM calc c
),
kpi_tier AS (
    SELECT k.*,
           CAST(CASE
               WHEN k.plan_achievement_percent >= 100 THEN 1.4
               WHEN k.plan_achievement_percent >= 90  THEN 1.0
               WHEN k.plan_achievement_percent >= 80  THEN 0.8
               ELSE 0
           END AS numeric) AS plan_kpi_percent
    FROM kpi k
),
kpi_rate AS (
    SELECT t.*,
           -- Xodim kartochkasida foiz kiritilgan bo'lsa (kpi_percent > 0) o'sha
           -- ISHLATILADI va reja pog'onasi e'tiborga olinmaydi — ya'ni do'kon
           -- rejani bajarmasa ham bu xodim KPI oladi. Kiritilmagan bo'lsa
           -- (0) umumiy qoida: pog'ona do'kon rejasidan chiqadi.
           CASE WHEN t.employee_kpi_percent > 0
                THEN t.employee_kpi_percent
                ELSE t.plan_kpi_percent
           END AS kpi_percent
    FROM kpi_tier t
),
final AS (
    SELECT r.*,
           -- Baza rolga qarab: zav do'kon aylanmasidan, qolganlar o'z
           -- savdosidan oladi. Xodimda ikkala rol ham bo'lsa zav ustun turadi.
           ROUND(CASE WHEN @zav_role = ANY(r.role_names)
                      THEN r.store_sales_amount
                      ELSE r.individual_sales_amount
                 END * r.kpi_percent / 100.0, 2) AS kpi_amount
    FROM kpi_rate r
)
INSERT INTO employee_payrolls (
    id, employee_id, company_id, store_id, store_name,
    first_name, last_name, full_name, position_snapshot, role, role_names,
    experience_years, worked_hours, avg_monthly_hours,
    salary_rate_amount, actual_salary_amount, individual_sales_amount,
    store_target_id, store_plan_amount, store_sales_amount,
    employee_plan_amount, expected_plan_amount, plan_achievement_percent,
    month_work_days, elapsed_work_days,
    plan_kpi_percent, employee_kpi_percent, kpi_percent, kpi_amount, bonus_amount,
    gross_salary_amount, net_pay_amount,
    status, year, month, calculated_at
)
SELECT
    uuid_generate_v4(), f.employee_id, f.company_id, f.store_id, f.store_name,
    f.first_name, f.last_name, f.full_name, f.position_snapshot, f.role, f.role_names,
    f.experience_years, f.worked_hours, f.avg_monthly_hours,
    f.salary_rate_amount, f.actual_salary_amount, f.individual_sales_amount,
    f.store_target_id, f.store_plan_amount, f.store_sales_amount,
    f.employee_plan_amount, f.expected_plan_amount, f.plan_achievement_percent,
    f.month_work_days, f.elapsed_work_days,
    f.plan_kpi_percent, f.employee_kpi_percent, f.kpi_percent,
    f.kpi_amount,
    f.bonus_amount,
    f.actual_salary_amount + f.kpi_amount + f.bonus_amount,
    f.actual_salary_amount + f.kpi_amount + f.bonus_amount,
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
    store_sales_amount       = EXCLUDED.store_sales_amount,
    employee_plan_amount     = EXCLUDED.employee_plan_amount,
    expected_plan_amount     = EXCLUDED.expected_plan_amount,
    plan_achievement_percent = EXCLUDED.plan_achievement_percent,
    month_work_days          = EXCLUDED.month_work_days,
    elapsed_work_days        = EXCLUDED.elapsed_work_days,
    plan_kpi_percent         = EXCLUDED.plan_kpi_percent,
    employee_kpi_percent     = EXCLUDED.employee_kpi_percent,
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
// Deyarli hamma narsa qatorning o'zida snapshot qilingan; yagona istisno —
// role_type, u employees'dan JONLI olinadi.
//
// Nega snapshot emas: role_type management ekranidan tahrirlanadi
// (PUT /employee/payroll/{id}/management) va faqat employees'ga yoziladi.
// Snapshot bo'lganida tahrirdan keyin cron ishlagunicha hisobotda eski qiymat
// turgan bo'lardi.
//
// Filtrlar ixtiyoriy — NULL berilsa o'sha shart tekshirilmaydi.
const employeePayrollsSelectQuery = `
SELECT
    p.id, p.employee_id, p.store_id, p.store_name,
    p.first_name, p.last_name, p.full_name, p.position_snapshot, p.role,
    COALESCE(e.role_type, '') AS role_type,
    p.experience_years, p.worked_hours, p.avg_monthly_hours,
    p.salary_rate_amount, p.actual_salary_amount, p.individual_sales_amount,
    p.store_target_id, p.store_plan_amount, p.store_sales_amount,
    p.employee_plan_amount, p.expected_plan_amount, p.plan_achievement_percent,
    p.month_work_days, p.elapsed_work_days,
    p.plan_kpi_percent, p.employee_kpi_percent, p.kpi_percent,
    p.kpi_amount, p.bonus_amount, p.gross_salary_amount,
    p.advance_card_amount, p.advance_cash_amount,
    p.deduction_term_amount, p.deduction_recount_amount, p.deduction_fine_amount,
    p.net_pay_amount, p.status, p.year, p.month, p.completed_at, p.calculated_at,
    COUNT(*) OVER () AS total_count
FROM employee_payrolls p
LEFT JOIN employees e ON e.id = p.employee_id` + payrollSelectFilterSQL + `
ORDER BY p.store_name, p.full_name
LIMIT NULLIF(@limit, 0) OFFSET @offset`

// payrollSelectFilterSQL — hisobotning WHERE bo'lagi. Ro'yxat ham, statistika
// ham AYNAN shu shartdan foydalanadi: aks holda ekrandagi qatorlar bilan
// yuqoridagi yig'ma raqamlar bir-biriga mos kelmay qolardi.
const payrollSelectFilterSQL = `
WHERE p.year = @year
  AND p.month = @month
  AND (CAST(@employee_id AS uuid) IS NULL OR p.employee_id = CAST(@employee_id AS uuid))
  AND (CAST(@store_id AS uuid)    IS NULL OR p.store_id    = CAST(@store_id AS uuid))
  AND (CAST(@company_id AS uuid)  IS NULL OR p.company_id  = CAST(@company_id AS uuid))
	AND (CAST(@search AS text)      IS NULL OR p.full_name ILIKE CAST(@search AS text))
  AND (CAST(@roles AS text[])     IS NULL OR p.role_names && CAST(@roles AS text[]))
  AND (NOT CAST(@only_worked AS boolean) OR p.worked_hours > 0)`

// payrollStatisticsQuery — hisobotning yig'ma ko'rsatkichlari.
//
// store_plan, store_sales va expected_plan do'kon darajasidagi qiymatlar bo'lib,
// do'konning har bir xodim qatorida takrorlanadi. Ularni filtered ustidan SUM
// qilsa, do'kondagi xodimlar soniga ko'payib ketardi — shuning uchun avval
// per_store'da do'kon bo'yicha bittaga tushiriladi, keyin qo'shiladi.
const payrollStatisticsQuery = `
WITH filtered AS (
    SELECT p.*
    FROM employee_payrolls p
    LEFT JOIN employees e ON e.id = p.employee_id` + payrollSelectFilterSQL + `
),
per_store AS (
    SELECT store_id,
           MAX(store_plan_amount)    AS store_plan_amount,
           MAX(store_sales_amount)   AS store_sales_amount,
           MAX(expected_plan_amount) AS expected_plan_amount
    FROM filtered
    GROUP BY store_id
)
SELECT
    COUNT(*)::bigint                   AS total_employees_count,
    COUNT(DISTINCT f.store_id)::bigint AS total_stores_count,

    COALESCE(SUM(f.worked_hours), 0)      AS total_worked_hours,
    COALESCE(SUM(f.avg_monthly_hours), 0) AS total_avg_monthly_hours,

    COALESCE(SUM(f.salary_rate_amount), 0)   AS total_salary_rate_amount,
    COALESCE(SUM(f.actual_salary_amount), 0) AS total_actual_salary_amount,

    COALESCE((SELECT SUM(store_plan_amount)    FROM per_store), 0) AS total_store_plan_amount,
    COALESCE((SELECT SUM(store_sales_amount)   FROM per_store), 0) AS total_store_sales_amount,
    COALESCE((SELECT SUM(expected_plan_amount) FROM per_store), 0) AS total_expected_plan_amount,

    COALESCE(SUM(f.kpi_amount), 0)          AS total_kpi_amount,
    COALESCE(SUM(f.bonus_amount), 0)        AS total_bonus_amount,
    COALESCE(SUM(f.gross_salary_amount), 0) AS total_gross_salary_amount,

    COALESCE(SUM(f.advance_card_amount), 0) AS total_advance_card_amount,
    COALESCE(SUM(f.advance_cash_amount), 0) AS total_advance_cash_amount,

    COALESCE(SUM(f.deduction_term_amount), 0)    AS total_deduction_term_amount,
    COALESCE(SUM(f.deduction_recount_amount), 0) AS total_deduction_recount_amount,
    COALESCE(SUM(f.deduction_fine_amount), 0)    AS total_deduction_fine_amount,

    COALESCE(SUM(f.net_pay_amount), 0) AS total_net_pay_amount
FROM filtered f`

// storePayrollTotalsQuery — do'kon kesimidagi yig'indilar. store_plan_amount
// har bir xodim qatorida takrorlanadi (bitta store_target'dan keladi), shuning
// uchun SUM emas, MAX olinadi.
const storePayrollTotalsQuery = `
SELECT
    p.store_id                     AS store_id,
    -- Yig'indilarga kirgan xodimlar soni. employee_payrolls'da bir xodimga oyiga
    -- bitta qator (UNIQUE employee_id, year, month), shuning uchun qator soni =
    -- xodim soni. Quyidagi WHERE'dan o'tganlar sanaladi, ya'ni rol mos kelmagan
    -- va smenaga chiqmagan xodimlar bu songa kirmaydi.
    --
    -- Do'kondagi umumiy xodimlar soni esa paginateStores'da employees/stores
    -- jadvallaridan olinadi — cron hali qamramaganlar ham o'sha yerda sanaladi.
    COUNT(*)::int                  AS payroll_count,
    SUM(p.worked_hours)            AS worked_hours,
    -- Do'konning umumiy norma soati: har bir xodimning oylik normasi qo'shiladi.
    -- worked_hours bilan yonma-yon turadi, shuning uchun ikkalasi ham AYNAN bir
    -- xil qatorlar bo'yicha yig'iladi — aks holda ularni solishtirib bo'lmasdi.
    SUM(p.avg_monthly_hours)       AS avg_monthly_hours,
    SUM(p.salary_rate_amount)      AS salary_rate_amount,
    SUM(p.actual_salary_amount)    AS actual_salary_amount,
    SUM(p.individual_sales_amount) AS individual_sales_amount,
    MAX(p.store_plan_amount)       AS store_plan_amount,
    -- Do'kon savdosi ham har bir xodim qatorida takrorlanadi
    MAX(p.store_sales_amount)      AS store_sales_amount,
    SUM(p.kpi_amount)              AS kpi_amount,
    SUM(p.bonus_amount)            AS bonus_amount,
    SUM(p.gross_salary_amount)     AS gross_salary_amount,
    -- Oylik xarajati do'kon aylanmasining necha foizini yeyapti.
    -- store_sales_amount har bir xodim qatorida takrorlanadi, shuning uchun
    -- maxrajda SUM emas, MAX. Savdo 0 bo'lsa foiz ham 0 (bo'lishdan himoya).
    COALESCE(ROUND(
        SUM(p.gross_salary_amount) / NULLIF(MAX(p.store_sales_amount), 0) * 100
    , 2), 0)                       AS salary_percent,
    -- Karta va naqd avans bitta songa qo'shiladi: do'kon bo'yicha jami avans
    SUM(p.advance_card_amount + p.advance_cash_amount) AS advance_amount,
    -- Uchala ushlab qolish turi ham bitta songa: muddat, qayta hisob va jarima
    SUM(p.deduction_term_amount
      + p.deduction_recount_amount
      + p.deduction_fine_amount)   AS total_deduction,
    SUM(p.net_pay_amount)          AS net_pay_amount
FROM employee_payrolls p
WHERE p.year = @year
  AND p.month = @month
  AND p.store_id = ANY(CAST(@store_ids AS uuid[]))
  -- GetEmployeePayrolls bilan bir xil doira: do'kon yig'indisi aynan xodimlar
  -- ro'yxatida ko'rinadigan qatorlardan chiqadi, shuning uchun ro'yxatni qo'lda
  -- qo'shib chiqqanda do'kon summasi bilan mos keladi.
  AND p.role_names && CAST(@roles AS text[])
  AND p.worked_hours > 0
GROUP BY p.store_id`

// storePayrollStatisticsQuery — do'konlar ro'yxatining yig'ma ko'rsatkichlari.
//
// Ikkita manba birlashtiriladi:
//
//	stores_f   — do'konlar va ulardagi xodimlar soni (paginateStores bilan bir xil
//	             filtr, lekin sahifasiz: filtrga mos BARCHA do'konlar)
//	payrolls_f — o'sha do'konlarning oylik qatorlari (storePayrollTotalsQuery
//	             bilan bir xil doira: rol mos va davomati bor xodimlar)
//
// store_plan va store_sales do'kon darajasidagi qiymatlar bo'lib, do'konning har
// bir xodim qatorida takrorlanadi. Ularni payrolls_f ustidan SUM qilsa,
// do'kondagi xodimlar soniga ko'payib ketardi — shuning uchun per_store'da
// avval do'kon bo'yicha bittaga tushiriladi.
const storePayrollStatisticsQuery = `
WITH stores_f AS (
    SELECT
        s.id,
        COALESCE(s.employee_count, 0) AS employee_count,
        (SELECT COUNT(*)
           FROM employees e
          WHERE e.store_id = s.id
            AND e.is_active = TRUE
            AND e.status = CAST(@status AS text)
            AND e.deleted_at IS NULL) AS active_employee_count
    FROM stores s
    WHERE s.deleted_at IS NULL
      AND s.is_active = TRUE
      AND (CAST(@company_id AS uuid) IS NULL OR s.company_id = CAST(@company_id AS uuid))
      AND (CAST(@store_id AS uuid)   IS NULL OR s.id         = CAST(@store_id AS uuid))
),
payrolls_f AS (
    SELECT p.*
    FROM employee_payrolls p
    WHERE p.year = @year
      AND p.month = @month
      AND p.store_id IN (SELECT id FROM stores_f)
      AND p.role_names && CAST(@roles AS text[])
      AND p.worked_hours > 0
),
per_store AS (
    SELECT store_id,
           MAX(store_plan_amount)  AS store_plan_amount,
           MAX(store_sales_amount) AS store_sales_amount
    FROM payrolls_f
    GROUP BY store_id
)
SELECT
    (SELECT COUNT(*) FROM stores_f)::bigint                             AS total_stores_count,
    COALESCE((SELECT SUM(employee_count) FROM stores_f), 0)::numeric    AS total_employee_count,
    COALESCE((SELECT SUM(active_employee_count) FROM stores_f), 0)::bigint AS total_active_store_employee_count,
    (SELECT COUNT(*) FROM payrolls_f)::bigint                           AS total_payroll_count,

    COALESCE(SUM(f.worked_hours), 0)      AS total_worked_hours,
    COALESCE(SUM(f.avg_monthly_hours), 0) AS total_avg_monthly_hours,

    COALESCE(SUM(f.salary_rate_amount), 0)      AS total_salary_rate_amount,
    COALESCE(SUM(f.actual_salary_amount), 0)    AS total_actual_salary_amount,
    COALESCE(SUM(f.individual_sales_amount), 0) AS total_individual_sales_amount,

    COALESCE((SELECT SUM(store_plan_amount)  FROM per_store), 0) AS total_store_plan_amount,
    COALESCE((SELECT SUM(store_sales_amount) FROM per_store), 0) AS total_store_sales_amount,

    COALESCE(SUM(f.kpi_amount), 0)          AS total_kpi_amount,
    COALESCE(SUM(f.bonus_amount), 0)        AS total_bonus_amount,
    COALESCE(SUM(f.gross_salary_amount), 0) AS total_gross_salary_amount,

    COALESCE(SUM(f.advance_card_amount + f.advance_cash_amount), 0) AS total_advance_amount,
    COALESCE(SUM(f.deduction_term_amount
               + f.deduction_recount_amount
               + f.deduction_fine_amount), 0)                       AS total_deduction_amount,

    COALESCE(SUM(f.net_pay_amount), 0) AS total_net_pay_amount
FROM payrolls_f f`
