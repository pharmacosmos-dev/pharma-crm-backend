package domain

import "time"

// InventoryParam structure
type InventoryParam struct {
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	InventoryId string `form:"inventory_id"`
	StoreId     string `form:"store_id"`
	Type        string `form:"type"`
	Status      string `form:"status"`
	Search      string `form:"search"`
	ProductId   string `form:"product_id"`
	Order       string `form:"order"`
}

// Inventory structure
type Inventory struct {
	Id              string     `gorm:"id" json:"id"`
	PublicId        string     `gorm:"public_id" json:"public_id"`
	StoreId         string     `gorm:"store_id" json:"store_id"`
	Name            string     `gorm:"name" json:"name"`
	InventoryType   string     `gorm:"inventory_type" json:"type"`
	CurrentCount    int64      `gorm:"current_count" json:"current_count"`
	FactCount       int64      `gorm:"fact_count" json:"fact_count"`
	DifferenceCount int64      `gorm:"difference_count" json:"difference_count"`
	Status          string     `gorm:"status" json:"status"`
	CreatedById     string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById     string     `gorm:"column:accepted_by" json:"updated_by_id"`
	CurrentSum      float64    `gorm:"current_sum" json:"current_sum"`
	FactSum         float64    `gorm:"fact_sum" json:"fact_sum"`
	DifferenceSum   float64    `gorm:"difference_sum" json:"difference_sum"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
	Store           *Store     `gorm:"foreignKey:StoreId" json:"store"`
	CreatedBy       *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy       *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
}

type InventoryStatusSummary struct {
	CurrentSum      float64 `json:"current_sum"`
	FactSum         float64 `json:"fact_sum"`
	DifferenceSum   float64 `json:"difference_sum"`
	CurrentCount    float64 `json:"current_count"`
	FactCount       float64 `json:"fact_count"`
	DifferenceCount float64 `json:"difference_count"`
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
	ProducerName       string     `gorm:"producer_name" json:"producer_name"`
	Barcode            string     `gorm:"barcode" json:"barcode"`
	CurrentQuantity    float64    `gorm:"current_quantity" json:"current_quantity"`
	CurrentUnit        float64    `gorm:"current_unit" json:"current_unit"`
	FactQuantity       float64    `gorm:"fact_quantity" json:"fact_quantity"`
	FactUnit           float64    `gorm:"fact_unit" json:"fact_unit"`
	DifferenceQuantity float64    `gorm:"difference_quantity" json:"difference_quantity"`
	DifferenceUnit     float64    `gorm:"difference_unit" json:"difference_unit"`
	CurrentSum         float64    `gorm:"current_sum" json:"current_sum"`
	FactSum            float64    `gorm:"fact_sum" json:"fact_sum"`
	DifferenceSum      float64    `gorm:"difference_sum" json:"difference_sum"`
	SupplyPrice        float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice        float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate         *time.Time `gorm:"expire_date" json:"expire_date"`
	TotalCount         int64      `gorm:"total_count" json:"-"`
}

// InventoryDetailRequest structure
type InventoryDetailRequest struct {
	InventoryId string `gorm:"inventory_id" json:"inventory_id"`
	ProductId   string `gorm:"product_id" json:"product_id"`
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
	Barcode      string  `gorm:"barcode" json:"barcode"`
	ExpireDate   string  `gorm:"expire_date" json:"expire_date"`
	RetailPrice  float64 `gorm:"retail_price" json:"retail_price"`
	Id           string  `gorm:"id" json:"id"`
}

type InventoryDetailSum struct {
	TotalFactSum       float64 `gorm:"total_fact_sum" json:"total_fact_sum"`
	TotalCurrentSum    float64 `gorm:"total_current_sum" json:"total_current_sum"`
	TotalDifferenceSum float64 `gorm:"total_difference_sum" json:"total_difference_sum"`
}

// 1C request Structure
type InventoryProduct1C struct {
	MaterialCode        int        `gorm:"material_code" json:"material_code"`
	Name                string     `gorm:"name" json:"name"`
	Barcode             string     `gorm:"barcode" json:"barcode"`
	Manufacturer        string     `gorm:"manufacturer" json:"manufacturer"`
	ProductSeriesNumber string     `gorm:"product_series_number" json:"product_series_number"`
	ExpireDate          *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity            float64    `gorm:"quantity" json:"quantity"`
	QuantityInventar    float64    `gorm:"quantity_inventar" json:"quantity_inventar"`
	RetailPrice         float64    `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat      float64    `gorm:"retail_price_vat" json:"retail_price_vat"`
	SupplyPrice         float64    `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat      float64    `gorm:"supply_price_vat" json:"supply_price_vat"`
	Sum                 float64    `gorm:"sum" json:"sum"`
	SumVat              float64    `gorm:"sum_vat" json:"sum_vat"`
}

type InventoryData1C struct {
	Dok    Document             `json:"Dok"`
	Apteka Apteka               `json:"Apteka"`
	Товары []InventoryProduct1C `json:"Товары"`
}

// inventory product price options
type InventoryPriceOption struct {
	ProductId   string     `gorm:"product_id" json:"product_id"`
	RetailPrice float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate  *time.Time `gorm:"expire_date" json:"expire_date"`
}

// inventory helper
type InventoryHelper struct {
	Method   string `gorm:"method" json:"method"`
	Payload  any    `gorm:"payload" json:"payload"`
	Response any    `gorm:"response" json:"response"`
	Action   string `gorm:"action" json:"action"`
	DocDate  string `gorm:"doc_date" json:"doc_date"`
	DocNum   string `gorm:"doc_num" json:"doc_num"`
	Status   string `gorm:"status" json:"status"`
}
