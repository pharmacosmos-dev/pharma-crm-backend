package domain

import "time"

// EmployeePayroll — xodimning oylik ish haqi vedomosti (bir xodimga oyiga bitta qator).
// Summalar hisoblanganda: gross_salary_amount = actual_salary + kpi_amount + bonus_amount,
// net_pay_amount = gross_salary_amount - (advance_* + deduction_*).
type EmployeePayroll struct {
	Id               string  `json:"id" gorm:"column:id;primaryKey"`
	EmployeeId       string  `json:"employee_id" gorm:"column:employee_id"`
	StoreId          *string `json:"store_id" gorm:"column:store_id"`
	PositionSnapshot string  `json:"position_snapshot" gorm:"column:position_snapshot"`
	ExperienceYears  float64 `json:"experience_years" gorm:"column:experience_years"`
	WorkedHours      float64 `json:"worked_hours" gorm:"column:worked_hours"`
	AvgMonthlyHours  float64 `json:"avg_monthly_hours" gorm:"column:avg_monthly_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount" gorm:"column:salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount" gorm:"column:actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount" gorm:"column:individual_sales_amount"`

	StoreTargetId   *string `json:"store_target_id" gorm:"column:store_target_id"`
	StorePlanAmount float64 `json:"store_plan_amount" gorm:"column:store_plan_amount"`

	KpiPercent float64 `json:"kpi_percent" gorm:"column:kpi_percent"`
	KpiAmount  float64 `json:"kpi_amount" gorm:"column:kpi_amount"`

	BonusAmount       float64 `json:"bonus_amount" gorm:"column:bonus_amount"`
	GrossSalaryAmount float64 `json:"gross_salary_amount" gorm:"column:gross_salary_amount"`

	AdvanceCardAmount float64 `json:"advance_card_amount" gorm:"column:advance_card_amount"`
	AdvanceCashAmount float64 `json:"advance_cash_amount" gorm:"column:advance_cash_amount"`

	DeductionTermAmount    float64 `json:"deduction_term_amount" gorm:"column:deduction_term_amount"`
	DeductionRecountAmount float64 `json:"deduction_recount_amount" gorm:"column:deduction_recount_amount"`
	DeductionFineAmount    float64 `json:"deduction_fine_amount" gorm:"column:deduction_fine_amount"`

	NetPayAmount float64 `json:"net_pay_amount" gorm:"column:net_pay_amount"`

	Status     string  `json:"status" gorm:"column:status"`
	ApprovedBy *string `json:"approved_by" gorm:"column:approved_by"`
	Month      int     `json:"month" gorm:"column:month"`
	Year       int     `json:"year" gorm:"column:year"`

	CreatedAt   *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   *time.Time `json:"updated_at" gorm:"column:updated_at"`
	CompletedAt *time.Time `json:"completed_at" gorm:"column:completed_at"`

	Employee    *Employee    `json:"employee,omitempty" gorm:"foreignKey:EmployeeId"`
	Store       *Store       `json:"store,omitempty" gorm:"foreignKey:StoreId"`
	StoreTarget *StoreTarget `json:"store_target,omitempty" gorm:"foreignKey:StoreTargetId"`
	Approver    *Employee    `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
}

func (EmployeePayroll) TableName() string {
	return "employee_payrolls"
}

// employee_payrolls.status qiymatlari
const (
	EmployeePayrollStatusDraft     = "draft"
	EmployeePayrollStatusApproved  = "approved"
	EmployeePayrollStatusCompleted = "completed"
)

// Query params
type EmployeePayrollQueryParams struct {
	EmployeeId string `form:"employee_id"`
	StoreId    string `form:"store_id"`
	Status     string `form:"status"`
	Year       int    `form:"year"`
	Month      int    `form:"month"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`

	CompanyId string `form:"-"`
}

// EmployeePayrollRow — bitta xodimning oylik ko'rsatkichlari. Joriy oy uchun jonli
// hisoblanadi (attendance/bonus/target jadvallaridan), o'tgan oylar uchun
// employee_payrolls'dan o'qiladi — ikkalasi ham shu strukturaga tushadi.
type EmployeePayrollRow struct {
	EmployeeId       string  `json:"employee_id"`
	StoreId          *string `json:"store_id"`
	StoreName        string  `json:"store_name"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	FullName         string  `json:"full_name"`
	PositionSnapshot string  `json:"position_snapshot"`

	ExperienceYears float64 `json:"experience_years"`
	WorkedHours     float64 `json:"worked_hours"`
	AvgMonthlyHours float64 `json:"avg_monthly_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount"`

	StoreTargetId   *string `json:"store_target_id"`
	StorePlanAmount float64 `json:"store_plan_amount"`

	KpiPercent float64 `json:"kpi_percent"`
	KpiAmount  float64 `json:"kpi_amount"`

	BonusAmount       float64 `json:"bonus_amount"`
	GrossSalaryAmount float64 `json:"gross_salary_amount"`

	AdvanceCardAmount float64 `json:"advance_card_amount"`
	AdvanceCashAmount float64 `json:"advance_cash_amount"`

	DeductionTermAmount    float64 `json:"deduction_term_amount"`
	DeductionRecountAmount float64 `json:"deduction_recount_amount"`
	DeductionFineAmount    float64 `json:"deduction_fine_amount"`

	NetPayAmount float64 `json:"net_pay_amount"`

	Status      string     `json:"status"`
	Month       int        `json:"month"`
	Year        int        `json:"year"`
	CompletedAt *time.Time `json:"completed_at"`
}

// StorePayroll — do'kon va uning ichidagi xodimlar oyligi. API 1 shu shaklda qaytadi.
type StorePayroll struct {
	StoreId   string `json:"store_id"`
	StoreName string `json:"store_name"`

	EmployeeCount int     `json:"employee_count"`
	WorkedHours   float64 `json:"worked_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount"`
	StorePlanAmount       float64 `json:"store_plan_amount"`
	KpiAmount             float64 `json:"kpi_amount"`
	BonusAmount           float64 `json:"bonus_amount"`
	GrossSalaryAmount     float64 `json:"gross_salary_amount"`
	NetPayAmount          float64 `json:"net_pay_amount"`

	Employees []EmployeePayrollRow `json:"employees"`
}

// PayrollPeriod — so'ralgan davr haqidagi meta. is_live=true bo'lsa ma'lumot jonli
// hisoblangan (joriy oy, bugungi kungacha), false bo'lsa employee_payrolls'dan olingan.
type PayrollPeriod struct {
	Year   int    `json:"year"`
	Month  int    `json:"month"`
	From   string `json:"from"`
	To     string `json:"to"`
	IsLive bool   `json:"is_live"`
}

type MyPayrollResponse struct {
	Period  PayrollPeriod      `json:"period"`
	Payroll EmployeePayrollRow `json:"payroll"`
}
