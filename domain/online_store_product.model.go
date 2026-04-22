package domain

import "time"

type OnlineStoreProduct struct {
	Id             string     `gorm:"id" json:"id"`
	StoreId        string     `gorm:"store_id" json:"store_id"`
	ProductId      string     `gorm:"product_id" json:"product_id"`
	Type           string     `gorm:"type" json:"type"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	OldSupplyPrice float64    `gorm:"old_supply_price" json:"old_supply_price"`
	CreatedBy      *string    `gorm:"created_by" json:"created_by"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

func (OnlineStoreProduct) TableName() string {
	return "online_store_products"
}

type OnlineStoreProductItem struct {
	ProductId      string  `json:"product_id"`
	RetailPrice    float64 `json:"retail_price"`
	SupplyPrice    float64 `json:"supply_price"`
	OldSupplyPrice float64 `json:"old_supply_price"`
}

// Admin bulk upsert uchun
type UpsertOnlineStoreProductsRequest struct {
	StoreId   string                   `json:"store_id"`
	Type      string                   `json:"type"` // "uzum", "yandex_eda", etc.
	Products  []OnlineStoreProductItem `json:"products"`
	CreatedBy *string                  `json:"created_by"`
}

type CreateOnlineStoreProductsRequest struct {
	StoreId      string  `json:"store_id"`
	PlatformType string  `json:"platform_type"`
	CreatedBy    *string `json:"created_by"`
}

type OnlineStoreProductQueryParam struct {
	StoreId string `form:"store_id"`
	Type    string `form:"type"`
}
