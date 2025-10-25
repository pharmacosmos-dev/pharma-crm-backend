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
	ReceivedCount     float64    `gorm:"received_count" json:"received_count"`
	AcceptedCount     float64    `gorm:"accepted_count" json:"accepted_count"`
	AcceptedAmountVat float64    `gorm:"accepted_amount_vat" json:"accepted_amount_vat"`
	ReceivedAmountVat float64    `gorm:"received_amount_vat" json:"received_amount_vat"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	Store             *Store     `gorm:"foreignKey:StoreID" json:"store"`
	Sender            *Employee  `gorm:"foreignKey:CreatedBy" json:"sender"`
	Receiver          *Employee  `gorm:"foreignKey:AcceptedBy" json:"receiver"`
}

type ImportStatusSummary struct {
	CompletedReceivedVatAmount float64 `json:"completed_received_vat_amount"`
	NewAcceptedVatAmount       float64 `json:"new_accepted_vat_amount"`
	CompletedAcceptedCount     float64 `json:"completed_accepted_count"`
	NewReceivedCount           float64 `json:"new_received_count"`
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
	ReceivedCount     float64    `gorm:"received_count" json:"received_count"`
	AcceptedCount     float64    `gorm:"accepted_count" json:"accepted_count"`
	ScannedCount      float64    `gorm:"scanned_count" json:"scanned_count"`
	CanceledCount     float64    `gorm:"canceled_count" json:"canceled_count"`
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
	ProducerCode      string     `gorm:"producer_code" json:"producer_code"`
	ProductName       string     `gorm:"product_name" json:"product_name,omitempty"`
	Barcode           string     `gorm:"barcode" json:"barcode,omitempty"`
	MaterialCode      int        `gorm:"material_code" json:"material_code,omitempty"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
	ImportedAt        *time.Time `gorm:"imported_at" json:"imported_at"`
	Mxik              string     `gorm:"mxik" json:"mxik"`
	UnitCode          string     `gorm:"unit_code" json:"unit_code"`
	UnitLabel         string     `gorm:"unit_label" json:"unit_label"`
	IsMarking         bool       `gorm:"is_marking" json:"is_marking"`
	Product           *Product   `gorm:"references:Id;foreignKey:ProductID" json:"product"`
	Import            *Import    `gorm:"foreignKey:ImportID" json:"import"`
	StoreProductId    string     `gorm:"store_product_id" json:"store_product_id"`
}

type ImportDetailRequest struct {
	ImportID       string            `gorm:"import_id" json:"import_id"`
	ProductID      *string           `gorm:"product_id" json:"product_id"`
	ReceivedCount  float64           `gorm:"received_count" json:"received_count"`
	AcceptedCount  float64           `gorm:"accepted_count" json:"accepted_count"`
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
	ScannedCount  float64 `json:"scanned_count"`
	ShortageCount float64 `json:"shortage_count"`
	TotalCount    float64 `json:"total_count"`
	SurplusCount  float64 `json:"surplus_count"`
}

// product markirovka structure
type ProductMarking struct {
	Id             string     `gorm:"id" json:"id"`
	ImportDetailId string     `gorm:"import_detail_id" json:"import_detail_id"`
	ProductId      string     `gorm:"product_id" json:"product_id"`
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
	MaterialCode        int     `gorm:"material_code" json:"material_code"`
	Name                string  `gorm:"name" json:"name"`
	Barcode             string  `gorm:"barcode" json:"barcode"`
	Manufacturer        string  `gorm:"manufacturer" json:"manufacturer"`
	ProductSeriesNumber string  `gorm:"product_series_number" json:"product_series_number"`
	Quantity            float64 `gorm:"quantity" json:"quantity"`
	QuantityFakt        float64 `gorm:"quantity_fakt" json:"quantity_fakt"`
}

// import detail query params
type ImportDetailQueryParams struct {
	Limit              int     `form:"limit"`
	Offset             int     `form:"offset"`
	ImportId           string  `form:"import_id"`
	Search             string  `form:"search"`
	ReceivedAmountFrom float64 `form:"received_amount_from"`
	ReceivedAmountTo   float64 `form:"received_amount_to"`
	NoMarking          bool    `form:"no_marking"`
	NoBarcode          bool    `form:"no_barcode"`
}

// Import data
type ImportProductData struct {
	Id          string     `gorm:"id" json:"id"`
	PublicId    int        `gorm:"public_id" json:"public_id"`
	EntryType   int        `gorm:"entry_type" json:"entry_type"`
	Count       string     `gorm:"-" json:"count"`
	UnitPerPack int        `gorm:"unit_per_pack" json:"-"`
	Quantity    float64    `gorm:"quantity" json:"quantity"`
	Sum         float64    `gorm:"sum" json:"sum"`
	Name        string     `gorm:"name" json:"name"`
	StoreName   string     `gorm:"store_name" json:"store_name"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	TotalCount  int64      `gorm:"total_count" json:"-"`
}
