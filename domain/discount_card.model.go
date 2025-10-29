package domain

import (
	"time"

	"gorm.io/gorm"
)

// discount card structure
type DiscountCard struct {
	ID         string `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	CustomerID string `gorm:"type:uuid"`
	Barcode    string `gorm:"size:13;unique;not null"`
	Percent    int    `gorm:"default:0"`
	CreatedBy  string
	UpdatedBy  *string
	DeletedBy  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

type SaleCustomerDiscount struct {
	ID              string     `gorm:"id" json:"id"`
	CustomerID      string     `gorm:"customer_id" json:"customer_id"`
	SaleId          string     `gorm:"sale_id" json:"sale_id"`
	DiscountCardId  string     `gorm:"discount_card_id" json:"discount_card_id"`
	DiscountPercent float64    `gorm:"discount_percent" json:"discount_percent"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
}

type CreateDiscountCardRequest struct {
	Barcode    string `json:"barcode" binding:"required,min=10,max=20"`
	CustomerId string `json:"customer_id"`
	Percent    int    `json:"percent" binding:"gte=0,lte=100"`
}

type UpdateDiscountCardRequest struct {
	Id         string `json:"-"`
	CustomerId string `json:"customer_id"`
	Percent    int    `json:"percent" binding:"gte=0,lte=100"`
	UpdatedBy  string `json:"-"`
}

type UpdateDiscountCard struct {
	Percent    int       `gorm:"default:0"`
	UpdatedBy  string    `gorm:"updated_by"`
	UpdatedAt  time.Time `gorm:"updated_at"`
	CustomerID *string   `gorm:"type:uuid"`
}
