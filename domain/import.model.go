package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Import structure
type Import struct {
	Id                string     `gorm:"id" json:"id"`
	PublicID          int        `gorm:"public_id" json:"public_id"`
	StoreID           string     `gorm:"store_id" json:"store_id"`
	StoreCode         int        `gorm:"store_code" json:"store_code"`
	CreatedBy         string     `gorm:"created_by" json:"created_by"`
	AcceptedBy        string     `gorm:"accepted_by" json:"accepted_by"`
	DocumentNumber    string     `gorm:"document_number" json:"document_number"`
	DocumentYear      int        `gorm:"document_year" json:"document_year"`
	Status            string     `gorm:"status" json:"status"`
	ImportDate        *time.Time `gorm:"import_date" json:"import_date"`
	AcceptedAmount    float64    `gorm:"accepted_amount" json:"accepted_amount"`
	ReceivedAmount    float64    `gorm:"received_amount" json:"received_amount"`
	ReceivedCount     int        `gorm:"received_count" json:"received_count"`
	AcceptedCount     int        `gorm:"accepted_count" json:"accepted_count"`
	AcceptedAmountVat float64    `gorm:"accepted_amount_vat" json:"accepted_amount_vat"`
	ReceivedAmountVat float64    `gorm:"received_amount_vat" json:"received_amount_vat"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	Store             *Store     `gorm:"foreignKey:StoreID" json:"store"`
	Sender            *Employee  `gorm:"foreignKey:CreatedBy" json:"sender"`
	Receiver          *Employee  `gorm:"foreignKey:AcceptedBy" json:"receiver"`
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
	Id                string     `gorm:"id" json:"id"`
	ImportID          string     `gorm:"import_id" json:"import_id"`
	ProductID         *string    `gorm:"product_id" json:"product_id"`
	ReceivedCount     int        `gorm:"received_count" json:"received_count"`
	AcceptedCount     int        `gorm:"accepted_count" json:"accepted_count"`
	ScannedCount      int        `gorm:"scanned_count" json:"scanned_count"`
	CanceledCount     int        `gorm:"canceled_count" json:"canceled_count"`
	SupplyPrice       float64    `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat    float64    `gorm:"supply_price_vat" json:"supply_price_vat"`
	RetailPrice       float64    `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat    float64    `gorm:"retail_price_vat" json:"retail_price_vat"`
	UnitName          string     `gorm:"unit_name" json:"unit_name"`
	ReceivedAmount    float64    `gorm:"received_amount" json:"received_amount"`
	ReceivedAmountVat float64    `gorm:"received_amount_vat" json:"received_amount_vat"`
	AcceptedAmount    float64    `gorm:"accepted_amount" json:"accepted_amount"`
	AcceptedAmountVat float64    `gorm:"accepted_amount_vat" json:"accepted_amount_vat"`
	SeriesNumber      string     `gorm:"series_number" json:"series_number"`
	ExpireDate        *time.Time `gorm:"expire_date" json:"expire_date"`
	Vat               int        `gorm:"vat" json:"vat"`
	VatSum            float64    `gorm:"vat_sum" json:"vat_sum"`
	SumVat            float64    `gorm:"sum_vat" json:"sum_vat"`
	UnitPerPack       int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	ProducerName      string     `gorm:"producer_name" json:"producer_name"`
	ProductName       string     `gorm:"product_name" json:"product_name,omitempty"`
	Barcode           string     `gorm:"barcode" json:"barcode,omitempty"`
	MaterialCode      int        `gorm:"material_code" json:"material_code,omitempty"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	Product           *Product   `gorm:"references:Id;foreignKey:ProductID" json:"product"`
	Import            *Import    `gorm:"foreignKey:ImportID" json:"import"`
}

type ImportDetailRequest struct {
	ImportID       string            `gorm:"import_id" json:"import_id"`
	ProductID      *string           `gorm:"product_id" json:"product_id"`
	ReceivedCount  int               `gorm:"received_count" json:"received_count"`
	AcceptedCount  int               `gorm:"accepted_count" json:"accepted_count"`
	SupplyPrice    float64           `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat float64           `gorm:"supply_price_vat" json:"supply_price_vat"`
	RetailPrice    float64           `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat float64           `gorm:"retail_price_vat" json:"retail_price_vat"`
	ExpireDate     string            `gorm:"expire_date" json:"expire_date"`
	Vat            int               `gorm:"vat" json:"vat"`
	VatSum         float64           `gorm:"vat_sum" json:"vat_sum"`
	SumVat         float64           `gorm:"sum_vat" json:"sum_vat"`
	SeriesNumber   string            `gorm:"series_number" json:"series_number"`
	Marking        utils.StringArray `gorm:"type:text[]" json:"marking"`
}

type ImportUpdateRequest struct {
	ScannedCount int `gorm:"scanned_count" json:"scanned_count"`
}

type AddScanRequest struct {
	ID       string `json:"id"`
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

// product markirovka structure
type ProductMarking struct {
	Id             string     `gorm:"id" json:"id"`
	ImportDetailId string     `gorm:"import_detail_id" json:"import_detail_id"`
	Marking        string     `gorm:"marking" json:"marking"`
	Status         int8       `gorm:"status" json:"status"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

// product markirovka create structure
type ProductMarkingReq struct {
	ImportDetailId string   `gorm:"import_detail_id" json:"import_detail_id"`
	ProductID      string   `gorm:"product_id" json:"product_id"`
	Marking        []string `gorm:"marking" json:"marking"`
}

// Accept imported Product
type AcceptImport1C struct {
	Dok    Document                 `json:"Dok"`
	Apteka Apteka                   `json:"Apteka"`
	Товары []AcceptImport1CResponse `json:"Товары"`
}

// AcceptImport1CResponse structure for 1C resonse API
type AcceptImport1CResponse struct {
	MaterialCode        int    `gorm:"material_code" json:"material_code"`
	Name                string `gorm:"name" json:"name"`
	Barcode             string `gorm:"barcode" json:"barcode"`
	Manufacturer        string `gorm:"manufacturer" json:"manufacturer"`
	ProductSeriesNumber string `gorm:"product_series_number" json:"product_series_number"`
	Quantity            int    `gorm:"quantity" json:"quantity"`
	QuantityFakt        int    `gorm:"quantity_fakt" json:"quantity_fakt"`
}
