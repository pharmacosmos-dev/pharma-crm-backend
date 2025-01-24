package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Product
type Product struct {
	Id           string            `gorm:"id" json:"id"`
	BrandId      string            `gorm:"-" json:"brand_id"`
	UnitTypeID   string            `gorm:"unit_type_id" json:"unit_type_id"`
	Name         string            `gorm:"name" json:"name"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64           `gorm:"supply_price" json:"supply_price"`
	Markup       int               `gorm:"markup" json:"markup"`
	RetailPrice  float64           `gorm:"retail_price" json:"retail_price"`
	Quantity     int               `gorm:"quantity" json:"quantity"`
	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	Vat          int               `gorm:"vat" json:"vat"`
	VatPrice     float64           `gorm:"vat_price" json:"vat_price"`
	Sum          float64           `gorm:"sum" json:"sum"`
	Description  string            `gorm:"description" json:"description"`
	Status       string            `gorm:"status" json:"status"`
	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
	MaterialCode int               `gorm:"material_code" json:"material_code"`
	ExpireDate   string            `gorm:"expire_date" json:"expire_date"`
	IsActive     bool              `gorm:"is_active" json:"is_active"`
	BonusPercent float64           `gorm:"bonus_percent" json:"bonus_percent"`
	BonusAmount  float64           `gorm:"bonus_amount" json:"bonus_amount"`
	CreatedAt    *time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time        `gorm:"updated_at" json:"updated_at"`
	Categories   []*Category       `gorm:"many2many:category_products;foreignKey:Id;joinForeignKey:ProductId;References:Id;joinReferences:CategoryId" json:"categories"`
	StoreProduct []*StoreProduct   `gorm:"foreignKey:ProductID" json:"store_product"`
	UnitType     *UnitType         `gorm:"foreignKey:UnitTypeID" json:"unit_type"`
	UnitName     string            `gorm:"unit_name" json:"unit_name"`
}

// Product create request
type ProductRequest struct {
	Id           string                `gorm:"id" json:"-"`
	UnitTypeID   string                `gorm:"unit_type_id" json:"unit_type_id"`
	MaterialCode int                   `gorm:"material_code" json:"material_code"`
	Name         string                `gorm:"name" json:"name"`
	Barcode      string                `gorm:"barcode" json:"barcode"`
	Photos       utils.StringArray     `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64               `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64               `gorm:"retail_price" json:"retail_price"`
	Quantity     int                   `gorm:"quantity" json:"quantity"`
	UnitPerPack  int                   `gorm:"unit_per_pack" json:"unit_per_pack"`
	Vat          int                   `gorm:"vat" json:"vat"`
	VatPrice     float64               `gorm:"vat_price" json:"vat_price"`
	Sum          float64               `gorm:"sum" json:"-"`
	Description  string                `gorm:"description" json:"description"`
	Status       string                `gorm:"status" json:"-" example:"active|inactive"`
	Manufacturer string                `gorm:"manufacturer" json:"manufacturer"`
	ExpireDate   *time.Time            `gorm:"expire_date" json:"expire_date"`
	BonusPercent float64               `gorm:"bonus_percent" json:"bonus_percent"`
	BonusAmount  float64               `gorm:"bonus_amount" json:"bonus_amount"`
	StoreProduct []StoreProductRequest `gorm:"-" json:"store_product"`
	CategoryIds  []string              `gorm:"-" json:"category_ids"`
}

// Product update request
type ProductUpdateRequest struct {
	Name         string                `gorm:"name" json:"name"`
	Barcode      string                `gorm:"barcode" json:"barcode"`
	UnitTypeID   string                `gorm:"unit_type_id" json:"unit_type_id"`
	Photos       utils.StringArray     `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64               `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64               `gorm:"retail_price" json:"retail_price"`
	Quantity     int                   `gorm:"quantity" json:"quantity"`
	UnitPerPack  int                   `gorm:"unit_per_pack" json:"unit_per_pack"`
	Vat          int                   `gorm:"vat" json:"vat"`
	VatPrice     float64               `gorm:"vat_price" json:"vat_price"`
	Sum          float64               `gorm:"sum" json:"-"`
	Description  string                `gorm:"description" json:"description"`
	Status       string                `gorm:"status" json:"-" example:"active|inactive"`
	Manufacturer string                `gorm:"manufacturer" json:"manufacturer"`
	ExpireDate   *time.Time            `gorm:"expire_date" json:"expire_date"`
	BonusPercent float64               `gorm:"bonus_percent" json:"bonus_percent"`
	BonusAmount  float64               `gorm:"bonus_amount" json:"bonus_amount"`
	StoreProduct []StoreProductRequest `gorm:"-" json:"store_product"`
	CategoryIds  []string              `gorm:"-" json:"category_ids"`
}

// Product Upload request
type ProductUploadReq struct {
	Id           string  `gorm:"id" json:"id"`
	Name         string  `gorm:"name" json:"name"`
	Barcode      string  `gorm:"barcode" json:"barcode"`
	SupplyPrice  float64 `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64 `gorm:"retail_price" json:"retail_price"`
	Quantity     int     `gorm:"quantity" json:"quantity"`
	Vat          float64 `gorm:"vat" json:"vat"`
	VatPrice     float64 `gorm:"vat_price" json:"vat_price"`
	Sum          float64 `gorm:"sum" json:"sum"`
	Status       string  `gorm:"status" json:"status"`
	Manufacturer string  `gorm:"manufacturer" json:"manufacturer"`
	ExpireDate   string  `gorm:"expire_date" json:"expire_date"`
	IsActive     bool    `gorm:"is_active" json:"is_active"`
}

// Product Producer
type ProductProducer struct {
	Id           string `gorm:"id" json:"id"`
	Manufacturer string `gorm:"manufacturer" json:"name"`
}

// Document structure for 1C API
type Document struct {
	ID             string `gorm:"id" json:"-"`
	DocumentDate   string `gorm:"document_date" json:"data_dok"`
	DocumentNumber string `gorm:"document_number" json:"nomer_dok"`
	StoreCode      int    `gorm:"store_code" json:"-"`
}

// Apteka structure for 1C API
type Apteka struct {
	StoreCode int    `gorm:"store_code" json:"store_code"`
	Name      string `gorm:"name" json:"name"`
}

// Request structure for 1C API
type ProductRequest1C struct {
	Id                  string  `gorm:"type:uuid;default:gen_random_uuid()" json:"-"`
	MaterialCode        int     `gorm:"material_code" json:"material_code"`
	Name                string  `gorm:"name" json:"name"`
	Manufacturer        string  `gorm:"manufacturer" json:"manufacturer"`
	Quantity            int     `gorm:"quantity" json:"quantity"`
	UnitPerPack         int     `gorm:"unit_per_pack" json:"unit_per_pack"`
	RetailPrice         float64 `gorm:"retail_price" json:"retail_price"`
	SupplyPrice         float64 `gorm:"supply_price" json:"supply_price"`
	Sum                 float64 `gorm:"sum" json:"sum"`
	VatPrice            float64 `gorm:"vat_price" json:"vat_price"`
	Vat                 string  `gorm:"vat" json:"vat"`
	VatSum              float64 `gorm:"vat_sum" json:"vat_sum"`
	ProductSeriesNumber string  `gorm:"product_series_number" json:"product_series_number"`
	ExpireDate          string  `gorm:"expire_date" json:"expire_date"`
	Barcode             string  `gorm:"barcode" json:"barcode"`
}

// Create Tovar structure for 1C API
type CreateProduct1C struct {
	Dok    Document           `json:"Dok"`
	Apteka Apteka             `json:"Apteka"`
	Товары []ProductRequest1C `json:"Товары"`
}
