package domain

import "time"

type ReportQueryParam struct {
	StoreId     string   `form:"store_id"`
	StartDate   string   `form:"start_date"`
	EndDate     string   `form:"end_date"`
	Limit       int      `form:"limit"`
	Offset      int      `form:"offset"`
	Search      string   `form:"search"`
	Order       string   `form:"order"`
	EmployeeId  string   `form:"employee_id"`
	StoreIds    []string `form:"store_ids"`
	ProducerIds []string `form:"producer_ids"`
	ProductIds  []string `form:"product_ids"`
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
	MaterialCode   int        `gorm:"material_code" json:"material_code"`
	StoreName      string     `gorm:"store_name" json:"store_name"`
	ProductName    string     `gorm:"product_name" json:"product_name"`
	ProducerName   string     `gorm:"producer_name" json:"producer_name"`
	SerialNumber   string     `gorm:"serial_number" json:"serial_number"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity       string     `gorm:"quantity" json:"quantity"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPriceSum float64    `gorm:"supply_price_sum" json:"supply_price_sum"`
	RetailPriceSum float64    `gorm:"retail_price_sum" json:"retail_price_sum"`
	MarkupSum      float64    `gorm:"markup_sum" json:"markup_sum"`
	VatSum         float64    `gorm:"vat_sum" json:"vat_sum"`
	CompletedAt    *time.Time `gorm:"completed_at" json:"completed_at"`
	FullName       string     `gorm:"full_name" json:"full_name"`
	SaleNumber     string     `gorm:"sale_number" json:"sale_number"`
	MarkingCount   int        `gorm:"marking_count" json:"marking_count"`
	TotalCount     int64      `gorm:"total_count" json:"-"`
}
