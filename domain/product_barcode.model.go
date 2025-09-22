package domain

import "time"

type ProductBarcode struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	ProductID  string    `json:"product_id"`
	OldBarcode string    `json:"old_barcode"`
	Barcode    string    `json:"barcode"`
	CreatedBy  string    `json:"created_by"`
	Status     string    `json:"status"` // pending, auto, confirmed
	CreatedAt  time.Time `json:"created_at"`
}

type ConfirmBarcodeResponse struct {
	Status     string `json:"status"`
	ProductID  string `json:"product_id"`
	OldBarcode string `json:"old_barcode"`
	NewBarcode string `json:"new_barcode"`
}
