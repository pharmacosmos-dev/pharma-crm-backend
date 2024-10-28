package domain

import "time"

// Product

type Product struct {
	Id                 string     `json:"id" db:"id"`
	StoreId            string     `json:"store_id" db:"store_id"`
	CategoryId         string     `json:"category_id" db:"category_id"`
	BrandId            string     `json:"brand_id" db:"brand_id"`
	SupplierId         string     `json:"supplier_id" db:"supplier_id"`
	UnitId             string     `json:"unit_id" db:"unit_id"`
	ProductType        string     `json:"product_type" db:"product_type"`
	ProductVariability string     `json:"product_variability" db:"product_variability"`
	Name               string     `json:"name" db:"name"`
	Sku                string     `json:"sku" db:"sku"`
	BarCode            string     `json:"bar_code" db:"barcode"`
	MainPhoto          string     `json:"main_photo" db:"main_photo"`
	Photos             []string   `json:"photos" db:"photos"`
	SupplyPrice        string     `json:"supply_price" db:"supply_price"`
	Markup             int        `json:"markup" db:"markup"`
	RetailPrice        string     `json:"retail_price" db:"retail_price"`
	Quantity           int        `json:"quantity" db:"quantity"`
	Sum                string     `json:"sum" db:"sum"`
	Description        string     `json:"description" db:"description"`
	Status             string     `json:"status" db:"status"`
	Manufacturer       string     `json:"manufacturer" db:"manufacturer"`
	ExpireDate         string     `json:"expire_date" db:"expire_date"`
	CreatedAt          *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at" db:"updated_at"`
}
