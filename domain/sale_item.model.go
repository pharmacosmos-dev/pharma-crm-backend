package domain

import "time"

// SaleItem structure
type SaleItem struct {
	ID             string     `gorm:"id" json:"id"`
	SaleID         string     `gorm:"sale_id" json:"sale_id"`
	ProductID      string     `gorm:"product_id" json:"product_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	Quantity       uint       `gorm:"quantity" json:"quantity"`
	UnitPrice      float64    `gorm:"unit_price" json:"unit_price"`
	DiscountType   string     `gorm:"discount_type" json:"discount_type"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Product        *Product   `gorm:"foreignKey:ProductID" json:"product"`
}

// SaleItemRequest structure for create || update
type SaleItemRequest struct {
	ID            string  `gorm:"id" json:"-"`
	SaleID        string  `gorm:"sale_id" json:"sale_id"`
	ProductID     string  `gorm:"product_id" json:"product_id"`
	EmployeeID    string  `gorm:"employee_id" json:"employee_id"`
	Quantity      uint    `gorm:"quantity" json:"quantity"`
	UnitPrice     float64 `gorm:"unit_price" json:"unit_price"`
	DiscountType  string  `gorm:"discount_type" json:"discount_type"`
	DiscountValue float64 `gorm:"discount_value" json:"discount_value"`
}

// SaleID structure for multiple sale items
type SaleID struct {
	SaleID string `gorm:"sale_id" json:"sale_id"`
}
