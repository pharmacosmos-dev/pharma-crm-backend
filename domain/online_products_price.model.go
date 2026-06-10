package domain

import "time"

// Append-only history — har bir narx o'zgarishi yangi row
type OnlineProductsPrice struct {
	Id             string                           `gorm:"column:id"           json:"id"`
	ProductId      string                           `gorm:"column:product_id"   json:"product_id"`
	MaterialCode   string                           `gorm:"column:material_code" json:"material_code"`
	Type           string                           `gorm:"column:type"         json:"type"`
	RetailPrice    float64                          `gorm:"column:retail_price" json:"retail_price"`
	CreatedBy      *string                          `gorm:"column:created_by"   json:"created_by"`
	CreatedAt      *time.Time                       `gorm:"column:created_at"   json:"created_at"`
	UpdatedAt      *time.Time                       `gorm:"column:updated_at"   json:"updated_at"`
	UpdatedBy      *string                          `gorm:"column:updated_by"   json:"updated_by"`
	StoreQuantity  float64                          `gorm:"column:store_quantity"  json:"store_quantity"`
	SoldQuantity   float64                          `gorm:"column:sold_quantity"   json:"sold_quantity"`
	Units          string                           `gorm:"-"                      json:"units"`
	UnitPerPack    int                              `gorm:"column:unit_per_pack"   json:"unit_per_pack"`
	ProductName    string                           `gorm:"column:product_name"    json:"-"`
	ProductBarcode string                           `gorm:"column:product_barcode" json:"-"`
	ProductPhoto   string                           `gorm:"column:product_photo"   json:"-"`
	Product        NullStruct[OnlineProductSummary] `gorm:"-"                      json:"product"`
}

func (OnlineProductsPrice) TableName() string {
	return "online_products_price"
}

// products bilan left join natijasidagi qisqa product ma'lumoti
type OnlineProductSummary struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Barcode string `json:"barcode"`
	Photos  string `json:"photos" nilable:"true"`
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
	Search       string `form:"search"`
	StoreId      string `form:"store_id"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}

// CRM dan material_code bo'yicha narx yangilash
type UpdateOnlinePriceRequest struct {
	MaterialCode string  `json:"material_code" binding:"required"`
	RetailPrice  float64 `json:"retail_price" binding:"required,gt=0"`
	Type         string  `json:"type"`
}

// CRM dan yangi online narx qo'shish
type CreateOnlinePriceRequest struct {
	MaterialCode string  `json:"material_code" binding:"required"`
	Type         string  `json:"type" binding:"required"`
	RetailPrice  float64 `json:"retail_price" binding:"required,gt=0"`
}
