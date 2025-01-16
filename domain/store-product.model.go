package domain

import "time"

// StoreProduct structure
type StoreProduct struct {
	ProductID           *string    `gorm:"product_id" json:"product_id"`
	ProductMaterialCode *int       `gorm:"product_material_code" json:"product_material_code"`
	StoreID             string     `gorm:"store_id" json:"store_id"`
	Quantity            int        `gorm:"quantity" json:"quantity"`
	PackQuantity        int        `gorm:"pack_quantity" json:"pack_quantity"`
	UnitQuantity        int        `gorm:"unit_quantity" json:"unit_quantity"`
	SmallQuantity       int        `gorm:"small_quantity" json:"small_quantity"`
	ExpireDate          *time.Time `gorm:"expire_date" json:"expire_date"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	Product             *Product   `gorm:"foreignKey:ProductID" json:"product"`
	Store               *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

type StoreProductUpdateRequest struct {
	StoreID       string `json:"store_id"`
	Quantity      int    `json:"quantity"`
	SmallQuantity int    `json:"small_quantity"`
}

// Store Product Request structure for creating
type StoreProductRequest struct {
	StoreID       string     `gorm:"store_id" json:"store_id"`
	ProductID     string     `gorm:"product_id" json:"product_id"`
	PackQuantity  int        `gorm:"pack_quantity" json:"pack_quantity"`
	UnitQuantity  int        `gorm:"unit_quantity" json:"unit_quantity"`
	SmallQuantity int        `gorm:"store_id" json:"small_quantity"`
	ExpireDate    *time.Time `gorm:"expire_date" json:"expire_date"`
}

type StoreProductResponse struct {
	ID                  string     `gorm:"id" json:"id"`
	ProductID           string     `gorm:"product_id" json:"product_id"`
	ProductMaterialCode int        `gorm:"product_material_code" json:"product_material_code"`
	StoreID             string     `gorm:"store_id" json:"store_id"`
	Quantity            string     `gorm:"quantity" json:"quantity"`
	Barcode             string     `gorm:"barcode" json:"barcode"`
	PackQuantity        int        `gorm:"pack_quantity" json:"pack_quantity"`
	UnitQuantity        int        `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPerPack         int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	SmallQuantity       int        `gorm:"small_quantity" json:"small_quantity"`
	RetailPrice         float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate          *time.Time `gorm:"expire_date" json:"expire_date"`
	ExpireDay           int        `gorm:"expire_day" json:"expire_day"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	Name                string     `gorm:"name" json:"name"`
	CategoryName        string     `gorm:"category_name" json:"category_name"`
}
