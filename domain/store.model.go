package domain

import "time"

// Store structure
type Store struct {
	Id            string     `gorm:"id" json:"id"`
	Name          string     `gorm:"name" json:"name"`
	Phone         string     `gorm:"phone" json:"phone"`
	DetailedName  string     `gorm:"detailed_name" json:"detailed_name"`
	Location      string     `gorm:"location" json:"location"`
	EmployeeCount int        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int        `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int        `gorm:"store_code" json:"store_code"`
	Address       string     `gorm:"address" json:"address"`
	PackQuantity  int        `gorm:"pack_quantity" json:"pack_quantity"`
	SmallQuantity int        `gorm:"small_quantity" json:"small_quantity"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Store Create Request
type StoreRequest struct {
	Id            string  `gorm:"id" json:"-"`
	Name          string  `gorm:"name" json:"name"`
	Phone         *string `gorm:"phone" json:"phone"`
	WorkHours     string  `gorm:"work_hours" json:"work_hours"`
	DetailedName  string  `gorm:"detailed_name" json:"detailed_name"`
	Address       string  `gorm:"address" json:"address"`
	EmployeeCount int     `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int     `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int     `gorm:"store_code" json:"store_code"`
	Location      string  `gorm:"location" json:"location"`
}

// Store Update Request
type StoreUpdateRequest struct {
	Id            string  `gorm:"id" json:"-"`
	Name          string  `gorm:"name" json:"name"`
	Phone         *string `gorm:"phone" json:"phone"`
	WorkHours     string  `gorm:"work_hours" json:"work_hours"`
	DetailedName  string  `gorm:"detailed_name" json:"detailed_name"`
	Address       string  `gorm:"address" json:"address"`
	EmployeeCount int     `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int     `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int     `gorm:"store_code" json:"store_code"`
	Location      string  `gorm:"location" json:"location"`
	UpdatedBy     string  `gorm:"updated_by" json:"-"`
}

// Store Request 1C
type StoreRequest1C struct {
	Id        string `gorm:"type:uuid;default:gen_random_uuid()" json:"-"`
	Name      string `gorm:"name" json:"name"`
	StoreCode int    `gorm:"store_code" json:"store_code"`
}

// Stores list with store product info
type StoreWithProducts struct {
	Id            string     `gorm:"id" json:"id"`
	Name          string     `gorm:"name" json:"name"`
	Phone         string     `gorm:"phone" json:"phone"`
	DetailedName  string     `gorm:"detailed_name" json:"detailed_name"`
	StoreCode     int        `gorm:"store_code" json:"store_code"`
	PackQuantity  int        `gorm:"pack_quantity" json:"pack_quantity"`
	UnitQuantity  int        `gorm:"unit_quantity" json:"unit_quantity"`
	SmallQuantity int        `gorm:"small_quantity" json:"small_quantity"`
	SupplyPrice   float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice   float64    `gorm:"retail_price" json:"retail_price"`
	Vat           int        `gorm:"vat" json:"vat"`
	Markup        int        `gorm:"markup" json:"markup"`
	ExpireDate    *time.Time `gorm:"expire_date" json:"expire_date"`
	BonusPercent  int        `gorm:"bonus_percent" json:"bonus_percent"`
	Location      string     `gorm:"location" json:"location"`
	Address       string     `gorm:"address" json:"address"`
}
