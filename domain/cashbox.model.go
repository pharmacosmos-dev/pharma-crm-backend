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

// Cashbox data
type CashboxOpenData struct {
	ID         string `gorm:"id" json:"id"`
	Name       string `gorm:"name" json:"name"`
	StoreID    string `gorm:"store_id" json:"store_id"`
	IsOpen     bool   `gorm:"is_open" json:"is_open"`
	IsActive   bool   `gorm:"is_active" json:"is_active"`
	StoreName  string `gorm:"store_name" json:"store_name"`
	FullName   string `gorm:"full_name" json:"full_name"`
	TotalCount int64  `gorm:"total_count" json:"-"`
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
	OperationID    int64      `gorm:"operation_id" json:"operation_id"`
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
	ID                 string     `gorm:"id" json:"-"`
	CashBoxID          string     `gorm:"cash_box_id" json:"cash_box_id"`
	StoreID            string     `gorm:"store_id" json:"store_id"`
	EmployeeID         string     `gorm:"employee_id" json:"-"`
	OpenedAmount       float64    `gorm:"opened_amount" json:"opened_amount"`
	OpenCashlessAmount float64    `gorm:"open_cashless_amount" json:"open_cashless_amount"`
	Description        string     `gorm:"description" json:"description"`
	IsOpen             bool       `gorm:"is_open" json:"is_open"`
	StartTime          *time.Time `gorm:"start_time" json:"-"`
}

// Close cashbox request
type CloseCashboxOperation struct {
	ClosedAmount        float64    `gorm:"closed_amount" json:"closed_amount"`
	CloseCashlessAmount float64    `gorm:"close_cashless_amount" json:"close_cashless_amount"`
	IsCompany           bool       `gorm:"-" json:"is_company"`
	IsOpen              bool       `gorm:"is_open" json:"is_open"`
	EndTime             *time.Time `gorm:"end_time" json:"-"`
}

type CashBoxCheckResponse struct {
	CashBoxOperationID string `gorm:"cash_box_operation_id" json:"cash_box_operation_id"`
	IsOpen             bool   `gorm:"is_open" json:"is_open"`
	SaleID             string `gorm:"-" json:"sale_id"`
}

// Cash Box Session structure
type CashboxOperationInfo struct {
	ID             string     `gorm:"id" json:"id"`
	OperationID    int64      `gorm:"operation_id" json:"operation_id"`
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

// CashboxOperation Payment amounts
type CashboxOperationAmount struct {
	Cash   float64 `gorm:"cash" json:"cash"`
	Uzcard float64 `gorm:"uzcard" json:"uzcard"`
	Humo   float64 `gorm:"humo" json:"humo"`
	Click  float64 `gorm:"click" json:"click"`
	Payme  float64 `gorm:"payme" json:"payme"`
	Uzum   float64 `gorm:"uzum" json:"uzum"`
}

// Cashbox operation shift
type CashboxOperationShift struct {
	Id                   string     `gorm:"id" json:"id"`
	OperationId          int64      `gorm:"operation_id" json:"operation_id"`
	CashboxName          string     `gorm:"cashbox_name" json:"cashbox_name"`
	StoreName            string     `gorm:"store_name" json:"store_name"`
	IsOpen               bool       `gorm:"is_open" json:"is_open"`
	OpenedCashAmount     float64    `gorm:"opened_amount" json:"opened_amount"`
	OpenedCashlessAmount float64    `gorm:"opened_cashless_amount" json:"opened_cashless_amount"`
	CashAmount           float64    `gorm:"cash_amount" json:"cash_amount"`
	CashlessAmount       float64    `gorm:"cashless_amount" json:"cashless_amount"`
	StartTime            *time.Time `gorm:"start_time" json:"start_time"`
	EndTime              *time.Time `gorm:"end_time" json:"end_time"`
}

type CashboxOperationStats struct {
	TotalCashAmount            float64 `gorm:"total_cash_amount" json:"total_cash_amount"`
	TotalCashlessAmount        float64 `gorm:"total_cashless_amount" json:"total_cashless_amount"`
	TotalExpenseCashAmount     float64 `gorm:"total_expense_cash_amount" json:"total_expense_cash_amount"`
	TotalExpenseCashlessAmount float64 `gorm:"total_expense_cashless_amount" json:"total_expense_cashless_amount"`
	TotalOpenedCashAmount      float64 `gorm:"total_opened_cash_amount" json:"total_opened_cash_amount"`
	TotalOpenedCashlessAmount  float64 `gorm:"total_opened_cashless_amount" json:"total_opened_cashless_amount"`
	CurrentCashAmount          float64 `gorm:"current_cash_amount" json:"current_cash_amount"`
	CurrentCashlessAmount      float64 `gorm:"current_cashless_amount" json:"current_cashless_amount"`
}

// Cashbox operation history for getting all operations history
type CashBoxOperationHistory struct {
	Id                 string     `gorm:"id" json:"id"`
	OperationId        int64      `gorm:"operation_id" json:"operation_id"`
	CashboxName        string     `gorm:"cashbox_name" json:"cashbox_name"`
	StoreName          string     `gorm:"store_name" json:"store_name"`
	StartTime          *time.Time `gorm:"start_time" json:"start_time"`
	EndTime            *time.Time `gorm:"end_time" json:"end_time"`
	IsOpen             bool       `gorm:"is_open" json:"is_open"`
	OpenedBy           string     `gorm:"opened_by" json:"opened_by"`
	ClosedBy           string     `gorm:"closed_by" json:"closed_by"`
	TotalExpenseAmount float64    `gorm:"total_expense_amount" json:"total_expense_amount"`
}

// Employee Cashbox structure for getting cashbox info which is open employee
type EmployeeCashbox struct {
	Id          string     `gorm:"id" json:"id"`
	OperationId int64      `gorm:"operation_id" json:"operation_id"`
	Name        string     `gorm:"name" json:"name"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
}
