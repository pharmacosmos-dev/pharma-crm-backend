package domain

type RejectedProductRequest struct {
	ProductID   *string `gorm:"product_id" json:"product_id"` // nullable
	ProductName *string `gorm:"product_name" json:"product_name" `
	StoreID     string  `gorm:"store_id" json:"store_id" binding:"required"`
	Reason      string  `gorm:"reason" json:"reason"`
	CreatedBy   string  `gorm:"created_by" json:"-"`
}
type RejectedProductQueryParam struct {
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	Search      string `form:"search"`
	StoreID     string `form:"store_id"`
	ProductID   string `form:"product_id"`
	ProductName string `form:"product_name"`
}

type RejectedProduct struct {
	Id          string `json:"id"`
	StoreID     string `json:"store_id"`
	StoreName   string `json:"store_name"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Count       int64  `json:"count"`
	TotalCount  int64  `json:"-"`
	Reason      string `json:"reason"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}
