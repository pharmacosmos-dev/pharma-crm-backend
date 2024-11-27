package domain

import "time"

// CartItem structure
type CartItem struct {
	ID             string     `gorm:"id" json:"id"`
	ProductID      string     `gorm:"product_id" json:"product_id"`
	Quantity       int        `gorm:"quantity" json:"quantity"`
	UnitPrice      float64    `gorm:"unit_price" json:"unit_price"`
	DiscountType   string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Product        *Product   `gorm:"foreignKey:ProductID" json:"product"`
}

// CartItemRequest structure
type CartItemRequest struct {
	ID            string  `gorm:"id" json:"-"`
	ProductID     string  `gorm:"product_id" json:"product_id"`
	Quantity      int     `gorm:"quantity" json:"quantity"`
	UnitPrice     float64 `gorm:"unit_price" json:"unit_price"`
	DiscountType  string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue float64 `gorm:"discount_value" json:"discount_value"`
}
