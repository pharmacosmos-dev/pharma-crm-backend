package domain

import "time"

type Shift struct {
	Id             string     `gorm:"id" json:"id"`
	CashBoxId      string     `gorm:"cash_box_id" json:"cash_box_id"`
	FromEmployeeId string     `gorm:"from_employee_id" json:"from_employee_id"`
	ToEmployeeId   string     `gorm:"to_employee_id" json:"to_employee_id"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	FromEmployee   *Employee  `gorm:"foreignKey:FromEmployeeId" json:"from_employee"`
	ToEmployee     *Employee  `gorm:"foreignKey:ToEmployeeId" json:"to_employee"`
	CashBox        *CashBox   `gorm:"foreignKey:CashBoxId" json:"cash_box"`
}

type ShiftRequest struct {
	CashBoxId      string `gorm:"cash_box_id" json:"cash_box_id"`
	FromEmployeeId string `gorm:"from_employee_id" json:"from_employee_id"`
	ToEmployeeId   string `gorm:"to_employee_id" json:"to_employee_id"`
}
