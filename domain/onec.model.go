package domain

import (
	"time"
)

// Create Tovar structure for 1C API
type CreateOnecImportDto struct {
	Dok    Document                `json:"Dok"`
	Apteka Apteka                  `json:"Apteka"`
	Товары []ProductRequestOnecDto `json:"Товары"`
}

// Document structure for 1C API
type Document struct {
	ID             string `gorm:"id" json:"-"`
	DocumentDate   string `gorm:"document_date" json:"data_dok"`
	DocumentNumber string `gorm:"document_number" json:"nomer_dok"`
	StoreCode      int    `gorm:"store_code" json:"-"`
}

// Apteka structure for 1C API
type Apteka struct {
	StoreCode int    `gorm:"store_code" json:"store_code"`
	Name      string `gorm:"name" json:"name"`
	Franshiza bool   `gorm:"franshiza" json:"franshiza"`
}

// Request structure for 1C API
type ProductRequestOnecDto struct {
	Id                  string   `gorm:"type:uuid;default:gen_random_uuid()" json:"-" validate:"omitempty,uuid4"`
	MaterialCode        int      `gorm:"material_code" json:"material_code" validate:"required,gt=0"`
	Name                string   `gorm:"name" json:"name" validate:"required,min=1,max=500"`
	Manufacturer        string   `gorm:"manufacturer" json:"manufacturer" validate:"required,min=1,max=255"`
	Quantity            float64  `gorm:"quantity" json:"quantity" validate:"required,gte=0"`
	RetailPrice         float64  `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat      float64  `gorm:"retail_price_vat" json:"retail_price_vat"`
	SupplyPrice         float64  `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat      float64  `gorm:"supply_price_vat" json:"supply_price_vat"`
	Sum                 float64  `gorm:"sum" json:"sum"`
	VatPrice            float64  `gorm:"vat_price" json:"vat_price"`
	Vat                 string   `gorm:"vat" json:"vat" validate:"required"`
	Markup              float64  `gorm:"markup" json:"markup"`
	VatSum              float64  `gorm:"vat_sum" json:"vat_sum"`
	ProductSeriesNumber string   `gorm:"product_series_number" json:"product_series_number" validate:"required"`
	ExpireDate          string   `gorm:"expire_date" json:"expire_date" validate:"required"`
	Barcode             string   `gorm:"barcode" json:"barcode" validate:"required,min=0,max=255"`
	SumVat              float64  `gorm:"sum_vat" json:"sum_vat"`
	Ikpu                string   `gorm:"ikpu" json:"ikpu" validate:"omitempty,len=min=14,max=255"`
	Mar                 bool     `gorm:"mar" json:"mar"`
	Markirovka          []string `gorm:"-" json:"markirovka"`
}

type RequestOnec struct {
	ID        *string    `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	Method    string     `gorm:"method" json:"method"`
	Payload   []byte     `gorm:"payload" json:"payload"`
	Response  []byte     `gorm:"response" json:"response"`
	Action    string     `gorm:"action" json:"action"`
	DocDate   string     `gorm:"doc_date" json:"doc_date"`
	DocNum    string     `gorm:"doc_num" json:"doc_num"`
	Status    string     `gorm:"status" json:"status"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// product ostatok data for 1c
type OnecProductRes struct {
	Id           string     `gorm:"id" json:"id"`
	MaterialCode int        `gorm:"material_code" json:"material_code"`
	Name         string     `gorm:"name" json:"name"`
	StoreName    string     `gorm:"store_name" json:"store_name"`
	StoreCode    int        `gorm:"store_code" json:"store_code"`
	Barcode      string     `gorm:"barcode" json:"barcode"`
	Manufacturer string     `gorm:"manufacturer" json:"manufacturer"`
	SerialNumber string     `gorm:"serial_number" json:"serial_number"`
	Quantity     float64    `gorm:"quantity" json:"quantity"`
	ExpireDate   *time.Time `gorm:"expire_date" json:"expire_date"`
	RetailPrice  float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPrice  float64    `gorm:"supply_price" json:"supply_price"`
	Sum          float64    `gorm:"sum" json:"sum"`
}

// Request structure for 1C API
type OnecProductRepricing struct {
	Id             string  `gorm:"id" json:"id"`
	MaterialCode   int     `gorm:"material_code" json:"material_code"`
	Name           string  `gorm:"name" json:"name"`
	Barcode        string  `gorm:"barcode" json:"barcode"`
	Manufacturer   string  `gorm:"manufacturer" json:"manufacturer"`
	SerialNumber   string  `gorm:"serial_number" json:"serial_number"`
	RetailPrice    float64 `gorm:"retail_price" json:"retail_price"`
	NewRetailPrice float64 `gorm:"new_retail_price" json:"new_retail_price"`
	SupplyPrice    float64 `gorm:"supply_price" json:"supply_price"`
	ExpireDate     string  `gorm:"expire_date" json:"expire_date"`
}

// Price revalution request
type OnecRepricingRequest struct {
	Dok    Document               `json:"Dok"`
	Apteka Apteka                 `json:"Apteka"`
	Товары []OnecProductRepricing `json:"Товары"`
}

type OnecUpdateQuantityRequest struct {
	Dok    Document                       `json:"Dok"`
	Товары []ProductUpdateQuantityRequest `json:"Товары"`
}

type ProductUpdateQuantityRequest struct {
	StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
	AcceptedCount  float64 `gorm:"accepted_count" json:"accepted_count"`
	GivenCount     float64 `gorm:"given_count" json:"given_count"`
}

// Multi repricing
type OnecMultiRepricingRequest struct {
	Dok     Document                      `json:"Dok"`
	Aptekas []AptekaWithProductsRepricing `json:"Aptekas"`
}

type AptekaWithProductsRepricing struct {
	Apteka Apteka                 `json:"Apteka"`
	Товары []OnecProductRepricing `json:"Товары"`
}

type ProductChangePriceItem struct {
	MaterialCode int     `json:"material_code"`
	MaxPrice     float64 `json:"max_price"`
}

type ProductChangePriceRequest struct {
	Products []ProductChangePriceItem `json:"products"`
}


