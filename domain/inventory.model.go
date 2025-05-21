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
	Id                string     `gorm:"id" json:"id"`
	PublicId          string     `gorm:"public_id" json:"public_id"`
	StoreId           string     `gorm:"store_id" json:"store_id"`
	Name              string     `gorm:"name" json:"name"`
	InventoryType     string     `gorm:"inventory_type" json:"type"`
	MeasurementCount  float64    `gorm:"measurement_count" json:"measurement_count"`
	Shortage          float64    `gorm:"shotage" json:"shortage"`
	Surplus           float64    `gorm:"surplus" json:"surplus"`
	DifferenceSum     float64    `gorm:"difference_sum" json:"difference_sum"`
	Status            string     `gorm:"status" json:"status"` // 0 -> new, 1 -> pending, 2 -> completed
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	CreatedById       string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById       string     `gorm:"column:accepted_by" json:"updated_by_id"`
	ShortageSupplySum float64    `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64    `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64    `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64    `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
	Store             *Store     `gorm:"foreignKey:StoreId" json:"store"`
	CreatedBy         *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy         *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
}

// InventoryRequest structure
type InventoryRequest struct {
	PublicId  string `gorm:"public_id" json:"public_id"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	Name      string `gorm:"name" json:"name"`
	Type      string `gorm:"type" json:"type"` // FULL || PARTIAL || IMPORT
	CreatedBy string `gorm:"created_by" json:"created_by"`
}

// InventoryRequest structure
type InventoryDetail struct {
	Id                 string     `gorm:"id" json:"id"`
	InventoryId        string     `gorm:"inventory_id" json:"inventory_id"`
	ProductId          string     `gorm:"product_id" json:"product_id"`
	MaterialCode       int        `gorm:"material_code" json:"material_code"`
	UnitPerPack        int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	Name               string     `gorm:"name" json:"name"`
	CurrentQuantity    float64    `gorm:"current_quantity" json:"current_quantity"`
	CurrentUnit        float64    `gorm:"current_unit" json:"current_unit"`
	FactQuantity       float64    `gorm:"fact_quantity" json:"fact_quantity"`
	FactUnit           float64    `gorm:"fact_unit" json:"fact_unit"`
	DifferenceQuantity float64    `gorm:"difference_quantity" json:"difference_quantity"`
	DifferenceUnit     float64    `gorm:"difference_unit" json:"difference_unit"`
	CurrentSum         float64    `gorm:"current_sum" json:"current_sum"`
	FactSum            float64    `gorm:"fact_sum" json:"fact_sum"`
	DifferenceSum      float64    `gorm:"difference_sum" json:"difference_sum"`
	RetailPrice        float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate         *time.Time `gorm:"expire_date" json:"expire_date"`
	TotalCount         int64      `gorm:"total_count" json:"-"`
}

// InventoryDetailRequest structure
type InventoryDetailRequest struct {
	InventoryId string `gorm:"inventory_id" json:"inventory_id"`
	ProductId   string `gorm:"product_id" json:"product_id"`
}

type InventoryDetailParam struct {
	InventoryId string `form:"inventory_id"`
	ProductId   string `form:"product_id"`
	Type        string `form:"type"`
	Search      string `form:"search"`
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	Order       string `form:"order"`
}

type InventoryDetailStatus struct {
	Scanned           float64 `gorm:"scanned" json:"scanned"`
	Shortage          float64 `gorm:"shortage" json:"shortage"`
	Surplus           float64 `gorm:"surplus" json:"surplus"`
	All               float64 `gorm:"all" json:"all"`
	New               float64 `gorm:"new" json:"new"`
	Accepted          float64 `gorm:"accepted" json:"accepted"`
	ShortageSupplySum float64 `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64 `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64 `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64 `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
}

type InventoryAddProduct struct {
	FactQuantity float64 `gorm:"fact_quantity" json:"fact_quantity"`
	FactUnit     float64 `gorm:"fact_unit" json:"fact_unit"`
	Id           string  `gorm:"id" json:"id"`
}
