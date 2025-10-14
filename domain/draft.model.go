package domain

import "time"

// Draft structure
type Draft struct {
	ID          string             `gorm:"id" json:"id"`
	StoreId     string             `gorm:"store_id" json:"store_id"`
	SaleId      string             `gorm:"sale_id" json:"sale_id"`
	CustomerId  string             `gorm:"customer_id" json:"customer_id"`
	CreatedBy   string             `gorm:"created_by" json:"created_by"`
	DraftNumber string             `gorm:"draft_number" json:"draft_number"`
	Quantity    int                `gorm:"quantity" json:"quantity"`
	TotalPrice  float64            `gorm:"total_price" json:"total_price"`
	Description string             `gorm:"description" json:"description"`
	DraftTime   string             `gorm:"draft_time" json:"draft_time"`
	CreatedAt   *time.Time         `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time         `gorm:"updated_at" json:"updated_at"`
	Store       *Store             `gorm:"foreignKey:StoreId" json:"store"`
	Customer    *Customer          `gorm:"foreignKey:CustomerId" json:"customer"`
	Employee    *Employee          `gorm:"foreignKey:CreatedBy" json:"employee"`
	CartItems   []CartItemResponse `gorm:"-" json:"cart_items"`
}

// DraftRequest structure for create, update
type DraftRequest struct {
	ID          string  `gorm:"id" json:"-"`
	StoreId     string  `gorm:"store_id" json:"store_id"`
	CustomerId  *string `gorm:"customer_id" json:"customer_id"`
	SaleId      string  `gorm:"sale_id" json:"sale_id"`
	CreatedBy   string  `gorm:"created_by" json:"created_by"`
	Description string  `gorm:"description" json:"description"`
	DraftTime   string  `gorm:"draft_time" json:"draft_time"`
	Status      string  `gorm:"status" json:"status"`
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
}

type DraftQueryParams struct {
	Search     string `form:"search"`
	StoreId    string `form:"store_id"`
	CustomerId string `form:"customer_id"`
	DraftDate  string `form:"draft_date"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

type CartItemDraft struct {
	ID         string    `gorm:"id" json:"id"`
	DraftID    string    `gorm:"draft_id" json:"draft_id"`
	CartItemID string    `gorm:"cart_item_id" json:"cart_item_id"`
	CartItem   *CartItem `gorm:"foreignKey:CartItemID" json:"cart_item"`
	Draft      *Draft    `gorm:"foreignKey:DraftID" json:"draft"`
}
