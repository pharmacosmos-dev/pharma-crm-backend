package domain

import "time"

// Import structure
type Import struct {
	Id             string     `gorm:"id" json:"id"`
	PublicID       int        `gorm:"public_id" json:"public_id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	StoreCode      int        `gorm:"store_code" json:"store_code"`
	CreatedBy      string     `gorm:"created_by" json:"created_by"`
	AcceptedBy     string     `gorm:"accepted_by" json:"accepted_by"`
	DocumentNumber string     `gorm:"document_number" json:"document_number"`
	DocumentYear   int        `gorm:"document_year" json:"document_year"`
	Status         string     `gorm:"status" json:"status"`
	ImportDate     *time.Time `gorm:"import_date" json:"import_date"`
	AcceptedAmount float64    `gorm:"accepted_amount" json:"accepted_amount"`
	ReceivedAmount float64    `gorm:"received_amount" json:"received_amount"`
	ReceivedCount  int        `gorm:"received_count" json:"received_count"`
	AcceptedCount  int        `gorm:"accepted_count" json:"accepted_count"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Store          *Store     `gorm:"foreignKey:StoreID" json:"store"`
	Sender         *Employee  `gorm:"foreignKey:CreatedBy" json:"sender"`
	Receiver       *Employee  `gorm:"foreignKey:AcceptedBy" json:"receiver"`
}

// ImportRequest structure
type ImportRequest struct {
	Id             string `gorm:"id" json:"id"`
	StoreID        string `gorm:"store_id" json:"store_id"`
	StoreCode      int    `gorm:"store_code" json:"store_code"`
	Status         string `gorm:"status" json:"status"`
	ImportDate     string `gorm:"import_date" json:"import_date"`
	DocumentNumber string `gorm:"document_number" json:"document_number"`
}

type ImportDetail struct {
	Id             string     `gorm:"id" json:"id"`
	ImportID       string     `gorm:"import_id" json:"import_id"`
	ProductID      *string    `gorm:"product_id" json:"product_id"`
	ReceivedCount  int        `gorm:"received_count" json:"received_count"`
	AcceptedCount  int        `gorm:"accepted_count" json:"accepted_count"`
	CanceledCount  int        `gorm:"canceled_count" json:"canceled_count"`
	ReceivedAmount float64    `gorm:"received_amount" json:"received_amount"`
	AcceptedAmount float64    `gorm:"accepted_amount" json:"accepted_amount"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Product        *Product   `gorm:"references:MaterialCode;foreignKey:ProductMaterialCode" json:"product"`
	Import         *Import    `gorm:"foreignKey:ImportID" json:"import"`
}

type ImportDetailRequest struct {
	ImportID       string  `gorm:"import_id" json:"import_id"`
	ProductID      *string `gorm:"product_id" json:"product_id"`
	ReceivedCount  int     `gorm:"received_count" json:"received_count"`
	AcceptedCount  int     `gorm:"accepted_count" json:"accepted_count"`
	ReceivedAmount float64 `gorm:"received_amount" json:"received_amount"`
	AcceptedAmount float64 `gorm:"accepted_amount" json:"accepted_amount"`
}

type ImportUpdateRequest struct {
	ScannedCount int `gorm:"accepted_count" json:"scanned_count"`
}

type AddScanRequest struct {
	ImportID string `json:"import_id"`
	Barcode  string `json:"barcode"`
	Count    int    `json:"count"`
}

type StockCountResponse struct {
	ScannedCount  int `json:"scanned_count"`
	ShortageCount int `json:"shortage_count"`
	TotalCount    int `json:"total_count"`
	SurplusCount  int `json:"surplus_count"`
}
