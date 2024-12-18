package domain

import "time"

// PaymentType structure
type PaymentType struct {
	ID          string     `gorm:"id" json:"id"`
	Name        string     `gorm:"name" json:"name"`
	Type        string     `gorm:"type" json:"type"`
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

// PaymentService structure
// for using save payment services data
type PaymentService struct {
	ID             string     `gorm:"id" json:"id"`
	CashBoxID      string     `gorm:"cash_box_id" json:"cash_box_id"`
	Name           string     `gorm:"name" json:"name"`
	MerchantID     int        `gorm:"merchant_id" json:"merchant_id"`
	MerchantUserID int        `gorm:"merchant_user_id" json:"merchant_user_id"`
	ServiceID      int        `gorm:"service_id" json:"service_id"`
	SecretKey      string     `gorm:"secret_key" json:"secret_key"`
	IsActive       bool       `gorm:"is_active" json:"is_active"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

// PaymentServiceRequest structure for create, update
type PaymentServiceRequest struct {
	ID             string `gorm:"id" json:"-"`
	CashBoxID      string `gorm:"cash_box_id" json:"cash_box_id"`
	Name           string `gorm:"name" json:"name"`
	MerchantID     int    `gorm:"merchant_id" json:"merchant_id"`
	MerchantUserID int    `gorm:"merchant_user_id" json:"merchant_user_id"`
	ServiceID      int    `gorm:"service_id" json:"service_id"`
	SecretKey      string `gorm:"secret_key" json:"secret_key"`
	IsActive       bool   `gorm:"is_active" json:"is_active"`
}

// SalePayment structure for sale payment
type SalePayment struct {
	ID               string       `gorm:"id" json:"id"`
	SaleID           string       `gorm:"sale_id" json:"sale_id"`
	PaymentServiceID string       `gorm:"payment_service_id" json:"payment_service_id"`
	PaymentTypeID    string       `gorm:"payment_type_id" json:"payment_type_id"`
	Amount           float64      `gorm:"amount" json:"amount"`
	TransactionID    string       `gorm:"transaction_id" json:"transaction_id,omitempty"`
	PaidAt           *time.Time   `gorm:"paid_at" json:"paid_at"`
	Status           string       `gorm:"status" json:"status"`
	CreatedAt        *time.Time   `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time   `gorm:"updated_at" json:"updated_at"`
	Sale             *Sale        `gorm:"foreignKey:SaleID" json:"sale"`
	PaymentType      *PaymentType `gorm:"foreignKey:PaymentTypeID" json:"payment_type"`
}

// SalePaymentRequest structure for create, update
type SalePaymentRequest struct {
	ID               string  `gorm:"id" json:"-"`
	SaleID           string  `gorm:"sale_id" json:"sale_id"`
	PaymentServiceID *string `gorm:"payment_service_id" json:"payment_service_id"`
	PaymentTypeID    string  `gorm:"payment_type_id" json:"payment_type_id"`
	Amount           float64 `gorm:"amount" json:"amount"`
	TransactionID    string  `gorm:"transaction_id" json:"transaction_id,omitempty"`
	PaidAt           string  `gorm:"paid_at" json:"paid_at"`
	Status           string  `gorm:"status" json:"status"`
}

// Transaction structure
type Transaction struct {
	ID               string     `gorm:"id" json:"id"`
	SalePaymentID    string     `gorm:"sale_payment_id" json:"sale_payment_id"`
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
