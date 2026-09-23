package domain

import "time"

// Reserve — rezerv hujjati (imports kabi): do'kon 1C rezerv ro'yxati asosida yig'adigan hujjat.
type Reserve struct {
	Id                string          `json:"id" gorm:"column:id;primaryKey"`
	DokNumber         string          `json:"dok_number" gorm:"column:dok_number"`
	StoreId           string          `json:"store_id" gorm:"column:store_id"`
	StoreName         string          `json:"store_name" gorm:"column:store_name"`
	Status            string          `json:"status" gorm:"column:status"`
	TotalQuantity     float64         `json:"total_quantity" gorm:"column:total_quantity"`
	TotalProductCount int             `json:"total_product_count" gorm:"column:total_product_count"`
	Comment           string          `json:"comment" gorm:"column:comment"`
	CreatedBy         string          `json:"created_by" gorm:"column:created_by"`
	CreatedByName     string          `json:"created_by_name" gorm:"column:created_by_name"`
	UpdatedBy         string          `json:"updated_by" gorm:"column:updated_by"`
	UpdatedByName     string          `json:"updated_by_name" gorm:"column:updated_by_name"`
	CompletedAt       *time.Time      `json:"completed_at" gorm:"column:completed_at"`
	CreatedAt         *time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         *time.Time      `json:"updated_at" gorm:"column:updated_at"`
	Details           []ReserveDetail `json:"details,omitempty" gorm:"-"`
}

func (Reserve) TableName() string {
	return "reserves"
}

// ReserveDetail — hujjatdagi bitta mahsulot qatori.
type ReserveDetail struct {
	Id                string     `json:"id" gorm:"column:id;primaryKey"`
	ReserveId         string     `json:"reserve_id" gorm:"column:reserve_id"`
	ProductId         string     `json:"product_id" gorm:"column:product_id"`
	ReservedProductId string     `json:"reserved_product_id" gorm:"column:reserved_product_id"`
	MaterialCode      string     `json:"material_code" gorm:"column:material_code"`
	ProductName       string     `json:"product_name" gorm:"column:product_name"`
	Barcode           string     `json:"barcode" gorm:"column:barcode"`
	UnitPerPack       int        `json:"unit_per_pack" gorm:"column:unit_per_pack"`
	Quantity          float64    `json:"quantity" gorm:"column:quantity"`
	CheckedQuantity   float64    `json:"checked_quantity" gorm:"column:checked_quantity"`
	AvailableQuantity float64    `json:"available_quantity" gorm:"column:available_quantity"`
	CreatedBy         string     `json:"created_by" gorm:"column:created_by"`
	UpdatedBy         string     `json:"updated_by" gorm:"column:updated_by"`
	UpdatedByName     string     `json:"updated_by_name" gorm:"column:updated_by_name"`
	CreatedAt         *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ReserveDetail) TableName() string {
	return "reserve_details"
}

// ReserveRequest — hujjat yaratish. store_id bo'sh bo'lsa token'dagi do'kon olinadi,
// dok_number bo'sh bo'lsa baza avtomatik beradi (RZ-1000, RZ-1001, ...).
type ReserveRequest struct {
	StoreId   string               `json:"store_id"`
	DokNumber string               `json:"dok_number"`
	Comment   string               `json:"comment"`
	Items     []ReserveItemRequest `json:"items"`
}

type ReserveItemRequest struct {
	ProductId string  `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"gte=0"`
}

// ReserveStatusRequest — new -> checking -> done. done bo'lgan hujjat o'zgarmaydi.
type ReserveStatusRequest struct {
	Status string `json:"status" binding:"required" example:"checking"`
}

// ReserveDetailRequest — hujjatga mahsulot qo'shish yoki miqdorini yangilash (upsert).
type ReserveDetailRequest struct {
	ReserveId string  `json:"reserve_id" binding:"required"`
	ProductId string  `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"gte=0"`
}

// ReserveQuickDetailRequest — hujjat ochmasdan qo'shish: do'konning ochiq hujjati o'zi
// topiladi (yo'q bo'lsa yangisi ochiladi), miqdor esa mavjud qator ustiga qo'shiladi.
// store_id bo'sh bo'lsa token'dagi do'kon olinadi.
type ReserveQuickDetailRequest struct {
	StoreId   string  `json:"store_id"`
	ProductId string  `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
}

// ReserveDetailUpdateRequest — faqat berilgan maydon yangilanadi.
type ReserveDetailUpdateRequest struct {
	Quantity        *float64 `json:"quantity"`
	CheckedQuantity *float64 `json:"checked_quantity"`
}

type ReserveQueryParams struct {
	StoreId   string `form:"store_id"`
	Status    string `form:"status"`
	Search    string `form:"search"` // dok_number yoki izoh bo'yicha
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

type ReserveDetailQueryParams struct {
	ReserveId string `form:"reserve_id" binding:"required"`
	Search    string `form:"search"` // mahsulot nomi yoki material_code
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}
