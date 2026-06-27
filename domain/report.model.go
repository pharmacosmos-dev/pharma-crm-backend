package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

type ReportQueryParam struct {
	StoreId    string      `form:"store_id"`
	StartDate  *CustomTime `form:"start_date"`
	EndDate    *CustomTime `form:"end_date"`
	Limit      int         `form:"limit"`
	Offset     int         `form:"offset"`
	Search     string      `form:"search"`
	Order      string      `form:"order"`
	EmployeeId string      `form:"employee_id"`
	ProducerId string      `form:"producer_id"`
	CompanyId  string      `form:"company_id"`
	StoreIds   []string    `json:"store_ids"`
	CompanyIds []string    `json:"company_ids"`
}

type ProductStatusReport struct {
	TotalQuantity               float64 `json:"total_quantity"`
	TotalQuantityReturned       float64 `json:"total_quantity_returned"`
	TotalRetailPriceSum         float64 `json:"total_retail_price_sum"`
	TotalRetailPriceSumReturned float64 `json:"total_retail_price_sum_returned"`
	TotalDiscountSum            float64 `json:"total_discount_sum"`
	TotalLoyaltyCardSum         float64 `json:"total_loyalty_card_sum"`
}

// Bonus report structure
type BonusReport struct {
	Id         string  `gorm:"id" json:"id"`
	PublicId   int     `gorm:"public_id" json:"public_id"`
	FullName   string  `gorm:"full_name" json:"full_name"`
	Phone      string  `gorm:"phone" json:"phone"`
	StoreName  string  `gorm:"store_name" json:"store_name"`
	Role       string  `gorm:"role" json:"role"`
	Amount     float64 `gorm:"amount" json:"amount"`
	Count      float64 `gorm:"count" json:"count"`
	TotalCount int64   `gorm:"total_count" json:"-"`
}

// get product report
type ProductReport struct {
	CartItemId     string     `gorm:"cart_item_id" json:"cart_item_id"`
	MaterialCode   int        `gorm:"material_code" json:"material_code"`
	StoreName      string     `gorm:"store_name" json:"store_name"`
	ProductName    string     `gorm:"product_name" json:"product_name"`
	UnitPerPack    int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	ProducerName   string     `gorm:"producer_name" json:"producer_name"`
	SerialNumber   string     `gorm:"serial_number" json:"serial_number"`
	TotalDiscount  float64    `gorm:"total_discount" json:"total_discount"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity       string     `gorm:"quantity" json:"quantity"`
	UnitQuantity   int        `gorm:"unit_quantity" json:"unit_quantity"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	SupplyPriceSum float64    `gorm:"supply_price_sum" json:"supply_price_sum"`
	RetailPriceSum float64    `gorm:"retail_price_sum" json:"retail_price_sum"`
	MarkupSum      float64    `gorm:"markup_sum" json:"markup_sum"`
	VatSum         float64    `gorm:"vat_sum" json:"vat_sum"`
	CompletedAt    *time.Time `gorm:"completed_at" json:"completed_at"`
	FullName       string     `gorm:"full_name" json:"full_name"`
	SaleNumber     int        `gorm:"sale_number" json:"sale_number"`
	SaleType       string     `gorm:"sale_type" json:"sale_type"`
	MarkingCount   int        `gorm:"marking_count" json:"marking_count"`
	TotalCount     int64      `gorm:"total_count" json:"-"`
}

// lfl report structure
type LflReport struct {
	FirstMonth  []LflReportDetail `json:"first_month"`
	SecondMonth []LflReportDetail `json:"second_month"`
}

type LflReportDetail struct {
	Id            int     `gorm:"id" json:"id"`
	Weekdate      string  `gorm:"weekdate" json:"weekdate"`
	Weekname      string  `gorm:"weekname" json:"weekname"`
	BranchCount   int     `gorm:"branch_count" json:"branch_count"`
	LcSum         float64 `gorm:"lc_sum" json:"lc_sum"`
	ParapharmaSum float64 `gorm:"parapharma_sum" json:"parapharma_sum"`
	TotalSum      float64 `gorm:"total_sum" json:"total_sum"`
	WeekNumber    int     `gorm:"week_number" json:"week_number"`
	Weekday       int     `gorm:"weekday" json:"weekday"`
}

// Store report amount with payment types
type StoreAmount struct {
	UID            int        `gorm:"uid" json:"uid"`
	ID             string     `gorm:"id" json:"id"`
	StoreCode      int        `gorm:"store_code" json:"store_code"`
	StoreName      string     `gorm:"store_name" json:"store_name"`
	SaleDate       *time.Time `gorm:"sale_date" json:"sale_date"`
	Cash           float64    `gorm:"cash" json:"cash"`
	Uzcard         float64    `gorm:"uzcard" json:"uzcard"`
	Humo           float64    `gorm:"humo" json:"humo"`
	Click          float64    `gorm:"click" json:"click"`
	Payme          float64    `gorm:"payme" json:"payme"`
	Alif           float64    `gorm:"alif" json:"alif"`
	Uzum           float64    `gorm:"uzum" json:"uzum"`
	UzumTezKor     float64    `gorm:"column:uzum_tez_kor" json:"uzum_tez_kor"`
	ReturnAmount   float64    `gorm:"return_amount" json:"return_amount"`
	TotalAmount    float64    `gorm:"total_amount" json:"total_amount"`
	DiscountAmount float64    `gorm:"discount_amount" json:"discount_amount"`
	LoyaltyCardAmount  float64   `gorm:"loyalty_card_amount" json:"loyalty_card_amount"`
	ChequeCount    int        `gorm:"cheque_count" json:"cheque_count"`
}

type StoreReportStats struct {
	TotalTransactionSum float64 `gorm:"total_transaction_sum" json:"total_transaction_sum"`
	TotalTransaction    int     `gorm:"total_transaction" json:"total_transaction"`
	TotalReturnalsSum   float64 `gorm:"total_returnals_sum" json:"total_returnals_sum"`
	TotalReturnedCount  int     `gorm:"total_returned_count" json:"total_returned_count"`
	TotalDiscountSum    float64 `gorm:"total_discount_sum" json:"total_discount_sum"`
	TotalDiscountCount  int     `gorm:"total_discount_count" json:"total_discount_count"`
	TotalCashSum        float64 `gorm:"total_cash_sum" json:"total_cash_sum"`
	TotalCashCount      int     `gorm:"total_cash_count" json:"total_cash_count"`
	TotalHumoSum        float64 `gorm:"total_humo_sum" json:"total_humo_sum"`
	TotalHumoCount      int     `gorm:"total_humo_count" json:"total_humo_count"`
	TotalUzcardSum      float64 `gorm:"total_uzcard_sum" json:"total_uzcard_sum"`
	TotalUzcardCount    int     `gorm:"total_uzcard_count" json:"total_uzcard_count"`
	TotalClickSum       float64 `gorm:"total_click_sum" json:"total_click_sum"`
	TotalClickCount     int     `gorm:"total_click_count" json:"total_click_count"`
	TotalPaymeSum       float64 `gorm:"total_payme_sum" json:"total_payme_sum"`
	TotalPaymeCount     int     `gorm:"total_payme_count" json:"total_payme_count"`
	TotalAlifSum        float64 `gorm:"total_alif_sum" json:"total_alif_sum"`
	TotalAlifCount      int     `gorm:"total_alif_count" json:"total_alif_count"`
	TotalUzumCount      int     `gorm:"total_uzum_count" json:"total_uzum_count"`
	TotalUzumSum        float64 `gorm:"total_uzum_sum" json:"total_uzum_sum"`
	TotalUzumTezKorCount int     `gorm:"total_uzum_tez_kor_count" json:"total_uzum_tez_kor_count"`
    TotalUzumTezKorSum  float64 `gorm:"total_uzum_tez_kor_sum" json:"total_uzum_tez_kor_sum"`
	TotalLoyaltyCardSum   float64 `gorm:"total_loyalty_card_sum" json:"total_loyalty_card_sum"`
	TotalLoyaltyCardCount int     `gorm:"total_loyalty_card_count" json:"total_loyalty_card_count"`
	TotalCashbackSum      float64 `gorm:"total_cashback_sum" json:"total_cashback_sum"`
	TotalCashbackCount    float64 `gorm:"total_cashback_count" json:"total_cashback_count"`
	TotalProductCount     int64   `gorm:"total_product_count" json:"total_product_count"`
}

type StoreSummary struct {
	Name                string  `json:"name"`
	SaleAmount          float64 `json:"sale_amount"`
	DiscountAmount      float64 `gorm:"discount_amount" json:"discount_amount"`
	LoyaltyCardAmount   float64 `gorm:"loyalty_card_amount" json:"loyalty_card_amount"`
	ImportAmount        float64 `json:"import_amount"`
	StockAmount         float64 `json:"stock_amount"`
	ImportStocAmount    float64 `gorm:"import_stock_amount" json:"import_stock_amount"`
	Total               float64 `json:"total"`
	ImportTotal         float64 `gorm:"import_total" json:"import_total"`
}

type StoreSummaryStats struct {
	TotalSaleAmount        float64 `json:"total_sale_amount"`
	TotalDiscountAmount    float64 `gorm:"total_discount_amount" json:"total_discount_amount"`
	TotalLoyaltyCardAmount float64 `gorm:"total_loyalty_card_amount" json:"total_loyalty_card_amount"`
	TotalImportAmount      float64 `json:"total_import_amount"`
	TotalStockAmount       float64 `json:"total_stock_amount"`
	TotalImportStockAmount float64 `gorm:"total_import_stock_amount" json:"total_import_stock_amount"`
	Total                  float64 `json:"total"`
	ImportTotal            float64 `gorm:"import_total" json:"import_total"`
}

type StoreProductsReport struct {
	ID           string            `gorm:"id" json:"id"`
	StoreID      string            `gorm:"store_id" json:"store_id"`
	StoreName    string            `gorm:"store_name" json:"store_name"`
	Name         string            `gorm:"name" json:"name"`
	UnitQuantity int               `gorm:"unit_quantity" json:"unit_quantity"`
	MaterialCode int               `gorm:"material_code" json:"material_code"`
	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
	Barcode      string            `gorm:"barcode" json:"barcode"`
	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
	MXIK         string            `gorm:"mxik" json:"mxik"`
	UnitCode     string            `gorm:"unit_code" json:"unit_code"`
	IsMarking    bool              `gorm:"is_marking" json:"is_marking"`
	CreatedAt    time.Time         `gorm:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"updated_at" json:"updated_at"`
	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
	UnitName     string            `gorm:"unit_name" json:"unit_name"`
	UnitLabel    string            `gorm:"unit_label" json:"unit_label"`
}

type DiscountCardReport struct {
	Id                   int     `json:"id"`
	StoreID              string  `json:"store_id"`
	StoreName            string  `json:"store_name"`
	CustomerID           string  `json:"customer_id"`
	CustomerName         string  `json:"customer_name"`
	CheckCount           int64   `json:"check_count"`
	Percent              int     `json:"percent"`
	TotalWithoutDiscount float64 `json:"total_without_discount"`
	TotalDiscount        float64 `json:"total_discount"`
	TotalWithDiscount    float64 `json:"total_with_discount"`
	TotalCount           int64   `json:"-"`
}

// type RemainingProduct struct {
// 	ID           string            `gorm:"id" json:"id"`
// 	StoreID      string            `gorm:"store_id" json:"store_id"`
// 	UnitQuantity int               `gorm:"unit_quantity" json:"unit_quantity"`
// 	MaterialCode int               `gorm:"material_code" json:"material_code"`
// 	Name         string            `gorm:"name" json:"name"`
// 	Photos       utils.StringArray `gorm:"type:text[]" json:"photos"`
// 	Barcode      string            `gorm:"barcode" json:"barcode"`
// 	UnitPerPack  int               `gorm:"unit_per_pack" json:"unit_per_pack"`
// 	MXIK         string            `gorm:"mxik" json:"mxik"`
// 	UnitCode     string            `gorm:"unit_code" json:"unit_code"`
// 	IsMarking    bool              `gorm:"is_marking" json:"is_marking"`
// 	CreatedAt    time.Time         `gorm:"created_at" json:"created_at"`
// 	UpdatedAt    time.Time         `gorm:"updated_at" json:"updated_at"`
// 	Manufacturer string            `gorm:"manufacturer" json:"manufacturer"`
// 	UnitName     string            `gorm:"unit_name" json:"unit_name"`
// 	UnitLabel    string            `gorm:"unit_label" json:"unit_label"`
// }

type StoreProductGivenDayParams struct {
	Date    string `form:"date"`
	StoreId string `form:"store_id"`
	Search  string `form:"search"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

type OstatokForDate struct {
	ProductId      string     `gorm:"product_id" json:"product_id"`
	Name           string     `gorm:"name" json:"name"`
	UnitPerPack    int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	MinSupplyPrice float64    `gorm:"min_supply_price" json:"min_supply_price"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	MinRetailPrice float64    `gorm:"min_retail_price" json:"min_retail_price"`
	UnitQuantity   float64    `gorm:"unit_quantity" json:"unit_quantity"`
}
