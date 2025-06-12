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
	Id            string  `gorm:"id" json:"id"`
	StoreId       string  `gorm:"store_id" json:"store_id"`
	Status        string  `gorm:"status" json:"status"`
	AutoOrderDate string  `gorm:"auto_order_date" json:"auto_order_date"`
	IntervalDay   float64 `gorm:"-" json:"interval_day"`
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
	MinStock               float64    `gorm:"min_stock" json:"min_stock"`
	MaxStock               float64    `gorm:"max_stock" json:"max_stock"`
	CurrentStock           float64    `gorm:"current_stock" json:"current_stock"`
	SaleCount              float64    `gorm:"sale_count" json:"sale_count"`
	DailySaleCount         float64    `gorm:"daily_sale_count" json:"daily_sale_count"`
	OrderCount             float64    `gorm:"order_count" json:"order_count"`
	ResponseOrderCount     float64    `gorm:"response_order_count" json:"response_order_count"`
	ImportDay              int        `gorm:"import_day" json:"import_day"`
	SalePeriod             int        `gorm:"sale_period" json:"sale_period"`
	StockOnDeliveryDate    float64    `gorm:"stock_on_delivery_date" json:"stock_on_delivery_date"`
	ReserveQuantity        float64    `gorm:"reserve_quantity" json:"reserve_quantity"`
	FutureStock            float64    `gorm:"future_stock" json:"future_stock"`
	FutureStockWithReserve float64    `gorm:"future_stock_with_reserve" json:"future_stock_with_reserve"`
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
	MinStock               float64 `gorm:"min_stock" json:"min_stock"`
	MaxStock               float64 `gorm:"max_stock" json:"max_stock"`
	CurrentStock           float64 `gorm:"current_stock" json:"current_stock"`
	SaleCount              float64 `gorm:"sale_count" json:"sale_count"`
	DailySaleCount         float64 `gorm:"daily_sale_count" json:"daily_sale_count"`
	OrderCount             float64 `gorm:"order_count" json:"order_count"`
	ImportDay              int     `gorm:"import_day" json:"import_day"`
	SalePeriod             int     `gorm:"sale_period" json:"sale_period"`
	StockOnDeliveryDate    float64 `gorm:"stock_on_delivery_date" json:"stock_on_delivery_date"`
	ReserveQuantity        float64 `gorm:"reserve_quantity" json:"reserve_quantity"`
	FutureStock            float64 `gorm:"future_stock" json:"future_stock"`
	FutureStockWithReserve float64 `gorm:"future_stock_with_reserve" json:"future_stock_with_reserve"`
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

// auto order product structure
type AutoOrderProduct struct {
	MaterialCode int    `json:"material_code"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Quantity     int    `json:"quantity"`
	QuantityFakt int    `json:"quantity_fakt"`
}
