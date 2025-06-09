package domain

import "time"

// PaymentType structure
type PaymentType struct {
	ID          string     `gorm:"id" json:"id"`
	Name        string     `gorm:"name" json:"name"`
	Type        string     `gorm:"type" json:"type"`
	IsActive    bool       `gorm:"is_active" json:"is_active"`
	Description string     `gorm:"description" json:"description"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
}

// PaymentTypeRequest structure for create, update
type PaymentTypeRequest struct {
	ID          string `gorm:"id" json:"-"`
	Name        string `gorm:"name" json:"name"`
	Type        string `gorm:"type" json:"type"`
	Description string `gorm:"description" json:"description"`
}

// CashboxPaymentType structure
type CashboxPaymentTypeResponse struct {
	ID            string     `gorm:"id" json:"id"`
	CashBoxId     string     `gorm:"cash_box_id" json:"cash_box_id"`
	PaymentTypeId string     `gorm:"payment_type_id" json:"payment_type_id"`
	IsActive      bool       `gorm:"is_active" json:"is_active"`
	Name          string     `gorm:"name" json:"name"`
	Type          string     `gorm:"type" json:"type"`
	Description   string     `gorm:"description" json:"description"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
}

// PaymentService structure
// for using save payment services data
type PaymentService struct {
	ID             string     `gorm:"id" json:"id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	Name           string     `gorm:"name" json:"name"`
	Type           string     `gorm:"type" json:"type"`
	MerchantID     int        `gorm:"merchant_id" json:"merchant_id"`
	MerchantUserID int        `gorm:"merchant_user_id" json:"merchant_user_id"`
	ServiceID      int        `gorm:"service_id" json:"service_id"`
	CashboxId      string     `gorm:"cashbox_id" json:"cashbox_id"`
	SecretKey      string     `gorm:"secret_key" json:"secret_key"`
	IsActive       bool       `gorm:"is_active" json:"is_active"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

// PaymentServiceRequest structure for create, update
type PaymentServiceRequest struct {
	ID             string     `gorm:"id" json:"-"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	PaymentTypeId  string     `gorm:"payment_type_id" json:"payment_type_id"`
	Name           string     `gorm:"name" json:"name" example:"Click|Payme|Uzum"`
	Type           string     `gorm:"type" json:"type" example:"click|payme|uzum"`
	MerchantID     int        `gorm:"merchant_id" json:"merchant_id"`
	MerchantUserID int        `gorm:"merchant_user_id" json:"merchant_user_id"`
	ServiceID      int        `gorm:"service_id" json:"service_id"`
	CashboxId      string     `gorm:"cashbox_id" json:"cashbox_id"`
	SecretKey      string     `gorm:"secret_key" json:"secret_key"`
	IsActive       bool       `gorm:"is_active" json:"is_active"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"-"`
}

// Transaction structure
type Transaction struct {
	ID               string     `gorm:"id" json:"id"`
	PaymentServiceID string     `gorm:"payment_service_id" json:"payment_service_id"`
	TransactionID    string     `gorm:"transaction_id" json:"transaction_id"`
	Status           string     `gorm:"status" json:"status"`
	ResponseData     string     `gorm:"response_data" json:"response_data"`
	CreatedAt        *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time `gorm:"updated_at" json:"updated_at"`
}

// TransactionRequest structure for create, update
type TransactionRequest struct {
	ID               string `gorm:"id" json:"-"`
	SalePaymentID    string `gorm:"sale_payment_id" json:"sale_payment_id"`
	PaymentServiceID string `gorm:"payment_service_id" json:"payment_service_id"`
	TransactionID    string `gorm:"transaction_id" json:"transaction_id"`
	Status           string `gorm:"status" json:"status"`
	ResponseData     string `gorm:"response_data" json:"response_data"`
}

// Payment request structure
type PaymentRequest struct {
	ID              *string    `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	RequestId       int64      `gorm:"request_id" json:"request_id"`
	Method          string     `gorm:"method" json:"method"`
	Payload         []byte     `gorm:"payload" json:"payload"`
	Response        []byte     `gorm:"response" json:"response"`
	TransactionID   string     `gorm:"transaction_id" json:"transaction_id"`
	PaymentProvider string     `gorm:"payment_provider" json:"payment_provider"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Click Pass request body
type ClickPassRequest struct {
	ServiceID     int     `json:"service_id"`
	OtpData       string  `json:"otp_data"`
	Amount        float64 `json:"amount"`
	CashboxCode   string  `json:"cashbox_code"`
	TransactionID string  `json:"transaction_id"`
}

// Click Pass response body
type ClickPassResponse struct {
	ErrorCode      int    `json:"error_code"`
	ErrorNote      string `json:"error_note"`
	PaymentID      string `json:"payment_id"`
	PaymentStatus  int    `json:"payment_status"`
	ConfirmMode    bool   `json:"confirm_mode"`
	CardType       string `json:"card_type"`
	ProcessingType string `json:"processing_type"`
	CardNumber     string `json:"card_number"`
	PhoneNumber    string `json:"phone_number"`
}

// Type Uzum request body
type UzumRequest struct {
	OrderId       string  `json:"order_id"`
	TransactionID string  `json:"transaction_id"`
	ServiceID     int     `json:"service_id"`
	Amount        float64 `json:"amount"`
	CashboxCode   string  `json:"cashbox_code"`
	OtpData       string  `json:"otp_data"`
}

// Type Uzum response body
type UzumResponse struct {
	PaymentID         string `json:"payment_id"`
	PaymentStatus     int    `json:"payment_status"`
	ErrorCode         int    `json:"error_code"`
	OperationTime     string `json:"operation_time"`
	ClientPhoneNumber string `json:"client_phone_number"`
}

// PaymeGo receipt create body
type PaymeGoReceiptCreate struct {
	Id     int64         `json:"id"`
	Method string        `json:"method"`
	Params PaymeGoParams `json:"params"`
}

type PaymeGoParams struct {
	Amount  float64 `json:"amount"`
	Account struct {
		OrderId string `json:"order_id"`
	} `json:"account"`
	Detail *PaymeGoDetail `json:"detail"`
}

type PaymeGoDetail struct {
	ReceiptType int `json:"receipt_type"`
	Shipping    *struct {
		Title string  `json:"title"`
		Price float64 `json:"price"`
	}
	Items []PaymeGoItem `json:"items"`
}

type PaymeGoItem struct {
	Discount    float64 `json:"discount"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Count       int     `json:"count"`
	Code        string  `json:"code"`
	Units       string  `json:"units"`
	VatPercent  int     `json:"vat_percent"`
	PackageCode string  `json:"package_code"`
}

// PaymeGo receipt pay body
type PaymeGoReceiptPay struct {
	Id     int64            `json:"id"`
	Method string           `json:"method"`
	Params PaymeGoPayParams `json:"params"`
}

type PaymeGoPayParams struct {
	Id    string `json:"id"`
	Token string `json:"token"`
}

// PaymeGo receipt cancel body
type PaymeGoReceiptCancel struct {
	Id     int64               `json:"id"`
	Method string              `json:"method"`
	Params PaymeGoCancelParams `json:"params"`
}

type PaymeGoCancelParams struct {
	Id string `json:"id"`
}

// PaymeGo response structures for proper handling
type PaymeGoResponse struct {
	JsonRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Result  PaymeGoResult `json:"result,omitempty"`
	Error   PaymeGoError  `json:"error,omitempty"`
}

type PaymeGoResult struct {
	Receipt PaymeGoReceipt `json:"receipt"`
}

type PaymeGoReceipt struct {
	ID           string    `json:"_id"`
	CreateTime   int64     `json:"create_time"`
	PayTime      int64     `json:"pay_time"`
	CancelTime   int64     `json:"cancel_time"`
	State        int       `json:"state"`
	Type         int       `json:"type"`
	External     bool      `json:"external"`
	Operation    int       `json:"operation"`
	Category     any       `json:"category"`
	Error        any       `json:"error"`
	Description  string    `json:"description"`
	Detail       *Detail   `json:"detail,omitempty"`
	Amount       int       `json:"amount"`
	Currency     *int      `json:"currency,omitempty"`
	Commission   int       `json:"commission"`
	Account      []Account `json:"account"`
	Card         any       `json:"card,omitempty"`
	Merchant     Merchant  `json:"merchant"`
	Meta         *Meta     `json:"meta,omitempty"`
	ProcessingID *int      `json:"processing_id,omitempty"`
}

type PaymeGoError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type Detail struct {
	Discount *PaymeGoDetailDiscount `json:"discount,omitempty"`
	Shipping *PaymeGoDetailShipping `json:"shipping,omitempty"`
	Items    []PaymeGoItem          `json:"items,omitempty"`
}

type PaymeGoDetailDiscount struct {
	Title string `json:"title"`
	Price int    `json:"price"`
}

type PaymeGoDetailShipping struct {
	Title string `json:"title"`
	Price int    `json:"price"`
}

type Account struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Value string `json:"value"`
	Main  *bool  `json:"main,omitempty"`
}

type Card struct {
	Number string `json:"number"`
	Expire string `json:"expire"`
}

type Merchant struct {
	ID           string  `json:"_id"`
	Name         string  `json:"name"`
	Organization string  `json:"organization"`
	Address      string  `json:"address"`
	BusinessID   *string `json:"business_id,omitempty"`
	Epos         Epos    `json:"epos"`
	Date         int64   `json:"date"`
	Logo         any     `json:"logo,omitempty"`
	Type         string  `json:"type"`
	Terms        any     `json:"terms,omitempty"`
	Payer        *Payer  `json:"payer,omitempty"`
}

type Epos struct {
	MerchantID string `json:"merchantId"`
	TerminalID string `json:"terminalId"`
}

type Payer struct {
	Phone string `json:"phone"`
}

type Meta struct {
	Source string `json:"source"`
	Owner  string `json:"owner"`
}

// Payme Go Set fiscal data
type FiscalDataRequest struct {
	Id     int64            `json:"id"`
	Method string           `json:"method"`
	Params FiscalDataParams `json:"params"`
}

// Fiscal data params
type FiscalDataParams struct {
	Id         string     `json:"id"`
	FiscalData FiscalData `json:"fiscal_data"`
}

// Fiscal data structure
type FiscalData struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	TerminalId string `json:"terminal_id"`
	ReceiptId  int    `json:"receipt_id"`
	Date       string `json:"date"`
	FiscalSign string `json:"fiscal_sign"`
	QrCodeUrl  string `json:"qr_code_url"`
}

// Payme GO error response
type PaymeGoErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data"`
}
