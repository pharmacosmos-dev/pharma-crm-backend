package domain

import (
	"fmt"
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
//	RoleType           — FAQAT employees. employee_payrolls'da bunday ustun yo'q
//	                     (u yerdagi role/role_names roles jadvalidan keladi va
//	                     boshqa tushuncha). Hisob-kitobga ta'sir qilmaydi.
//	Avanslar           — faqat employee_payrolls, ular shu oyga tegishli.
type EmployeePayrollAdvanceRequest struct {
	KpiPercent *float64 `json:"kpi_percent" binding:"omitempty,min=0"`
	Salary     *float64 `json:"salary" binding:"omitempty,min=0"`
	// RoleType — xodimning tizimdagi roli (employees.role_type), masalan
	// "CASHIER", "HEADOFCASHIER", "MANAGER". Berilsa xodim kartochkasida
	// yangilanadi va keyingi oylarga ham amal qiladi.
	RoleType *string `json:"role_type" binding:"omitempty,max=55" example:"CASHIER"`
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
		r.ShiftType == nil && r.RoleType == nil &&
		r.AdvanceCardAmount == nil && r.AdvanceCashAmount == nil
}

// TouchesEmployee — employees jadvali ham yangilanishi kerakligini bildiradi.
func (r EmployeePayrollAdvanceRequest) TouchesEmployee() bool {
	return r.KpiPercent != nil || r.Salary != nil ||
		r.DailyWorkHours != nil || r.ShiftType != nil || r.RoleType != nil
}

// EmployeePayrollAdvanceQueryParams — tahrirlash ro'yxatining filtrlari.
// Year/Month berilmasa joriy oy olinadi; kelajakdagi oy qabul qilinmaydi.
type EmployeePayrollAdvanceQueryParams struct {
	StoreId string `form:"store_id"`
	Search  string `form:"search"`
	Year    int    `form:"year"`
	Month   int    `form:"month"`
	// Date — "YYYY-MM-DD" (yoki RFC3339). Berilsa Year/Month shundan olinadi va
	// alohida berilgan year/month e'tiborga olinmaydi. Payroll qatorlari oylik
	// bo'lgani uchun kun qismi faqat oyni aniqlash uchun ishlatiladi.
	Date   string `form:"date"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`

	CompanyId string `form:"-"`
}

// ApplyDate — Date berilgan bo'lsa undan Year/Month'ni ajratib oladi.
// Bo'sh bo'lsa hech narsa qilmaydi (year/month o'z holicha qoladi).
func (p *EmployeePayrollAdvanceQueryParams) ApplyDate() error {
	year, month, err := yearMonthFromDate(p.Date)
	if err != nil || year == 0 {
		return err
	}
	p.Year, p.Month = year, month
	return nil
}

// yearMonthFromDate — "YYYY-MM-DD", "YYYY-MM" yoki RFC3339 satridan yil va oyni
// ajratadi. Bo'sh satrda (0, 0, nil) qaytadi — "sana berilmagan" degani.
func yearMonthFromDate(date string) (int, int, error) {
	if date == "" {
		return 0, 0, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01"} {
		if t, err := time.Parse(layout, date); err == nil {
			return t.Year(), int(t.Month()), nil
		}
	}
	return 0, 0, fmt.Errorf("date must be YYYY-MM-DD: %s", date)
}

// PayrollManagementStatistics — oylik tahrirlash ro'yxatining yig'ma ko'rsatkichlari.
// Ro'yxat bilan BIR XIL filtrlardan o'tadi (store_id, search, davr, kompaniya va
// "Кассир"/"Заведующий" doirasi), shuning uchun raqamlar
// ekrandagi ro'yxatga mos keladi — sahifalashdan qat'i nazar hammasi bo'yicha.
type PayrollManagementStatistics struct {
	TotalStores    int64 `json:"total_stores"`
	TotalEmployees int64 `json:"total_employees"`
	// TotalSalary — employees.salary yig'indisi, ya'ni oylik fond stavkasi.
	// Ishlagan soatga bog'liq emas (actual_salary_amount'dan farqli).
	TotalSalary float64 `json:"total_salary"`
	// TotalAdvanceAmount — karta va naqd avanslar birga.
	TotalAdvanceAmount float64 `json:"total_advance_amount"`

	// RoleTypeCounts — employees.role_type bo'yicha xodimlar soni. Kalitlar
	// bazadagi haqiqiy qiymatlar ("CASHIER", "HEADOFCASHIER", "ROP_APTEKA",
	// "INTERN", ...), shuning uchun yangi rol qo'shilsa kod o'zgarmaydi va
	// hech kim sanoqdan tushib qolmaydi.
	//
	// role_type to'ldirilmagan xodimlar bo'sh kalit ("") ostida turadi — shu
	// sababli qiymatlar yig'indisi doim TotalEmployees'ga teng.
	//
	// gorm:"-" shart: asosiy so'rov bitta qator qaytaradi, map esa alohida
	// GROUP BY so'rovdan to'ldiriladi.
	RoleTypeCounts map[string]int64 `json:"role_type_counts" gorm:"-"`
}

type EmployeePayrollAdvanceRow struct {
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
	// Date — "YYYY-MM-DD" (yoki YYYY-MM / RFC3339). Berilsa Year/Month shundan
	// olinadi. Payroll qatorlari oylik, shuning uchun kun qismi faqat oyni
	// aniqlash uchun ishlatiladi.
	Date   string `form:"date"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`

	CompanyId string `form:"-"`
}

// ApplyDate — Date berilgan bo'lsa undan Year/Month'ni ajratib oladi.
func (p *EmployeePayrollQueryParams) ApplyDate() error {
	year, month, err := yearMonthFromDate(p.Date)
	if err != nil || year == 0 {
		return err
	}
	p.Year, p.Month = year, month
	return nil
}

// PayrollStatistics — xodimlar hisoboti (/employee/payroll/employees) bo'yicha
// yig'ma ko'rsatkichlar. Ro'yxat bilan BIR XIL filtrlardan o'tadi, lekin
// sahifalanmaydi — limit/offset ta'sir qilmaydi.
//
// DIQQAT: store_plan, store_sales va expected_plan do'kon darajasidagi
// qiymatlar bo'lib, do'konning HAR BIR xodim qatorida takrorlanadi. Ularni
// oddiy SUM qilib bo'lmaydi — do'kondagi xodimlar soniga ko'payib ketardi.
// Shuning uchun ular avval do'kon bo'yicha yig'ilib, keyin qo'shiladi.
type PayrollStatistics struct {
	TotalEmployeesCount int64 `json:"total_employees_count"`
	TotalStoresCount    int64 `json:"total_stores_count"`

	TotalWorkedHours     float64 `json:"total_worked_hours"`
	TotalAvgMonthlyHours float64 `json:"total_avg_monthly_hours"`

	TotalSalaryRateAmount   float64 `json:"total_salary_rate_amount"`
	TotalActualSalaryAmount float64 `json:"total_actual_salary_amount"`

	// Do'kon bo'yicha bir marta sanaladi (xodimlar soniga ko'paymaydi)
	TotalStorePlanAmount    float64 `json:"total_store_plan_amount"`
	TotalStoreSalesAmount   float64 `json:"total_store_sales_amount"`
	TotalExpectedPlanAmount float64 `json:"total_expected_plan_amount"`

	TotalKpiAmount         float64 `json:"total_kpi_amount"`
	TotalBonusAmount       float64 `json:"total_bonus_amount"`
	TotalGrossSalaryAmount float64 `json:"total_gross_salary_amount"`

	TotalAdvanceCardAmount float64 `json:"total_advance_card_amount"`
	TotalAdvanceCashAmount float64 `json:"total_advance_cash_amount"`

	TotalDeductionTermAmount    float64 `json:"total_deduction_term_amount"`
	TotalDeductionRecountAmount float64 `json:"total_deduction_recount_amount"`
	TotalDeductionFineAmount    float64 `json:"total_deduction_fine_amount"`

	TotalNetPayAmount float64 `json:"total_net_pay_amount"`
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
	// Role — roles jadvalidagi nom(lar)i, payroll qatoriga snapshot qilingan
	// ("Кассир", "Заведующий"). RoleType esa employees.role_type — tizimdagi
	// rol kodi ("CASHIER", "HEADOFCASHIER"). Ikkalasi turli tushuncha.
	//
	// RoleType snapshot emas, employees'dan jonli olinadi: u management
	// ekranidan tahrirlanadi va o'zgarish darhol ko'rinishi kerak.
	Role     string `json:"role"`
	RoleType string `json:"role_type"`

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
	// IsFranchise — do'kon kompaniyasi franshizami. Ro'yxat shu maydon bo'yicha
	// tartiblangan: franshiza do'konlari eng oxirida keladi.
	IsFranchise bool `json:"is_franchise"`

	EmployeeCount            int `json:"employee_count"`
	ActiveStoreEmployeeCount int `json:"active_store_employee_count"`
	PayrollCount             int `json:"payroll_count"`

	WorkedHours     float64 `json:"worked_hours"`
	AvgMonthlyHours float64 `json:"avg_monthly_hours"`

	SalaryRateAmount      float64 `json:"salary_rate_amount"`
	ActualSalaryAmount    float64 `json:"actual_salary_amount"`
	IndividualSalesAmount float64 `json:"individual_sales_amount"`
	StorePlanAmount       float64 `json:"store_plan_amount"`
	// StoreSalesAmount — do'konning savdosi. Xodim qatorlarida takrorlanadi,
	// shuning uchun yig'indi emas, bittasidan olinadi (MAX).
	StoreSalesAmount float64 `json:"store_sales_amount"`
	// KpiAmount — xodimlar KPI'sining yig'indisi, ya'ni do'konning umumiy KPI xarajati.
	KpiAmount         float64 `json:"kpi_amount"`
	BonusAmount       float64 `json:"bonus_amount"`
	GrossSalaryAmount float64 `json:"gross_salary_amount"`
	// SalaryPercent — gross_salary_amount / store_sales_amount * 100, ya'ni oylik
	// xarajati do'kon aylanmasining necha foizini tashkil qiladi. Faqat javobda
	// hisoblanadi, bazada bunday ustun yo'q. Savdo 0 bo'lsa 0 qaytadi.
	SalaryPercent float64 `json:"salary_percent"`
	AdvanceAmount float64 `json:"advance_amount"`
	// TotalDeduction — uchala ushlab qolishning yig'indisi: deduction_term +
	// deduction_recount + deduction_fine.
	TotalDeduction float64 `json:"total_deduction"`
	NetPayAmount   float64 `json:"net_pay_amount"`
}

// StorePayrollStatistics — do'konlar ro'yxatining (/employee/payroll/stores)
// yig'ma ko'rsatkichlari. Ro'yxat bilan bir xil filtrlardan o'tadi, lekin
// sahifalanmaydi: limit/offset ta'sir qilmaydi, filtrga mos BARCHA do'konlar.
//
// Uchta xodim sanog'i uchta boshqa narsani bildiradi:
//
//	TotalEmployeeCount            — stores.employee_count yig'indisi (kartochkadagi son)
//	TotalActiveStoreEmployeeCount — haqiqatan faol xodimlar (employees'dan jonli)
//	TotalPayrollCount             — oylik hisobga kirgan xodimlar (rol + davomat doirasi)
//
// Odatda TotalPayrollCount <= TotalActiveStoreEmployeeCount: rol mos kelmagan
// yoki smenaga chiqmagan xodimlar hisobga kirmaydi.
type StorePayrollStatistics struct {
	TotalStoresCount              int64 `json:"total_stores_count"`
	TotalEmployeeCount            int64 `json:"total_employee_count"`
	TotalActiveStoreEmployeeCount int64 `json:"total_active_store_employee_count"`
	TotalPayrollCount             int64 `json:"total_payroll_count"`

	TotalWorkedHours     float64 `json:"total_worked_hours"`
	TotalAvgMonthlyHours float64 `json:"total_avg_monthly_hours"`

	TotalSalaryRateAmount      float64 `json:"total_salary_rate_amount"`
	TotalActualSalaryAmount    float64 `json:"total_actual_salary_amount"`
	TotalIndividualSalesAmount float64 `json:"total_individual_sales_amount"`

	// Do'kon darajasidagi qiymatlar — har bir do'kon bo'yicha BIR MARTA
	// sanaladi, xodimlar soniga ko'paymaydi.
	TotalStorePlanAmount  float64 `json:"total_store_plan_amount"`
	TotalStoreSalesAmount float64 `json:"total_store_sales_amount"`

	TotalKpiAmount         float64 `json:"total_kpi_amount"`
	TotalBonusAmount       float64 `json:"total_bonus_amount"`
	TotalGrossSalaryAmount float64 `json:"total_gross_salary_amount"`

	// TotalAdvanceAmount — karta va naqd avans birga.
	TotalAdvanceAmount float64 `json:"total_advance_amount"`
	// TotalDeductionAmount — uchala ushlab qolish birga: muddat, qayta hisob, jarima.
	TotalDeductionAmount float64 `json:"total_deduction_amount"`

	TotalNetPayAmount float64 `json:"total_net_pay_amount"`
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
