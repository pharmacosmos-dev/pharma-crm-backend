package domain

import "time"

// Draft structure
type Draft struct {
	ID          string     `gorm:"id" json:"id"`
	StoreID     string     `gorm:"store_id" json:"store_id"`
	ProductID   string     `gorm:"product_id" json:"product_id"`
	CashBoxId   string     `gorm:"cash_box_id" json:"cash_box_id"`
	DraftNumber string     `gorm:"draft_number" json:"draft_number"`
	Quantity    int        `gorm:"quantity" json:"quantity"`
	UnitPrice   float64    `gorm:"unit_price" json:"unit_price"`
	TotalAmount float64    `gorm:"total_amount" json:"total_amount"`
	Description string     `gorm:"description" json:"description"`
	DraftTime   string     `gorm:"draft_time" json:"draft_time"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
	Store       *Store     `gorm:"foreignKey:StoreID" json:"store"`
	CashBox     *CashBox   `gorm:"foreignKey:CashBoxId" json:"cash_box"`
	Product     *Product   `gorm:"foreignKey:ProductID" json:"product"`
}

// DraftRequest structure for create, update
type DraftRequest struct {
	ID          string  `gorm:"id" json:"-"`
	StoreID     string  `gorm:"store_id" json:"store_id"`
	ProductID   string  `gorm:"product_id" json:"product_id"`
	CashBoxId   string  `gorm:"cash_box_id" json:"cash_box_id"`
	DraftNumber string  `gorm:"draft_number" json:"-"`
	Quantity    int     `gorm:"quantity" json:"quantity"`
	UnitPrice   float64 `gorm:"unit_price" json:"unit_price"`
	TotalAmount float64 `gorm:"total_amount" json:"total_amount"`
	Description string  `gorm:"description" json:"description"`
	DraftTime   string  `gorm:"draft_time" json:"draft_time"`
}
