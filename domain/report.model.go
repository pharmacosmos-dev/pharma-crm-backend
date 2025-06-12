package domain

import "time"

type ReportQueryParam struct {
	StoreId    string   `form:"store_id"`
	StartDate  string   `form:"start_date"`
	EndDate    string   `form:"end_date"`
	Limit      int      `form:"limit"`
	Offset     int      `form:"offset"`
	Search     string   `form:"search"`
	Order      string   `form:"order"`
	EmployeeId string   `form:"employee_id"`
	ProducerId string   `form:"producer_id"`
	StoreIds   []string `json:"store_ids"`
}

// Bonus report structure
type BonusReport struct {
	Id         string  `gorm:"id" json:"id"`
	PublicId   int     `gorm:"public_id" json:"public_id"`
	FullName   string  `gorm:"full_name" json:"full_name"`
	Phone      string  `gorm:"phone" json:"phone"`
	StoreName  string  `gorm:"store_name" json:"store_name"`
	Role       string  `gorm:"role" json:"role"`
	Amount     float64 `gorm:"amount" json:"amount"`
	Count      float64 `gorm:"count" json:"count"`
	TotalCount int64   `gorm:"total_count" json:"-"`
}

// get product report
type ProductReport struct {
	CartItemId     string     `gorm:"cart_item_id" json:"cart_item_id"`
	MaterialCode   int        `gorm:"material_code" json:"material_code"`
	StoreName      string     `gorm:"store_name" json:"store_name"`
	ProductName    string     `gorm:"product_name" json:"product_name"`
	ProducerName   string     `gorm:"producer_name" json:"producer_name"`
	SerialNumber   string     `gorm:"serial_number" json:"serial_number"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity       float64    `gorm:"quantity" json:"quantity"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPriceSum float64    `gorm:"supply_price_sum" json:"supply_price_sum"`
	RetailPriceSum float64    `gorm:"retail_price_sum" json:"retail_price_sum"`
	MarkupSum      float64    `gorm:"markup_sum" json:"markup_sum"`
	VatSum         float64    `gorm:"vat_sum" json:"vat_sum"`
	CompletedAt    *time.Time `gorm:"completed_at" json:"completed_at"`
	FullName       string     `gorm:"full_name" json:"full_name"`
	SaleNumber     int        `gorm:"sale_number" json:"sale_number"`
	SaleType       string     `gorm:"sale_type" json:"sale_type"`
	MarkingCount   int        `gorm:"marking_count" json:"marking_count"`
	TotalCount     int64      `gorm:"total_count" json:"-"`
}

// lfl report structure
type LflReport struct {
	FirstMonth  []LflReportDetail `json:"first_month"`
	SecondMonth []LflReportDetail `json:"second_month"`
}

type LflReportDetail struct {
	Id            int     `gorm:"id" json:"id"`
	Weekdate      string  `gorm:"weekdate" json:"weekdate"`
	Weekname      string  `gorm:"weekname" json:"weekname"`
	BranchCount   int     `gorm:"branch_count" json:"branch_count"`
	LcSum         float64 `gorm:"lc_sum" json:"lc_sum"`
	ParapharmaSum float64 `gorm:"parapharma_sum" json:"parapharma_sum"`
	TotalSum      float64 `gorm:"total_sum" json:"total_sum"`
	WeekNumber    int     `gorm:"week_number" json:"week_number"`
	Weekday       int     `gorm:"weekday" json:"weekday"`
}

// Store report amount with payment types
type StoreAmount struct {
	UID          int     `gorm:"uid" json:"uid"`
	ID           string  `gorm:"id" json:"id"`
	StoreCode    int     `gorm:"store_code" json:"store_code"`
	StoreName    string  `gorm:"store_name" json:"store_name"`
	SaleDate     string  `gorm:"sale_date" json:"sale_date"`
	Cash         float64 `gorm:"cash" json:"cash"`
	Uzcard       float64 `gorm:"uzcard" json:"uzcard"`
	Humo         float64 `gorm:"humo" json:"humo"`
	Click        float64 `gorm:"click" json:"click"`
	Payme        float64 `gorm:"payme" json:"payme"`
	Alif         float64 `gorm:"alif" json:"alif"`
	ReturnAmount float64 `gorm:"return_amount" json:"return_amount"`
	TotalAmount  float64 `gorm:"total_amount" json:"total_amount"`
}

type StoreReportStats struct {
	TotalAmount  float64 `gorm:"total_amount" json:"total_amount"`
	ReturnAmount float64 `gorm:"return_amount" json:"return_amount"`
	Cash         float64 `gorm:"cash" json:"cash"`
	Uzcard       float64 `gorm:"uzcard" json:"uzcard"`
	Humo         float64 `gorm:"humo" json:"humo"`
	Click        float64 `gorm:"click" json:"click"`
	Payme        float64 `gorm:"payme" json:"payme"`
}
