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
	ID             string       `gorm:"id" json:"id"`
	StoreID        string       `gorm:"store_id" json:"store_id"`
	PaymentTypeId  string       `gorm:"payment_type_id" json:"payment_type_id"`
	Name           string       `gorm:"name" json:"name"`
	Type           string       `gorm:"type" json:"type"`
	MerchantID     int          `gorm:"merchant_id" json:"merchant_id"`
	MerchantUserID int          `gorm:"merchant_user_id" json:"merchant_user_id"`
	ServiceID      int          `gorm:"service_id" json:"service_id"`
	CashboxId      string       `gorm:"cashbox_id" json:"cashbox_id"`
	SecretKey      string       `gorm:"secret_key" json:"secret_key"`
	IsActive       bool         `gorm:"is_active" json:"is_active"`
	CreatedAt      *time.Time   `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time   `gorm:"updated_at" json:"updated_at"`
	Store          *Store       `gorm:"foreignKey:StoreID" json:"store"`
	PaymentType    *PaymentType `gorm:"foreignKey:PaymentTypeId" json:"payment_type"`
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

// ChangePaymentTypeRequest structure
type ChangePaymentTypeRequest struct {
	SaleId          string `json:"sale_id"`
	FromPaymentType string `json:"from_payment_type"`
	ToPaymentType   string `json:"to_payment_type"`
}

// Alif payment request and response structures
// AlifPaymentRequest and AlifPaymentResponse are used for Alif payment integration
type AlifPaymentRequest struct {
	ID     string     `json:"id"`
	Amount int64      `json:"amount"`
	Method AlifMethod `json:"method"`
}
type AlifMethod struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}
