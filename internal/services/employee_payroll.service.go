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

// Oylik hisob-kitob formulalari (bitta manba — API ham, cron ham shu SQL'dan foydalanadi):
//
//	worked_hours         = SUM(employee_attendance_days.worked_minutes) / 60
//	individual_sales     = SUM(employee_attendance_days.sales_amount)
//	bonus_amount         = SUM(employee_bonus.bonus_amount)
//	actual_salary_amount = salary_rate * (worked_hours / avg_monthly_hours)
//	kpi_amount           = individual_sales * kpi_percent / 100
//	gross_salary_amount  = actual_salary + kpi_amount + bonus_amount
//	net_pay_amount       = gross - (avanslar + ushlab qolishlar)
//
// Avans va ushlab qolishlar qo'lda kiritiladi — jonli hisobda ular 0, o'tgan oylarda
// employee_payrolls'dagi saqlangan qiymat ishlatiladi.
//
// %s — WHERE filtri (faqat kod ichidagi doimiy satrlar qo'yiladi, foydalanuvchi kiritmasi emas).
const payrollLiveSQLTemplate = `
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
           COALESCE(ROUND(salary_rate_amount * (worked_hours / NULLIF(avg_monthly_hours, 0)), 2), 0) AS actual_salary_amount,
           ROUND(individual_sales_amount * kpi_percent / 100.0, 2)                                   AS kpi_amount
    FROM base
)
SELECT calc.*,
       (actual_salary_amount + kpi_amount + bonus_amount) AS gross_salary_amount,
       0::numeric                                         AS advance_card_amount,
       0::numeric                                         AS advance_cash_amount,
       0::numeric                                         AS deduction_term_amount,
       0::numeric                                         AS deduction_recount_amount,
       0::numeric                                         AS deduction_fine_amount,
       (actual_salary_amount + kpi_amount + bonus_amount) AS net_pay_amount
FROM calc
ORDER BY store_name, full_name`

// O'tgan oylar uchun — employee_payrolls'dan o'qish. %s — WHERE filtri.
const payrollStoredSQLTemplate = `
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

// tashkentNow — Toshkent devor-vaqti (loyihada hamma joyda UTC+5 ishlatiladi)
func tashkentNow() time.Time {
	return time.Now().UTC().Add(domain.TashkentTimeDif)
}

// resolvePayrollPeriod — so'ralgan yil/oyni tekshiradi va davr chegaralarini qaytaradi.
// Joriy oy bo'lsa oxirgi kun = bugun (is_live=true), aks holda oyning oxirgi kuni.
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

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	if start.After(now) {
		return domain.PayrollPeriod{}, fmt.Errorf("period is in the future: %d-%02d", year, month)
	}

	isLive := year == now.Year() && month == int(now.Month())
	end := start.AddDate(0, 1, -1) // oyning oxirgi kuni
	if isLive {
		end = now // joriy oy — faqat bugungi kungacha
	}

	return domain.PayrollPeriod{
		Year:   year,
		Month:  month,
		From:   start.Format(constants.TimeOnlyDateFormat),
		To:     end.Format(constants.TimeOnlyDateFormat),
		IsLive: isLive,
	}, nil
}

// fetchPayrollRows — davrga qarab jonli hisoblaydi yoki employee_payrolls'dan o'qiydi.
// filter — SQL WHERE bo'lagi (doimiy satr), args — uning parametrlari.
func (s *Services) fetchPayrollRows(
	ctx context.Context, period domain.PayrollPeriod, liveFilter, storedFilter string, args ...any,
) ([]domain.EmployeePayrollRow, error) {
	var (
		rows  []domain.EmployeePayrollRow
		query string
		full  []any
	)

	if period.IsLive {
		query = fmt.Sprintf(payrollLiveSQLTemplate, liveFilter)
		full = append([]any{
			period.From, period.To, // att
			period.From, period.To, // bon
			period.Year, period.Month, // store_targets
			constants.GeneralStatusActive, // e.status
		}, args...)
	} else {
		query = fmt.Sprintf(payrollStoredSQLTemplate, storedFilter)
		full = append([]any{period.Year, period.Month}, args...)
	}

	if err := s.db.WithContext(ctx).Raw(query, full...).Scan(&rows).Error; err != nil {
		s.log.Errorf("could not get payroll rows: %v", err)
		return nil, domain.InternalServerError
	}

	// Jonli hisobda bu maydonlar SQL'dan kelmaydi — davrdan to'ldiramiz
	if period.IsLive {
		for i := range rows {
			rows[i].Status = domain.EmployeePayrollStatusDraft
			rows[i].Year = period.Year
			rows[i].Month = period.Month
		}
	}
	return rows, nil
}

// GetStorePayrolls — API 1: do'konlar bo'yicha sahifalangan ro'yxat, har bir do'kon
// ichida uning xodimlarining oylik ko'rsatkichlari. Pagination do'konlarga qo'yiladi.
func (s *Services) GetStorePayrolls(
	ctx context.Context, params *domain.EmployeePayrollQueryParams,
) ([]domain.StorePayroll, int64, domain.PayrollPeriod, error) {
	period, err := resolvePayrollPeriod(params.Year, params.Month)
	if err != nil {
		return nil, 0, period, domain.BadRequestError
	}

	// 1-qadam: do'konlarni sahifalash.
	// Count va Find uchun query alohida quriladi — GORM'da finisher (Count)
	// chaqirilgandan keyin bir xil *gorm.DB'ni qayta ishlatish statement'ni ifloslantiradi.
	buildStoreQuery := func() *gorm.DB {
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
	if err := buildStoreQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count stores for payroll: %v", err)
		return nil, 0, period, domain.InternalServerError
	}

	var stores []struct {
		Id   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := buildStoreQuery().
		Select("id, name").
		Order("name").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&stores).Error; err != nil {
		s.log.Errorf("could not get stores for payroll: %v", err)
		return nil, 0, period, domain.InternalServerError
	}
	if len(stores) == 0 {
		return []domain.StorePayroll{}, totalCount, period, nil
	}

	storeIds := make([]string, 0, len(stores))
	for _, st := range stores {
		storeIds = append(storeIds, st.Id)
	}

	// 2-qadam: shu sahifadagi do'konlarning xodimlari uchun bitta so'rov
	rows, err := s.fetchPayrollRows(ctx, period, "e.store_id IN ?", "p.store_id IN ?", storeIds)
	if err != nil {
		return nil, 0, period, err
	}

	// 3-qadam: do'konlar bo'yicha guruhlash (bo'sh do'kon ham ro'yxatda qoladi)
	byStore := make(map[string][]domain.EmployeePayrollRow, len(stores))
	for _, r := range rows {
		if r.StoreId == nil {
			continue
		}
		byStore[*r.StoreId] = append(byStore[*r.StoreId], r)
	}

	res := make([]domain.StorePayroll, 0, len(stores))
	for _, st := range stores {
		sp := domain.StorePayroll{
			StoreId:   st.Id,
			StoreName: st.Name,
			Employees: byStore[st.Id],
		}
		if sp.Employees == nil {
			sp.Employees = []domain.EmployeePayrollRow{}
		}
		sp.EmployeeCount = len(sp.Employees)
		for _, e := range sp.Employees {
			sp.WorkedHours += e.WorkedHours
			sp.SalaryRateAmount += e.SalaryRateAmount
			sp.ActualSalaryAmount += e.ActualSalaryAmount
			sp.IndividualSalesAmount += e.IndividualSalesAmount
			sp.KpiAmount += e.KpiAmount
			sp.BonusAmount += e.BonusAmount
			sp.GrossSalaryAmount += e.GrossSalaryAmount
			sp.NetPayAmount += e.NetPayAmount
		}
		// Do'kon plani xodimlar bo'yicha takrorlanadi — yig'indi emas, bitta qiymat
		if len(sp.Employees) > 0 {
			sp.StorePlanAmount = sp.Employees[0].StorePlanAmount
		}
		res = append(res, sp)
	}

	return res, totalCount, period, nil
}

// GetMyPayroll — API 2: token egasining o'z oyligi.
func (s *Services) GetMyPayroll(
	ctx context.Context, employeeId string, year, month int,
) (*domain.MyPayrollResponse, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		return nil, domain.BadRequestError
	}

	rows, err := s.fetchPayrollRows(ctx, period, "e.id = ?", "p.employee_id = ?", employeeId)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.NotFoundError
	}

	return &domain.MyPayrollResponse{Period: period, Payroll: rows[0]}, nil
}

// AutoCreateMonthlyPayrolls — har oyning 1-kuni cron tomonidan chaqiriladi.
// Tugagan o'tgan oy uchun har bir xodimning yakuniy ko'rsatkichlarini hisoblab,
// employee_payrolls'ga yozib qo'yadi (store_target'lardagi kabi snapshot).
// UNIQUE(employee_id, year, month) tufayli qayta ishga tushsa dublikat yaratmaydi.
func (s *Services) AutoCreateMonthlyPayrolls() {
	ctx := context.Background()

	prev := tashkentNow().AddDate(0, -1, 0)
	year, month := prev.Year(), int(prev.Month())

	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("cron AutoCreateMonthlyPayrolls: %v", err)
		return
	}
	// O'tgan oy — to'liq oy bo'yicha hisoblanishi shart
	period.IsLive = true

	query := fmt.Sprintf(payrollLiveSQLTemplate, "TRUE")
	var rows []domain.EmployeePayrollRow
	err = s.db.WithContext(ctx).Raw(query,
		period.From, period.To,
		period.From, period.To,
		year, month,
		constants.GeneralStatusActive,
	).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("cron AutoCreateMonthlyPayrolls: could not calculate: %v", err)
		return
	}

	created := 0
	for _, r := range rows {
		payroll := domain.EmployeePayroll{
			Id:                    uuid.New().String(),
			EmployeeId:            r.EmployeeId,
			StoreId:               r.StoreId,
			PositionSnapshot:      r.PositionSnapshot,
			ExperienceYears:       r.ExperienceYears,
			WorkedHours:           r.WorkedHours,
			AvgMonthlyHours:       r.AvgMonthlyHours,
			SalaryRateAmount:      r.SalaryRateAmount,
			ActualSalaryAmount:    r.ActualSalaryAmount,
			IndividualSalesAmount: r.IndividualSalesAmount,
			StoreTargetId:         r.StoreTargetId,
			StorePlanAmount:       r.StorePlanAmount,
			KpiPercent:            r.KpiPercent,
			KpiAmount:             r.KpiAmount,
			BonusAmount:           r.BonusAmount,
			GrossSalaryAmount:     r.GrossSalaryAmount,
			NetPayAmount:          r.NetPayAmount,
			Status:                domain.EmployeePayrollStatusDraft,
			Year:                  year,
			Month:                 month,
		}

		// UNIQUE(employee_id, year, month) buzilsa jim o'tkazib yuboradi —
		// cron takror ishga tushsa mavjud yozuvni buzmaydi
		err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&payroll).Error
		if err != nil {
			s.log.Errorf("cron AutoCreateMonthlyPayrolls: employee %s: %v", r.EmployeeId, err)
			continue
		}
		created++
	}

	s.log.Infof("AutoCreateMonthlyPayrolls completed for %d-%02d: %d created", year, month, created)
}
