package domain

import "time"

// CashBox structure
type CashBox struct {
	ID        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	StoreID   string     `gorm:"store_id" json:"store_id"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
	Store     *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

// Cash Register Request for create, update
type CashBoxRequest struct {
	ID      string `gorm:"id" json:"-"`
	Name    string `gorm:"name" json:"name"`
	StoreID string `gorm:"store_id" json:"store_id"`
}

// Cash Box Session structure
type CashBoxSession struct {
	ID             string     `gorm:"id" json:"id"`
	CashBoxID      string     `gorm:"cash_box_id" json:"cash_box_id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	Type           string     `gorm:"type" json:"type"`
	OpeningBalance float64    `gorm:"opening_balance" json:"opening_balance"`
	ClosingBalance float64    `gorm:"closing_balance" json:"closing_balance"`
	StartTime      *time.Time `gorm:"start_time" json:"start_time"`
	EndTime        *time.Time `gorm:"end_time" json:"end_time"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Store          *Store     `gorm:"foreignKey:StoreID" json:"store"`
	CashBox        *CashBox   `gorm:"foreignKey:CashBoxID" json:"cash_box"`
	Employee       *Employee  `gorm:"foreignKey:EmployeeID" json:"employee"`
}

// Cash Box Session Request for create, update
type CashBoxSessionRequest struct {
	ID             string     `gorm:"id" json:"-"`
	CashBoxID      string     `gorm:"cash_box_id" json:"cash_box_id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	Type           string     `gorm:"type" json:"type" example:"with_cash|without_cash"`
	OpeningBalance float64    `gorm:"opening_balance" json:"opening_balance"`
	ClosingBalance float64    `gorm:"closing_balance" json:"closing_balance"`
	StartTime      *time.Time `gorm:"start_time" json:"-"`
}
