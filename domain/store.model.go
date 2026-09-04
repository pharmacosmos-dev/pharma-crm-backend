package domain

import (
	"time"

	"github.com/lib/pq"
)

// Store structure
type Store struct {
	Id                 string         `gorm:"id" json:"id"`
	StoreCode          int            `gorm:"store_code" json:"store_code"`
	Name               string         `gorm:"name" json:"name"`
	DetailedName       string         `gorm:"detailed_name" json:"detailed_name"`
	CompanyId          string         `gorm:"company_id" json:"company_id"`
	Phone              string         `gorm:"phone" json:"phone"`
	Contact            string         `gorm:"contact" json:"contact"`
	Inn                string         `gorm:"inn" json:"inn"`
	EmployeeCount      float64        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount       int            `gorm:"cash_box_count" json:"cash_box_count"`
	Address            string         `gorm:"address" json:"address"`
	Location           string         `gorm:"location" json:"location"`
	WorkHours          string         `gorm:"work_hours" json:"work_hours"`
	IsFullday          bool           `gorm:"is_fullday" json:"is_fullday"`
	AverageTargetSales float64        `gorm:"column:average_target_sales" json:"average_target_sales"`
	IsOnlineOrder      bool           `gorm:"column:is_online_order" json:"is_online_order"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	TerminalID         pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

type StoreDto struct {
	Id                 string         `gorm:"id" json:"id"`
	StoreCode          int            `gorm:"store_code" json:"store_code"`
	Name               string         `gorm:"name" json:"name"`
	DetailedName       string         `gorm:"detailed_name" json:"detailed_name"`
	CompanyId          string         `gorm:"company_id" json:"company_id"`
	Phone              string         `gorm:"phone" json:"phone"`
	Contact            string         `gorm:"contact" json:"contact"`
	Inn                string         `gorm:"inn" json:"inn"`
	EmployeeCount      float64        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount       int            `gorm:"cash_box_count" json:"cash_box_count"`
	Address            string         `gorm:"address" json:"address"`
	Location           string         `gorm:"location" json:"location"`
	Coordinates        Point          `gorm:"column:coordinates" json:"coordinates"`
	WorkHours          string         `gorm:"work_hours" json:"work_hours"`
	IsFullday          bool           `gorm:"is_fullday" json:"is_fullday"`
	TargetAmount       float64        `gorm:"column:target_amount" json:"target_amount"`
	AverageTargetSales float64        `gorm:"column:average_target_sales" json:"average_target_sales"`
	CreatedAt          *time.Time     `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time     `gorm:"updated_at" json:"updated_at"`
	TerminalID         pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

// Store Create Request
type StoreRequest struct {
	Id            string         `gorm:"id" json:"-"`
	Name          string         `gorm:"name" json:"name"`
	CompanyId     string         `gorm:"company_id" json:"company_id"`
	Phone         *string        `gorm:"phone" json:"phone"`
	WorkHours     string         `gorm:"work_hours" json:"work_hours"`
	DetailedName  string         `gorm:"detailed_name" json:"detailed_name"`
	Address       string         `gorm:"address" json:"address"`
	Inn           string         `gorm:"inn" json:"inn"`
	EmployeeCount float64        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int            `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int            `gorm:"store_code" json:"store_code"`
	Location      string         `gorm:"location" json:"location"`
	IsFullday     bool           `gorm:"is_fullday" json:"is_fullday"`
	TerminalID    pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

// Store Update Request
type StoreUpdateRequest struct {
	Id            string         `gorm:"id" json:"-"`
	Name          string         `gorm:"name" json:"name"`
	Phone         *string        `gorm:"phone" json:"phone"`
	WorkHours     string         `gorm:"work_hours" json:"work_hours"`
	DetailedName  string         `gorm:"detailed_name" json:"detailed_name"`
	Address       string         `gorm:"address" json:"address"`
	Inn           string         `gorm:"inn" json:"inn"`
	EmployeeCount float64        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int            `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int            `gorm:"store_code" json:"store_code"`
	CompanyId     string         `gorm:"company_id" json:"company_id"`
	Location      string         `gorm:"location" json:"location"`
	IsFullday     bool           `gorm:"is_fullday" json:"is_fullday"`
	UpdatedBy     string         `gorm:"updated_by" json:"-"`
	TerminalID    pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

type UpdateOnlineOrderRequest struct {
	StoreIds      []string `json:"store_ids" binding:"required,min=1"`
	IsOnlineOrder bool     `json:"is_online_order"`
}

type StoreQueryParams struct {
	CompanyId     string   `form:"company_id"`
	CompanyIds    []string `form:"-"`
	StoreId       string   `form:"-"`
	StoreIds      []string `form:"-"`
	Search        string   `form:"search"`
	IsFranchise   *bool    `form:"is_franchise"`
	IsOnlineOrder *bool    `form:"is_online"`
	Limit         int      `form:"limit"`
	Offset        int      `form:"offset"`
}

type StoreEmployeeCountQueryParams struct {
	CompanyId   string   `form:"company_id"`
	CompanyIds  []string `form:"-"`
	StoreId     string   `form:"-"`
	StoreIds    []string `form:"-"`
	Search      string   `form:"search"`
	IsFranchise *bool    `form:"is_franchise"`
	Limit       int      `form:"limit"`
	Offset      int      `form:"offset"`
}

// StoreEmployeeCount holds the planned headcount of a store next to the number
// of employees actually assigned to it.
// EmployeeCount rejadagi son bo'lgani uchun kasr bo'lishi mumkin (yarim stavka:
// 2.5). ActualEmployeeCount esa haqiqiy xodimlar sanog'i — u doim butun.
type StoreEmployeeCount struct {
	Id                  string  `json:"id"`
	StoreCode           int     `json:"store_code"`
	Name                string  `json:"name"`
	CompanyId           string  `json:"company_id"`
	EmployeeCount       float64 `json:"employee_count"`
	CashBoxCount        int     `json:"cash_box_count"`
	ActualEmployeeCount int     `json:"actual_employee_count"`
	Difference          float64 `json:"difference"`
}

// StoreEmployeeCountStat is the StoreEmployeeCount list rolled up into totals.
type StoreEmployeeCountStat struct {
	TotalStores         int64   `json:"total_stores"`
	TotalPlanEmployees  float64 `json:"total_plan_employees"`
	ActualEmployeeCount int64   `json:"actual_employee_count"`
	TotalDiff           float64 `json:"total_diff"`
}

// StoreEmployeeCountRequest updates the planned headcount of a single store.
// The pointer keeps an explicit 0 distinguishable from a missing field.
// Kasr qiymat qabul qilinadi (yarim stavka uchun 2.5).
type StoreEmployeeCountRequest struct {
	EmployeeCount *float64 `json:"employee_count" binding:"required,min=0"`
}

type StoreMapInfoQueryParams struct {
	Search      string `form:"search"`
	IsFranchise *bool  `form:"is_franchise"`
	IsPharma    *bool  `form:"is_pharma"`
	IsOnline    *bool  `form:"is_online"`
}

type StoreMapInfo struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	WorkHours     string `json:"work_hours"`
	Phone         string `json:"phone"`
	IsOnlineOrder bool   `json:"is_online_order"`
	IsOpen        bool   `json:"is_open"`
	Address       string `json:"address"`
	Inn           string `json:"inn"`
	StoreCode     string `json:"store_code"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Coordinates   Point  `json:"coordinates"`
}

type StoreMapInfoDetail struct {
	StoreMapInfo
	SalesAmount        float64                   `json:"sales_amount"`
	SalesCount         int64                     `json:"sales_count"`
	SalesAggregateSum  float64                   `json:"sales_aggregate_sum"`
	AverageSalesAmount float64                   `json:"average_sales_amount"`
	CashBoxCount       int64                     `json:"cash_box_count"`
	EmployeeCount      int64                     `json:"employee_count"`
	Employees          []StoreMapInfoEmployee    `json:"employees"`
	PaymentTypes       []StoreMapInfoPaymentType `json:"payment_types"`
	OpenedAt           *time.Time                `json:"opened_at,omitempty"`
	ClosedAt           *time.Time                `json:"closed_at,omitempty"`
}

type StoreMapInfoEmployee struct {
	Id         string     `json:"id"`
	FullName   string     `json:"full_name"`
	CheckInAt  *time.Time `json:"check_in_at,omitempty"`
	CheckOutAt *time.Time `json:"check_out_at,omitempty"`
	LastEventType string     `json:"last_event_type,omitempty"`
	LastEventAt   *time.Time `json:"last_event_at,omitempty"`
}

type StoreMapInfoPaymentType struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}
