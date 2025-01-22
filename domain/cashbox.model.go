package domain

import "time"

// CashBox structure
type CashBox struct {
	ID        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	StoreID   string     `gorm:"store_id" json:"store_id"`
	IsOpen    bool       `gorm:"is_open" json:"is_open"`
	IsEnable  bool       `gorm:"is_enable" json:"is_enable"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
	Store     *Store     `gorm:"foreignKey:StoreID" json:"store"`
}

// Cash Register Request for create, update
type CashBoxRequest struct {
	ID           string               `gorm:"id" json:"-"`
	Name         string               `gorm:"name" json:"name"`
	StoreID      string               `gorm:"store_id" json:"store_id"`
	IsOpen       bool                 `gorm:"is_open" json:"-"`
	IsEnable     bool                 `gorm:"is_enable" json:"is_enable"`
	PaymentTypes []CashboxPaymentType `gorm:"-" json:"payment_types"`
}

// Cash Box Session structure
type CashboxOperation struct {
	ID             string     `gorm:"id" json:"id"`
	CashBoxID      string     `gorm:"cash_box_id" json:"cash_box_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	CashAmount     float64    `gorm:"cash_amount" json:"cash_amount"`
	CashlessAmount float64    `gorm:"cashless_amount" json:"cashless_amount"`
	ClosedAmount   float64    `gorm:"closed_amount" json:"closed_amount"`
	OpenedAmount   float64    `gorm:"opened_amount" json:"opened_amount"`
	IsOpen         bool       `gorm:"is_open" json:"is_open"`
	Description    string     `gorm:"description" json:"description"`
	StartTime      *time.Time `gorm:"start_time" json:"start_time"`
	EndTime        *time.Time `gorm:"end_time" json:"end_time"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	CashBox        *CashBox   `gorm:"foreignKey:CashBoxID" json:"cash_box"`
	Employee       *Employee  `gorm:"foreignKey:EmployeeID" json:"employee"`
}

// Cash Box Session Request for create, update
type CashboxOperationRequest struct {
	ID           string     `gorm:"id" json:"-"`
	CashBoxID    string     `gorm:"cash_box_id" json:"cash_box_id"`
	EmployeeID   string     `gorm:"employee_id" json:"-"`
	OpenedAmount float64    `gorm:"opened_amount" json:"opened_amount"`
	Description  string     `gorm:"description" json:"description"`
	IsOpen       bool       `gorm:"is_open" json:"is_open"`
	StartTime    *time.Time `gorm:"start_time" json:"-"`
}

// Close cashbox request
type CloseCashboxOperation struct {
	CashAmount     float64    `gorm:"cash_amount" json:"cash_amount"`
	CashlessAmount float64    `gorm:"cashless_amount" json:"cashless_amount"`
	ClosedAmount   float64    `gorm:"closed_amount" json:"closed_amount"`
	IsCompany      bool       `gorm:"-" json:"is_company"`
	IsOpen         bool       `gorm:"is_open" json:"is_open"`
	EndTime        *time.Time `gorm:"end_time" json:"-"`
}

type CashBoxCheckResponse struct {
	CashBoxOperationID string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	IsOpen             bool   `gorm:"is_open" json:"is_open"`
	SaleID             string `gorm:"-" json:"sale_id"`
}

// Cash Box Session structure
type CashboxOperationInfo struct {
	ID             string     `gorm:"id" json:"id"`
	CashBoxID      string     `gorm:"cash_box_id" json:"cash_box_id"`
	EmployeeID     string     `gorm:"employee_id" json:"employee_id"`
	CashAmount     float64    `gorm:"cash_amount" json:"cash_amount"`
	CashlessAmount float64    `gorm:"cashless_amount" json:"cashless_amount"`
	ClosedAmount   float64    `gorm:"closed_amount" json:"closed_amount"`
	OpenedAmount   float64    `gorm:"opened_amount" json:"opened_amount"`
	IsOpen         bool       `gorm:"is_open" json:"is_open"`
	Description    string     `gorm:"description" json:"description"`
	StartTime      *time.Time `gorm:"start_time" json:"start_time"`
	EndTime        *time.Time `gorm:"end_time" json:"end_time"`
	FirstName      string     `gorm:"first_name" json:"first_name"`
	StoreName      string     `gorm:"store_name" json:"store_name"`
}

// PaymentType structure
type CashboxPaymentType struct {
	ID            *string    `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	CashBoxId     string     `gorm:"cash_box_id" json:"cash_box_id"`
	PaymentTypeId string     `gorm:"payment_type_id" json:"payment_type_id"`
	IsActive      bool       `gorm:"is_active" json:"is_active"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
}
