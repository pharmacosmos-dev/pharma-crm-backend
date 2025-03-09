package domain

import "time"

// finance category structure
type FinanceCategory struct {
	Id           int64             `gorm:"id" json:"id"`
	ParentId     int64             `gorm:"parent_id" json:"parent_id"`
	Name         string            `gorm:"name" json:"name"`
	Description  string            `gorm:"description" json:"description"`
	AccountGroup string            `gorm:"account_group" json:"account_group"`
	Status       string            `gorm:"status" json:"status"`
	CreatedAt    *time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time        `gorm:"updated_at" json:"updated_at"`
	Children     []FinanceCategory `gorm:"foreignKey:ParentId" json:"children"`
}

// FinanceCategoryRequest structure
type FinanceCategoryRequest struct {
	Id           *int64                   `gorm:"id" json:"id"`
	ParentId     *int64                   `gorm:"parent_id" json:"parent_id"`
	Name         string                   `gorm:"name" json:"name"`
	Description  string                   `gorm:"description" json:"description"`
	AccountGroup string                   `gorm:"account_group" json:"account_group" example:"income, expense"`
	Status       string                   `gorm:"status" json:"status"`
	Children     []FinanceCategoryRequest `gorm:"-" json:"children"`
}
