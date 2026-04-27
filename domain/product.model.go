package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Product structure
type Product struct {
	Id              string               `gorm:"id" json:"id"`
	BrandId         string               `gorm:"-" json:"brand_id"`
	UnitTypeID      string               `gorm:"unit_type_id" json:"unit_type_id"`
	ShelfID         string               `gorm:"shelf_id" json:"shelf_id"`
	ProducerID      string               `gorm:"producer_id" json:"producer_id"`
	Name            string               `gorm:"name" json:"name"`
	Barcode         string               `gorm:"barcode" json:"barcode"`
	Photos          utils.StringArray    `gorm:"type:text[]" json:"photos"`
	SupplyPrice     float64              `gorm:"supply_price" json:"supply_price"`
	RetailPrice     float64              `gorm:"retail_price" json:"retail_price"`
	RetailUnitPrice float64              `gorm:"retail_unit_price" json:"retail_unit_price"`
	Quantity        float64              `gorm:"quantity" json:"quantity"`
	UnitPerPack     int                  `gorm:"unit_per_pack" json:"unit_per_pack"`
	Vat             float64              `gorm:"vat" json:"vat"`
	Markup          float64              `gorm:"markup" json:"markup"`
	VatPrice        float64              `gorm:"vat_price" json:"vat_price"`
	MarkupPrice     float64              `gorm:"markup_price" json:"markup_price"`
	Sum             float64              `gorm:"sum" json:"sum"`
	Description     string               `gorm:"description" json:"description"`
	Status          string               `gorm:"status" json:"status"`
	Manufacturer    string               `gorm:"manufacturer" json:"manufacturer"`
	MaterialCode    int                  `gorm:"material_code" json:"material_code"`
	ExpireDate      string               `gorm:"expire_date" json:"expire_date"`
	IsActive        bool                 `gorm:"is_active" json:"is_active"`
	BonusPercent    float64              `gorm:"bonus_percent" json:"bonus_percent"`
	BonusAmount     float64              `gorm:"bonus_amount" json:"bonus_amount"`
	MaxPrice        float64              `gorm:"max_price" json:"max_price"`
	IsMarking       bool                 `gorm:"is_marking" json:"is_marking"`
	CreatedAt       *time.Time           `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time           `gorm:"updated_at" json:"updated_at"`
	UnitName        string               `gorm:"unit_name" json:"unit_name"`
	UnitType        NullStruct[UnitType] `gorm:"-" json:"unit_type"`
	Shelf           NullStruct[Shelf]    `gorm:"-" json:"shelf"`
	Producer        NullStruct[Producer] `gorm:"-" json:"producer"`
	CategoryName    string               `gorm:"-" json:"category_name"`
	Markings        []string             `gorm:"-" json:"markings"`
	Categories           []Category           `gorm:"-" json:"categories"`
	RequiresPrescription bool                 `gorm:"requires_prescription" json:"requires_prescription"`
	CountryId            *string              `gorm:"country_id" json:"country_id"`
	Country              NullStruct[Country]  `gorm:"-" json:"country"`
	IsReturn             bool                 `gorm:"is_return" json:"is_return"`
}

// Product create request
type ProductRequest struct {
	Id           string            `gorm:"id" json:"-"`
	UnitTypeId   string            `gorm:"unit_type_id" json:"unit_type_id"`
	ShelfId      *string           `gorm:"shelf_id" json:"shelf_id"`
	ProducerId   *string           `gorm:"producer_id" json:"producer_id"`
	MaterialCode int               `gorm:"material_code" json:"material_code"`
	Name         string            `gorm:"name" json:"name"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	Description  string            `gorm:"description" json:"description"`
	Status       string            `gorm:"status" json:"-" example:"active|inactive"`
	CategoryIds  []string          `gorm:"-" json:"category_ids"`
}

// ProductExcludeRequest request
type ProductExcludeRequest struct {
	ProducerID *string   `json:"producer_id" binding:"omitempty,uuid"`
	StoreID    []*string `json:"store_id"` // optional
	ProductID  []string  `json:"product_id"`
}

// ExcludedProductResponse response
type ExcludedProductResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	StoreID     *string   `json:"store_id,omitempty"`
	StoreName   *string   `json:"store_name,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// Product update request
type ProductUpdateRequest struct {
	Name        string            `gorm:"name" json:"name"`
	Description string            `gorm:"description" json:"description"`
	Barcode     string            `gorm:"barcode" json:"barcode"`
	UnitTypeId  string            `gorm:"unit_type_id" json:"unit_type_id"`
	ProducerId  string            `gorm:"producer_id" json:"producer_id"`
	ShelfId     string            `gorm:"shelf_id" json:"shelf_id"`
	Photos      utils.StringArray `gorm:"type:text[]" json:"photos"`
	UnitPerPack int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	CategoryId  string            `gorm:"category_id" json:"category_id"`
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

// Product response with cart items
// Product structure
type ProductRes struct {
	Id             string            `gorm:"id" json:"id"`
	StoreProductId string            `gorm:"store_product_id" json:"store_product_id"`
	Name           string            `gorm:"name" json:"name"`
	Barcode        string            `gorm:"barcode" json:"barcode"`
	IsMarking      bool              `gorm:"is_marking" json:"is_marking"`
	Photos         utils.StringArray `gorm:"type:text[]" json:"photos"`
	UnitPrice      float64           `gorm:"unit_price" json:"unit_price"`
	PackPrice      float64           `gorm:"pack_price" json:"pack_price"`
	TotalPrice     float64           `gorm:"total_price" json:"total_price"`
	TotalDiscount  float64           `gorm:"total_discount" json:"total_discount"`
	Quantity       float64           `gorm:"quantity" json:"quantity"`
	UnitQuantity   float64           `gorm:"unit_quantity" json:"unit_quantity"`
	MarkingCount   int               `gorm:"marking_count" json:"marking_count"`
	Description    string            `gorm:"description" json:"description"`
	BonusAmount    float64           `gorm:"bonus_amount" json:"bonus_amount"`
	ShortName      string            `gorm:"short_name" json:"short_name"`
	ClassCode      string            `gorm:"class_code" json:"class_code"`
	PackageName    string            `gorm:"package_name" json:"package_name"`
	Vat            float64           `gorm:"vat" json:"vat"`
	VatPercent     float64           `gorm:"vat_percent" json:"vat_percent"`
	DiscountAmount float64           `gorm:"discount_amount" json:"discount_amount"`
}

type ProductStats struct {
	TotalQuantity    float64 `gorm:"total_quantity" json:"total_quantity"`
	TotalCount       float64 `gorm:"total_count" json:"total_count"`
	ActiveCount      float64 `gorm:"active_count" json:"active_count"`
	TotalStockAmount float64 `gorm:"total_stock_amount" json:"total_stock_amount"`
	InactiveCount    float64 `gorm:"inactive_count" json:"inactive_count"`
	ZeroStockCount   float64 `gorm:"zero_stock_count" json:"zero_stock_count"`
	LowStockCount    float64 `gorm:"low_stock_count" json:"low_stock_count"`
	ImminentCount    float64 `gorm:"imminent_count" json:"imminent_count"`
	ExpiredCount     float64 `gorm:"expired_count" json:"expired_count"`
}

// product list query params
type ProductQueryParam struct {
	ProductId       string      `form:"product_id"`
	StoreId         string      `form:"store_id"`
	CompanyId       string      `form:"company_id"`
	ProducerId      string      `form:"producer_id"`
	ImportId        string      `form:"import_id"`
	CategoryId      string      `form:"category_id"`
	SearchField     string      `form:"search"`
	SupplyPriceFrom float64     `form:"supply_price_from"`
	SupplyPriceTo   float64     `form:"supply_price_to"`
	RetailPriceFrom float64     `form:"retail_price_from"`
	RetailPriceTo   float64     `form:"retail_price_to"`
	Status          string      `form:"status"`
	NoBarcode       bool        `form:"no_barcode"`
	IsReturn		*bool      `form:"is_return"`
	Order           string      `form:"order"`
	Category        int         `form:"category"`
	ExpiryFrom      string      `form:"expiry_from"`
	ExpiryTo        string      `form:"expiry_to"`
	StoreIds        []string    `from:"store_ids"`
	CompanyIds      []string    `form:"company_ids"`
	Limit           int         `form:"limit"`
	Offset          int         `form:"offset"`
	StartDate       *CustomTime `form:"start_date"`
	EndDate         *CustomTime `form:"end_date"`
}

// update barcode structure
type UpdateBarcodeRequest struct {
	Id         string `json:"id"`
	Barcode    string `json:"barcode"`
	Mxik       string `json:"mxik"`
	UnitCode   string `json:"unit_code"`
	UnitLabel  string `json:"unit_label"`
	ExpireDate string `json:"expire_date"` // format: 2006-01-02
}

// update is marking request
type UpdateIsMarking struct {
	ID         string `json:"id"` // this id is store_product_id
	ProductId  string `json:"product_id"`
	IsMarking  *bool  `json:"is_marking"`
	IsChecking *bool  `json:"is_checking"` // this field is used to check if the product is checking or not
}

// product list data
type ProductData struct {
	ID           string            `gorm:"id" json:"id"`
	StoreName    string            `gorm:"store_name" json:"store_name"`
	MaterialCode int               `gorm:"material_code" json:"material_code"`
	Name         string            `gorm:"name" json:"name"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	Barcodes     utils.StringArray `gorm:"type:text[]" json:"barcodes"`
	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	MXIK         string            `gorm:"mxik" json:"mxik"`
	UnitCode     string            `gorm:"unit_code" json:"unit_code"`
	IsMarking    bool              `gorm:"is_marking" json:"is_marking"`
	CreatedAt    time.Time         `gorm:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"updated_at" json:"updated_at"`
	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
	UnitName     string            `gorm:"unit_name" json:"unit_name"`
	ShortName    string            `gorm:"short_name" json:"short_name"`
	UnitLabel    string            `gorm:"unit_label" json:"unit_label"`
	ExpireDate   *time.Time        `gorm:"expire_date" json:"expire_date"`
	ExpireDay    int               `gorm:"expire_day" json:"expire_day"`
	SerialNumber string            `gorm:"serial_number" json:"serial_number"`
	UnitQuantity int               `gorm:"unit_quantity" json:"unit_quantity"`
	Quantity     int               `gorm:"quantity" json:"quantity"`
	Units        string            `gorm:"units" json:"units"`
	CategoryName string            `gorm:"category_name" json:"category_name"`
	SupplyPrice  float64           `gorm:"supply_price" json:"supply_price"`
	RetailPrice  float64           `gorm:"retail_price" json:"retail_price"`
	Vat          int               `gorm:"vat" json:"vat"`
	Markup       float64           `gorm:"markup" json:"markup"`
	VatPrice     float64           `gorm:"vat_price" json:"vat_price"`
	MarkupPrice          float64   `gorm:"markup_price" json:"markup_price"`
	Sum                  float64   `gorm:"sum" json:"sum"`
	RequiresPrescription bool      `gorm:"requires_prescription" json:"requires_prescription"`
	//CountryId            *string   `gorm:"country_id" json:"country_id"`
	Country              string    `gorm:"country" json:"country"`
	IsReturn             bool      `gorm:"is_return" json:"is_return"`
}

// product response structure for arzon apteka
type ProductArzon struct {
	Id           string  `gorm:"id" json:"id"`
	Name         string  `gorm:"name" json:"name"`
	ProducerName string  `gorm:"producer_name" json:"producer_name"`
	RetailPrice  float64 `gorm:"retail_price" json:"retail_price"`
}

type SingeProductDashoard struct {
	UnitQuantity         int     `gorm:"unit_quantity" json:"unit_quantity"`
	SaleCount            int     `gorm:"sale_count" json:"sale_count"`
	SaleAmount           float64 `gorm:"sale_amount" json:"sale_amount"`
	ReturnSaleCount      int     `gorm:"return_sale_count" json:"return_sale_count"`
	ReturnSaleAmount     float64 `gorm:"return_sale_amount" json:"return_sale_amount"`
	ImportCount          int     `gorm:"import_count" json:"import_count"`
	ImportAmount         float64 `gorm:"import_amount" json:"import_amount"`
	ReturnToSkladCount   int     `gorm:"return_to_sklad_count" json:"return_to_sklad_count"`
	ReturnToSkladAmount  float64 `gorm:"return_to_sklad_amount" json:"return_to_sklad_amount"`
	TransferOutCount     int     `gorm:"transfer_out_count" json:"transfer_out_count"`
	TransferOutAmount    float64 `gorm:"transfer_out_amount" json:"transfer_out_amount"`
	TransferInCount      int     `gorm:"transfer_in_count" json:"transfer_in_count"`
	TransferInAmount     float64 `gorm:"transfer_in_amount" json:"transfer_in_amount"`
	InventoryPlusCount   int     `gorm:"inventory_plus_count" json:"inventory_plus_count"`
	InventoryMinusCount  int     `gorm:"inventory_minus_count" json:"inventory_minus_count"`
	InventoryPlusAmount  float64 `gorm:"inventory_plus_amount" json:"inventory_plus_amount"`
	InventoryMinusAmount float64 `gorm:"inventory_minus_amount" json:"inventory_minus_amount"`
}

type MultiProductDashboardItem struct {
	ProductId            string  `gorm:"product_id" json:"product_id"`
	ProductName          string  `gorm:"product_name" json:"product_name"`
	UnitPerPack          float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
	UnitQuantity         int     `gorm:"unit_quantity" json:"unit_quantity"`
	SaleCount            int     `gorm:"sale_count" json:"sale_count"`
	SaleAmount           float64 `gorm:"sale_amount" json:"sale_amount"`
	ReturnSaleCount      int     `gorm:"return_sale_count" json:"return_sale_count"`
	ReturnSaleAmount     float64 `gorm:"return_sale_amount" json:"return_sale_amount"`
	ImportCount          int     `gorm:"import_count" json:"import_count"`
	ImportAmount         float64 `gorm:"import_amount" json:"import_amount"`
	ReturnToSkladCount   int     `gorm:"return_to_sklad_count" json:"return_to_sklad_count"`
	ReturnToSkladAmount  float64 `gorm:"return_to_sklad_amount" json:"return_to_sklad_amount"`
	TransferOutCount     int     `gorm:"transfer_out_count" json:"transfer_out_count"`
	TransferOutAmount    float64 `gorm:"transfer_out_amount" json:"transfer_out_amount"`
	TransferInCount      int     `gorm:"transfer_in_count" json:"transfer_in_count"`
	TransferInAmount     float64 `gorm:"transfer_in_amount" json:"transfer_in_amount"`
	InventoryPlusCount   int     `gorm:"inventory_plus_count" json:"inventory_plus_count"`
	InventoryMinusCount  int     `gorm:"inventory_minus_count" json:"inventory_minus_count"`
	InventoryPlusAmount  float64 `gorm:"inventory_plus_amount" json:"inventory_plus_amount"`
	InventoryMinusAmount float64 `gorm:"inventory_minus_amount" json:"inventory_minus_amount"`
}

type ProductBarcodeItem struct {
	ID        string    `json:"id"         gorm:"column:id"`
	Barcode   string    `json:"barcode"    gorm:"column:barcode"`
	Mxik      string    `json:"mxik"       gorm:"column:mxik"`
	UnitCode  string    `json:"unit_code"  gorm:"column:unit_code"`
	IsMarking bool      `json:"is_marking" gorm:"column:is_marking"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
	CreatedBy string    `json:"created_by" gorm:"column:created_by"`
}

type CreateProductBarcode struct {
	Barcode   string `json:"barcode"`
	Mxik      string `json:"mxik"`
	UnitCode  string `json:"unit_code"`
	IsMarking bool   `json:"is_marking"`
	CreatedBy string `json:"-"`
}

type UpdateProductBarcodeRequest struct {
	ID        string `json:"id"`
	Barcode   string `json:"barcode"`
	Mxik      string `json:"mxik"`
	UnitCode  string `json:"unit_code"`
	IsMarking bool   `json:"is_marking"`
	UpdatedBy string `json:"-"`
}

type DeleteProductBarcodeRequest struct {
	ID string `json:"id"` // faqat bitta ID
}

// region product by import

// product structure for update ikpu page
type ProductByImport struct {
	Id           string     `gorm:"id" json:"id"`
	ProductID    string     `gorm:"product_id" json:"product_id"`
	MaterialCode int        `gorm:"material_code" json:"material_code"`
	StoreName    string     `gorm:"store_name" json:"store_name"`
	ImportNumber string     `gorm:"import_number" json:"import_number"`
	Name         string     `gorm:"name" json:"name"`
	ProducerName string     `gorm:"producer_name" json:"producer_name"`
	Barcode      string     `gorm:"barcode" json:"barcode"`
	IsMarking    bool       `gorm:"is_marking" json:"is_marking"`
	IsChecking   bool       `gorm:"is_checking" json:"is_checking"`
	Manufacturer string     `gorm:"manufacturer" json:"manufacturer"`
	SerialNumber string     `gorm:"serial_number" json:"serial_number"`
	Quantity     int        `gorm:"quantity" json:"quantity"`
	UnitQuantity int        `gorm:"unit_quantity" json:"unit_quantity"`
	UQuantity    int        `gorm:"u_quantity" json:"u_quantity"`
	UnitPerPack  int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	Mxik         string     `gorm:"mxik" json:"mxik"`
	UnitCode     string     `gorm:"unit_code" json:"unit_code"`
	UnitLabel    string     `gorm:"unit_label" json:"unit_label"`
	ExpireDate   *time.Time `gorm:"expire_date" json:"expire_date"`
	RetailPrice  float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPrice  float64    `gorm:"supply_price" json:"supply_price"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// region min max

type StoreProductThreshold struct {
	ID          int        `gorm:"id" json:"id"`
	ProductID   string     `gorm:"product_id" json:"product_id"`
	StoreID     string     `gorm:"store_id" json:"store_id"`
	Kvant       float64    `gorm:"kvant" json:"kvant"`
	MinQuantity float64    `gorm:"min_quantity" json:"min_quantity"`
	MaxQuantity float64    `gorm:"max_quantity" json:"max_quantity"`
	IsActive    bool       `gorm:"is_active" json:"is_active"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
	Store       *Store     `gorm:"foreignKey:StoreID" json:"store"`
	Product     *Product   `gorm:"foreignKey:ProductID" json:"product"`
}

// create min max product strucute
type MinMaxProductRequest struct {
	ProductID   string `gorm:"product_id" json:"product_id"`
	StoreID     string `gorm:"store_id" json:"store_id"`
	Kvant       int    `gorm:"kvant" json:"kvant"`
	MinQuantity int    `gorm:"min_quantity" json:"min_quantity"`
	MaxQuantity int    `gorm:"max_quantity" json:"max_quantity"`
	IsActive    bool   `gorm:"is_active" json:"is_active"`
}

// update min max product structure
type MinMaxProductUpdate struct {
	ID          int    `gorm:"id" json:"id"`
	ProductID   string `gorm:"product_id" json:"product_id"`
	Kvant       int    `gorm:"kvant" json:"kvant"`
	MinQuantity int    `gorm:"min_quantity" json:"min_quantity"`
	MaxQuantity int    `gorm:"max_quantity" json:"max_quantity"`
	IsActive    bool   `gorm:"is_active" json:"is_active"`
}

// product structure for min, max
type MinMaxProduct struct {
	Id           int        `gorm:"id" json:"id"`
	StoreID      string     `gorm:"store_id" json:"store_id"`
	ProductID    string     `gorm:"product_id" json:"product_id"`
	MaterialCode int        `gorm:"material_code" json:"material_code"`
	StoreName    string     `gorm:"store_name" json:"store_name"`
	Name         string     `gorm:"name" json:"name"`
	Kvant        float64    `gorm:"kvant" json:"kvant"`
	MinQuantity  float64    `gorm:"min_quantity" json:"min_quantity"`
	MaxQuantity  float64    `gorm:"max_quantity" json:"max_quantity"`
	IsActive     bool       `gorm:"is_active" json:"is_active"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// region Noor
// Noor API response structure
type NoorProduct struct {
	Id            string            `gorm:"id" json:"id"`
	Name          string            `gorm:"name" json:"name"`
	Photos        utils.StringArray `gorm:"type:text[]" json:"photos"`
	Description   string            `gorm:"description" json:"description"`
	DescriptionRu string            `gorm:"description_ru" json:"description_ru"`
	DescriptionUz string            `gorm:"description_uz" json:"description_uz"`
	DescriptionKr string            `gorm:"description_kr" json:"description_kr"`
	CategoryId    string            `gorm:"category_id" json:"category_id"`
}

// Noor API store_products structure
type NoorStoreProduct struct {
	StoreId   string `gorm:"store_id" json:"shop_id"`
	ProductId string `gorm:"product_id" json:"product_id"`
	Quantity  int    `gorm:"quantity" json:"quantity"`
	Price     int    `gorm:"price" json:"price"`
}

// Store external API response structure
type NoorStore struct {
	Id        string `gorm:"id" json:"id"`
	Name      string `gorm:"name" json:"name"`
	Phone     string `gorm:"phone" json:"phone"`
	Address   string `gorm:"address" json:"address"`
	Location  string `gorm:"location" json:"-"`
	Location1 Point  `gorm:"-" json:"location"`
	WorkHours string `gorm:"work_hours" json:"work_hours"`
	IsFullday bool   `gorm:"is_fullday" json:"is_fullday"`
	IsActive  bool   `gorm:"is_active" json:"is_active"`
}

type NoorQueryParam struct {
	UpdatedAt string `form:"updatedAt"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	Search    string `form:"search"`
	ShopId    string `form:"shopId"`
}

type NoorCategory struct {
	Id       string `gorm:"id" json:"id"`
	NameUz   string `gorm:"name_uz" json:"name_uz"`
	NameRu   string `gorm:"name_ru" json:"name_ru"`
	NameEn   string `gorm:"name_en" json:"name_en"`
	NameKr   string `gorm:"name_kr" json:"name_kr"`
	ParentId string `gorm:"parent_id" json:"parent_id"`
	Photo    string `gorm:"photo" json:"photo"`
}

// end region

type UpdatePackagingRequest struct {
	ProductId   string `json:"product_id" binding:"required"`
	UnitPerPack int    `json:"unit_per_pack" binding:"required,min=1"`
}

// domain/product_photo_alert.go

type ProductPhotoAlert struct {
	ID          string            `json:"id"`
	ProductID   string            `json:"product_id"`
	Name        string            `json:"name"`
	Photos      utils.StringArray `gorm:"type:text[]" json:"photos"`
	UnitPerPack int               `json:"unit_per_pack"`
	Category    int               `json:"category"`
	Reason      string            `json:"reason"`
	CreatedBy   *string           `json:"created_by"`
	Status      string            `json:"status"`
	ResolvedBy  *string           `json:"resolved_by"`
	ResolvedAt  *time.Time        `json:"resolved_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CreateProductPhotoAlert struct {
	ProductID string    `json:"product_id"`
	Category  int       `json:"category"`
	Reason    string    `json:"reason"`
	CreatedBy *string   `json:"created_by"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductPhotoAlertCreate struct {
	ProductID string `json:"product_id" binding:"required"`
	Category  int    `json:"category" binding:"required"`
	Reason    string `json:"reason"`
	CreatedBy string `json:"-"`
}

type GenerateOnecTokenRequest struct {
	Password string `json:"password" validate:"required"`
}

type GenerateOnecTokenResponse struct {
	Token string `json:"token"`
}

type MovementUnitsResponse struct {
	ProductId           string  `gorm:"product_id" json:"product_id"`
	StoreProductId      string  `gorm:"store_product_id" json:"store_product_id"`
	StoreId             string  `gorm:"store_id" json:"store_id"`
	Name                string  `gorm:"name" json:"name"`
	UnitPerPack         float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
	UnitQuantity        float64 `gorm:"unit_quantity" json:"unit_quantity"`
	ImportQuantity      float64 `gorm:"import_quantity" json:"import_quantity"`
	SoldQuantity        float64 `gorm:"sold_quantity" json:"sold_quantity"`
	ReturnedQuantity    float64 `gorm:"returned_quantity" json:"returned_quantity"`
	VozvratQuantity     float64 `gorm:"vozvrat_quantity" json:"vozvrat_quantity"`
	TransferInQuantity  float64 `gorm:"transfer_in_quantity" json:"transfer_in_quantity"`
	TransferOutQuantity float64 `gorm:"transfer_out_quantity" json:"transfer_out_quantity"`
	CorrectQuantity     float64 `gorm:"correct_quantity" json:"correct_quantity"`
	Diff                float64 `gorm:"diff" json:"diff"`
}

type MovementUnitsByDateParam struct {
	StoreId   string      `form:"store_id"`
	CompanyId string      `form:"company_id"`
	FromDate  *CustomTime `form:"from_date"`
	ToDate    *CustomTime `form:"to_date"`
	Limit     int         `form:"limit"`
	Offset    int         `form:"offset"`
}

type ProductByImportParam struct {
	ProductId       string   `form:"product_id"`
	StoreId         string   `form:"store_id"`
	CompanyId       string   `form:"company_id"`
	ProducerId      string   `form:"producer_id"`
	ImportId        string   `form:"import_id"`
	CategoryId      string   `form:"category_id"`
	SearchField     string   `form:"search"`
	SupplyPriceFrom float64  `form:"supply_price_from"`
	SupplyPriceTo   float64  `form:"supply_price_to"`
	RetailPriceFrom float64  `form:"retail_price_from"`
	RetailPriceTo   float64  `form:"retail_price_to"`
	Status          string   `form:"status"`
	NoBarcode       bool     `form:"no_barcode"`
	Order           string   `form:"order"`
	Category        int      `form:"category"`
	ExpiryFrom      string   `form:"expiry_from"`
	ExpiryTo        string   `form:"expiry_to"`
	StoreIds        []string `from:"store_ids"`
	CompanyIds      []string `form:"company_ids"`
	Limit           int      `form:"limit"`
	Offset          int      `form:"offset"`
	StartDate       string   `form:"start_date"`
	EndDate         string   `form:"end_date"`
}
