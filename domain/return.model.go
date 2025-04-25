package domain

import "time"

// return  structure
type Return struct {
	Id                string     `gorm:"id" json:"id"`
	PublicId          string     `gorm:"public_id" json:"public_id"`
	FromStoreId       string     `gorm:"from_store_id" json:"store_id"`
	Name              string     `gorm:"name" json:"name"`
	Status            string     `gorm:"status" json:"status"`
	Comment           string     `gorm:"comment" json:"comment"`
	ReturnCount       int64      `gorm:"return_count" json:"return_count"`
	ReceivedSupplySum float64    `gorm:"received_supply_sum" json:"received_supply_sum"`
	ReceivedRetailSum float64    `gorm:"received_retail_sum" json:"received_retail_sum"`
	AcceptedSupplySum float64    `gorm:"accepted_supply_sum" json:"accepted_supply_sum"`
	AcceptedRetailSum float64    `gorm:"accepted_retail_sum" json:"accepted_retail_sum"`
	CreatedById       string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById       string     `gorm:"column:accepted_by" json:"updated_by_id"`
	AcceptedById      string     `gorm:"column:accepted_by" json:"accepted_by_id"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	AcceptedAt        *time.Time `gorm:"accepted_at" json:"accepted_at"`
	Store             *Store     `gorm:"foreignKey:FromStoreId" json:"store"`
	CreatedBy         *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy         *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
	AcceptedBy        *Employee  `gorm:"foreignKey:AcceptedById" json:"accepted_by"`
}

// return off create request
type ReturnRequest struct {
	PublicId  string `gorm:"public_id" json:"public_id"`
	Name      string `gorm:"name" json:"name"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	CreatedBy string `gorm:"created_by" json:"created_by"`
	Status    string `gorm:"status" json:"status"`
	Comment   string `gorm:"comment" json:"comment"`
}

// ReturnDetail structure
type ReturnDetail struct {
	Id            string     `gorm:"id" json:"id"`
	ReturnId      string     `gorm:"return_id" json:"return_id"`
	ProductId     string     `gorm:"product_id" json:"product_id"`
	ReceivedCount int        `gorm:"received_count" json:"received_count"`
	ScannedCount  int        `gorm:"scanned_count" json:"scanned_count"`
	Name          string     `gorm:"name" json:"name"`
	MaterialCode  int        `gorm:"material_code" json:"material_code"`
	Barcode       string     `gorm:"barcode" json:"barcode"`
	ShortName     string     `gorm:"short_name" json:"short_name"`
	SupplyPrice   float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice   float64    `gorm:"retail_price" json:"retail_price"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
}

// return query param  structure
type ReturnParam struct {
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	StoreId   string `form:"store_id"`
	Status    string `form:"status"`
	Search    string `form:"search"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

// return detail query param structure
type ReturnDetailParam struct {
	ReturnId string `form:"return_id"`
	Search   string `form:"search"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
	Type     string `form:"type"`
}

type ReturnDetailStatus struct {
	Scanned           int     `gorm:"scanned" json:"scanned"`
	Shortage          int     `gorm:"shortage" json:"shortage"`
	Surplus           int     `gorm:"surplus" json:"surplus"`
	All               int     `gorm:"all" json:"all"`
	New               int     `gorm:"new" json:"new"`
	Accepted          int     `gorm:"accepted" json:"accepted"`
	ShortageSupplySum float64 `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64 `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64 `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64 `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
}

type ReturnAddProduct struct {
	Barcode   string `gorm:"barcode" json:"barcode"`
	Count     int    `gorm:"count" json:"count"`
	ProductId string `gorm:"product_id" json:"product_id"`
	Type      string `gorm:"type" json:"type"`
}
