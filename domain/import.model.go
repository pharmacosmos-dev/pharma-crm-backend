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
	Status         string `gorm:"status" json:"status"`
	ImportDate     string `gorm:"import_date" json:"import_date"`
	DocumentNumber string `gorm:"document_number" json:"document_number"`
}

// ImportDetail structure
type ImportDetail struct {
	Id             string     `gorm:"id" json:"id"`
	ImportID       string     `gorm:"import_id" json:"import_id"`
	ProductID      *string    `gorm:"product_id" json:"product_id"`
	ReceivedCount  int        `gorm:"received_count" json:"received_count"`
	AcceptedCount  int        `gorm:"accepted_count" json:"accepted_count"`
	CanceledCount  int        `gorm:"canceled_count" json:"canceled_count"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	UnitName       string     `gorm:"unit_name" json:"unit_name"`
	ReceivedAmount float64    `gorm:"received_amount" json:"received_amount"`
	AcceptedAmount float64    `gorm:"accepted_amount" json:"accepted_amount"`
	SeriesNumber   string     `gorm:"series_number" json:"series_number"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	Vat            int        `gorm:"vat" json:"vat"`
	UnitPerPack    int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Product        *Product   `gorm:"references:Id;foreignKey:ProductID" json:"product"`
	Import         *Import    `gorm:"foreignKey:ImportID" json:"import"`
}

type ImportDetailRequest struct {
	ImportID      string   `gorm:"import_id" json:"import_id"`
	ProductID     *string  `gorm:"product_id" json:"product_id"`
	ReceivedCount int      `gorm:"received_count" json:"received_count"`
	AcceptedCount int      `gorm:"accepted_count" json:"accepted_count"`
	SupplyPrice   float64  `gorm:"supply_price" json:"supply_price"`
	RetailPrice   float64  `gorm:"retail_price" json:"retail_price"`
	ExpireDate    string   `gorm:"expire_date" json:"expire_date"`
	Vat           int      `gorm:"vat" json:"vat"`
	VatSum        float64  `gorm:"vat_sum" json:"vat_sum"`
	SeriesNumber  string   `gorm:"series_number" json:"series_number"`
	Markirovka    []string `gorm:"-" json:"markirovka"`
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
