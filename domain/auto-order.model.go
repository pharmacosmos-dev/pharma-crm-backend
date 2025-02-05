package domain

import "time"

// Auto Order table structure
type AutoOrder struct {
	Id                    string     `gorm:"id" json:"id"`
	PublicID              int        `gorm:"public_id" json:"public_id"`
	StoreId               string     `gorm:"store_id" json:"store_id"`
	Status                string     `gorm:"status" json:"status"`
	AutoOrderDate         *time.Time `gorm:"auto_order_date" json:"auto_order_date"`
	CreatedAt             *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt             *time.Time `gorm:"updated_at" json:"updated_at"`
	AdjustedOrderQuantity float64    `gorm:"adjusted_order_quantity" json:"adjusted_order_quantity"`
	ResponseOrderQuantity float64    `gorm:"response_order_quantity" json:"response_order_quantity"`
	Store                 *Store     `gorm:"foreignKey:StoreId" json:"store"`
}

// Auto Order create request structure
type AutoOrderRequest struct {
	Id            string `gorm:"id" json:"id"`
	StoreId       string `gorm:"store_id" json:"store_id"`
	Status        string `gorm:"status" json:"status"`
	AutoOrderDate string `gorm:"auto_order_date" json:"auto_order_date"`
	IntervalDay   int    `gorm:"-" json:"interval_day"`
}

type AutoOrderConfirm struct {
	StoreId       string  `gorm:"store_id" json:"store_id"`
	ProductId     string  `gorm:"product_id" json:"product_id"`
	AdjustedOrder float64 `gorm:"adjusted_order" json:"adjusted_order"`
}

// auto order detail table structure
type AutoOrderDetail struct {
	Id                     string     `gorm:"id" json:"id"`
	AutoOrderId            string     `gorm:"auto_order_id" json:"auto_order_id"`
	ProductId              string     `gorm:"product_id" json:"product_id"`
	ProductName            string     `gorm:"product_name" json:"product_name"`
	Kvant                  int        `gorm:"kvant" json:"kvant"`
	MinStock               int        `gorm:"min_stock" json:"min_stock"`
	MaxStock               int        `gorm:"max_stock" json:"max_stock"`
	CurrentStock           int        `gorm:"current_stock" json:"current_stock"`
	MonthSaleStock         int        `gorm:"month_sale_stock" json:"month_sale_stock"`
	DaySaleStock           int        `gorm:"day_sale_stock" json:"day_sale_stock"`
	OrderGrowth            float64    `gorm:"order_growth" json:"order_growth"`
	OrderLeadTime          float64    `gorm:"order_lead_time" json:"order_lead_time"`
	SuggestedOrderQuantity int        `gorm:"suggested_order_quantity" json:"suggested_order_quantity"`
	AdjustedOrderQuantity  int        `gorm:"adjusted_order_quantity" json:"adjusted_order_quantity"`
	ResponseOrderQuantity  int        `gorm:"response_order_quantity" json:"response_order_quantity"`
	UnitName               string     `gorm:"unit_name" json:"unit_name"`
	CreatedAt              *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt              *time.Time `gorm:"updated_at" json:"updated_at"`
	AutoOrder              *AutoOrder `gorm:"foreignKey:AutoOrderId" json:"auto_order"`
}

// auto order detail request structure
type AutoOrderDetailRequest struct {
	AutoOrderId            string  `gorm:"auto_order_id" json:"auto_order_id"`
	ProductId              string  `gorm:"product_id" json:"product_id"`
	Kvant                  int     `gorm:"kvant" json:"kvant"`
	MinStock               int     `gorm:"min_stock" json:"min_stock"`
	MaxStock               int     `gorm:"max_stock" json:"max_stock"`
	CurrentStock           int     `gorm:"current_stock" json:"current_stock"`
	MonthSaleStock         int     `gorm:"month_sale_stock" json:"month_sale_stock"`
	DaySaleStock           int     `gorm:"day_sale_stock" json:"day_sale_stock"`
	OrderGrowth            float64 `gorm:"order_growth" json:"order_growth"`
	OrderLeadTime          float64 `gorm:"order_lead_time" json:"order_lead_time"`
	SuggestedOrderQuantity int     `gorm:"suggested_order_quantity" json:"suggested_order_quantity"`
}

// auto order detail adjusted quantity
type AdjustedOrderQuantity struct {
	AdjustedOrderQuantity int `gorm:"adjusted_order_quantity" json:"adjusted_order_quantity"`
}
