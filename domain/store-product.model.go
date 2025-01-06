package domain

import "time"

// StoreProduct structure
type StoreProduct struct {
	ProductID           *string    `gorm:"product_id" json:"product_id"`
	ProductMaterialCode *int       `gorm:"product_material_code" json:"product_material_code"`
	StoreID             string     `gorm:"store_id" json:"store_id"`
	Quantity            int        `gorm:"quantity" json:"quantity"`
	SmallQuantity       int        `gorm:"small_quantity" json:"small_quantity"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	Product             *Product   `gorm:"foreignKey:ProductID" json:"product"`
	Store               *Store     `gorm:"foreignKey:StoreID" json:"store"`
}
