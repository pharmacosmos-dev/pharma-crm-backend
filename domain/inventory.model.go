package domain

import "time"

// InventoryParam structure
type InventoryParam struct {
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
	StoreId string `form:"store_id"`
	Type    string `form:"type"`
	Status  string `form:"status"`
	Search  string `form:"search"`
}

// Inventory structure
type Inventory struct {
	Id               string     `gorm:"id" json:"id"`
	PublicId         string     `gorm:"public_id" json:"public_id"`
	StoreId          string     `gorm:"store_id" json:"store_id"`
	Name             string     `gorm:"name" json:"name"`
	Type             string     `gorm:"type" json:"type"`
	MeasurementCount int64      `gorm:"measurement_count" json:"measurement_count"`
	Shortage         int64      `gorm:"shotage" json:"shortage"`
	Surplus          int64      `gorm:"surplus" json:"surplus"`
	DifferenceSum    float64    `gorm:"difference_sum" json:"difference_sum"`
	Status           int        `gorm:"status" json:"status"` // 0 -> new, 1 -> pending, 2 -> completed
	CreatedAt        *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time `gorm:"updated_at" json:"updated_at"`
	CreatedById      string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById      string     `gorm:"column:updated_by" json:"updated_by_id"`
	Store            *Store     `gorm:"foreignKey:StoreId" json:"store"`
	CreatedBy        *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy        *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
}

// InventoryRequest structure
type InventoryRequest struct {
	PublicId  string `gorm:"public_id" json:"public_id"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	Name      string `gorm:"name" json:"name"`
	Type      string `gorm:"type" json:"type"` // FULL || PARTIAL || IMPORT
	CreatedBy string `gorm:"created_by" json:"created_by"`
	Products  []struct {
		ProductId string `json:"product_id"`
	} `json:"products"`
}

// InventoryRequest structure
type InventoryDetail struct {
	Id           string     `gorm:"id" json:"id"`
	InventoryId  string     `gorm:"inventory_id" json:"inventory_id"`
	ProductId    string     `gorm:"product_id" json:"product_id"`
	StockCount   int        `gorm:"stock_count" json:"stock_count"`
	ScannedCount int        `gorm:"scanned_count" json:"scanned_count"`
	Name         string     `gorm:"name" json:"name"`
	MaterialCode int        `gorm:"material_code" json:"material_code"`
	Barcode      string     `gorm:"barcode" json:"barcode"`
	ShortName    string     `gorm:"short_name" json:"short_name"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// InventoryDetailRequest structure
type InventoryDetailRequest struct {
	InventoryId string `gorm:"inventory_id" json:"inventory_id"`
	ProductId   string `gorm:"product_id" json:"product_id"`
}

type InventoryDetailParam struct {
	InventoryId string `form:"inventory_id"`
	Type        string `form:"type"`
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
}

type InventoryDetailStatus struct {
	Scanned           int     `gorm:"scanned" json:"scanned"`
	Shortage          int     `gorm:"shortage" json:"shortage"`
	Surplus           int     `gorm:"surplus" json:"surplus"`
	All               int     `gorm:"all" json:"all"`
	New               int     `gorm:"new" json:"new"`
	Accepted          int     `gorm:"accepted" json:"accepted"`
	ShortageSupplySum float64 `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64 `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64 `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64 `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
}

type InventoryAddProduct struct {
	Barcode   string `gorm:"barcode" json:"barcode"`
	Count     int    `gorm:"count" json:"count"`
	ProductId string `gorm:"product_id" json:"product_id"`
	Type      string `gorm:"type" json:"type"`
}
