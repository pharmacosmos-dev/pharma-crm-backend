package domain

import "time"

// FinanceOperation structure
type FinanceOperation struct {
	Id                int64      `gorm:"id" json:"id"`
	CashboxId         string     `gorm:"cashbox_id" json:"cashbox_id"`
	PaymentTypeId     string     `gorm:"payment_type_id" json:"payment_type_id"`
	FinanceCategoryId int64      `gorm:"finance_category_id" json:"finance_category_id"`
	EmployeeId        string     `gorm:"employee_id" json:"employee_id"`
	OperationType     string     `gorm:"operation_type" json:"operation_type"`
	Comment           string     `gorm:"comment" json:"comment"`
	Amount            float64    `gorm:"amount" json:"amount"`
	Status            string     `gorm:"status" json:"status"`
	ReportType        string     `gorm:"report_type" json:"report_type"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
}

// FinanceOperationRequest structure
type FinanceOperationRequest struct {
	CashboxId         string  `gorm:"cashbox_id" json:"cashbox_id"`
	FinanceCategoryId *int64  `gorm:"finance_category_id" json:"finance_category_id"`
	PaymentTypeId     string  `gorm:"payment_type_id" json:"payment_type_id"`
	EmployeeId        string  `gorm:"employee_id" json:"employee_id"`
	OperationType     string  `gorm:"operation_type" json:"operation_type" example:"income||expense||collection"`
	Comment           string  `gorm:"comment" json:"comment"`
	Amount            float64 `gorm:"amount" json:"amount"`
	Status            string  `gorm:"status" json:"status"`
	ReportType        string  `gorm:"report_type" json:"report_type" example:"ДДC||ПиУ||СКВОЗНОЙ"`
}

// Finance operation total stats
type FinanceOperationStats struct {
	TotalCashIncome      float64 `json:"total_cash_income"`
	TotalCashlessIncome  float64 `json:"total_cashless_income"`
	TotalCashExpense     float64 `json:"total_cash_expense"`
	TotalCashlessExpense float64 `json:"total_cashless_expense"`
}

// query param structure
type FinanceQueryParams struct {
	Limit             int     `form:"limit"`
	Offset            int     `form:"offset"`
	Search            string  `form:"search"`
	CashboxId         string  `form:"cashbox_id"`
	EmployeeId        string  `form:"employee_id"`
	FinanceCategoryId int64   `form:"finance_category_id"`
	ReportType        string  `form:"report_type"`
	StartDate         string  `form:"start_date"`
	EndDate           string  `form:"end_date"`
	FromAmount        float64 `form:"from_amount"`
	ToAmount          float64 `form:"to_amount"`
}
