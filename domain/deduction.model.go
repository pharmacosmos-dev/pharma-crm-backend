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

	Amount  float64    `json:"amount" gorm:"column:amount"`
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
