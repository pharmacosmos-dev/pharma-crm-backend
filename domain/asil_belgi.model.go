package domain

import "time"

type AsilBelgiTokenRequest struct {
	Token     string `json:"token" binding:"required"`
	ExpiresAt string `json:"expires_at"`
}

type AsilBelgiToken struct {
	ID        string    `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"not null" json:"token"`
	IssuedAt  time.Time `gorm:"default:now()" json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type AsilBelgiRequest struct {
	Markirovka  string `json:"markirovka" example:"010481260800223421Hu4FxE1AFYCLn"`
	ProductName string `json:"productName" example:"АРПЕФЛЮ Таблетки покрытые пленочной оболочкой 100 мг №20"`
	ProductID   string `json:"productId" example:"uuid"`
	UserID      string `json:"-" example:"uuid"`
}

type CisInfo struct {
	ProductName string `json:"productName"`
	Gtin        string `json:"gtin"`
}

type AsilBelgiBarcodeResponse struct {
	Id                   string  `json:"id"`
	Status               string  `json:"status"`
	OldBarcode           string  `json:"old_barcode"`
	NewBarcode           string  `json:"new_barcode"`
	AsilBelgiProductName string  `json:"asil_belgi_product_name"`
	RequestName          string  `json:"request_name"`
	Similarity           float64 `json:"similarity"`
}
