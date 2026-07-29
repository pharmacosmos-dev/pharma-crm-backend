package domain

import "time"

// EmployeeAttendanceDay — bitta xodim + bitta kun uchun kunlik davomat yig'indisi.
// attendance_logs'dagi xom check-in/check-out voqealaridan tungi cron orqali to'ldiriladi
// (AggregateEmployeeAttendanceDays). is_manual_override=true bo'lgan kunlarga cron tegmaydi.
type EmployeeAttendanceDay struct {
	Id               string     `gorm:"id" json:"id"`
	EmployeeId       string     `gorm:"employee_id" json:"employee_id"`
	StoreId          *string    `gorm:"store_id" json:"store_id,omitempty"`
	WorkDate         time.Time  `gorm:"work_date" json:"work_date"`
	PlannedStartAt   *time.Time `gorm:"planned_start_at" json:"planned_start_at,omitempty"`
	FirstCheckIn     *time.Time `gorm:"first_check_in" json:"first_check_in,omitempty"`
	LastCheckOut     *time.Time `gorm:"last_check_out" json:"last_check_out,omitempty"`
	WorkedMinutes    int        `gorm:"worked_minutes" json:"worked_minutes"`
	LateMinutes      int        `gorm:"late_minutes" json:"late_minutes"`
	SalesAmount      float64    `gorm:"sales_amount" json:"sales_amount"`
	IsAbsent         bool       `gorm:"is_absent" json:"is_absent"`
	IsManualOverride bool       `gorm:"is_manual_override" json:"is_manual_override"`
	Comment          *string    `gorm:"comment" json:"comment,omitempty"`
	UpdatedBy        *string    `gorm:"updated_by" json:"updated_by,omitempty"`
	CreatedAt        *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time `gorm:"updated_at" json:"updated_at"`
	Employee         *Employee  `gorm:"foreignKey:EmployeeId" json:"employee,omitempty"`
	Store            *Store     `gorm:"foreignKey:StoreId" json:"store,omitempty"`
}

func (EmployeeAttendanceDay) TableName() string {
	return "employee_attendance_days"
}
