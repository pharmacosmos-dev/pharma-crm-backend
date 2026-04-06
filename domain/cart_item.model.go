package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// CartItem structure
type CartItem struct {
	Id             string            `gorm:"column:id;primaryKey" json:"id"`
	StoreProductId string            `gorm:"column:store_product_id" json:"store_product_id"`
	ProductId      string            `gorm:"column:product_id" json:"product_id"`
	EmployeeId     string            `gorm:"column:employee_id" json:"employee_id"`
	SaleId         string            `gorm:"column:sale_id" json:"sale_id"`
	Quantity       int               `gorm:"column:quantity" json:"quantity"`
	Markings       utils.StringArray `gorm:"column:markings;type:text[]" json:"markings"`
	UnitQuantity   int               `gorm:"column:unit_quantity" json:"unit_quantity"`
	UnitPrice      float64           `gorm:"column:unit_price" json:"unit_price"`
	DiscountPrice  float64           `gorm:"column:discount_price" json:"discount_price"`
	DiscountType   string            `gorm:"column:discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64           `gorm:"column:discount_value" json:"discount_value"`
	DiscountAmount float64           `gorm:"column:discount_amount" json:"discount_amount"`
	TotalPrice     float64           `gorm:"column:total_price" json:"total_price"`
	BonusAmount    float64           `gorm:"column:bonus_amount" json:"bonus_amount"`
	UnitPerPack    int               `gorm:"column:unit_per_pack" json:"unit_per_pack"`
	CreatedAt      *time.Time        `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      *time.Time        `gorm:"column:updated_at" json:"updated_at"`
	Barcode        string            `gorm:"column:barcode" json:"barcode"`
	IsMarking      bool              `gorm:"column:is_marking" json:"is_marking"`
	SkipAutoOrder  bool              `gorm:"skip_auto_order" json:"skip_auto_order"`
}

// AppendMarkingRequest structure
type AppendMarkingRequest struct {
	Marking string `json:"marking" binding:"required"`
}

// UpdateAutoOrderRequest structure
type UpdateAutoOrderRequest struct {
	IsAutoOrder bool `json:"is_auto_order"`
}

// CartItemRequest structure
type CartItemRequest struct {
	Id             string  `gorm:"id" json:"-"`
	EmployeeId     string  `gorm:"employee_id" json:"-"`
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	ProductId      string  `gorm:"product_id" json:"product_id"`
	Barcode        string  `gorm:"barcode" json:"barcode"`
	SaleId         string  `gorm:"sale_id" json:"sale_id"`
	Quantity       int     `gorm:"quantity" json:"-"`
	UnitQuantity   int     `gorm:"unit_quantity" json:"-"`
	UnitPrice      float64 `gorm:"unit_price" json:"-"`
	DiscountType   string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64 `gorm:"discount_value" json:"discount_value"`
	TotalPrice     float64 `gorm:"total_price" json:"-"`
	Status         string  `gorm:"status" json:"-"`
	DiscountAmount float64 `gorm:"discount_amount" json:"-"`
}

// Cart Item update product unit
type CartItemUpdateUnit struct {
	Id             string `gorm:"id" json:"id"`
	StoreProductId string `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int    `gorm:"quantity" json:"quantity"`
	UnitQuantity   int    `gorm:"unit_quantity" json:"unit_quantity"`
}

// CartItemBySaleIDUpdateRequest structure
type CartItemDiscountRequest struct {
	DiscountType  string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue float64 `gorm:"discount_value" json:"discount_value"`
}

// Ids
type Ids struct {
	Ids []string `json:"ids"`
}

// CartItemResponse structure
type CartItemData struct {
	Data           []CartItemResponse `gorm:"-" json:"data"`
	TotalAmount    float64            `gorm:"total_amount" json:"total_amount"`
	Sum            float64            `gorm:"sum" json:"sum"`
	DiscountAmount float64            `gorm:"discount_amount" json:"discount_amount"`
	CardPercent    float64            `gorm:"card_percent" json:"card_percent"`
	VatSum         float64            `gorm:"vat_sum" json:"vat_sum"`
	Count          int64              `gorm:"count" json:"count"`
	ItemCount      int64              `gorm:"item_count" json:"item_count"`
	SkipAutoOrder  bool               `gorm:"skip_auto_order" json:"skip_auto_order"`
}

// CartItemResponse structure with product data
type CartItemResponse struct {
	ID                 string            `gorm:"id" json:"id"`
	StoreProductID     string            `gorm:"store_product_id" json:"store_product_id"`
	EmployeeID         string            `gorm:"employee_id" json:"employee_id"`
	SaleId             string            `gorm:"sale_id" json:"sale_id"`
	Quantity           int               `gorm:"quantity" json:"quantity"`
	UnitQuantity       int               `gorm:"unit_quantity" json:"unit_quantity"`
	UnitAmount         float64           `gorm:"unit_amount" json:"unit_amount"`
	UnitPrice          float64           `gorm:"unit_price" json:"unit_price"`
	UnitQuantityPrice  float64           `gorm:"unit_quantity_price" json:"unit_quantity_price"`
	DiscountPrice      float64           `gorm:"discount_price" json:"discount_price"`
	DiscountType       string            `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue      float64           `gorm:"discount_value" json:"discount_value"`
	DiscountAmount     float64           `gorm:"discount_amount" json:"discount_amount"`
	DiscountUnitAmount float64           `gorm:"discount_unit_amount" json:"discount_unit_amount"`
	TotalPrice         float64           `gorm:"total_price" json:"total_price"`
	CreatedAt          *time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time        `gorm:"updated_at" json:"updated_at"`
	Name               string            `gorm:"name" json:"name"`
	Markings           utils.StringArray `gorm:"type:text[]" json:"markings"`
	ProductID          string            `gorm:"product_id" json:"product_id"`
	Description        string            `gorm:"description" json:"description"`
	Vat                float64           `gorm:"vat" json:"vat"`
	VatPrice           float64           `gorm:"vat_price" json:"vat_price"`
	UnitVatPrice       float64           `gorm:"unit_vat_price" json:"unit_vat_price"`
	Label              string            `gorm:"label" json:"label"`
	VatPercent         float64           `gorm:"vat_percent" json:"vat_percent"`
	Barcode            string            `gorm:"barcode" json:"barcode"`
	UnitName           string            `gorm:"unit_name" json:"unit_name"`
	ShortName          string            `gorm:"short_name" json:"short_name"`
	UnitPerPack        int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	Shelf              string            `gorm:"shelf" json:"shelf"`
	ProducerName       string            `gorm:"producer_name" json:"producer_name"`
	ClassCode          string            `gorm:"class_code" json:"class_code"`
	PackageCode        string            `gorm:"package_code" json:"package_code"`
	PackageName        string            `gorm:"package_name" json:"package_name"`
	CategoryName       string            `gorm:"category_name" json:"category_name"`
	BonusAmount        float64           `gorm:"bonus_amount" json:"bonus_amount"`
	BonusStartDate     *time.Time        `gorm:"bonus_start_date" json:"-"`
	BonusEndDate       *time.Time        `gorm:"bonus_end_date" json:"-"`
	BonusPercent       float64           `gorm:"bonus_percent" json:"bonus_percent"`
	QuantityStock      int               `gorm:"quantity_stock" json:"quantity_stock"`
	UnitQuantityStock  int               `gorm:"unit_quantity_stock" json:"unit_quantity_stock"`
	ExpireDate         *time.Time        `gorm:"expire_date" json:"expire_date"`
	IsMarking          bool              `gorm:"is_marking" json:"is_marking"`
	IsChecking         bool              `gorm:"is_checking" json:"is_checking"`
	SkipAutoOrder      bool              `gorm:"skip_auto_order" json:"skip_auto_order"`
}

type SumResult struct {
	TotalPrice    float64
	DiscountPrice float64
}

type CartItemR struct {
	ID             string  `gorm:"id" json:"id"`
	StoreProductID string  `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int     `gorm:"quantity" json:"quantity"`
	UnitQuantity   int     `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPrice      float64 `gorm:"unit_price" json:"unit_price"`
	TotalPrice     float64 `gorm:"total_price" json:"total_price"`
	Status         string  `gorm:"status" json:"status"`
	BonusAmount    float64 `gorm:"bonus_amount" json:"bonus_amount"`
}

// cart item create request for online sale
type CartItemOnlineRequest struct {
	SaleId         string  `gorm:"sale_id" json:"sale_id"`
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int     `gorm:"quantity" json:"-"`
	UnitQuantity   int     `gorm:"unit_quantity" json:"-"`
	UnitPrice      float64 `gorm:"unit_price" json:"-"`
	TotalPrice     float64 `gorm:"total_price" json:"-"`
	ProductId      string  `gorm:"product_id" json:"-"`
}

type CartItemForDMED struct {
	ID             string  `gorm:"id" json:"id"`
	UnitPrice      float64 `gorm:"unit_price" json:"unit_price"`
	Quantity       int     `gorm:"quantity" json:"quantity"`
	UnitQuantity   int     `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPerPack    int     `gorm:"unit_per_pack" json:"unit_per_pack"`
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	Barcode        string  `gorm:"barcode" json:"barcode"`
	SerialNumber   string  `gorm:"serial_number" json:"serial_number"`
}

type CartItemCheckMarkingRequest struct {
	CartItemId string `json:"cart_item_id" binding:"required,uuid"`
	ProductId  string `json:"product_id" binding:"required,uuid"`
	Barcode    string `json:"barcode" binding:"required"`
}

type CartItemWithProduct struct {
	Id                       string            `gorm:"column:id;primaryKey" json:"id"`
	StoreProductId           string            `gorm:"column:store_product_id" json:"store_product_id"`
	ProductId                string            `gorm:"column:product_id" json:"product_id"`
	EmployeeId               string            `gorm:"column:employee_id" json:"employee_id"`
	SaleId                   string            `gorm:"column:sale_id" json:"sale_id"`
	Quantity                 int               `gorm:"column:quantity" json:"quantity"`
	Markings                 utils.StringArray `gorm:"column:markings;type:text[]" json:"markings"`
	UnitQuantity             int               `gorm:"column:unit_quantity" json:"unit_quantity"`
	UnitPrice                float64           `gorm:"column:unit_price" json:"unit_price"`
	DiscountPrice            float64           `gorm:"column:discount_price" json:"discount_price"`
	DiscountType             string            `gorm:"column:discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue            float64           `gorm:"column:discount_value" json:"discount_value"`
	DiscountAmount           float64           `gorm:"column:discount_amount" json:"discount_amount"`
	TotalPrice               float64           `gorm:"column:total_price" json:"total_price"`
	BonusAmount              float64           `gorm:"column:bonus_amount" json:"bonus_amount"`
	BonusStartDate           *time.Time        `gorm:"column:bonus_start_date" json:"bonus_start_date"`
	BonusEndDate             *time.Time        `gorm:"column:bonus_end_date" json:"bonus_end_date"`
	UnitPerPack              int               `gorm:"column:unit_per_pack" json:"unit_per_pack"`
	CreatedAt                *time.Time        `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                *time.Time        `gorm:"column:updated_at" json:"updated_at"`
	Barcode                  string            `gorm:"column:barcode" json:"barcode"`
	IsMarking                bool              `gorm:"column:is_marking" json:"is_marking"`
	ProductName              string            `gorm:"column:product_name"`
	StoreProductUnitQuantity int               `gorm:"column:sp_unit_quantity"`
}
