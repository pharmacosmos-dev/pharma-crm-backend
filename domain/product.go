package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Product
type Product struct {
	Id                 string            `gorm:"id" json:"id" db:"id"`
	StoreId            string            `gorm:"-" json:"store_id" db:"store_id"`
	CategoryId         string            `gorm:"category_id" json:"category_id" db:"category_id"`
	BrandId            string            `gorm:"-" json:"brand_id" db:"brand_id"`
	SupplierId         string            `gorm:"-" json:"supplier_id" db:"supplier_id"`
	UnitId             string            `gorm:"-" json:"unit_id" db:"unit_id"`
	ProductVariability string            `gorm:"product_variability" json:"product_variability" db:"product_variability"`
	Name               string            `gorm:"name" json:"name" db:"name"`
	Sku                string            `gorm:"sku" json:"sku" db:"sku"`
	Barcode            string            `gorm:"barcode" json:"barcode" db:"barcode"`
	MainPhoto          string            `gorm:"main_photo" json:"main_photo" db:"main_photo"`
	Photos             utils.StringArray `gorm:"type:text[]" json:"photos" db:"photos"`
	SupplyPrice        float64           `gorm:"supply_price" json:"supply_price" db:"supply_price"`
	Markup             int               `gorm:"markup" json:"markup" db:"markup"`
	RetailPrice        float64           `gorm:"retail_price" json:"retail_price" db:"retail_price"`
	Quantity           int               `gorm:"quantity" json:"quantity" db:"quantity"`
	Vat                int               `gorm:"vat" json:"vat" db:"vat"`
	VatPrice           float64           `gorm:"vat_price" json:"vat_price" db:"vat_price"`
	Sum                float64           `gorm:"sum" json:"sum" db:"sum"`
	Description        string            `gorm:"description" json:"description" db:"description"`
	Status             string            `gorm:"status" json:"status" db:"status"`
	Manufacturer       string            `gorm:"manufacturer" json:"manufacturer" db:"manufacturer"`
	ExpireDate         string            `gorm:"expire_date" json:"expire_date" db:"expire_date"`
	Category           *Category         `gorm:"foreignKey:CategoryId" json:"category" db:"category"`
	Store              *Store            `gorm:"foreignKey:StoreId" json:"store" db:"store"`
	CreatedAt          *time.Time        `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time        `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}

// Product create request
type ProductRequest struct {
	Id           string            `gorm:"id" json:"-"`
	StoreId      string            `gorm:"store_id" json:"store_id"`
	CategoryId   string            `gorm:"category_id" json:"category_id"`
	Name         string            `gorm:"name" json:"name"`
	Sku          string            `gorm:"sku" json:"sku"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	MainPhoto    string            `gorm:"main_photo" json:"main_photo"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64           `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64           `gorm:"retail_price" json:"retail_price"`
	Quantity     int               `gorm:"quantity" json:"quantity"`
	Vat          int               `gorm:"vat" json:"vat"`
	VatPrice     float64           `gorm:"vat_price" json:"vat_price"`
	Sum          float64           `gorm:"sum" json:"sum"`
	Description  string            `gorm:"description" json:"description"`
	Status       string            `gorm:"status" json:"status"`
	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
	ExpireDate   string            `gorm:"expire_date" json:"expire_date"`
}

// Product update request
type ProductUpdateRequest struct {
	Id           string            `gorm:"id" json:"id"`
	StoreId      string            `gorm:"store_id" json:"store_id"`
	CategoryId   string            `gorm:"category_id" json:"category_id"`
	Name         string            `gorm:"name" json:"name"`
	Sku          string            `gorm:"sku" json:"sku"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	MainPhoto    string            `gorm:"main_photo" json:"main_photo"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64           `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64           `gorm:"retail_price" json:"retail_price"`
	Quantity     int               `gorm:"quantity" json:"quantity"`
	Vat          int               `gorm:"vat" json:"vat"`
	VatPrice     float64           `gorm:"vat_price" json:"vat_price"`
	Sum          float64           `gorm:"sum" json:"sum"`
	Description  string            `gorm:"description" json:"description"`
	Status       string            `gorm:"status" json:"status"`
	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
	ExpireDate   string            `gorm:"expire_date" json:"expire_date"`
}

// Product Upload request
type ProductUploadReq struct {
	Id           string   `gorm:"id" json:"id" db:"id"`
	CategoryId   string   `gorm:"category_id" json:"category_id" db:"category_id"`
	Name         string   `gorm:"name" json:"name" db:"name"`
	Barcode      string   `gorm:"barcode" json:"barcode" db:"barcode"`
	SupplyPrice  float64  `gorm:"supply_price" json:"supply_price" db:"supply_price"`
	RetailPrice  float64  `gorm:"retail_price" json:"retail_price" db:"retail_price"`
	Quantity     int      `gorm:"quantity" json:"quantity" db:"quantity"`
	Vat          int      `gorm:"vat" json:"vat" db:"vat"`
	VatPrice     float64  `gorm:"vat_price" json:"vat_price" db:"vat_price"`
	Sum          float64  `gorm:"sum" json:"sum" db:"sum"`
	Status       string   `gorm:"status" json:"status" db:"status"`
	Manufacturer string   `gorm:"manufacturer" json:"manufacturer" db:"manufacturer"`
	ExpireDate   string   `gorm:"expire_date" json:"expire_date" db:"expire_date"`
	Category     Category `gorm:"foreignKey:CategoryId" json:"category" db:"category"`
}

// Product Filter request
type ProductFilterReq struct {
	StoreId         string  `json:"store_id"`
	CategoryId      string  `json:"category_id"`
	ProducerId      string  `json:"producer_id"`
	SupplyPriceFrom float64 `json:"supply_price_from"`
	SupplyPriceTo   float64 `json:"supply_price_to"`
	RetailPriceFrom float64 `json:"retail_price_from"`
	RetailPriceTo   float64 `json:"retail_price_to"`
}

// Product Producer
type ProductProducer struct {
	Id           string `gorm:"id" json:"id"`
	Manufacturer string `gorm:"manufacturer" json:"manufacturer"`
}
