package domain

// TotalCountStats structure
type TotalCountStats struct {
	TotalSaleCount    float64 `gorm:"total_sale_count" json:"total_sale_count"`
	TotalSaleAmount   float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
	TotalProductCount int64   `gorm:"total_product_count" json:"total_product_count"`
	TotalStoreCount   int64   `gorm:"total_store_count" json:"total_store_count"`
}

// ChartResponse structure
type ChartResponse struct {
	ID          string  `gorm:"id" json:"id"`
	Count       int64   `gorm:"count" json:"count"`
	TotalAmount float64 `gorm:"total_amount" json:"total_amount"`
	CreatedAt   string  `gorm:"created_at" json:"created_at"`
}
