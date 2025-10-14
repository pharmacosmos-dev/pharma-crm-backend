package domain

import "time"

// product bonus structure
type ProductBonus struct {
	Id          int64      `gorm:"id" json:"id"`
	ProductId   string     `gorm:"product_id" json:"product_id"`
	StoreId     string     `gorm:"store_id" json:"store_id"`
	BonusAmount float64    `gorm:"bonus_amount" json:"bonus_amount"`
	Status      int        `gorm:"status" json:"status"`
	StartDate   string     `gorm:"start_date" json:"start_date"`
	EndDate     string     `gorm:"end_date" json:"end_date"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
	Product     *Product   `gorm:"foreignKey:ProductId" json:"product"`
	Store       *Store     `gorm:"foreignKey:StoreId" json:"store"`
}

// product bonus request structure

type ProductBonusRequest struct {
	ProductId   string  `json:"product_id"`
	StoreId     *string `json:"store_id"`
	BonusAmount float64 `json:"bonus_amount"`
	CompanyId   string  `json:"company_id"`
	Status      int     `json:"status"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
}

var EmployeeBonusBalance struct {
	TotalBonus    float64 `json:"total_bonus"`
	TotalSales    int64   `json:"total_sales"`
	TotalProducts int64   `json:"total_products"`
}
