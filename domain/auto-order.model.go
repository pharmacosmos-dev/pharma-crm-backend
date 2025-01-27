package domain

type AutoOrder struct {
	StoreId         string  `gorm:"store_id" json:"store_id"`
	ProductId       string  `gorm:"product_id" json:"product_id"`
	StoreName       string  `gorm:"store_name" json:"store_name"`
	ProductName     string  `gorm:"product_name" json:"product_name"`
	CurrentStock    int     `gorm:"current_stock" json:"current_stock"`
	MonthlyQuantity float64 `gorm:"monthly_quantity" json:"monthly_quantity"`
	WeeklyQuantity  float64 `gorm:"weekly_quantity" json:"weekly_quantity"`
	SuggestedOrder  float64 `gorm:"suggested_order" json:"suggested_order"`
	AdjustedOrder   float64 `gorm:"adjusted_order" json:"adjusted_order"`
	TotalCount      int64   `gorm:"total_count" json:"total_count"`
}
