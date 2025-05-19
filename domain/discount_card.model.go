package domain

import "time"

// discount card structure
type DiscountCard struct {
	Id         string     `gorm:"id" json:"id"`
	CustomerID string     `gorm:"customer_id" json:"customer_id"`
	Barcode    string     `gorm:"barcode" json:"barcode"`
	Percent    string     `gorm:"pencent" json:"percent"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}

type CustomerDiscount struct {
	ID              string     `gorm:"id" json:"id"`
	CustomerID      string     `gorm:"customer_id" json:"customer_id"`
	SaleId          string     `gorm:"sale_id" json:"sale_id"`
	DiscountPercent string     `gorm:"discount_percent" json:"discount_percent"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
}
