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
	ID               string       `gorm:"id" json:"id,omitempty"`
	SaleID           string       `gorm:"sale_id" json:"sale_id,omitempty"`
	PaymentServiceID string       `gorm:"payment_service_id" json:"payment_service_id,omitempty"`
	PaymentTypeID    string       `gorm:"payment_type_id" json:"payment_type_id,omitempty"`
	Amount           float64      `gorm:"amount" json:"amount"`
	NetAmount        float64      `gorm:"net_amount" json:"net_amount"`
	ExpenseAmount    float64      `gorm:"expense_amount" json:"expense_amount"`
	DifferenceAmount float64      `gorm:"difference_amount" json:"difference_amount"`
	TransactionID    string       `gorm:"transaction_id" json:"transaction_id,omitempty"`
	PaidAt           *time.Time   `gorm:"paid_at" json:"paid_at,omitempty"`
	Status           string       `gorm:"status" json:"status,omitempty"`
	CreatedAt        *time.Time   `gorm:"created_at" json:"created_at,omitempty"`
	UpdatedAt        *time.Time   `gorm:"updated_at" json:"updated_at,omitempty"`
	Sale             *Sale        `gorm:"foreignKey:SaleID" json:"sale,omitempty"`
	PaymentType      *PaymentType `gorm:"foreignKey:PaymentTypeID" json:"payment_type,omitempty"`
}

// SalePayment structure for close cashbox
type SalePaymentCloseCashBox struct {
	ID               string  `gorm:"id" json:"id,omitempty"`
	Name             string  `gorm:"name" json:"name"`
	Amount           float64 `gorm:"amount" json:"amount"`
	NetAmount        float64 `gorm:"net_amount" json:"net_amount"`
	ExpenseAmount    float64 `gorm:"expense_amount" json:"expense_amount"`
	DifferenceAmount float64 `gorm:"difference_amount" json:"difference_amount"`
}

// SalePaymentRequest structure for create, update
type SalePaymentRequest struct {
	ID               string  `gorm:"id" json:"-"`
	SaleID           string  `gorm:"sale_id" json:"sale_id"`
	CashBoxID        string  `gorm:"cash_box_id" json:"cash_box_id"`
	PaymentServiceID *string `gorm:"payment_service_id" json:"payment_service_id"`
	PaymentTypeID    string  `gorm:"payment_type_id" json:"payment_type_id"`
	Amount           float64 `gorm:"amount" json:"amount"`
	TransactionID    string  `gorm:"transaction_id" json:"transaction_id,omitempty"`
	PaidAt           string  `gorm:"paid_at" json:"paid_at"`
	Status           string  `gorm:"status" json:"status"`
}

// Sale Payment Total amounts struct
type SalePaymentTotalAmount struct {
	TotalAmount           float64 `json:"total_amount"`
	TotalNetAmount        float64 `json:"total_net_amount"`
	TotalExpenseAmount    float64 `json:"total_expense_amount"`
	TotalDifferenceAmount float64 `json:"total_difference_amount"`
}

// Sale Payment Update amount struct
type SalePaymentUpdateAmount struct {
	NetAmount float64 `json:"net_amount"`
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
