package domain

// TotalCountStats structure
type TotalCountStats struct {
	TotalSaleCount     float64 `gorm:"total_sale_count" json:"total_sale_count"`
	TotalSaleAmount    float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
	TotalProductCount  int64   `gorm:"total_product_count" json:"total_product_count"`
	TotalStoreCount    int64   `gorm:"total_store_count" json:"total_store_count"`
	StockTotalAmount   float64 `gorm:"stock_total_amount" json:"stock_total_amount"`
	ExpiringSoonCount  int64   `gorm:"expiring_soon_count" json:"expiring_soon_count"`
	ExpiringSoonAmount float64 `gorm:"expiring_soon_amount" json:"expiring_soon_amount"`
	TotalNetIncome     float64 `gorm:"total_net_income" json:"total_net_income"`
	BonusAmount        float64 `gorm:"bonus_amount" json:"bonus_amount"`
}

// ChartResponse structure
type ChartResponse struct {
	ID          string  `gorm:"id" json:"id"`
	Count       int64   `gorm:"count" json:"count"`
	TotalAmount float64 `gorm:"total_amount" json:"total_amount"`
	CreatedAt   string  `gorm:"created_at" json:"created_at"`
}

// Top Stores structure
type TopStores struct {
	Id          string  `gorm:"id" json:"id"`
	Name        string  `gorm:"name" json:"name"`
	Count       int64   `gorm:"count" json:"count"`
	TotalAmount float64 `gorm:"total_amount" json:"total_amount"`
}

// Top Products structure
type TopProducts struct {
	Id          string  `gorm:"id" json:"id"`
	Name        string  `gorm:"name" json:"name"`
	Count       string  `gorm:"count" json:"count"`
	TotalAmount float64 `gorm:"total_amount" json:"total_amount"`
}

// Dashboard query param
type DashboardQueryParam struct {
	StoreId   string `form:"store_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Type      string `form:"type"`
}
