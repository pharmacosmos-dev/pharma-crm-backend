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
	FaceIdUrl  *string    `gorm:"column:face_id_url" json:"face_id_url,omitempty"`
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
	FaceIdUrl string `json:"face_id_url"`
}

// ManualCreateAttendanceLogRequest — admin tomonidan xodim uchun qo'lda check-in/check-out
// yozuvi qo'shish so'rovi (face id orqali belgilash ishlamay qolgan hollar uchun).
// event_type qat'iy "check-in" yoki "check-out" bo'lishi kerak.
type ManualCreateAttendanceLogRequest struct {
	EmployeeId string    `json:"employee_id" binding:"required" example:"cd378978-2454-4c55-b5a4-3c3267d1c4c4"`
	EventType  string    `json:"event_type" binding:"required" example:"check-in"`
	EventAt    time.Time `json:"event_at" binding:"required" example:"2026-08-03T15:34:27+05:00"`
}

// AttendanceLogQueryParams — check-in/check-out ro'yxati uchun filter parametrlari.
// start_date va end_date DashboardQueryParam/SaleStatistic bilan bir xil ishlaydi:
// end_date berilmasa, start_date kuni yakunigacha (default 23:59) qamrab olinadi.
type AttendanceLogQueryParams struct {
	StoreId    string      `form:"store_id"`
	EmployeeId string      `form:"employee_id"`
	EventType  string      `form:"event_type"`
	StartDate  *CustomTime `form:"start_date"`
	EndDate    *CustomTime `form:"end_date"`
	Limit      int         `form:"limit"`
	Offset     int         `form:"offset"`
}

// AttendanceLogListItem — GET list javobi uchun, xodim va do'kon nomi bilan birga.
type AttendanceLogListItem struct {
	Id           string     `json:"id"`
	StoreId      *string    `json:"store_id"`
	StoreName    string     `json:"store_name"`
	EmployeeId   string     `json:"employee_id"`
	EmployeeName string     `json:"employee_name"`
	EventType    string     `json:"event_type"`
	EventAt      time.Time  `json:"event_at"`
	FaceIdUrl    *string    `json:"face_id_url"`
	CreatedAt    *time.Time `json:"created_at"`
}
