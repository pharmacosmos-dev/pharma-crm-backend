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
