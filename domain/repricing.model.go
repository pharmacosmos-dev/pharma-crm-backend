package domain

import "time"

type PriceRevalution struct {
	Id          int        `gorm:"id" json:"id"`
	StoreID     string     `gorm:"store_id" json:"store_id"`
	Name        string     `gorm:"name" json:"name"`
	Status      string     `gorm:"status" json:"status"`
	Type        string     `gorm:"type" json:"type"`
	CreatedByID string     `gorm:"created_by_id" json:"created_by_id"`
	UpdatedByID string     `gorm:"updated_by_id" json:"updated_by_id"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
	CreatedBy   *Employee  `gorm:"foreignKey:CreatedByID" json:"created_by"`
	UpdatedBy   *Employee  `gorm:"foreignKey:UpdatedByID" json:"updated_by"`
	Store       *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

type PriceRevalutionDetail struct {
	Id                int        `gorm:"id" json:"id"`
	PriceRevalutionId int        `gorm:"price_revalution_id" json:"price_revalution_id"`
	StoreProductID    string     `gorm:"store_product_id" json:"store_product_id"`
	ProductID         string     `gorm:"product_id" json:"product_id"`
	OldSupplyPrice    float64    `gorm:"old_supply_price" json:"old_supply_price"`
	NewSupplyPrice    float64    `gorm:"new_supply_price" json:"new_supply_price"`
	OldRetailPrice    float64    `gorm:"old_retail_price" json:"old_retail_price"`
	NewRetailPrice    float64    `gorm:"new_retail_price" json:"new_retail_price"`
	OldExpireDate     *time.Time `gorm:"old_expire_date" json:"old_expire_date"`
	NewExpireDate     *time.Time `gorm:"new_expire_date" json:"new_expire_date"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
}
