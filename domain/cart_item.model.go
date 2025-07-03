package domain

import "time"

// CartItem structure
type CartItem struct {
	ID             string     `gorm:"id" json:"id"`
	StoreProductID string     `gorm:"store_product_id" json:"store_product_id"`
	ProductId      string     `gorm:"product_id" json:"product_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	SaleId         string     `gorm:"sale_id" json:"sale_id"`
	Quantity       int        `gorm:"quantity" json:"quantity"`
	UnitQuantity   int        `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPrice      float64    `gorm:"unit_price" json:"unit_price"`
	DiscountPrice  float64    `gorm:"discount_price" json:"discount_price"`
	DiscountType   string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	BonusAmount    float64    `gorm:"bonus_amount" json:"bonus_amount"`
	UnitPerPack    int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	IsMarking      bool       `gorm:"is_marking" json:"is_marking"`
}

// CartItemRequest structure
type CartItemRequest struct {
	ID             string  `gorm:"id" json:"-"`
	EmployeeID     string  `gorm:"employee_id" json:"-"`
	StoreProductID string  `gorm:"store_product_id" json:"store_product_id"`
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
type CartItemUpdateProductUnit struct {
	StoreProductID string `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int    `gorm:"quantity" json:"quantity"`
	UnitQuantity   int    `gorm:"unit_quantity" json:"unit_quantity"`
}

// CartItemBySaleIDUpdateRequest structure
type CartItemBySaleIDUpdateRequest struct {
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
}

// CartItemResponse structure with product data
type CartItemResponse struct {
	ID                 string     `gorm:"id" json:"id"`
	StoreProductID     string     `gorm:"store_product_id" json:"store_product_id"`
	EmployeeID         string     `gorm:"employee_id" json:"employee_id"`
	SaleId             string     `gorm:"sale_id" json:"sale_id"`
	Quantity           int        `gorm:"quantity" json:"quantity"`
	UnitQuantity       int        `gorm:"unit_quantity" json:"unit_quantity"`
	UnitAmount         float64    `gorm:"unit_amount" json:"unit_amount"`
	UnitPrice          float64    `gorm:"unit_price" json:"unit_price"`
	UnitQuantityPrice  float64    `gorm:"unit_quantity_price" json:"unit_quantity_price"`
	DiscountPrice      float64    `gorm:"discount_price" json:"discount_price"`
	DiscountType       string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue      float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount     float64    `gorm:"discount_amount" json:"discount_amount"`
	DiscountUnitAmount float64    `gorm:"discount_unit_amount" json:"discount_unit_amount"`
	TotalPrice         float64    `gorm:"total_price" json:"total_price"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time `gorm:"updated_at" json:"updated_at"`
	Name               string     `gorm:"name" json:"name"`
	Description        string     `gorm:"description" json:"description"`
	Vat                float64    `gorm:"vat" json:"vat"`
	VatPrice           float64    `gorm:"vat_price" json:"vat_price"`
	UnitVatPrice       float64    `gorm:"unit_vat_price" json:"unit_vat_price"`
	Label              string     `gorm:"label" json:"label"`
	VatPercent         float64    `gorm:"vat_percent" json:"vat_percent"`
	Barcode            string     `gorm:"barcode" json:"barcode"`
	UnitName           string     `gorm:"unit_name" json:"unit_name"`
	ShortName          string     `gorm:"short_name" json:"short_name"`
	UnitPerPack        int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	Shelf              string     `gorm:"shelf" json:"shelf"`
	ProducerName       string     `gorm:"producer_name" json:"producer_name"`
	ClassCode          string     `gorm:"class_code" json:"class_code"`
	PackageCode        string     `gorm:"package_code" json:"package_code"`
	PackageName        string     `gorm:"package_name" json:"package_name"`
	CategoryName       string     `gorm:"category_name" json:"category_name"`
	BonusAmount        float64    `gorm:"bonus_amount" json:"bonus_amount"`
	BonusPercent       float64    `gorm:"bonus_percent" json:"bonus_percent"`
	QuantityStock      int        `gorm:"quantity_stock" json:"quantity_stock"`
	UnitQuantityStock  int        `gorm:"unit_quantity_stock" json:"unit_quantity_stock"`
	ExpireDate         *time.Time `gorm:"expire_date" json:"expire_date"`
	IsMarking          bool       `gorm:"is_marking" json:"is_marking"`
	IsChecking         bool       `gorm:"is_checking" json:"is_checking"`
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
	StoreProductID string  `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int     `gorm:"quantity" json:"-"`
	UnitQuantity   int     `gorm:"unit_quantity" json:"-"`
	UnitPrice      float64 `gorm:"unit_price" json:"-"`
	TotalPrice     float64 `gorm:"total_price" json:"-"`
}
