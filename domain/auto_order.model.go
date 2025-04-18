package domain

import "time"

// Auto order query param
type AutoOrderParam struct {
	StoreID   string `form:"store_id"`
	Search    string `form:"search"`
	Status    string `form:"status"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	UserId    string `form:"user_id"`
}

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
	MinStock              int `gorm:"min_stock" json:"min_stock"`
	MaxStock              int `gorm:"max_stock" json:"max_stock"`
	Kvant                 int `gorm:"kvant" json:"kvant"`
}

// auto order detail send request structure
type AutoOrderDetailSendRequest struct {
	Dok    AutoOrderDocument  `json:"Dok"`
	Apteka Apteka             `json:"Apteka"`
	Товары []ProductAutoOrder `json:"Товары"`
}

type ProductAutoOrder struct {
	MaterialCode int    `gorm:"material_code" json:"material_code"`
	Name         string `gorm:"name" json:"name"`
	Manufacturer string `gorm:"manufacturer" json:"manufacturer"`
	Quantity     int    `gorm:"quantity" json:"quantity"`
}

type AutoOrderDocument struct {
	DataDok  string `gorm:"data_dok" json:"data_dok"`
	NomerDok string `gorm:"nomer_dok" json:"nomer_dok"`
}

// response from 1c
type AutoOrderResponse struct {
	Ok       string             `json:"ok"`
	Code     int                `json:"code"`
	Message  string             `json:"message"`
	Data     string             `json:"data"`
	Products []AutoOrderProduct `json:"Товары"`
}

type AutoOrderProduct struct {
	MaterialCode int    `json:"material_code"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Quantity     int    `json:"quantity"`
	QuantityFakt int    `json:"quantity_fakt"`
}
