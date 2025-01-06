package domain

import "time"

// Store structure
type Store struct {
	Id            string     `gorm:"id" json:"id"`
	Name          string     `gorm:"name" json:"name"`
	Location      string     `gorm:"location" json:"location"`
	EmployeeCount int        `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int        `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int        `gorm:"store_code" json:"store_code"`
	Address       string     `gorm:"address" json:"address"`
	Quantity      int        `gorm:"quantity" json:"quantity"`
	SmallQuantity int        `gorm:"small_quantity" json:"small_quantity"`
	CreatedAt     *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Store Create Request
type StoreRequest struct {
	Id            string `gorm:"id" json:"-"`
	Name          string `gorm:"name" json:"name"`
	Address       string `gorm:"address" json:"address"`
	EmployeeCount int    `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int    `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int    `gorm:"store_code" json:"store_code"`
	CreatedBy     string `gorm:"created_by" json:"-"`
}

// Store Update Request
type StoreUpdateRequest struct {
	Id            string `gorm:"id" json:"-"`
	Name          string `gorm:"name" json:"name"`
	Address       string `gorm:"address" json:"address"`
	EmployeeCount int    `gorm:"employee_count" json:"employee_count"`
	CashBoxCount  int    `gorm:"cash_box_count" json:"cash_box_count"`
	StoreCode     int    `gorm:"store_code" json:"store_code"`
	UpdatedBy     string `gorm:"updated_by" json:"-"`
}

// Store Request 1C
type StoreRequest1C struct {
	Id        string `gorm:"type:uuid;default:gen_random_uuid()" json:"-"`
	Name      string `gorm:"name" json:"name"`
	StoreCode int    `gorm:"store_code" json:"store_code"`
}
