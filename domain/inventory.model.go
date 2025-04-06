package domain

import "time"

// "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
// "public_id" VARCHAR(20),
// "store_id" UUID NOT NULL REFERENCES stores(id),
// "name" VARCHAR(255),
// "type" VARCHAR(55) DEFAULT 'FULL', -- FULL || PARTIAL || IMPORT
// "status" INT DEFAULT 0, -- 0 -> new, 1 -> pending, 2 -> completed
// "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
// "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
// );

// CREATE TABLE IF NOT EXISTS inventory_details(
// "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
// "inventory_id" UUID NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
// "store_product_id" UUID NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
// "scanned_count" INT DEFAULT 0,
// "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
// "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
// );

// Inventory structure
type Inventory struct {
	Id        string     `gorm:"id" json:"id"`
	PublicId  string     `gorm:"public_id" json:"public_id"`
	StoreId   string     `gorm:"store_id" json:"store_id"`
	Name      string     `gorm:"name" json:"name"`
	Type      string     `gorm:"type" json:"type"`     // FULL || PARTIAL || IMPORT
	Status    int        `gorm:"status" json:"status"` // 0 -> new, 1 -> pending, 2 -> completed
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// InventoryRequest structure
type InventoryRequest struct {
	PublicId string `gorm:"public_id" json:"public_id"`
	StoreId  string `gorm:"store_id" json:"store_id"`
	Name     string `gorm:"name" json:"name"`
	Type     string `gorm:"type" json:"type"` // FULL || PARTIAL || IMPORT
	Products []struct {
		ProductId string `json:"product_id"`
	} `json:"products"`
}

// InventoryRequest structure
type InventoryDetail struct {
	Id           string     `gorm:"id" json:"id"`
	InventoryId  string     `gorm:"inventory_id" json:"inventory_id"`
	ProductId    string     `gorm:"product_id" json:"product_id"`
	ScannedCount int        `gorm:"scanned_count" json:"scanned_count"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// InventoryDetailRequest structure
type InventoryDetailRequest struct {
	InventoryId string `gorm:"inventory_id" json:"inventory_id"`
	ProductId   string `gorm:"product_id" json:"product_id"`
}
