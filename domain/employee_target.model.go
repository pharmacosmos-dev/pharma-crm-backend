package domain

import "time"

type EmployeeTarget struct {
	Id            string     `json:"id" gorm:"column:id"`
	StoreTargetId string     `json:"store_target_id" gorm:"column:store_target_id"`
	EmployeeId    string     `json:"employee_id" gorm:"column:employee_id"`
	StoreId       string     `json:"store_id" gorm:"column:store_id"`
	CompanyId     string     `json:"company_id" gorm:"column:company_id"`
	Amount        float64    `json:"amount" gorm:"column:amount"`
	Sales         float64    `json:"sales" gorm:"column:sales"`
	Year          int        `json:"year" gorm:"column:year"`
	Month         int        `json:"month" gorm:"column:month"`
	CreatedAt     *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"column:updated_at"`

	Employee    *Employee    `json:"employee,omitempty" gorm:"foreignKey:EmployeeId"`
	StoreTarget *StoreTarget `json:"store_target,omitempty" gorm:"foreignKey:StoreTargetId"`
}

func (EmployeeTarget) TableName() string {
	return "employee_targets"
}

// Employee's current month target + actual sales
type EmployeeTargetWithSales struct {
	Id                 string  `json:"id"`
	EmployeeId         string  `json:"employee_id"`
	EmployeeName       string  `json:"employee_name"`
	MonthlyTarget      float64 `json:"monthly_target"`
	DailyTarget        float64 `json:"daily_target"`
	ActualMonthlySales float64 `json:"actual_monthly_sales"`
	ActualDailySales   float64 `json:"actual_daily_sales"`
	Year               int     `json:"year"`
	Month              int     `json:"month"`
	DaysInMonth        int     `json:"days_in_month"`
}

// History all employees by store
type EmployeeTargetHistoryItem struct {
	EmployeeId   string  `json:"employee_id"`
	EmployeeName string  `json:"employee_name"`
	Amount       float64 `json:"amount"`
	Sales        float64 `json:"sales"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// Query params
type EmployeeTargetQueryParams struct {
	EmployeeId string `form:"employee_id"`
	StoreId    string `form:"store_id"`
	CompanyId  string `form:"-"`
	Year       int    `form:"year"`
	Month      int    `form:"month"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}
