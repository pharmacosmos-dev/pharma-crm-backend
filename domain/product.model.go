package domain

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Product structure
type Product struct {
	Id           string            `gorm:"id" json:"id"`
	BrandId      string            `gorm:"-" json:"brand_id"`
	UnitTypeID   string            `gorm:"unit_type_id" json:"unit_type_id"`
	ShelfID      string            `gorm:"shelf_id" json:"shelf_id"`
	ProducerID   string            `gorm:"producer_id" json:"producer_id"`
	Name         string            `gorm:"name" json:"name"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	SupplyPrice  float64           `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64           `gorm:"retail_price" json:"retail_price"`
	Quantity     int               `gorm:"quantity" json:"quantity"`
	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	Vat          float64           `gorm:"vat" json:"vat"`
	Markup       float64           `gorm:"markup" json:"markup"`
	VatPrice     float64           `gorm:"vat_price" json:"vat_price"`
	MarkupPrice  float64           `gorm:"markup_price" json:"markup_price"`
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
	UnitName     string            `gorm:"unit_name" json:"unit_name"`
	Categories   []*Category       `gorm:"many2many:category_products;foreignKey:Id;joinForeignKey:ProductId;References:Id;joinReferences:CategoryId" json:"categories"`
	StoreProduct []*StoreProduct   `gorm:"foreignKey:ProductID" json:"store_product"`
	UnitType     *UnitType         `gorm:"foreignKey:UnitTypeID" json:"unit_type"`
	Shelf        *Shelf            `gorm:"foreignKey:ShelfID" json:"shelf"`
	Producer     *Producer         `gorm:"foreignKey:ProducerID" json:"producer"`
}

// Product create request
type ProductRequest struct {
	Id           string                `gorm:"id" json:"-"`
	UnitTypeID   string                `gorm:"unit_type_id" json:"unit_type_id"`
	ShelfID      *string               `gorm:"shelf_id" json:"shelf_id"`
	ProducerID   *string               `gorm:"producer_id" json:"producer_id"`
	MaterialCode int                   `gorm:"material_code" json:"material_code"`
	Name         string                `gorm:"name" json:"name"`
	Barcode      string                `gorm:"barcode" json:"barcode"`
	Photos       utils.StringArray     `gorm:"type:text[]" json:"photos"`
	UnitPerPack  int                   `gorm:"unit_per_pack" json:"unit_per_pack"`
	Description  string                `gorm:"description" json:"description"`
	Status       string                `gorm:"status" json:"-" example:"active|inactive"`
	StoreProduct []StoreProductRequest `gorm:"-" json:"store_product"`
	CategoryIds  []string              `gorm:"-" json:"category_ids"`
}

// Product update request
type ProductUpdateRequest struct {
	Name         string                `gorm:"name" json:"name"`
	Description  string                `gorm:"description" json:"description"`
	Barcode      string                `gorm:"barcode" json:"barcode"`
	UnitTypeID   *string               `gorm:"unit_type_id" json:"unit_type_id"`
	ProducerID   *string               `gorm:"producer_id" json:"producer_id"`
	ShelfID      *string               `gorm:"shelf_id" json:"shelf_id"`
	Photos       utils.StringArray     `gorm:"type:text[]" json:"photos"`
	UnitPerPack  int                   `gorm:"unit_per_pack" json:"unit_per_pack"`
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
	Id                  string   `gorm:"type:uuid;default:gen_random_uuid()" json:"-" validate:"omitempty,uuid4"`
	MaterialCode        int      `gorm:"material_code" json:"material_code" validate:"required,gt=0"`
	Name                string   `gorm:"name" json:"name" validate:"required,min=1,max=500"`
	Manufacturer        string   `gorm:"manufacturer" json:"manufacturer" validate:"required,min=1,max=255"`
	Quantity            int      `gorm:"quantity" json:"quantity" validate:"required,gte=0"`
	RetailPrice         float64  `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat      float64  `gorm:"retail_price_vat" json:"retail_price_vat"`
	SupplyPrice         float64  `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat      float64  `gorm:"supply_price_vat" json:"supply_price_vat"`
	Sum                 float64  `gorm:"sum" json:"sum"`
	VatPrice            float64  `gorm:"vat_price" json:"vat_price"`
	Vat                 string   `gorm:"vat" json:"vat" validate:"required"`
	Markup              int      `gorm:"markup" json:"markup"`
	VatSum              float64  `gorm:"vat_sum" json:"vat_sum"`
	ProductSeriesNumber string   `gorm:"product_series_number" json:"product_series_number" validate:"required"`
	ExpireDate          string   `gorm:"expire_date" json:"expire_date" validate:"required"`
	Barcode             string   `gorm:"barcode" json:"barcode" validate:"required,min=0,max=255"`
	SumVat              float64  `gorm:"sum_vat" json:"sum_vat"`
	Ikpu                string   `gorm:"ikpu" json:"ikpu" validate:"omitempty,len=min=14,max=255"`
	Markirovka          []string `gorm:"-" json:"markirovka"`
}

var validate *validator.Validate

// Validate checks the struct fields.
func (p *ProductRequest1C) Validate() error {

	// Check struct validations
	if err := validate.Struct(p); err != nil {
		return err
	}

	return nil
}

// Create Tovar structure for 1C API
type CreateProduct1C struct {
	Dok    Document           `json:"Dok"`
	Apteka Apteka             `json:"Apteka"`
	Товары []ProductRequest1C `json:"Товары"`
}

// Product response with cart items
// Product structure
type ProductRes struct {
	Id             string            `gorm:"id" json:"id"`
	StoreProductId string            `gorm:"store_product_id" json:"store_product_id"`
	Name           string            `gorm:"name" json:"name"`
	Barcode        string            `gorm:"barcode" json:"barcode"`
	Photos         utils.StringArray `gorm:"type:text[]" json:"photos"`
	TotalPrice     float64           `gorm:"total_price" json:"total_price"`
	TotalDiscount  float64           `gorm:"total_discount" json:"total_discount"`
	Quantity       int               `gorm:"quantity" json:"quantity"`
	UnitQuantity   int               `gorm:"unit_quantity" json:"unit_quantity"`
	Description    string            `gorm:"description" json:"description"`
	BonusPercent   float64           `gorm:"bonus_percent" json:"bonus_percent"`
	BonusAmount    float64           `gorm:"bonus_amount" json:"bonus_amount"`
	ShortName      string            `gorm:"short_name" json:"short_name"`
}

type TotalStatusCount struct {
	TotalCount     int `gorm:"total_count" json:"total_count"`
	ActiveCount    int `gorm:"active_count" json:"active_count"`
	InactiveCount  int `gorm:"inactive_count" json:"inactive_count"`
	ZeroStockCount int `gorm:"zero_stock_count" json:"zero_stock_count"`
	LowStockCount  int `gorm:"low_stock_count" json:"low_stock_count"`
	ImminentCount  int `gorm:"imminent_count" json:"imminent_count"`
	ExpiredCount   int `gorm:"expired_count" json:"expired_count"`
}

// External API response structure
type ProductExternal struct {
	Id          string            `gorm:"id" json:"id"`
	Name        string            `gorm:"name" json:"name"`
	Barcode     string            `gorm:"barcode" json:"barcode"`
	Photos      utils.StringArray `gorm:"type:text[]" json:"photos"`
	Quantity    int               `gorm:"quantity" json:"quantity"`
	Description string            `gorm:"description" json:"description"`
	UnitName    string            `gorm:"unit_name" json:"unit_name"`
	Stores      []StoreExternal   `gorm:"-" json:"stores"`
	Categories  []string          `gorm:"-" json:"categories"`
}

// Store external API response structure
type StoreExternal struct {
	Id           string     `gorm:"id" json:"id"`
	Name         string     `gorm:"name" json:"name"`
	Address      string     `gorm:"address" json:"address"`
	Location     string     `gorm:"location" json:"location"`
	Quantity     int        `gorm:"quantity" json:"quantity"`
	UnitQuantity int        `gorm:"unit_quantity" json:"unit_quantity"`
	RetailPrice  float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate   *time.Time `gorm:"expire_date" json:"expire_date"`
}
