package domain

import (
	"time"
)

// Sale structure
type Sale struct {
	Id                 string         `gorm:"id" json:"id"`
	DeviceId           string         `gorm:"device_id" json:"device_id"`
	StoreId            string         `gorm:"store_id" json:"store_id"`
	EmployeeId         string         `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string         `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string         `gorm:"cashbox_id" json:"cashbox_id"`
	CustomerId         string         `gorm:"customer_id" json:"customer_id"`
	ParentId           string         `gorm:"parent_id" json:"parent_id"`
	SaleNumber         int            `gorm:"sale_number" json:"sale_number"`
	TotalDiscount      float64        `gorm:"total_discount" json:"total_discount"`
	TotalAmount        float64        `gorm:"total_amount" json:"total_amount"`
	ReturnedAmount     float64        `gorm:"returned_amount" json:"returned_amount"`
	ProductCount       int            `gorm:"product_count" json:"product_count"`
	Type               string         `gorm:"type" json:"type"`
	SaleType           string         `gorm:"sale_type" json:"sale_type"`
	Status             string         `gorm:"status" json:"status"`
	Stage              int            `gorm:"stage" json:"stage"`
	OnlineStatus       int            `gorm:"online_status" json:"online_status"`
	ServiceType        string         `gorm:"service_type" json:"service_type"`
	IsDelivered        bool           `gorm:"is_delivered" json:"is_delivered"`
	IsReturned         bool           `gorm:"is_returned" json:"is_returned"`
	Cash               float64        `gorm:"cash" json:"cash"`
	Click              float64        `gorm:"click" json:"click"`
	Humo               float64        `gorm:"humo" json:"humo"`
	Uzcard             float64        `gorm:"uzcard" json:"uzcard"`
	Payme              float64        `gorm:"payme" json:"payme"`
	Alif               float64        `gorm:"alif" json:"alif"`
	Uzum               float64        `gorm:"uzum" json:"uzum"`
	UzumTezKor        float64         `gorm:"uzum_tez_kor" json:"uzum_tez_kor"`
	LoyaltyCard        float64        `gorm:"loyalty_card" json:"loyalty_card"`
	IsPaid             bool           `gorm:"is_paid" json:"is_paid"`
	IsCorporate        bool           `gorm:"is_corporate" json:"is_corporate"`
	OtpCode            string         `gorm:"otp_code" json:"otp_code"`
	PaymentReceiptId   string         `gorm:"payment_receipt_id" json:"payment_receipt_id"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	CompletedAt        *time.Time     `gorm:"completed_at" json:"completed_at"`
	Employee           *Employee      `gorm:"-" json:"employee"`
	Customer           *Customer      `gorm:"-" json:"customer"`
	SalePayments       []*SalePayment `gorm:"foreignKey:SaleID" json:"sale_payments"`
	CartItems          []*CartItem    `gorm:"foreignKey:SaleId" json:"cart_items"`
	CashBack           float64        `gorm:"cash_back" json:"cash_back"`
	VendorOrderId      string         `gorm:"vendor_order_id" json:"vendor_order_id"`
	ClientComment      string         `gorm:"client_comment" json:"client_comment"`
}

type SaleQueryParams struct {
	StoreId         string      `form:"store_id"`
	StoreIds        []string    `form:"store_ids"`
	CompanyId       string      `form:"company_id"`
	CompanyIds      []string    `form:"-"`
	Search          string      `form:"search"`
	StartDate       *CustomTime `form:"start_date" validate:"date"`
	EndDate         *CustomTime `form:"end_date" validate:"date"`
	VendorId        string      `form:"vendor_id"`
	PaymentTypeId   string      `form:"payment_type_id"`
	CashboxId       string      `form:"cashbox_id"`
	Limit           int         `form:"limit"`
	Offset          int         `form:"offset"`
	TotalAmountTo   float64     `form:"total_amount_to"`
	TotalAmountFrom float64     `form:"total_amount_from"`
	Status          string      `form:"status"`
	SaleType        string      `form:"sale_type"`
	Cash            bool        `form:"cash"`
	Humo            bool        `form:"humo"`
	Uzcard          bool        `form:"uzcard"`
	Click           bool        `form:"click"`
	Payme           bool        `form:"payme"`
	Alif            bool        `form:"alif"`
	Uzum            bool        `form:"uzum"`
	UzumTezKor      bool        `form:"uzum_tez_kor"`
	IsCorporate     bool        `form:"is_corporate"`
	Stage           int         `form:"stage"`
	OnlineStatus    *int        `form:"online_status"`
	ProductId       string      `form:"product_id"`
	CustomerId      string      `form:"customer_id"`
}

// SaleRequest structure for create
type SaleRequest struct {
	Id                 string  `gorm:"id" json:"id"`
	EmployeeId         string  `gorm:"employee_id" json:"employee_id"`
	StoreId            string  `gorm:"store_id" json:"store_id"`
	CashBoxOperationId string  `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string  `gorm:"cashbox_id" json:"cashbox_id"`
	ServiceType        *string `gorm:"service_type" json:"service_type"`
}

// SaleReturnRequest structure for create
type SaleReturnRequest struct {
	SaleId             string     `gorm:"sale_id" json:"sale_id"`
	EmployeeId         string     `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string     `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string     `gorm:"cashbox_id" json:"cashbox_id"`
	SaleType           string     `gorm:"sale_type" json:"sale_type"`
	Items              []SaleItem `gorm:"-" json:"sale_items"`
}

type OnlineSaleCreate struct {
	Id            string                  `gorm:"id" json:"id"`
	StoreId       string                  `gorm:"store_id" json:"store_id"`
	VendorOrderId string                  `gorm:"vendor_order_id" json:"vendor_order_id"`
	ServiceType   string                  `gorm:"service_type" json:"service_type"`
	CustomerId    string                  `gorm:"customer_id" json:"customer_id"`
	ClientComment string                  `gorm:"client_comment" json:"client_comment"`
	Items         []CartItemOnlineRequest `gorm:"-" json:"items"`
}

// SaleItem structure for create return

type SaleItem struct {
	SaleId         string `gorm:"sale_id" json:"-"`
	StoreProductId string `gorm:"store_product_id" json:"store_product_id"`
	Quantity       int    `gorm:"quantity" json:"quantity"`
	UnitQuantity   int    `gorm:"unit_quantity" json:"unit_quantity"`
}

type SaleResponse struct {
	Id              string     `gorm:"id" json:"id"`
	DisplayId       int        `gorm:"display_id" json:"display_id"`
	VendorOrderId   string     `gorm:"vendor_order_id" json:"vendor_order_id"`
	ParentId        string     `gorm:"parent_id" json:"parent_id"`
	SaleNumber      int        `gorm:"sale_number" json:"sale_number"`
	TotalDiscount   float64    `gorm:"total_discount" json:"total_discount"`
	TotalAmount     float64    `gorm:"total_amount" json:"total_amount"`
	VatSum          float64    `gorm:"vat_sum" json:"vat_sum"`
	ReturnedAmount  float64    `gorm:"returned_amount" json:"returned_amount"`
	Cash            float64    `gorm:"cash" json:"cash"`
	Uzcard          float64    `gorm:"uzcard" json:"uzcard"`
	Humo            float64    `gorm:"humo" json:"humo"`
	Click           float64    `gorm:"click" json:"click"`
	Payme           float64    `gorm:"payme" json:"payme"`
	Alif            float64    `gorm:"alif" json:"alif"`
	Uzum            float64    `gorm:"uzum" json:"uzum"`
	UzumTezKor      float64    `gorm:"uzum_tez_kor" json:"uzum_tez_kor"`
	LoyaltyCard     float64    `gorm:"loyalty_card" json:"loyalty_card"`
	ProductCount    float64    `gorm:"product_count" json:"product_count"`
	Status          string     `gorm:"status" json:"status"`
	Stage           int        `gorm:"stage" json:"stage"`
	OnlineStatus    int        `gorm:"online_status" json:"online_status"`
	Type            string     `gorm:"type" json:"type"`
	SaleType        string     `gorm:"sale_type" json:"sale_type"`
	ServiceType     string     `gorm:"service_type" json:"service_type"`
	DiscountBarcode string     `gorm:"discount_barcode" json:"discount_barcode"`
	IsDelivered     bool       `gorm:"is_delivered" json:"is_delivered"`
	IsReturned      bool       `gorm:"is_returned" json:"is_returned"`
	FiscalSign      string     `gorm:"fiscal_sign" json:"fiscal_sign"`
	CheckUrl        string     `gorm:"check_url" json:"check_url"`
	OtpCode         string     `gorm:"otp_code" json:"otp_code"`
	IsSentToTax     string     `gorm:"is_sent_to_tax" json:"is_sent_to_tax"`
	TaxFree         bool       `gorm:"tax_free" json:"tax_free"`
	IsCorporate     bool       `gorm:"is_corporate" json:"is_corporate"`
	ClientComment   string     `gorm:"client_comment" json:"client_comment"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
	CompletedAt     *time.Time `gorm:"completed_at" json:"completed_at"`
	CashBack        float64    `gorm:"cash_back" json:"cash_back"`

	CashBoxOperationId string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	StoreName          string `gorm:"store_name" json:"store_name"`
	CashBoxName        string `gorm:"cash_box_name" json:"cash_box_name"`

	EmployeeId string `gorm:"employee_id" json:"employee_id"`
	FullName   string `gorm:"full_name" json:"full_name"`
	Phone      string `gorm:"phone" json:"phone"`

	CustomerId    string           `gorm:"customer_id" json:"customer_id"`
	CustomerName  *string          `gorm:"customer_name" json:"customer_name"`
	CustomerPhone *string          `gorm:"customer_phone" json:"customer_phone"`
	Employee      *EmployeeForSale `gorm:"-" json:"employee"`
	Customer      *CustomerForSale `gorm:"-" json:"customer"`
	SalePayments  []*SalePayment   `gorm:"-" json:"sale_payments"`
	CartItems     []*CartItem      `gorm:"-" json:"cart_items"`
	Product       []ProductRes     `gorm:"-" json:"products"`
	EposResponse  *EposResponse    `gorm:"-" json:"epos_response"`
}

// SaleUpdateRequest structure for update
type SaleUpdateRequest struct {
	ID            string  `gorm:"id" json:"-"`
	TotalDiscount float64 `gorm:"total_discount" json:"total_discount"`
	TotalAmount   float64 `gorm:"total_amount" json:"total_amount"`
}

// FinalSale structure
type FinalSale struct {
	StoreId            string             `gorm:"store_id" json:"store_id"`
	SaleId             string             `gorm:"sale_id" json:"sale_id"`
	PrescriptionId     string             `gorm:"prescription_id" json:"prescription_id"`
	CustomerId         *string            `gorm:"customer_id" json:"customer_id"`
	CashBoxOperationId string             `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	TotalAmount        float64            `gorm:"total_amount" json:"total_amount"`
	ServiceType        string             `gorm:"service_type" json:"service_type"`
	TaxFree            bool               `gorm:"tax_free" json:"tax_free"`
	Cash               float64            `gorm:"cash" json:"cash"`
	Humo               float64            `gorm:"humo" json:"humo"`
	Uzcard             float64            `gorm:"uzcard" json:"uzcard"`
	Click              float64            `gorm:"click" json:"click"`
	Payme              float64            `gorm:"payme" json:"payme"`
	Alif               float64            `gorm:"alif" json:"alif"`
	LoyaltyCard        float64            `gorm:"loyalty_card" json:"loyalty_card"`
	Uzum               float64            `gorm:"uzum" json:"uzum"`
	UzumTezKor		   float64            `gorm:"uzum_tez_kor" json:"uzum_tez_kor"`
	ReturnAmount       float64            `gorm:"return_amount" json:"return_amount"`
	LoyaltyCardBarcode string             `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	OtpCode            string             `gorm:"otp_code" json:"otp_code"`
	IsCorporate        bool               `gorm:"is_corporate" json:"is_corporate"`
	PaymentTypes       []FinalPaymentType `json:"payment_types"`
	MarkingData        []MarkingData      `json:"marking_data"`
	EposData           [][]EposItem       `json:"epos_data"`
}

type MarkingData struct {
	Id           string   `json:"id" gorm:"id"`
	DmedId       int      `json:"dmed_id"`
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
	AppType       string  `gorm:"app_type" json:"app_type" example:"click|payme|uzum|uzum_tez_kor"`
	Type          string  `gorm:"type" json:"type" example:"card|cash|app"`
	OtpData       string  `gorm:"otp_data" json:"otp_data"`
}

// Total amount struct
type SaleTotalAmount struct {
	TotalAmount      float64 `gorm:"total_amount" json:"total_amount"`
	CashAmount       float64 `gorm:"cash_amount" json:"cash_amount"`
	HumoAmount       float64 `gorm:"humo_amount" json:"humo_amount"`
	UzcardAmount     float64 `gorm:"uzcard_amount" json:"uzcard_amount"`
	VisaAmount       float64 `gorm:"visa_amount" json:"visa_amount"`
	ClickAmount      float64 `gorm:"click_amount" json:"click_amount"`
	PaymeAmount      float64 `gorm:"payme_amount" json:"payme_amount"`
	UzumAmount       float64 `gorm:"uzum_amount" json:"uzum_amount"`
	UzumTezKorAmount float64 `gorm:"uzum_tez_kor_amount" json:"uzum_tez_kor_amount"`
	BalanceAmount    float64 `gorm:"balance_amount" json:"balance_amount"`
}

// SaleStats structure
type SaleStats struct {
	TotalTransactionSum float64 `gorm:"total_transaction_sum" json:"total_transaction_sum"`
	TotalTransaction    int     `gorm:"total_transaction" json:"total_transaction"`
	TotalReturnalsSum   float64 `gorm:"total_returnals_sum" json:"total_returnals_sum"`
	TotalReturnedCount  int     `gorm:"total_returned_count" json:"total_returned_count"`
	TotalDiscountSum    float64 `gorm:"total_discount_sum" json:"total_discount_sum"`
	TotalDiscountCount  int     `gorm:"total_discount_count" json:"total_discount_count"`
	TotalLoyaltyCardSum float64 `gorm:"total_loyalty_card_sum" json:"total_loyalty_card_sum"`
	TotalLoyaltyCardCount int   `gorm:"total_loyalty_card_count" json:"total_loyalty_card_count"`
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
	TotalUzumSum        float64 `gorm:"total_uzum_sum" json:"total_uzum_sum"`
	TotalUzumCount      int     `gorm:"total_uzum_count" json:"total_uzum_count"`
	TotalUzumTezKorSum  float64 `gorm:"total_uzum_tez_kor_sum" json:"total_uzum_tez_kor_sum"`
	TotalUzumTezKorCount int     `gorm:"total_uzum_tez_kor_count" json:"total_uzum_tez_kor_count"`
	TotalCashbackSum    float64 `gorm:"total_cashback_sum" json:"total_cashback_sum"`
	TotalCashbackCount  float64 `gorm:"total_cashback_count" json:"total_cashback_count"`
	TotalProductCount   int64   `gorm:"total_product_count" json:"total_product_count"`
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
	CustomerId string `json:"customer_id"`
	SaleId     string `json:"sale_id"`
	Barcode    string `json:"barcode"`
	Percent    int    `json:"percent"`
}

// region Online sale

// create online order request structure
// noor online order request
type OnlineOrderRequest struct {
	ShopId        string                  `json:"shop_id" binding:"required,uuid"`
	Products      []OnlineCartItemRequest `json:"product_ids" binding:"required,dive"`
	ClientInfo    NoorClientInfo          `json:"client_info" binding:"required"`
	DeliveryTime  string                  `json:"delivery_time"`
	Destination   Point                   `json:"destination" binding:"required"`
	ClientComment string                  `json:"client_comment"`
}

// noor online order product
type OnlineCartItemRequest struct {
	ProductId string `json:"productId" binding:"required,uuid"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

// noor online order client info
type NoorClientInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// Order create response struct
type OnlineOrderResponse struct {
	Message string `json:"message"`
	OrderId int    `json:"order_id"`
}

type ConfirmOnlineSaleRequest struct {
	SaleId             string `gorm:"sale_id" json:"sale_id"`
	CashBoxOperationId string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CashboxId          string `gorm:"cashbox_id" json:"cashbox_id"`
	EmployeeId         string `gorm:"employee_id" json:"employee_id"`
}

type OnlineSaleDto struct {
	Id            string     `gorm:"id" json:"id"`
	EmployeeId    string     `gorm:"employee_id" json:"employee_id"`
	StoreId       string     `gorm:"store_id" json:"store_id"`
	SaleNumber    int        `gorm:"sale_number" json:"sale_number"`
	VendorOrderId string     `gorm:"vendor_order_id" json:"vendor_order_id"`
	OnlineStatus  int        `gorm:"online_status" json:"online_status"`
	Stage         int        `gorm:"stage" json:"stage"`
	Type          string     `gorm:"type" json:"type"`
	SaleType      string     `gorm:"sale_type" json:"sale_type"`
	ServiceType   string     `gorm:"service_type" json:"service_type"`
	TotalAmount   float64    `gorm:"total_amount" json:"total_amount"`
	ProductCount  int        `gorm:"product_count" json:"product_count"`
	ClientComment string     `gorm:"client_comment" json:"client_comment"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
}

// end region

type SaleDifference struct {
	Difference      float64 `gorm:"column:difference"`
	TotalDifference float64 `gorm:"column:total_difference"`
}

type PendingSaleResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type BarcodeResponse struct {
	Id          string `gorm:"column:id" json:"id"`
	CartItemId  string `gorm:"column:cart_item_id" json:"cart_item_id"`
	Barcode     string `gorm:"column:barcode" json:"barcode"`
	ClassCode   string `gorm:"column:mxik" json:"classCode"`
	PackageCode string `gorm:"column:unit_code" json:"packageCode"`
}

type MarkingItemsResponse struct {
	Items []BarcodeResponse `json:"items"`
}

type OnlineOrderDto struct {
	Id            string                          `json:"id"`
	SaleNumber    int                             `json:"sale_number"`
	VendorOrderId string                          `json:"vendor_order_id"`
	TotalAmount   float64                         `json:"total_amount"`
	TotalDiscount float64                         `json:"total_discount"`
	PaymentType   string                          `json:"payment_type"`
	Stage         int                             `json:"stage"`
	OnlineStatus  int                             `json:"online_status"`
	ServiceType   string                          `json:"service_type"`
	IsPaid        bool                            `json:"is_paid"`
	CreatedAt     *time.Time                      `json:"created_at"`
	CompletedAt   *time.Time                      `json:"completed_at"`
	Cusomer       NullStruct[OnlineOrderCustomer] `json:"customer"`
	Store         NullStruct[OnlineOrderStore]    `json:"store"`
}

type OnlineOrderCustomer struct {
	Id       string `json:"id"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type OnlineOrderStore struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type UpdateOnlineSale struct {
	OnlineStatus int `json:"online_status" binding:"required"`
}

type OnlineOrderStatistic struct {
	TotalCount     int64   `json:"total_count"`
	TotalAmount    float64 `json:"total_amount"`
	WaitingCount   int64   `json:"waiting_count"`
	CompletedCount int64   `json:"completed_count"`
	CancelledCount int64   `json:"cancelled_count"`
}
