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
	DiscountType   string     `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue  float64    `gorm:"discount_value" json:"discount_value"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	TotalPrice     float64    `gorm:"total_price" json:"total_price"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Product        *Product   `gorm:"foreignKey:ProductID" json:"product"`
	Employee       *Employee  `gorm:"foreignKey:EmployeeID" json:"employee"`
}

// CartItemRequest structure
type CartItemRequest struct {
	ID                 string  `gorm:"id" json:"-"`
	EmployeeID         string  `gorm:"employee_id" json:"employee_id"`
	ProductID          string  `gorm:"product_id" json:"product_id"`
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
	ProductID     string   `gorm:"product_id" json:"product_id"`
	Quantity      int      `gorm:"quantity" json:"quantity"`
	DrugCount     int      `gorm:"drug_count" json:"drug_count"`
	DiscountType  *string  `gorm:"discount_type" json:"discount_type" example:"percent|cash"`
	DiscountValue *float64 `gorm:"discount_value" json:"discount_value"`
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
type CartItemResponse struct {
	Data           []CartItem `json:"data"`
	TotalAmount    float64    `json:"total_amount"`
	DiscountAmount float64    `json:"discount_amount"`
	Count          int64      `json:"count"`
}

// CartItemUpdateRequest structure
type CartItemUpdateRequest struct {
	Quantity           int     `gorm:"quantity" json:"quantity"`
	TotalPrice         float64 `gorm:"total_price" json:"total_price"`
	TotalDiscountPrice float64 `gorm:"total_discount_price" json:"total_discount_price"`
	DiscountAmount     float64 `gorm:"discount_amount" json:"discount_amount"`
	DrugCount          int     `gorm:"drug_count" json:"drug_count"`
}

type SumResult struct {
	TotalPrice    float64
	DiscountPrice float64
}
