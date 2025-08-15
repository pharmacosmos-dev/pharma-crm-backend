package domain

type RejectedProductRequest struct {
	ProductID     *string `gorm:"product_id" json:"product_id"` // nullable
	ProductName   string  `gorm:"product_name" json:"product_name" `
	StoreID       string  `gorm:"store_id" json:"store_id" binding:"required"`
	Reason        string  `gorm:"reason" json:"reason"`
	RejectedTimes float64 `gorm:"rejected_times" json:"-"`
	CreatedBy     *string `gorm:"created_by" json:"-"`
}
