package domain

import "time"

type Transfer struct {
	Id               string     `gorm:"id" json:"id"`
	PublicId         string     `gorm:"public_id" json:"public_id"`
	FromStoreId      string     `gorm:"from_store_id" json:"from_store_id"`
	ToStoreId        string     `gorm:"to_store_id" json:"to_store_id"`
	Name             string     `gorm:"name" json:"name"`
	Status           string     `gorm:"status" json:"status"`
	MeasurementCount int        `gorm:"measurement_count" json:"measurement_count"`
	Comment          string     `gorm:"comment" json:"comment"`
	CreatedAt        *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time `gorm:"updated_at" json:"updated_at"`
	AcceptedAt       *time.Time `gorm:"accepted_at" json:"accepted_at"`
	SupplyPriceSum   float64    `gorm:"supply_price_sum" json:"supply_price_sum"`
	RetailPriceSum   float64    `gorm:"retail_price_sum" json:"retail_price_sum"`
	CreatedById      string     `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById      string     `gorm:"column:updated_by" json:"updated_by_id"`
	AcceptedById     string     `gorm:"column:accepted_by" json:"accepted_by_id"`
	FromStore        *Store     `gorm:"foreignKey:FromStoreId" json:"store"`
	ToStore          *Store     `gorm:"foreignKey:ToStoreId" json:"to_store"`
	CreatedBy        *Employee  `gorm:"foreignKey:CreatedById" json:"created_by"`
	UpdatedBy        *Employee  `gorm:"foreignKey:UpdatedById" json:"updated_by"`
	AcceptedBy       *Employee  `gorm:"foreignKey:AcceptedById" json:"accepted_by"`
}

// return off create request
type TransferRequest struct {
	PublicId    string `gorm:"public_id" json:"public_id"`
	Name        string `gorm:"name" json:"name"`
	FromStoreId string `gorm:"from_store_id" json:"from_store_id"`
	ToStoreId   string `gorm:"to_store_id" json:"to_store_id"`
	CreatedBy   string `gorm:"created_by" json:"created_by"`
	Status      string `gorm:"status" json:"status"`
	Comment     string `gorm:"comment" json:"comment"`
}

type TransferDetail struct {
	Id            string     `gorm:"id" json:"id"`
	TransferId    string     `gorm:"transfer_id" json:"transfer_id"`
	ProductId     string     `gorm:"product_id" json:"product_id"`
	ReceivedCount int        `gorm:"received_count" json:"received_count"`
	AcceptedCount int        `gorm:"accepted_count" json:"accepted_count"`
	ScannedCount  int        `gorm:"scanned_count" json:"scanned_count"`
	ExpireDate    string     `gorm:"expire_date" json:"expire_date"`
	SerialNumber  string     `gorm:"serial_number" json:"serial_number"`
	SupplyPrice   float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice   float64    `gorm:"retail_price" json:"retail_price"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
	Name          string     `gorm:"name" json:"name"`
	MaterialCode  int        `gorm:"material_code" json:"material_code"`
	Barcode       string     `gorm:"barcode" json:"barcode"`
	ShortName     string     `gorm:"short_name" json:"short_name"`
	ReceivedSum   float64    `gorm:"received_sum" json:"received_sum"`
	ScannedSum    float64    `gorm:"scanned_sum" json:"scanned_sum"`
}

type TransferDetailStatus struct {
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
