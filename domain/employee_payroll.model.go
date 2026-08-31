package domain

import (
	"time"

	"github.com/lib/pq"
)

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

	StoreTargetId    *string `json:"store_target_id" gorm:"column:store_target_id"`
	StorePlanAmount  float64 `json:"store_plan_amount" gorm:"column:store_plan_amount"`
	StoreSalesAmount float64 `json:"store_sales_amount" gorm:"column:store_sales_amount"`

	PlanKpiPercent     float64 `json:"plan_kpi_percent" gorm:"column:plan_kpi_percent"`
	EmployeeKpiPercent float64 `json:"employee_kpi_percent" gorm:"column:employee_kpi_percent"`
	KpiPercent         float64 `json:"kpi_percent" gorm:"column:kpi_percent"`
	KpiAmount          float64 `json:"kpi_amount" gorm:"column:kpi_amount"`

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

// EmployeePayrollAdvanceRequest — oylik kartochkasidagi qo'lda kiritiladigan
// maydonlar. Hammasi ixtiyoriy: berilgani yoziladi, berilmagani eski qiymatida
// qoladi. Hech biri bo'lmasa so'rov rad etiladi.
//
// Qaysi maydon qaysi jadvalga tushadi:
//
//	KpiPercent, Salary — IKKALASI ham: employees (xodim kartochkasi, keyingi
//	                     oylarga ta'sir qiladi) va employee_payrolls (shu oy).
//	DailyWorkHours     — FAQAT employees. employee_payrolls'da bunday ustun yo'q,
//	                     lekin oylik normasiga ta'sir qiladi: berilsa payroll
//	                     qatoridagi avg_monthly_hours qayta hisoblanadi.
//	ShiftType          — FAQAT employees. Hisob-kitobga umuman kirmaydi.
//	Avanslar           — faqat employee_payrolls, ular shu oyga tegishli.
type EmployeePayrollAdvanceRequest struct {
	KpiPercent *float64 `json:"kpi_percent" binding:"omitempty,min=0"`
	Salary     *float64 `json:"salary" binding:"omitempty,min=0"`
	// DailyWorkHours — faqat 4, 7 yoki 8 soat. Tip ataylab int: validator'ning
	// `oneof` qoidasi float maydonda panic beradi (Bad field type float64).
	DailyWorkHours *int `json:"daily_work_hours" binding:"omitempty,oneof=4 7 8" example:"8"`
	// ShiftType — smena turi: "day" (День) yoki "night" (Ночь).
	ShiftType         *string  `json:"shift_type" binding:"omitempty,oneof=day night" example:"night"`
	AdvanceCardAmount *float64 `json:"advance_card_amount" binding:"omitempty,min=0"`
	AdvanceCashAmount *float64 `json:"advance_cash_amount" binding:"omitempty,min=0"`
}

// IsEmpty — hech qanday maydon berilmaganini bildiradi.
func (r EmployeePayrollAdvanceRequest) IsEmpty() bool {
	return r.KpiPercent == nil && r.Salary == nil && r.DailyWorkHours == nil &&
		r.ShiftType == nil && r.AdvanceCardAmount == nil && r.AdvanceCashAmount == nil
}

// TouchesEmployee — employees jadvali ham yangilanishi kerakligini bildiradi.
func (r EmployeePayrollAdvanceRequest) TouchesEmployee() bool {
	return r.KpiPercent != nil || r.Salary != nil ||
		r.DailyWorkHours != nil || r.ShiftType != nil
}

// EmployeePayrollAdvanceQueryParams — tahrirlash ro'yxatining filtrlari.
// Year/Month berilmasa joriy oy olinadi; kelajakdagi oy qabul qilinmaydi.
type EmployeePayrollAdvanceQueryParams struct {
	StoreId string `form:"store_id"`
	Search  string `form:"search"`
	Year    int    `form:"year"`
	Month   int    `form:"month"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`

	CompanyId string `form:"-"`
}

// EmployeePayrollAdvanceRow — tahrirlash ro'yxatining bitta qatori:
// xodim kartochkasidagi qiymatlar + shu oyning payroll qatoridan olingan
// kpi_percent va avanslar.
type EmployeePayrollAdvanceRow struct {
	// Id — employee_payrolls qatorining id'si, update shu bo'yicha ketadi.
	Id         string         `json:"id"`
	EmployeeId string         `json:"employee_id"`
	RoleType   string         `json:"role_type"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Phone      string         `json:"phone"`
	StoreName  *string        `json:"store_name"`
	Roles      pq.StringArray `json:"roles" gorm:"type:text[]" swaggertype:"array,string"`

	// KpiPercent employee_payrolls'dan olinadi — shu oyda AMALDA ishlatilgan foiz.
	KpiPercent      float64 `json:"kpi_percent"`
	Salary          float64 `json:"salary"`
	DailyWorkHours  float64 `json:"daily_work_hours"`
	ShiftType       *string `json:"shift_type"`
	ExperienceYears float64 `json:"experience_years"`

	AdvanceCardAmount float64 `json:"advance_card_amount"`
	AdvanceCashAmount float64 `json:"advance_cash_amount"`
}

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
	// Id — employee_payrolls qatorining id'si. Avansni yangilash uchun kerak:
	// PUT /employee/payroll/{id}/advance.
	Id               string  `json:"id"`
	EmployeeId       string  `json:"employee_id"`
	StoreId          *string `json:"store_id"`
	StoreName        string  `json:"store_name"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	FullName         string  `json:"full_name"`
	PositionSnapshot string  `json:"position_snapshot"`
	Role             string  `json:"role"`

	ExperienceYears float64 `json:"experience_years"`
	WorkedHours     float64 `json:"worked_hours"`
	AvgMonthlyHours float64 `json:"avg_monthly_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount"`

	StoreTargetId    *string `json:"store_target_id"`
	StorePlanAmount  float64 `json:"store_plan_amount"`  // do'konning to'liq oylik rejasi
	StoreSalesAmount float64 `json:"store_sales_amount"` // do'konning shu kungacha savdosi

	// Reja bajarilishi DO'KON bo'yicha o'lchanadi: do'kon rejasi o'tgan ish
	// kunlariga proporsional kesiladi va do'kon savdosi shunga solishtiriladi.
	// Quyidagi uchtasi do'kondagi barcha xodimlarda BIR XIL bo'ladi — ular
	// kpi_percent'ni belgilaydi. KPI SUMMASI esa har kimning o'z savdosidan
	// hisoblanadi, ya'ni kpi_amount xodimdan xodimga farq qiladi.
	//
	// employee_plan_amount hisobga kirmaydi, faqat ma'lumot uchun qoladi.
	EmployeePlanAmount     float64 `json:"employee_plan_amount"`     // xodimning to'liq oylik rejasi (ma'lumot uchun)
	ExpectedPlanAmount     float64 `json:"expected_plan_amount"`     // do'konning shu kungacha kutilgan rejasi
	PlanAchievementPercent float64 `json:"plan_achievement_percent"` // store_sales / expected_plan * 100
	MonthWorkDays          int     `json:"month_work_days"`          // oydagi jami ish kuni
	ElapsedWorkDays        int     `json:"elapsed_work_days"`        // o'tgan ish kuni

	// KPI foizining ikkala manbai ham qaytadi — frontend qaysi biri ishlaganini
	// employee_kpi_percent > 0 sharti bilan aniqlaydi va ikkalasini ko'rsata oladi.
	// Foizlarni bir-biriga solishtirib aniqlab bo'lmaydi: qo'lda kiritilgan qiymat
	// pog'ona qiymatiga teng bo'lib qolishi mumkin.
	PlanKpiPercent     float64 `json:"plan_kpi_percent"`     // reja bajarilishidan chiqqan pog'ona
	EmployeeKpiPercent float64 `json:"employee_kpi_percent"` // xodim kartochkasidagi foiz, 0 = kiritilmagan
	KpiPercent         float64 `json:"kpi_percent"`          // AMALDAGI foiz — kpi_amount shundan hisoblangan
	KpiAmount          float64 `json:"kpi_amount"`

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
	// CalculatedAt — cron oxirgi marta qachon qayta hisoblagani.
	CalculatedAt *time.Time `json:"calculated_at"`
}

// StorePayroll — do'kon va uning ichidagi xodimlar oyligi. API 1 shu shaklda qaytadi.
type StorePayroll struct {
	StoreId   string `json:"store_id"`
	StoreName string `json:"store_name"`

	// EmployeeCount — do'konga biriktirilgan faol xodimlar soni (employees
	// jadvalidan). Cron ishlagan-ishlamaganidan qat'i nazar to'g'ri qiymat.
	EmployeeCount int `json:"employee_count"`
	// PayrollCount — shu oy uchun payroll qatori bor xodimlar soni. Quyidagi
	// summalar aynan shu xodimlar bo'yicha yig'ilgan. EmployeeCount'dan kichik
	// bo'lsa, demak ba'zi xodimlar hali hisobga tushmagan (masalan oy o'rtasida
	// qo'shilgan va cron hali ishlamagan).
	PayrollCount int `json:"payroll_count"`

	WorkedHours float64 `json:"worked_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount"`
	StorePlanAmount       float64 `json:"store_plan_amount"`
	// StoreSalesAmount — do'konning savdosi. Xodim qatorlarida takrorlanadi,
	// shuning uchun yig'indi emas, bittasidan olinadi (MAX).
	StoreSalesAmount float64 `json:"store_sales_amount"`
	// KpiAmount — xodimlar KPI'sining yig'indisi, ya'ni do'konning umumiy KPI xarajati.
	KpiAmount float64 `json:"kpi_amount"`
	BonusAmount           float64 `json:"bonus_amount"`
	GrossSalaryAmount     float64 `json:"gross_salary_amount"`
	NetPayAmount          float64 `json:"net_pay_amount"`
}

// Eslatma: bu strukturada xodimlar ro'yxati YO'Q va bo'lmasligi kerak.
// Do'kon qatorlari GORM'ning Scan'i orqali to'ldiriladi; ichida []struct maydon
// bo'lsa GORM uni bog'lanish (relation) deb hisoblab, foreign key talab qiladi
// va so'rov "invalid field found for struct ... define a valid foreign key"
// xatosi bilan yiqiladi. Xodimlar alohida /employee/payroll/employees
// endpointidan store_id bo'yicha olinadi.

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

// PayrollRecalcResult — qo'lda qayta hisoblash natijasi. Cron tunda ishlamay
// qolganda ertalab nima qilinganini ko'rish uchun.
type PayrollRecalcResult struct {
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	CalculatedTo string `json:"calculated_to"` // hisob shu kungacha olib borildi
	RowsAffected int64  `json:"rows_affected"`
	DurationMs   int64  `json:"duration_ms"`
}
