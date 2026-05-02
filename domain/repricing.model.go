package domain

import "time"

// Repricing structure
type PriceRevalution struct {
	Id                  int        `gorm:"id" json:"id"`
	StoreID             string     `gorm:"store_id" json:"store_id"`
	Name                string     `gorm:"name" json:"name"`
	Status              string     `gorm:"status" json:"status"`
	Type                string     `gorm:"type" json:"type"`
	CreatedByID         string     `gorm:"created_by_id" json:"created_by_id"`
	UpdatedByID         string     `gorm:"updated_by_id" json:"updated_by_id"`
	Count               float64    `gorm:"count" json:"count"`
	TotalOldRetailPrice float64    `gorm:"total_old_retail_price" json:"total_old_retail_price"`
	TotalNewRetailPrice float64    `gorm:"total_new_retail_price" json:"total_new_retail_price"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	CreatedBy           *Employee  `gorm:"foreignKey:CreatedByID" json:"created_by"`
	UpdatedBy           *Employee  `gorm:"foreignKey:UpdatedByID" json:"updated_by"`
	Store               *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

type RepricingStatusSummary struct {
	Count               int64   `json:"count"`
	TotalOldRetailPrice float64 `json:"total_old_retail_price"`
	TotalNewRetailPrice float64 `json:"total_new_retail_price"`
}

// repricing off create request
type RepricingRequest struct {
	Name           string  `gorm:"name" json:"name"`
	StoreId        string  `gorm:"store_id" json:"store_id"`
	ImportId       string  `gorm:"import_id" json:"import_id"`
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	CreatedBy      *string `gorm:"created_by" json:"created_by"`
	Status         string  `gorm:"status" json:"status"`
	Type           string  `gorm:"type" json:"type" example:"retail_price|supply_price|expire_date"`
}

// repricing detail structure
type PriceRevalutionDetail struct {
	Id                string     `gorm:"id" json:"id"`
	PriceRevalutionId int        `gorm:"price_revalution_id" json:"price_revalution_id"`
	StoreProductID    string     `gorm:"store_product_id" json:"store_product_id"`
	ProductID         string     `gorm:"product_id" json:"product_id"`
	ScannedCount      float64    `gorm:"scanned_count" json:"scanned_count"`
	OldSupplyPrice    float64    `gorm:"old_supply_price" json:"old_supply_price"`
	NewSupplyPrice    float64    `gorm:"new_supply_price" json:"new_supply_price"`
	OldRetailPrice    float64    `gorm:"old_retail_price" json:"old_retail_price"`
	NewRetailPrice    float64    `gorm:"new_retail_price" json:"new_retail_price"`
	OldExpireDate     *time.Time `gorm:"old_expire_date" json:"old_expire_date"`
	NewExpireDate     *time.Time `gorm:"new_expire_date" json:"new_expire_date"`
	OldMarkup         float64    `gorm:"old_markup" json:"old_markup"`
	NewMarkup         float64    `gorm:"new_markup" json:"new_markup"`
	PriceDifference   float64    `gorm:"price_difference" json:"price_difference"`
	SerialNumber      string     `gorm:"serial_number" json:"serial_number"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	Name              string     `gorm:"name" json:"name"`
	Barcode           string     `gorm:"barcode" json:"barcode"`
	MaxPrice          float64    `gorm:"max_price" json:"max_price"`
	PackQuantity       int        `gorm:"pack_quantity" json:"pack_quantity"`
	UnitQuantity       int        `gorm:"unit_quantity" json:"unit_quantity"`
	SmallQuantity      int        `gorm:"small_quantity" json:"small_quantity"`
	StoreProductCount  int        `gorm:"store_product_count" json:"store_product_count"`
	TotalCount         int64      `gorm:"total_count" json:"-"`
}

// price revalution detail request
type PriceRevalutionDetailRequest struct {
	PriceRevalutionId int     `gorm:"price_revalution_id" json:"price_revalution_id"`
	ProductId         string  `gorm:"product_id" json:"product_id"`
	StoreProductId    string  `gorm:"store_product_id" json:"store_product_id"`
	OldRetailPrice    float64 `gorm:"old_retail_price" json:"old_retail_price"`
	NewRetailPrice    float64 `gorm:"new_retail_price" json:"new_retail_price"`
	OldSupplyPrice    float64 `gorm:"old_supply_price" json:"old_supply_price"`
	OldExpireDate     string  `gorm:"old_expire_date" json:"old_expire_date"`
	SerialNumber      string  `gorm:"serial_number" json:"serial_number"`
}

// update new price structure
type UpdateNewPrice struct {
	Id             string  `gorm:"id" json:"id"`
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	Percent        float64 `gorm:"percent" json:"percent"`
	NewRetailPrice float64 `gorm:"new_retail_price" json:"new_retail_price"`
	NewExpireDate  string  `gorm:"new_expire_date" json:"new_expire_date"`
}

type RepricingDetailStatusSummary struct {
	Count               int64   `json:"count"`
	TotalOldRetailPrice float64 `json:"total_old_retail_price"`
	TotalNewRetailPrice float64 `json:"total_new_retail_price"`
	TotalOldSupplyPrice float64 `json:"total_old_supply_price"`
	AvgOldMarkup        float64 `json:"avg_old_markup"`
	AvgNewMarkup        float64 `json:"avg_new_markup"`
}
