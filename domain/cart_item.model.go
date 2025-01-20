package domain

import "time"

// CartItem structure
type CartItem struct {
	ID             string     `gorm:"id" json:"id"`
	ProductID      string     `gorm:"product_id" json:"product_id"`
	StoreProductID string     `gorm:"store_product_id" json:"store_product_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	SaleId         string     `gorm:"sale_id" json:"sale_id"`
	Quantity       int        `gorm:"quantity" json:"quantity"`
	UnitPrice      float64    `gorm:"unit_price" json:"unit_price"`
	DiscountPrice  float64    `gorm:"discount_price" json:"discount_price"`
	DiscountType   string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

// CartItemRequest structure
type CartItemRequest struct {
	ID                 string  `gorm:"id" json:"-"`
	EmployeeID         string  `gorm:"employee_id" json:"employee_id"`
	StoreProductID     string  `gorm:"store_product_id" json:"store_product_id"`
	SaleId             string  `gorm:"sale_id" json:"sale_id"`
	Quantity           int     `gorm:"quantity" json:"quantity"`
	UnitQuantity       int     `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPrice          float64 `gorm:"unit_price" json:"unit_price"`
	DiscountType       string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue      float64 `gorm:"discount_value" json:"discount_value"`
	TotalPrice         float64 `gorm:"total_price" json:"-"`
	TotalDiscountPrice float64 `gorm:"total_discount_price" json:"-"`
	Status             string  `gorm:"status" json:"-"`
}

// Cart Item update product unit
type CartItemUpdateProductUnit struct {
	StoreProductID string   `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int      `gorm:"quantity" json:"quantity"`
	UnitQuantity   int      `gorm:"unit_quantity" json:"unit_quantity"`
	DiscountType   *string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  *float64 `gorm:"discount_value" json:"discount_value"`
}

// CartItemBySaleIDUpdateRequest structure
type CartItemBySaleIDUpdateRequest struct {
	DiscountType   string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64 `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64 `gorm:"discount_amount" json:"-"`
}

// Ids
type Ids struct {
	Ids []string `json:"ids"`
}

// CartItemResponse structure
type CartItemData struct {
	Data           []CartItemResponse `gorm:"-" json:"data"`
	TotalAmount    float64            `gorm:"total_amount" json:"total_amount"`
	DiscountAmount float64            `gorm:"discount_amount" json:"discount_amount"`
	Count          int64              `gorm:"count" json:"count"`
}

// CartItemResponse structure with product data
type CartItemResponse struct {
	ID             string     `gorm:"id" json:"id"`
	StoreProductID string     `gorm:"store_product_id" json:"store_product_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	SaleId         string     `gorm:"sale_id" json:"sale_id"`
	Quantity       int        `gorm:"quantity" json:"quantity"`
	UnitQuantity   int        `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPrice      float64    `gorm:"unit_price" json:"unit_price"`
	DiscountPrice  float64    `gorm:"discount_price" json:"discount_price"`
	DiscountType   string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Name           string     `gorm:"name" json:"name"`
	Barcode        string     `gorm:"barcode" json:"barcode"`
	BonusAmount    float64    `gorm:"bonus_amount" json:"bonus_amount"`
	BonusPercent   int        `gorm:"bonus_percent" json:"bonus_percent"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
}

type SumResult struct {
	TotalPrice    float64
	DiscountPrice float64
}
