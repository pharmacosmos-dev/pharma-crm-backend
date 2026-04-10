package domain

import (
	"time"

	"github.com/lib/pq"
)

// Store structure
type Store struct {
	Id                  string     `gorm:"id" json:"id"`
	StoreCode           int        `gorm:"store_code" json:"store_code"`
	Name                string     `gorm:"name" json:"name"`
	DetailedName        string     `gorm:"detailed_name" json:"detailed_name"`
	CompanyId           string     `gorm:"company_id" json:"company_id"`
	Phone               string     `gorm:"phone" json:"phone"`
	Contact             string     `gorm:"contact" json:"contact"`
	Inn                 string     `gorm:"inn" json:"inn"`
	EmployeeCount       int        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount        int        `gorm:"cash_box_count" json:"cash_box_count"`
	Address             string     `gorm:"address" json:"address"`
	Location            string     `gorm:"location" json:"location"`
	WorkHours           string     `gorm:"work_hours" json:"work_hours"`
	IsFullday           bool       `gorm:"is_fullday" json:"is_fullday"`
	AverageTargetSales  float64    `gorm:"column:average_target_sales" json:"average_target_sales"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	TerminalID          pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

type StoreDto struct {
	Id            string     `gorm:"id" json:"id"`
	StoreCode     int        `gorm:"store_code" json:"store_code"`
	Name          string     `gorm:"name" json:"name"`
	DetailedName  string     `gorm:"detailed_name" json:"detailed_name"`
	CompanyId     string     `gorm:"company_id" json:"company_id"`
	Phone         string     `gorm:"phone" json:"phone"`
	Contact       string     `gorm:"contact" json:"contact"`
	Inn           string     `gorm:"inn" json:"inn"`
	EmployeeCount int        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int        `gorm:"cash_box_count" json:"cash_box_count"`
	Address       string     `gorm:"address" json:"address"`
	Location      string     `gorm:"location" json:"location"`
	Coordinates   Point      `gorm:"column:coordinates" json:"coordinates"`
	WorkHours     string     `gorm:"work_hours" json:"work_hours"`
	IsFullday     bool       `gorm:"is_fullday" json:"is_fullday"`
	TargetAmount        float64    `gorm:"column:target_amount" json:"target_amount"`
	AverageTargetSales  float64    `gorm:"column:average_target_sales" json:"average_target_sales"`
	CreatedAt           *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `gorm:"updated_at" json:"updated_at"`
	TerminalID          pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

// Store Create Request
type StoreRequest struct {
	Id            string  `gorm:"id" json:"-"`
	Name          string  `gorm:"name" json:"name"`
	CompanyId     string  `gorm:"company_id" json:"company_id"`
	Phone         *string `gorm:"phone" json:"phone"`
	WorkHours     string  `gorm:"work_hours" json:"work_hours"`
	DetailedName  string  `gorm:"detailed_name" json:"detailed_name"`
	Address       string  `gorm:"address" json:"address"`
	Inn           string  `gorm:"inn" json:"inn"`
	EmployeeCount int     `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int     `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int     `gorm:"store_code" json:"store_code"`
	Location      string  `gorm:"location" json:"location"`
	IsFullday     bool    `gorm:"is_fullday" json:"is_fullday"`
	TerminalID    pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

// Store Update Request
type StoreUpdateRequest struct {
	Id            string  `gorm:"id" json:"-"`
	Name          string  `gorm:"name" json:"name"`
	Phone         *string `gorm:"phone" json:"phone"`
	WorkHours     string  `gorm:"work_hours" json:"work_hours"`
	DetailedName  string  `gorm:"detailed_name" json:"detailed_name"`
	Address       string  `gorm:"address" json:"address"`
	Inn           string  `gorm:"inn" json:"inn"`
	EmployeeCount int     `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int     `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int     `gorm:"store_code" json:"store_code"`
	Location      string  `gorm:"location" json:"location"`
	IsFullday     bool    `gorm:"is_fullday" json:"is_fullday"`
	UpdatedBy     string  `gorm:"updated_by" json:"-"`
	TerminalID    pq.StringArray `gorm:"type:varchar(255)[];column:terminal_id" json:"terminal_ids" swaggertype:"array,string"`
}

type StoreQueryParams struct {
	CompanyId  string   `form:"company_id"`
	CompanyIds []string `form:"-"`
	Search     string   `form:"search"`
	Limit      int      `form:"limit"`
	Offset     int      `form:"offset"`
}
