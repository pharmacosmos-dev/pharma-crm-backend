package domain

import "time"

const (
	AttendanceEventCheckIn  = "check-in"
	AttendanceEventCheckOut = "check-out"
)

// AttendanceLog — xodimning check-in/check-out voqealari jurnali.
type AttendanceLog struct {
	Id         string     `gorm:"column:id" json:"id"`
	StoreId    *string    `gorm:"column:store_id" json:"store_id,omitempty"`
	EmployeeId string     `gorm:"column:employee_id" json:"employee_id"`
	EventType  string     `gorm:"column:event_type" json:"event_type"`
	EventAt    time.Time  `gorm:"column:event_at" json:"event_at"`
	CreatedAt  *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AttendanceLog) TableName() string {
	return "attendance_logs"
}

// CreateAttendanceLogRequest — check-in/check-out yaratish so'rovi.
// event_type qat'iy "check-in" yoki "check-out" bo'lishi kerak.
type CreateAttendanceLogRequest struct {
	EventType string `json:"event_type" binding:"required" example:"check-in"`
}
