package domain

import "time"

// Oylikdan ushlab qolishlar: shtraf, pereuchyot va boshqalar.
//
// Uch qatlam:
//
//	DeductionType   — turlar lug'ati (TERM, RECOUNT, FINE, ...)
//	Deduction       — do'kon + oy sarlavhasi: to'liq to'landimi yoki ochiqmi
//	DeductionDetail — xodim bo'yicha aniq summa
//
// Hozircha employee_payrolls'dan mustaqil: oylikdagi deduction_* ustunlari
// avvalgidek qo'lda kiritiladi.

// region Types

// DeductionType — ushlab qolish turi.
//
// Code — kodda ishlatiladigan barqaror kalit; Name tahrirlansa ham o'zgarmaydi,
// shuning uchun hisobotlar rol nomlaridagi kabi buzilib qolmaydi.
type DeductionType struct {
	Id        string     `json:"id" gorm:"column:id;primaryKey"`
	Code      string     `json:"code" gorm:"column:code"`
	Name      string     `json:"name" gorm:"column:name"`
	IsActive  bool       `json:"is_active" gorm:"column:is_active"`
	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (DeductionType) TableName() string { return "deduction_types" }

// Migratsiyada oldindan yaratilgan turlar. Yangilari API orqali qo'shiladi,
// shuning uchun bu ro'yxat to'liq emas — faqat employee_payrolls'dagi uchta
// ustunga mos keladiganlari.
const (
	DeductionCodeTerm    = "TERM"    // Срок
	DeductionCodeRecount = "RECOUNT" // Переучёт
	DeductionCodeFine    = "FINE"    // Штраф
)

// deductions.status qiymatlari
const (
	DeductionStatusOpen = "open" // hali to'liq yopilmagan
	DeductionStatusPaid = "paid" // to'liq to'langan
)

// region Deduction

// Deduction — bitta do'konning bitta oydagi ushlab qolishlar sarlavhasi.
//
// TotalAmount va PaidAmount saqlanadi (har safar detallardan qayta yig'ilmaydi):
// "qancha qoldi" darhol ko'rinadi va yopilgan oyning summasi keyin xodim
// qatorlari o'zgarsa ham siljib ketmaydi.
type Deduction struct {
	Id        string  `json:"id" gorm:"column:id;primaryKey"`
	StoreId   string  `json:"store_id" gorm:"column:store_id"`
	CompanyId *string `json:"company_id" gorm:"column:company_id"`
	Year      int     `json:"year" gorm:"column:year"`
	Month     int     `json:"month" gorm:"column:month"`

	TotalAmount float64 `json:"total_amount" gorm:"column:total_amount"`
	PaidAmount  float64 `json:"paid_amount" gorm:"column:paid_amount"`

	Status  string  `json:"status" gorm:"column:status"`
	Comment *string `json:"comment" gorm:"column:comment"`

	CreatedBy   *string    `json:"created_by" gorm:"column:created_by"`
	UpdatedBy   *string    `json:"updated_by" gorm:"column:updated_by"`
	ApprovedBy  *string    `json:"approved_by" gorm:"column:approved_by"`
	ApprovedAt  *time.Time `json:"approved_at" gorm:"column:approved_at"`
	CompletedAt *time.Time `json:"completed_at" gorm:"column:completed_at"`

	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Deduction) TableName() string { return "deductions" }

// region Detail

// DeductionDetail — xodimga yozilgan bitta ushlab qolish.
//
// StoreId/Year/Month sarlavhada ham bor, lekin bu yerda takrorlanadi: xodim
// bo'yicha hisobot sarlavhaga JOIN qilmasdan o'qiladi.
//
// Bir xodimga bir oyda bir necha qator bo'lishi mumkin (masalan ikkita shtraf) —
// shuning uchun UNIQUE cheklov qo'yilmagan.
type DeductionDetail struct {
	Id              string `json:"id" gorm:"column:id;primaryKey"`
	DeductionId     string `json:"deduction_id" gorm:"column:deduction_id"`
	DeductionTypeId string `json:"deduction_type_id" gorm:"column:deduction_type_id"`
	EmployeeId      string `json:"employee_id" gorm:"column:employee_id"`
	StoreId         string `json:"store_id" gorm:"column:store_id"`
	Year            int    `json:"year" gorm:"column:year"`
	Month           int    `json:"month" gorm:"column:month"`

	Amount float64 `json:"amount" gorm:"column:amount"`
	// MonthsCount — necha oyga bo'lib to'lanadi. Shuncha deduction_installments
	// qatori hosil bo'ladi.
	MonthsCount int `json:"months_count" gorm:"column:months_count"`
	// IsPaid — to'lov jadvalidan HOSILA: barcha to'lovlar to'langanda true
	// bo'ladi. Qo'lda o'rnatilmaydi.
	IsPaid  bool       `json:"is_paid" gorm:"column:is_paid"`
	PaidAt  *time.Time `json:"paid_at" gorm:"column:paid_at"`
	Comment *string    `json:"comment" gorm:"column:comment"`

	CreatedBy  *string    `json:"created_by" gorm:"column:created_by"`
	UpdatedBy  *string    `json:"updated_by" gorm:"column:updated_by"`
	ApprovedBy *string    `json:"approved_by" gorm:"column:approved_by"`
	ApprovedAt *time.Time `json:"approved_at" gorm:"column:approved_at"`

	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (DeductionDetail) TableName() string { return "deduction_details" }

// region Installments

// DeductionInstallment — qarzning bitta oyga to'g'ri keladigan to'lovi.
//
// EmployeeId/StoreId qarzda ham bor, lekin bu yerda takrorlanadi: "shu
// xodimning shu oydagi qarzi" so'rovi JOIN'siz o'qiladi — aynan shu so'rov
// har oy oylik ekranida ishlatiladi.
//
// Seq — nechanchi to'lov (1..months_count). Bo'linishda qolgan tiyinlar
// OXIRGI to'lovga qo'shiladi, shuning uchun to'lovlar yig'indisi qarz
// summasiga aniq teng bo'ladi.
type DeductionInstallment struct {
	Id                string `json:"id" gorm:"column:id;primaryKey"`
	DeductionDetailId string `json:"deduction_detail_id" gorm:"column:deduction_detail_id"`
	EmployeeId        string `json:"employee_id" gorm:"column:employee_id"`
	StoreId           string `json:"store_id" gorm:"column:store_id"`
	Year              int    `json:"year" gorm:"column:year"`
	Month             int    `json:"month" gorm:"column:month"`
	Seq               int    `json:"seq" gorm:"column:seq"`

	Amount  float64    `json:"amount" gorm:"column:amount"`
	IsPaid  bool       `json:"is_paid" gorm:"column:is_paid"`
	PaidAt  *time.Time `json:"paid_at" gorm:"column:paid_at"`
	Comment *string    `json:"comment" gorm:"column:comment"`

	UpdatedBy  *string    `json:"updated_by" gorm:"column:updated_by"`
	ApprovedBy *string    `json:"approved_by" gorm:"column:approved_by"`
	ApprovedAt *time.Time `json:"approved_at" gorm:"column:approved_at"`

	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (DeductionInstallment) TableName() string { return "deduction_installments" }

// EmployeeDebt — bitta xodimning qarz holati.
//
// CurrentMonth* — so'ralgan oyga to'g'ri keladigan to'lov: oylik ekranida
// "shu oyda qancha ushlab qolinadi" shu yerdan olinadi.
type EmployeeDebt struct {
	EmployeeId string `json:"employee_id"`

	TotalAmount     float64 `json:"total_amount"`     // jami qarz
	PaidAmount      float64 `json:"paid_amount"`      // to'langani
	RemainingAmount float64 `json:"remaining_amount"` // qolgani

	// So'ralgan oydagi to'lov. Qator bo'lmasa hammasi 0/false.
	CurrentMonthAmount   float64 `json:"current_month_amount"`
	CurrentMonthPaid     bool    `json:"current_month_paid"`
	CurrentMonthApproved bool    `json:"current_month_approved"`

	UnpaidCount int `json:"unpaid_count"` // qolgan to'lovlar soni
}

// region Requests

type DeductionTypeRequest struct {
	Code     string `json:"code" binding:"required,max=50" example:"SHORTAGE"`
	Name     string `json:"name" binding:"required,max=255" example:"Kamomad"`
	IsActive *bool  `json:"is_active"`
}

// DeductionRequest — do'kon+oy sarlavhasini yaratish.
//
// total_amount/paid_amount/status bu yerda yo'q: ular detallardan avtomatik
// hisoblanadi, qo'lda kiritilsa sarlavha va qatorlar bir-biriga mos kelmay
// qolardi.
type DeductionRequest struct {
	StoreId string  `json:"store_id" binding:"required"`
	Year    int     `json:"year" binding:"required,min=2000,max=2100"`
	Month   int     `json:"month" binding:"required,min=1,max=12"`
	Comment *string `json:"comment"`
}

// DeductionUpdateRequest — hammasi ixtiyoriy: berilgani yoziladi.
//
// Approve true bo'lsa approved_by joriy foydalanuvchiga, approved_at hozirgi
// vaqtga qo'yiladi. Tasdiqlashni bekor qilish uchun false yuboriladi.
type DeductionUpdateRequest struct {
	Comment *string `json:"comment"`
	Approve *bool   `json:"approve"`
}

func (r DeductionUpdateRequest) IsEmpty() bool {
	return r.Comment == nil && r.Approve == nil
}

// DeductionDetailRequest — xodimga qarz ajratish.
//
// MonthsCount — necha oyga bo'lib to'lanadi. Berilmasa 1 (bir oyda to'liq).
// Yaratilganda shuncha to'lov qatori avtomatik hosil bo'ladi: birinchisi
// sarlavha oyidan boshlanadi, qolganlari ketma-ket keyingi oylarga.
type DeductionDetailRequest struct {
	DeductionId     string  `json:"deduction_id" binding:"required"`
	DeductionTypeId string  `json:"deduction_type_id" binding:"required"`
	EmployeeId      string  `json:"employee_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,min=0"`
	MonthsCount     int     `json:"months_count" binding:"omitempty,min=1" example:"4"`
	Comment         *string `json:"comment"`
}

// DeductionInstallmentUpdateRequest — bitta oylik to'lovni tahrirlash.
//
// Summani bu yerdan o'zgartirib bo'lmaydi: to'lovlar yig'indisi qarz
// summasiga teng bo'lishi kerak, bittasini o'zgartirsa bu buzilardi.
// Summani o'zgartirish uchun qarzning o'zi (amount yoki months_count)
// yangilanadi va jadval qaytadan tuziladi.
type DeductionInstallmentUpdateRequest struct {
	IsPaid  *bool   `json:"is_paid"`
	Approve *bool   `json:"approve"`
	Comment *string `json:"comment"`
}

func (r DeductionInstallmentUpdateRequest) IsEmpty() bool {
	return r.IsPaid == nil && r.Approve == nil && r.Comment == nil
}

type DeductionInstallmentQueryParams struct {
	DeductionDetailId string `form:"deduction_detail_id"`
	EmployeeId        string `form:"employee_id"`
	StoreId           string `form:"store_id"`
	IsPaid            *bool  `form:"is_paid"`
	Year              int    `form:"year"`
	Month             int    `form:"month"`
	Limit             int    `form:"limit"`
	Offset            int    `form:"offset"`
}

// DeductionDetailUpdateRequest — hammasi ixtiyoriy.
//
// store_id/year/month bu yerda yo'q: ular sarlavhadan olinadi va qatorni
// boshqa oyga ko'chirish sarlavha yig'indilarini buzardi. Boshqa oyga
// ko'chirish kerak bo'lsa qator o'chirilib, yangisi yaratiladi.
// is_paid bu yerda YO'Q: u to'lov jadvalidan hosila. Oylik to'lovni to'langan
// deb belgilash uchun PUT /deduction-installment/{id} ishlatiladi.
//
// Amount yoki MonthsCount o'zgarsa to'lov jadvali qaytadan tuziladi. Shuning
// uchun allaqachon to'langan to'lovi bor qarzda ular o'zgartirilmaydi — aks
// holda to'lov tarixi yo'qolardi.
type DeductionDetailUpdateRequest struct {
	DeductionTypeId *string  `json:"deduction_type_id"`
	Amount          *float64 `json:"amount" binding:"omitempty,min=0"`
	MonthsCount     *int     `json:"months_count" binding:"omitempty,min=1"`
	Comment         *string  `json:"comment"`
	Approve         *bool    `json:"approve"`
}

func (r DeductionDetailUpdateRequest) IsEmpty() bool {
	return r.DeductionTypeId == nil && r.Amount == nil && r.MonthsCount == nil &&
		r.Comment == nil && r.Approve == nil
}

// RebuildsSchedule — to'lov jadvali qaytadan tuzilishi kerakligini bildiradi.
func (r DeductionDetailUpdateRequest) RebuildsSchedule() bool {
	return r.Amount != nil || r.MonthsCount != nil
}

// region Query params

type DeductionQueryParams struct {
	StoreId string `form:"store_id"`
	Status  string `form:"status"`
	Year    int    `form:"year"`
	Month   int    `form:"month"`
	Date    string `form:"date"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`

	CompanyId string `form:"-"`
}

type DeductionDetailQueryParams struct {
	DeductionId     string `form:"deduction_id"`
	EmployeeId      string `form:"employee_id"`
	StoreId         string `form:"store_id"`
	DeductionTypeId string `form:"deduction_type_id"`
	IsPaid          *bool  `form:"is_paid"`
	Year            int    `form:"year"`
	Month           int    `form:"month"`
	Date            string `form:"date"`
	Limit           int    `form:"limit"`
	Offset          int    `form:"offset"`

	CompanyId string `form:"-"`
}
