package domain

import "time"

// Append-only history — har bir narx o'zgarishi yangi row
type OnlineProductsPrice struct {
	Id           string     `gorm:"id" json:"id"`
	ProductId    string     `gorm:"product_id" json:"product_id"`
	MaterialCode string     `gorm:"material_code" json:"material_code"`
	Type         string     `gorm:"type" json:"type"`
	RetailPrice  float64    `gorm:"retail_price" json:"retail_price"`
	CreatedBy    *string    `gorm:"created_by" json:"created_by"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
	UpdatedBy    *string    `gorm:"updated_by" json:"updated_by"`
}

func (OnlineProductsPrice) TableName() string {
	return "online_products_price"
}

// 1C dan keluvchi narx item
type UzumTezKorProductRepriceItem struct {
	MaterialCode string  `json:"material_code"`
	RetailPrice  float64 `json:"retail_price"`
}

// 1C request — faqat items, type, created_by token dan olinadi
type UzumTezkorProductRepriceFromOnecRequest struct {
	Items []UzumTezKorProductRepriceItem `json:"items"`
}

// CRM list uchun query param
type UzumTezkorProductQueryParam struct {
	Type         string `form:"type"`
	ProductId    string `form:"product_id"`
	MaterialCode string `form:"material_code"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}

// CRM dan material_code bo'yicha narx yangilash
type UpdateOnlinePriceRequest struct {
	MaterialCode string  `json:"material_code" binding:"required"`
	RetailPrice  float64 `json:"retail_price" binding:"required, gt=0"`
	Type         string  `json:"type"`
}
