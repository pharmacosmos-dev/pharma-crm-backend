package domain

import "time"

// write off structure
type WriteOff struct {
	Id             string     `gorm:"id" json:"id"`
	PublicId       int64      `gorm:"public_id" json:"public_id"`
	StoreId        string     `gorm:"store_id" json:"store_id"`
	Name           string     `gorm:"name" json:"name"`
	Status         string     `gorm:"status" json:"status"`
	Comment        string     `gorm:"comment" json:"comment"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	WriteoffCount  int64      `gorm:"writeoff_count" json:"writeoff_count"`
	SupplyPriceSum float64    `gorm:"supply_price_sum" json:"supply_price_sum"`
	RetailPriceSum float64    `gorm:"retail_price_sum" json:"retail_price_sum"`
	CreatedById    string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById    string     `gorm:"column:accepted_by" json:"updated_by_id"`
	Store          *Store     `gorm:"foreignKey:StoreId" json:"store"`
	CreatedBy      *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy      *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
}

// write off create request
type WriteOffRequest struct {
	Name      string `gorm:"name" json:"name"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	CreatedBy string `gorm:"created_by" json:"created_by"`
	Status    string `gorm:"status" json:"status"`
	Comment   string `gorm:"comment" json:"comment"`
}

// WriteOffDetail structure
type WriteOffDetail struct {
	Id           string     `gorm:"id" json:"id"`
	WriteoffId   string     `gorm:"writeoff_id" json:"writeoff_id"`
	ProductId    string     `gorm:"product_id" json:"product_id"`
	ScannedCount int        `gorm:"scanned_count" json:"scanned_count"`
	Name         string     `gorm:"name" json:"name"`
	MaterialCode int        `gorm:"material_code" json:"material_code"`
	Barcode      string     `gorm:"barcode" json:"barcode"`
	ShortName    string     `gorm:"short_name" json:"short_name"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// writeoff query param  structure
type WriteOffParam struct {
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	StoreId   string `form:"store_id"`
	Status    string `form:"status"`
	Search    string `form:"search"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

// writeoff detail query param structure
type WriteOffDetailParam struct {
	WriteOffId string `form:"writeoff_id"`
	Search     string `form:"search"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
	Type       string `form:"type"`
}
