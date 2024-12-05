package domain

import "time"

// Sale structure
type Sale struct {
	ID            string     `gorm:"id" json:"id"`
	EmployeeID    string     `gorm:"employee_id" json:"employee_id"`
	CashBoxId     string     `gorm:"cash_box_id" json:"cash_box_id"`
	SaleNumber    string     `gorm:"sale_number" json:"sale_number"`
	TotalDiscount float64    `gorm:"total_discount" json:"total_discount"`
	TotalAmount   float64    `gorm:"total_amount" json:"total_amount"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
	Employee      *Employee  `gorm:"foreignKey:EmployeeID" json:"employee"`
	CashBox       *CashBox   `gorm:"foreignKey:CashBoxId" json:"cash_box"`
}

// SaleRequest structure for create
type SaleRequest struct {
	ID         string `gorm:"id" json:"-"`
	EmployeeID string `gorm:"employee_id" json:"employee_id"`
	CashBoxId  string `gorm:"cash_box_id" json:"cash_box_id"`
	SaleNumber string `gorm:"sale_number" json:"-"`
}

// SaleUpdateRequest structure for update
type SaleUpdateRequest struct {
	ID            string  `gorm:"id" json:"-"`
	EmployeeID    string  `gorm:"employee_id" json:"employee_id"`
	CashBoxId     string  `gorm:"cash_box_id" json:"cash_box_id"`
	SaleNumber    string  `gorm:"sale_number" json:"sale_number"`
	TotalDiscount float64 `gorm:"total_discount" json:"total_discount"`
	TotalAmount   float64 `gorm:"total_amount" json:"total_amount"`
}
