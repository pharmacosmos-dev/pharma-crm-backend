package domain

import "time"

// Draft structure
type Draft struct {
	ID          string      `gorm:"id" json:"id"`
	StoreID     string      `gorm:"store_id" json:"store_id"`
	CustomerID  string      `gorm:"customer_id" json:"customer_id"`
	CreatedBy   string      `gorm:"created_by" json:"created_by"`
	DraftNumber string      `gorm:"draft_number" json:"draft_number"`
	Quantity    int         `gorm:"quantity" json:"quantity"`
	TotalPrice  float64     `gorm:"total_price" json:"total_price"`
	Description string      `gorm:"description" json:"description"`
	DraftTime   string      `gorm:"draft_time" json:"draft_time"`
	CreatedAt   *time.Time  `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time  `gorm:"updated_at" json:"updated_at"`
	Store       *Store      `gorm:"foreignKey:StoreID" json:"store"`
	Customer    *Customer   `gorm:"foreignKey:CustomerID" json:"customer"`
	Employee    *Employee   `gorm:"foreignKey:CreatedBy" json:"employee"`
	CartItems   []*CartItem `gorm:"-" json:"cart_items"` // Add this field for cart items
}

// DraftRequest structure for create, update
type DraftRequest struct {
	ID          string  `gorm:"id" json:"-"`
	StoreID     string  `gorm:"store_id" json:"store_id"`
	CustomerID  *string `gorm:"customer_id" json:"customer_id"`
	SaleID      string  `gorm:"sale_id" json:"sale_id"`
	CreatedBy   string  `gorm:"created_by" json:"created_by"`
	DraftNumber string  `gorm:"draft_number" json:"-"`
	Description string  `gorm:"description" json:"description"`
	DraftTime   string  `gorm:"draft_time" json:"draft_time"`
}

type DraftCreate struct {
	ID          string  `gorm:"id" json:"-"`
	StoreID     string  `gorm:"store_id" json:"store_id"`
	SaleID      string  `gorm:"sale_id" json:"sale_id"`
	CustomerID  *string `gorm:"customer_id" json:"customer_id"`
	CreatedBy   string  `gorm:"created_by" json:"created_by"`
	DraftNumber string  `gorm:"draft_number" json:"-"`
	Description string  `gorm:"description" json:"description"`
	DraftTime   string  `gorm:"draft_time" json:"draft_time"`
	ProductID   string  `gorm:"product_id" json:"product_id"`
	Quantity    int     `gorm:"quantity" json:"quantity"`
	UnitPrice   float64 `gorm:"unit_price" json:"unit_price"`
	TotalPrice  float64 `gorm:"total_price" json:"total_price"`
}

type CartItemDraft struct {
	ID         string    `gorm:"id" json:"id"`
	DraftID    string    `gorm:"draft_id" json:"draft_id"`
	CartItemID string    `gorm:"cart_item_id" json:"cart_item_id"`
	CartItem   *CartItem `gorm:"foreignKey:CartItemID" json:"cart_item"`
	Draft      *Draft    `gorm:"foreignKey:DraftID" json:"draft"`
}
