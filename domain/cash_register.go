package domain

import "time"

// CashRegister structure
type CashRegister struct {
	ID        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	StoreID   string     `gorm:"store_id" json:"store_id"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
	Store     *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

// Cash Register Request for create, update
type CashRegisterRequest struct {
	ID      string `gorm:"id" json:"-"`
	Name    string `gorm:"name" json:"name"`
	StoreID string `gorm:"store_id" json:"store_id"`
}

// Cash Register Session structure
type CashRegisterSession struct {
	ID             string        `gorm:"id" json:"id"`
	CashRegisterID string        `gorm:"cash_register_id" json:"cash_register_id"`
	StoreID        string        `gorm:"store_id" json:"store_id"`
	EmployeeID     string        `gorm:"employee_id" json:"employee_id"`
	Type           string        `gorm:"type" json:"type"`
	OpeningBalance float64       `gorm:"opening_balance" json:"opening_balance"`
	ClosingBalance float64       `gorm:"closing_balance" json:"closing_balance"`
	StartTime      *time.Time    `gorm:"start_time" json:"start_time"`
	EndTime        *time.Time    `gorm:"end_time" json:"end_time"`
	CreatedAt      *time.Time    `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time    `gorm:"updated_at" json:"updated_at"`
	Store          *Store        `gorm:"foreignKey:StoreID" json:"store"`
	CashRegister   *CashRegister `gorm:"foreignKey:CashRegisterID" json:"cash_register"`
	Employee       *Employee     `gorm:"foreignKey:EmployeeID" json:"employee"`
}

// Cash Register Session Request for create, update
type CashRegisterSessionRequest struct {
	ID             string     `gorm:"id" json:"-"`
	CashRegisterID string     `gorm:"cash_register_id" json:"cash_register_id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	Type           string     `gorm:"type" json:"type" example:"with_cash|without_cash"`
	OpeningBalance float64    `gorm:"opening_balance" json:"opening_balance"`
	ClosingBalance float64    `gorm:"closing_balance" json:"closing_balance"`
	StartTime      *time.Time `gorm:"start_time" json:"-"`
}
