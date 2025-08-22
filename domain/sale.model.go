package domain

import (
	"time"
)

// Sale structure
type Sale struct {
	ID                 string         `gorm:"id" json:"id"`
	DeviceID           string         `gorm:"device_id" json:"device_id"`
	StoreId            string         `gorm:"store_id" json:"store_id"`
	EmployeeID         string         `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string         `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string         `gorm:"cashbox_id" json:"cashbox_id"`
	CustomerID         string         `gorm:"customer_id" json:"customer_id"`
	SaleNumber         int            `gorm:"sale_number" json:"sale_number"`
	TotalDiscount      float64        `gorm:"total_discount" json:"total_discount"`
	TotalAmount        float64        `gorm:"total_amount" json:"total_amount"`
	ReturnedAmount     float64        `gorm:"returned_amount" json:"returned_amount"`
	ProductCount       int            `gorm:"product_count" json:"product_count"`
	Type               string         `gorm:"type" json:"type"`
	SaleType           string         `gorm:"sale_type" json:"sale_type"`
	Status             string         `gorm:"status" json:"status"`
	OnlineStatus       int            `gorm:"online_status" json:"online_status"`
	IsDelivered        bool           `gorm:"is_delivered" json:"is_delivered"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	CompletedAt        *time.Time     `gorm:"completed_at" json:"completed_at"`
	Employee           *Employee      `gorm:"foreignKey:EmployeeID" json:"employee"`
	Customer           *Customer      `gorm:"foreignKey:CustomerID" json:"customer"`
	SalePayments       []*SalePayment `gorm:"foreignKey:SaleID" json:"sale_payments"`
	CartItems          []*CartItem    `gorm:"foreignKey:SaleId" json:"cart_items"`
}

// SaleRequest structure for create
type SaleRequest struct {
	ID                 string  `gorm:"id" json:"id"`
	EmployeeID         string  `gorm:"employee_id" json:"employee_id"`
	StoreId            string  `gorm:"store_id" json:"store_id"`
	CashBoxOperationId string  `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string  `gorm:"cashbox_id" json:"cashbox_id"`
	ServiceType        *string `gorm:"service_type" json:"service_type"`
}

// SaleReturnRequest structure for create
type SaleReturnRequest struct {
	SaleId             string     `gorm:"sale_id" json:"sale_id"`
	EmployeeID         string     `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string     `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string     `gorm:"cashbox_id" json:"cashbox_id"`
	SaleType           string     `gorm:"sale_type" json:"sale_type"`
	Items              []SaleItem `gorm:"-" json:"sale_items"`
}

// SaleItem structure for create return

type SaleItem struct {
	SaleId         string `gorm:"sale_id" json:"-"`
	StoreProductId string `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int    `gorm:"quantity" json:"quantity"`
	UnitQuantity   int    `gorm:"unit_quantity" json:"unit_quantity"`
}

type SaleResponse struct {
	ID                 string         `gorm:"id" json:"id"`
	ParentId           string         `gorm:"parent_id" json:"parent_id"`
	EmployeeID         string         `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string         `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CustomerID         string         `gorm:"customer_id" json:"customer_id"`
	SaleNumber         int            `gorm:"sale_number" json:"sale_number"`
	TotalDiscount      float64        `gorm:"total_discount" json:"total_discount"`
	TotalAmount        float64        `gorm:"total_amount" json:"total_amount"`
	VatSum             float64        `gorm:"vat_sum" json:"vat_sum"`
	ReturnedAmount     float64        `gorm:"returned_amount" json:"returned_amount"`
	ProductCount       float64        `gorm:"product_count" json:"product_count"`
	Status             string         `gorm:"status" json:"status"`
	OnlineStatus       int            `gorm:"online_status" json:"online_status"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	CompletedAt        *time.Time     `gorm:"completed_at" json:"completed_at"`
	StoreName          string         `gorm:"store_name" json:"store_name"`
	CashBoxName        string         `gorm:"cash_box_name" json:"cash_box_name"`
	FullName           string         `gorm:"full_name" json:"full_name"`
	Phone              string         `gorm:"phone" json:"phone"`
	Type               string         `gorm:"type" json:"type"`
	SaleType           string         `gorm:"sale_type" json:"sale_type"`
	Cash               float64        `gorm:"cash" json:"cash"`
	Uzcard             float64        `gorm:"uzcard" json:"uzcard"`
	Humo               float64        `gorm:"humo" json:"humo"`
	Click              float64        `gorm:"click" json:"click"`
	Payme              float64        `gorm:"payme" json:"payme"`
	IsDelivered        bool           `gorm:"is_delivered" json:"is_delivered"`
	FiscalSign         string         `gorm:"fiscal_sign" json:"fiscal_sign"`
	CustomerName       *string        `gorm:"customer_name" json:"customer_name"`
	CustomerPhone      *string        `gorm:"customer_phone" json:"customer_phone"`
	Employee           *Employee      `gorm:"foreignKey:EmployeeID" json:"employee"`
	Customer           *Customer      `gorm:"foreignKey:CustomerID" json:"customer"`
	SalePayments       []*SalePayment `gorm:"foreignKey:SaleID" json:"sale_payments"`
	CartItems          []*CartItem    `gorm:"foreignKey:SaleId" json:"cart_items"`
	Product            []ProductRes   `gorm:"-" json:"products"`
	EposResponse       *EposResponse  `gorm:"-" json:"epos_response"`
}

// SaleUpdateRequest structure for update
type SaleUpdateRequest struct {
	ID            string  `gorm:"id" json:"-"`
	TotalDiscount float64 `gorm:"total_discount" json:"total_discount"`
	TotalAmount   float64 `gorm:"total_amount" json:"total_amount"`
}

// FinalSale structure
type FinalSale struct {
	StoreID            string             `gorm:"store_id" json:"store_id"`
	SaleID             string             `gorm:"sale_id" json:"sale_id"`
	PrescriptionID     string             `gorm:"prescription_id" json:"prescription_id"`
	CustomerID         *string            `gorm:"customer_id" json:"customer_id"`
	CashBoxOperationId string             `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	TotalAmount        float64            `gorm:"total_amount" json:"total_amount"`
	ServiceType        *string            `gorm:"service_type" json:"service_type"`
	PaymentTypes       []FinalPaymentType `json:"payment_types"`
	MarkingData        []MarkingData      `json:"marking_data"`
	EposData           [][]EposItem       `json:"epos_data"`
}

type MarkingData struct {
	Id           string   `json:"id" gorm:"id"`
	MarkingCount int      `json:"marking_count" gorm:"marking_count"`
	MarkingList  []string `json:"marking_list" gorm:"marking_list"`
}
type EposItem struct {
	Barcode     string `json:"barcode"`
	Amount      int    `json:"amount"`
	Price       int    `json:"price"`
	Discount    int    `json:"discount"`
	VatPercent  int    `json:"vatPercent"`
	Vat         int    `json:"vat"`
	Label       string `json:"label"`
	Name        string `json:"name"`
	ClassCode   string `json:"classCode"`
	PackageCode string `json:"packageCode"`
	Other       int    `json:"other"`
	OwnerType   int    `json:"ownerType"`
}

// FinalPaymentType structure
type FinalPaymentType struct {
	PaymentTypeID string  `gorm:"payment_type_id" json:"payment_type_id"`
	Amount        float64 `gorm:"amount" json:"amount"`
	ReturnAmount  float64 `gorm:"return_amount" json:"return_amount"`
	AppType       string  `gorm:"app_type" json:"app_type" example:"click|payme|uzum"`
	Type          string  `gorm:"type" json:"type" example:"card|cash|app"`
	OtpData       string  `gorm:"otp_data" json:"otp_data"`
}

// Total amount struct
type SaleTotalAmount struct {
	TotalAmount   float64 `gorm:"total_amount" json:"total_amount"`
	CashAmount    float64 `gorm:"cash_amount" json:"cash_amount"`
	HumoAmount    float64 `gorm:"humo_amount" json:"humo_amount"`
	UzcardAmount  float64 `gorm:"uzcard_amount" json:"uzcard_amount"`
	VisaAmount    float64 `gorm:"visa_amount" json:"visa_amount"`
	ClickAmount   float64 `gorm:"click_amount" json:"click_amount"`
	PaymeAmount   float64 `gorm:"payme_amount" json:"payme_amount"`
	UzumAmount    float64 `gorm:"uzum_amount" json:"uzum_amount"`
	BalanceAmount float64 `gorm:"balance_amount" json:"balance_amount"`
}

// SaleStats structure
type SaleStats struct {
	TotalTransactionsSum float64            `gorm:"total_transactions_sum" json:"total_transactions_sum"`
	TotalReturnalsSum    float64            `gorm:"total_returnals_sum" json:"total_returnals_sum"`
	TotalCount           int64              `gorm:"total_count" json:"total_count"`
	TotalProductCount    int64              `gorm:"total_product_count" json:"total_product_count"`
	PaymentTypeStats     []PaymentTypeStats `gorm:"-" json:"payment_type_stats"`
}

// PaymentTypeStats structure
type PaymentTypeStats struct {
	Id   string  `gorm:"id" json:"id"`
	Name string  `gorm:"name" json:"name"`
	Type string  `gorm:"type" json:"type"`
	Sum  float64 `gorm:"sum" json:"sum"`
}

// Epos Response structure
type EposResponse struct {
	Id        string     `gorm:"id" json:"id"`
	SaleId    string     `gorm:"sale_id" json:"sale_id"`
	Response  string     `gorm:"response" json:"response"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// EposResponse Request structure
type EposResponseRequest struct {
	SaleId       string `gorm:"sale_id" json:"sale_id"`
	Error        bool   `gorm:"-" json:"error"`
	Status       int    `gorm:"status" json:"status"`
	Response     []byte `gorm:"response" json:"-"`
	ResponseData any    `gorm:"-" json:"response_data"`
}

type EposResponseData struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// region EPOS response structure

// SuccessResponse for successful case
type EposSuccessResponse struct {
	Error   bool               `json:"error"`
	Message EposSuccessMessage `json:"message"`
	Info    Info               `json:"info"`
}

type EposSuccessMessage struct {
	DateTime   string `json:"dateTime"`
	QrCodeURL  string `json:"qrCodeURL"`
	FiscalSign string `json:"fiscalSign"`
	ReceiptSeq string `json:"receiptSeq"`
	TerminalId string `json:"terminalId"`
	Amount     any    `json:"amount"`
	Card       any    `json:"card"`
	Cash       any    `json:"cash"`
	QrCodeUrl  string `json:"qrCodeUrl"`
}

// Info struct for success case
type Info struct {
	DateTime   string `json:"dateTime"`
	QrCodeURL  string `json:"qrCodeURL"`
	FiscalSign string `json:"fiscalSign"`
	ReceiptSeq string `json:"receiptSeq"`
	TerminalId string `json:"terminalId"`
}

// ErrorResponse for error case
type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// Add discount card structure
type AddDiscountCard struct {
	CustomerID string `json:"customer_id"`
	SaleID     string `json:"sale_id"`
	Barcode    string `json:"barcode"`
}

// region Online sale

// create online order request structure
// noor online order request
type OnlineOrderRequest struct {
	ShopId       string                  `json:"shop_id"`
	Products     []OnlineCartItemRequest `json:"product_ids"`
	ClientInfo   NoorClientInfo          `json:"client_info"`
	DeliveryTime string                  `json:"delivery_time"`
	Destination  Point                   `json:"destination"`
}

// noor online order product
type OnlineCartItemRequest struct {
	ProductId string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// noor online order client info
type NoorClientInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// Order create response struct
type OnlineOrderResponse struct {
	Message string `json:"message"`
	OrderID int    `json:"order_id"`
}

type ConfirmOnlineSaleRequest struct {
	SaleID             string `gorm:"sale_id" json:"sale_id"`
	CashBoxOperationID string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxID          string `gorm:"cashbox_id" json:"cashbox_id"`
	EmployeeID         string `gorm:"employee_id" json:"employee_id"`
}

// end region
