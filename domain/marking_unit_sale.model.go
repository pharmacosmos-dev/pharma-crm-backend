package domain

import "time"

// MarkingUnitSale: markirovka bo'yicha sotilgan va qolgan donalar.
type MarkingUnitSale struct {
	Id             string     `gorm:"column:id" json:"id"`
	Marking        string     `gorm:"column:marking" json:"marking"`
	ProductId      string     `gorm:"column:product_id" json:"product_id"`
	ProductName    string     `gorm:"column:product_name" json:"product_name"`
	StoreId        string     `gorm:"column:store_id" json:"store_id"`
	StoreName      string     `gorm:"column:store_name" json:"store_name"`
	StoreProductId string     `gorm:"column:store_product_id" json:"store_product_id"`
	UnitPerPack    int        `gorm:"column:unit_per_pack" json:"unit_per_pack"`
	SoldQuantity   int        `gorm:"column:sold_quantity" json:"sold_quantity"`
	StockQuantity  int        `gorm:"column:stock_quantity" json:"stock_quantity"`
	CreatedAt      *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// MarkingUnitSaleQueryParams structure
type MarkingUnitSaleQueryParams struct {
	Marking string `form:"marking"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}
