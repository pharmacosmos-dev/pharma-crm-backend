package domain

import "time"

// Sale structure
type Sale struct {
	ID                 string         `gorm:"id" json:"id"`
	EmployeeID         string         `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string         `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	CustomerID         string         `gorm:"customer_id" json:"customer_id"`
	SaleNumber         int            `gorm:"sale_number" json:"sale_number"`
	TotalDiscount      float64        `gorm:"total_discount" json:"total_discount"`
	TotalAmount        float64        `gorm:"total_amount" json:"total_amount"`
	ProductCount       int            `gorm:"product_count" json:"product_count"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	Employee           *Employee      `gorm:"foreignKey:EmployeeID" json:"employee"`
	Customer           *Customer      `gorm:"foreignKey:CustomerID" json:"customer"`
	SalePayments       []*SalePayment `gorm:"foreignKey:SaleID" json:"sale_payments"`
	CartItems          []*CartItem    `gorm:"foreignKey:SaleId" json:"cart_items"`
}

// SaleRequest structure for create
type SaleRequest struct {
	ID                 string `gorm:"id" json:"-"`
	EmployeeID         string `gorm:"employee_id" json:"employee_id"`
	CashBoxOperationId string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
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
	CashBoxOperationId string             `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	TotalAmount        float64            `gorm:"total_amount" json:"total_amount"`
	PaymentTypes       []FinalPaymentType `json:"payment_types"`
}

// FinalPaymentType structure
type FinalPaymentType struct {
	PaymentTypeID string  `gorm:"payment_type_id" json:"payment_type_id"`
	Amount        float64 `gorm:"amount" json:"amount"`
	AppType       string  `gorm:"app_type" json:"app_type" example:"click|payme|uzum"`
	Type          string  `gorm:"type" json:"type" example:"card|cash|app"`
	OtpData       string  `gorm:"otp_data" json:"otp_data"`
}
