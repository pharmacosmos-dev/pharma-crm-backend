package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storeRef struct {
	Id   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
}
type employeePayrollPageRow struct {
	domain.EmployeePayrollRow `gorm:"embedded"`

	TotalCount int64 `gorm:"column:total_count"`
}
type payrollFilter struct {
	EmployeeId string   // bitta xodim
	StoreIds   []string // shu do'konlarning xodimlari
	CompanyId  string   // bitta kompaniya
	Roles      []string // xodimda shu rollardan biri bo'lishi shart
	Limit      int      // 0 = cheklovsiz
	Offset     int
}

var payrollSalesRoles = []string{constants.RoleNameCashier, constants.RoleNameZavStore}


func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}


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

	// Limit yo'q: sahifadagi do'konlarning BARCHA xodimlari kerak, aks holda
	// yig'indi to'liq chiqmaydi.
	page, err := s.payrollPage(ctx, period, payrollFilter{StoreIds: storeIdsOf(stores)})
	if err != nil {
		return nil, 0, period, err
	}

	storePayrolls := buildStorePayrolls(stores, payrollRowsOf(page))

	// Yig'indilar allaqachon hisoblangan — javobda faqat do'kon qatorlari qoladi.
	// Xodimlar ro'yxati GetEmployeePayrolls orqali alohida so'raladi.
	for i := range storePayrolls {
		storePayrolls[i].Employees = nil
	}

	return storePayrolls, totalCount, period, nil
}

// region EmployeePayrolls

func (s *Services) GetEmployeePayrolls(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]domain.EmployeePayrollRow, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, 0, period, domain.BadRequestError
	}

	filter := payrollFilter{
		CompanyId: params.CompanyId,
		Roles:     payrollSalesRoles,
		Limit:     params.Limit,
		Offset:    params.Offset,
	}
	if params.StoreId != "" {
		filter.StoreIds = []string{params.StoreId}
	}

	page, err := s.payrollPage(ctx, period, filter)
	if err != nil {
		return nil, 0, period, err
	}

	// Umumiy son har bir qatorda takrorlanadi (COUNT(*) OVER ()), shuning uchun
	// birinchisidan olinadi. Sahifa bo'sh bo'lsa — 0.
	var totalCount int64
	if len(page) > 0 {
		totalCount = page[0].TotalCount
	}

	return payrollRowsOf(page), totalCount, period, nil
}

// region MyPayroll

func (s *Services) GetMyPayroll(
	ctx context.Context, employeeId string, year, month int,
) (*domain.MyPayrollResponse, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("payroll: invalid period: %v", err)
		return nil, domain.BadRequestError
	}

	page, err := s.payrollPage(ctx, period, payrollFilter{EmployeeId: employeeId, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(page) == 0 {
		return nil, domain.NotFoundError
	}

	return &domain.MyPayrollResponse{Period: period, Payroll: page[0].EmployeePayrollRow}, nil
}


func (s *Services) payrollPage(
	ctx context.Context, period domain.PayrollPeriod, filter payrollFilter,
) ([]employeePayrollPageRow, error) {
	var page []employeePayrollPageRow

	err := s.db.WithContext(ctx).Raw(employeePayrollsQuery, map[string]any{
		"from":        period.From,
		"to":          period.To,
		"year":        period.Year,
		"month":       period.Month,
		"status":      constants.GeneralStatusActive,
		"draft":       domain.EmployeePayrollStatusDraft,
		"employee_id": nullIfEmpty(filter.EmployeeId),
		"company_id":  nullIfEmpty(filter.CompanyId),
		// Bo'sh massiv → NULL → o'sha shart tekshirilmaydi (pq.StringArray shunday ishlaydi).
		"store_ids": pq.StringArray(filter.StoreIds),
		"roles":     pq.StringArray(filter.Roles),
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	}).Scan(&page).Error
	if err != nil {
		s.log.Errorf("payroll: could not get employee payrolls: %v", err)
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

// region Cron

// AutoCreateMonthlyPayrolls — har oyning 1-kuni cron tomonidan chaqiriladi.
func (s *Services) AutoCreateMonthlyPayrolls() {
	const op = "cron AutoCreateMonthlyPayrolls"

	ctx := context.Background()
	prevMonth := tashkentNow().AddDate(0, -1, 0)

	period, err := resolvePayrollPeriod(prevMonth.Year(), int(prevMonth.Month()))
	if err != nil {
		s.log.Errorf("%s: %v", op, err)
		return
	}

	// Filtrsiz: barcha faol xodimlar, rolidan qat'i nazar. Limit ham yo'q —
	// snapshot hammasi uchun yozilishi kerak.
	page, err := s.payrollPage(ctx, period, payrollFilter{})
	if err != nil {
		s.log.Errorf("%s: could not calculate: %v", op, err)
		return
	}

	created := 0
	for _, row := range payrollRowsOf(page) {
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


const employeePayrollsQuery = `
WITH page AS (
    SELECT
        e.id                     AS employee_id,
        e.store_id               AS store_id,
        COALESCE(s.name, '')     AS store_name,
        e.first_name             AS first_name,
        e.last_name              AS last_name,
        e.full_name              AS full_name,
        COALESCE(e.position, '') AS position_snapshot,
        e.experience_years       AS experience_years,
        e.avg_monthly_hours      AS avg_monthly_hours,
        e.salary                 AS salary_rate_amount,
        e.kpi_percent            AS kpi_percent,
        COALESCE(rl.role, '')    AS role,
        COUNT(*) OVER ()         AS total_count
    FROM employees e
    LEFT JOIN stores s ON s.id = e.store_id
    LEFT JOIN LATERAL (
        SELECT string_agg(r.name, ', ' ORDER BY r.name) AS role
        FROM employee_roles er
        JOIN roles r ON r.id = er.role_id
        WHERE er.employee_id = e.id
    ) rl ON TRUE
    WHERE e.is_active
      AND e.status = @status
      -- Filtrlar ixtiyoriy: NULL berilsa o'sha shart tekshirilmaydi.
      AND (CAST(@employee_id AS uuid) IS NULL OR e.id = CAST(@employee_id AS uuid))
      AND (CAST(@store_ids AS uuid[]) IS NULL OR e.store_id = ANY(CAST(@store_ids AS uuid[])))
      AND (CAST(@company_id AS uuid) IS NULL OR e.company_id = CAST(@company_id AS uuid))
      AND (CAST(@roles AS text[]) IS NULL OR EXISTS (
          SELECT 1
          FROM employee_roles er
          JOIN roles r ON r.id = er.role_id
          WHERE er.employee_id = e.id
            AND r.name = ANY(CAST(@roles AS text[]))
      ))
    ORDER BY store_name, e.full_name
    LIMIT NULLIF(@limit, 0) OFFSET @offset
),
totals AS (
    SELECT
        p.*,
        COALESCE(att.worked_hours, 0)            AS worked_hours,
        COALESCE(att.individual_sales_amount, 0) AS individual_sales_amount,
        COALESCE(bon.bonus_amount, 0)            AS bonus_amount,
        st.id                                    AS store_target_id,
        COALESCE(st.amount, 0)                   AS store_plan_amount
    FROM page p
    LEFT JOIN LATERAL (
        SELECT SUM(d.worked_minutes)::numeric / 60.0 AS worked_hours,
               SUM(d.sales_amount)                   AS individual_sales_amount
        FROM employee_attendance_days d
        WHERE d.employee_id = p.employee_id
          AND d.work_date BETWEEN @from AND @to
    ) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT SUM(b.bonus_amount) AS bonus_amount
        FROM employee_bonus b
        WHERE b.employee_id = p.employee_id
          AND b.deleted_at IS NULL
          AND (b.created_at + interval '5 hours')::date BETWEEN @from AND @to
    ) bon ON TRUE
    LEFT JOIN store_targets st
           ON st.store_id = p.store_id
          AND st.year = @year
          AND st.month = @month
),
calc AS (
    SELECT
        t.*,
        -- avg_monthly_hours = 0 bo'lsa nolga bo'lish o'rniga 0 qaytadi
        COALESCE(ROUND(t.salary_rate_amount * (t.worked_hours / NULLIF(t.avg_monthly_hours, 0)), 2), 0) AS actual_salary_amount,
        ROUND(t.individual_sales_amount * t.kpi_percent / 100.0, 2)                                   AS kpi_amount
    FROM totals t
)
SELECT
    c.employee_id,
    c.store_id,
    c.store_name,
    c.first_name,
    c.last_name,
    c.full_name,
    c.role,
    c.total_count,

    -- Snapshot bor bo'lsa muzlatilgan qiymat, aks holda jonli hisob.
    COALESCE(sn.position_snapshot, c.position_snapshot)             AS position_snapshot,
    COALESCE(sn.experience_years, c.experience_years)               AS experience_years,
    COALESCE(sn.worked_hours, c.worked_hours)                       AS worked_hours,
    COALESCE(sn.avg_monthly_hours, c.avg_monthly_hours)             AS avg_monthly_hours,
    COALESCE(sn.salary_rate_amount, c.salary_rate_amount)           AS salary_rate_amount,
    COALESCE(sn.actual_salary_amount, c.actual_salary_amount)       AS actual_salary_amount,
    COALESCE(sn.individual_sales_amount, c.individual_sales_amount) AS individual_sales_amount,
    COALESCE(sn.store_target_id, c.store_target_id)                 AS store_target_id,
    COALESCE(sn.store_plan_amount, c.store_plan_amount)             AS store_plan_amount,
    COALESCE(sn.kpi_percent, c.kpi_percent)                         AS kpi_percent,
    COALESCE(sn.kpi_amount, c.kpi_amount)                           AS kpi_amount,
    COALESCE(sn.bonus_amount, c.bonus_amount)                       AS bonus_amount,
    COALESCE(sn.gross_salary_amount,
             c.actual_salary_amount + c.kpi_amount + c.bonus_amount) AS gross_salary_amount,

    -- Avans va ushlab qolishlar qo'lda kiritiladi: manba jadvali yo'q, shuning
    -- uchun ular faqat snapshot'da bo'ladi, jonli hisobda doim 0.
    COALESCE(sn.advance_card_amount, 0)      AS advance_card_amount,
    COALESCE(sn.advance_cash_amount, 0)      AS advance_cash_amount,
    COALESCE(sn.deduction_term_amount, 0)    AS deduction_term_amount,
    COALESCE(sn.deduction_recount_amount, 0) AS deduction_recount_amount,
    COALESCE(sn.deduction_fine_amount, 0)    AS deduction_fine_amount,

    COALESCE(sn.net_pay_amount,
             c.actual_salary_amount + c.kpi_amount + c.bonus_amount) AS net_pay_amount,

    COALESCE(sn.status, @draft) AS status,
    sn.completed_at             AS completed_at,
    CAST(@year AS int)          AS year,
    CAST(@month AS int)         AS month
FROM calc c
LEFT JOIN employee_payrolls sn
       ON sn.employee_id = c.employee_id
      AND sn.year = @year
      AND sn.month = @month
ORDER BY c.store_name, c.full_name`

