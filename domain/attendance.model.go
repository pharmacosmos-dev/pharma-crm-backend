package domain

import "time"

const (
	AttendanceEventCheckIn  = "check-in"
	AttendanceEventCheckOut = "check-out"
)

// AttendanceLog — xodimning check-in/check-out voqealari jurnali.
//
// created_by — yozuvni yaratgan foydalanuvchi (JWT user_id): face-id orqali
// check-in qilinganda xodimning o'zi, qo'lda kiritilganda kiritgan admin,
// cron (auto-close) yoki to'g'ridan-to'g'ri SQL yozganda esa NULL.
// updated_by — event_at'ni oxirgi marta qo'lda tuzatgan foydalanuvchi.
type AttendanceLog struct {
	Id           string     `gorm:"column:id" json:"id"`
	StoreId      *string    `gorm:"column:store_id" json:"store_id,omitempty"`
	EmployeeId   string     `gorm:"column:employee_id" json:"employee_id"`
	EventType    string     `gorm:"column:event_type" json:"event_type"`
	EventAt      time.Time  `gorm:"column:event_at" json:"event_at"`
	FaceIdUrl    *string    `gorm:"column:face_id_url" json:"face_id_url,omitempty"`
	IsAutoClosed bool       `gorm:"column:is_auto_closed" json:"is_auto_closed"`
	CreatedBy    *string    `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy    *string    `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt    *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AttendanceLog) TableName() string {
	return "attendance_logs"
}

type CreateAttendanceLogRequest struct {
	EventType string `json:"event_type" binding:"required" example:"check-in"`
	FaceIdUrl string `json:"face_id_url"`
}

type ManualCreateAttendanceLogRequest struct {
	EmployeeId string    `json:"employee_id" binding:"required" example:"cd378978-2454-4c55-b5a4-3c3267d1c4c4"`
	EventType  string    `json:"event_type" binding:"required" example:"check-in"`
	EventAt    time.Time `json:"event_at" binding:"required" example:"2026-08-03T15:34:27+05:00"`
}

// UpdateAttendanceLogRequest — check-in/check-out vaqtini qo'lda tuzatish.
//
// Faqat event_at o'zgaradi: employee_id yoki event_type'ni bu yerdan
// almashtirish yozuvni butunlay boshqa voqeaga aylantirardi — buning uchun
// eskisini o'chirib, yangisini qo'lda yaratish to'g'riroq.
type UpdateAttendanceLogRequest struct {
	EventAt time.Time `json:"event_at" binding:"required" example:"2026-09-03T09:15:00+05:00"`
}

type AttendanceLogQueryParams struct {
	StoreId      string      `form:"store_id"`
	EmployeeId   string      `form:"employee_id"`
	EventType    string      `form:"event_type"`
	Search       string      `form:"search"`
	StartDate    *CustomTime `form:"start_date"`
	EndDate      *CustomTime `form:"end_date"`
	IsAutoClosed *bool       `form:"is_auto_closed"`
	Limit  int               `form:"limit"`
	Offset int               `form:"offset"`
}

type AttendanceStatsQueryParams struct {
	Date    string `form:"date"`
	StoreId string `form:"store_id"`
	IsFranchise *bool `form:"is_franchise"`
	IsPharma    *bool `form:"is_pharma"`
	CompanyId string `form:"-"`
}

type AttendanceStoreStats struct {
	Total  int64 `json:"total"`
	Open   int64 `json:"open"`
	Closed int64 `json:"closed"`
}

type AttendanceEmployeeStats struct {
	Total      int64 `json:"total"`
	Working    int64 `json:"working"`
	NotWorking int64 `json:"not_working"`
	Came       int64 `json:"came"`
	Left       int64 `json:"left"`
	Absent     int64 `json:"absent"`
}

type AttendanceStats struct {
	Date      string                  `json:"date"`
	Stores    AttendanceStoreStats    `json:"stores"`
	Employees AttendanceEmployeeStats `json:"employees"`
}

// AttendanceLogListItem — GET list javobi uchun, xodim va do'kon nomi bilan birga.
type AttendanceLogListItem struct {
	Id            string     `json:"id"`
	StoreId       *string    `json:"store_id"`
	StoreName     string     `json:"store_name"`
	EmployeeId    string     `json:"employee_id"`
	EmployeeName  string     `json:"employee_name"`
	EmployeePhone string     `json:"employee_phone"`
	EventType     string     `json:"event_type"`
	EventAt       time.Time  `json:"event_at"`
	FaceIdUrl     *string    `json:"face_id_url"`
	IsAutoClosed  bool       `json:"is_auto_closed"`
	CreatedBy     *string    `json:"created_by"`
	CreatedByName string     `json:"created_by_name"`
	UpdatedBy     *string    `json:"updated_by"`
	UpdatedByName string     `json:"updated_by_name"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}


type AttendanceFaceIdCleanupResult struct {
	SelectedCount int `json:"selected_count" example:"120"`
	UpdatedCount int `json:"updated_count" example:"117"`
	DeletedFileCount int `json:"deleted_file_count" example:"110"`
	FileNotFoundCount int `json:"file_not_found_count" example:"5"`
	DeleteErrorCount int `json:"delete_error_count" example:"0"`
	SkippedInvalidPathCount int `json:"skipped_invalid_path_count" example:"3"`
	StillInUseFileCount int `json:"still_in_use_file_count" example:"0"`
}

type EmployeeAttendanceDayQueryParams struct {
	StoreId    string `form:"store_id"`
	EmployeeId string `form:"employee_id"`
	Search     string `form:"search"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

// EmployeeAttendanceDayListItem — GET list javobi uchun, xodim va do'kon nomi bilan birga.
type EmployeeAttendanceDayListItem struct {
	Id               string     `json:"id"`
	EmployeeId       string     `json:"employee_id"`
	EmployeeName     string     `json:"employee_name"`
	EmployeePhone    string     `json:"employee_phone"`
	StoreId          *string    `json:"store_id"`
	StoreName        string     `json:"store_name"`
	WorkDate         time.Time  `json:"work_date"`
	PlannedStartAt   *time.Time `json:"planned_start_at"`
	FirstCheckIn     *time.Time `json:"first_check_in"`
	LastCheckOut     *time.Time `json:"last_check_out"`
	WorkedMinutes    int        `json:"worked_minutes"`
	LateMinutes      int        `json:"late_minutes"`
	SalesAmount      float64    `json:"sales_amount"`
	IsAbsent         bool       `json:"is_absent"`
	IsManualOverride bool       `json:"is_manual_override"`
	Comment          *string    `json:"comment"`
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

type StoreWorkingHoursQueryParams struct {
	StoreId   string      `form:"store_id"`
	Search    string      `form:"search"`
	StartDate *CustomTime `form:"start_date"`
	EndDate   *CustomTime `form:"end_date"`
	Limit     int         `form:"limit"`
	Offset    int         `form:"offset"`
}

type StoreWorkingHoursListItem struct {
	StoreId       string     `json:"store_id"`
	StoreName     string     `json:"store_name"`
	WorkDate      string     `json:"work_date"`
	WorkedMinutes int        `json:"worked_minutes"`
	WorkedHours   float64    `json:"worked_hours"`
	OpenDate      *time.Time `json:"open_date"`
	CloseDate     *time.Time `json:"close_date"`
}
