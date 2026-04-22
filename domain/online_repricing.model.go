package domain

import "time"

type OnlinePriceRevaluation struct {
	Id           int        `gorm:"id" json:"id"`
	StoreId      string     `gorm:"store_id" json:"store_id"`
	PlatformType string     `gorm:"platform_type" json:"platform_type"`
	Name         string     `gorm:"name" json:"name"`
	Status       string     `gorm:"status" json:"status"`
	Count        int64      `gorm:"count" json:"count"`
	CreatedById  string     `gorm:"created_by" json:"created_by_id"`
	UpdatedById  string     `gorm:"updated_by" json:"updated_by_id"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
	CreatedBy    *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy    *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
	Store        *Store     `gorm:"foreignKey:StoreId" json:"store"`
}

func (OnlinePriceRevaluation) TableName() string {
	return "online_price_revaluations"
}

type OnlinePriceRevalutionDetail struct {
	Id                       string     `gorm:"id" json:"id"`
	OnlinePriceRevaluationId int        `gorm:"online_price_revaluation_id" json:"online_price_revaluation_id"`
	StoreId                  string     `gorm:"store_id" json:"store_id"`
	ProductId                string     `gorm:"product_id" json:"product_id"`
	OldRetailPrice           float64    `gorm:"old_retail_price" json:"old_retail_price"`
	NewRetailPrice           float64    `gorm:"new_retail_price" json:"new_retail_price"`
	OldSupplyPrice           float64    `gorm:"old_supply_price" json:"old_supply_price"`
	Name                     string     `gorm:"name" json:"name"`
	Barcode                  string     `gorm:"barcode" json:"barcode"`
	TotalCount               int64      `gorm:"total_count" json:"-"`
	CreatedAt                *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt                *time.Time `gorm:"updated_at" json:"updated_at"`
}

type OnlineRepricingRequest struct {
	StoreId      string  `json:"store_id"`
	PlatformType string  `json:"platform_type"`
	Name         string  `json:"name"`
	CreatedBy    *string `json:"created_by"`
}

type UpdateOnlineDetailPrice struct {
	Id             string  `json:"id"`
	NewRetailPrice float64 `json:"new_retail_price"`
}

type OnlineRepricingQueryParam struct {
	StoreId      string `form:"store_id"`
	PlatformType string `form:"platform_type"`
	Status       string `form:"status"`
	Search       string `form:"search"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}
