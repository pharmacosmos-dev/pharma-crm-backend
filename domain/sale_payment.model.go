package domain

import "time"

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

// SalePaymentRequest structure for create, update
type SalePaymentRequest struct {
	ID                 string     `gorm:"id" json:"-"`
	SaleID             string     `gorm:"sale_id" json:"sale_id"`
	CashBoxOperationID string     `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	PaymentServiceID   *string    `gorm:"payment_service_id" json:"payment_service_id"`
	PaymentTypeID      string     `gorm:"payment_type_id" json:"payment_type_id"`
	Amount             float64    `gorm:"amount" json:"amount"`
	PaidAt             *time.Time `gorm:"paid_at" json:"paid_at"`
	Status             string     `gorm:"status" json:"status"`
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

// Sale Payment summary
type SalePaymentSummary struct {
	CashBoxOperationID string     `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	PaymentTypeID      string     `gorm:"payment_type_id" json:"payment_type_id"`
	TotalAmount        float64    `gorm:"total_amount" json:"total_amount"`
	TotalNetAmount     float64    `gorm:"total_net_amount" json:"total_net_amount"`
	TotalExpenseAmount float64    `gorm:"total_expense_amount" json:"total_expense_amount"`
	TotalDifference    float64    `gorm:"total_difference_amount" json:"total_difference_amount"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time `gorm:"updated_at" json:"updated_at"`
}

// SalePayment structure for close cashbox
type SalePaymentCloseCashBox struct {
	ID               string  `gorm:"id" json:"id"`
	Name             string  `gorm:"name" json:"name"`
	Amount           float64 `gorm:"amount" json:"amount"`
	NetAmount        float64 `gorm:"net_amount" json:"net_amount"`
	ExpenseAmount    float64 `gorm:"expense_amount" json:"expense_amount"`
	DifferenceAmount float64 `gorm:"difference_amount" json:"difference_amount"`
}
