package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// GetTodayLastAttendanceEventType — xodimning bugungi kun (Toshkent vaqti) bo'yicha
// eng oxirgi attendance_logs voqeasi turini qaytaradi. Bugun hech qanday voqea
// bo'lmasa bo'sh string qaytaradi (xato emas).
func (s *Services) GetTodayLastAttendanceEventType(ctx context.Context, employeeId string) (string, error) {
	var last struct {
		EventType string `gorm:"column:event_type"`
	}
	err := s.db.WithContext(ctx).Raw(`
		SELECT event_type
		FROM attendance_logs
		WHERE employee_id = ?
		  AND (event_at + interval '5 hours')::date = CURRENT_DATE
		ORDER BY event_at DESC
		LIMIT 1
	`, employeeId).Take(&last).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		s.log.Errorf("could not get last attendance log: %v", err)
		return "", domain.InternalServerError
	}

	return last.EventType, nil
}

// CreateAttendanceLog — JWT orqali aniqlangan xodim uchun check-in yoki check-out
// voqeasini attendance_logs jadvaliga yozadi. Faqat bugungi kun (Toshkent vaqti)
// bo'yicha oxirgi voqeaga qarab tekshiriladi: hech qanday voqea yo'q yoki oxirgisi
// check-out bo'lsa faqat check-in, oxirgisi check-in bo'lsa faqat check-out qilish
// mumkin — kechagi kun voqealari hisobga olinmaydi.
func (s *Services) CreateAttendanceLog(ctx context.Context, employeeId, storeId, eventType string) (*domain.AttendanceLog, error) {
	if eventType != domain.AttendanceEventCheckIn && eventType != domain.AttendanceEventCheckOut {
		return nil, domain.InvalidEventTypeError
	}

	lastEventType, err := s.GetTodayLastAttendanceEventType(ctx, employeeId)
	if err != nil {
		return nil, err
	}

	switch lastEventType {
	case "":
		if eventType == domain.AttendanceEventCheckOut {
			return nil, domain.AttendanceCheckInRequiredError
		}
	case eventType:
		if eventType == domain.AttendanceEventCheckIn {
			return nil, domain.AttendanceAlreadyCheckedInError
		}
		return nil, domain.AttendanceCheckInRequiredError
	}

	var storeIdPtr *string
	if storeId != "" {
		storeIdPtr = &storeId
	}

	log := domain.AttendanceLog{
		Id:         uuid.New().String(),
		StoreId:    storeIdPtr,
		EmployeeId: employeeId,
		EventType:  eventType,
		EventAt:    time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
		s.log.Errorf("could not create attendance log: %v", err)
		return nil, domain.InternalServerError
	}

	return &log, nil
}

// GetAttendanceLogList — check-in/check-out yozuvlari ro'yxati, store_id, employee_id
// va date (Toshkent vaqti bo'yicha bitta kun) filtrlari bilan.
func (s *Services) GetAttendanceLogList(ctx context.Context, params *domain.AttendanceLogQueryParams) ([]domain.AttendanceLogListItem, int64, error) {
	countQuery := s.db.WithContext(ctx).Table("attendance_logs al")
	query := s.db.WithContext(ctx).Table("attendance_logs al").
		Joins("LEFT JOIN employees e ON e.id = al.employee_id").
		Joins("LEFT JOIN stores s ON s.id = al.store_id")

	if params.StoreId != "" {
		countQuery = countQuery.Where("al.store_id = ?", params.StoreId)
		query = query.Where("al.store_id = ?", params.StoreId)
	}

	if params.EmployeeId != "" {
		countQuery = countQuery.Where("al.employee_id = ?", params.EmployeeId)
		query = query.Where("al.employee_id = ?", params.EmployeeId)
	}

	if params.Date != "" {
		if _, err := time.Parse(constants.TimeOnlyDateFormat, params.Date); err != nil {
			return nil, 0, domain.InvalidTimeFormatError
		}
		countQuery = countQuery.Where("(al.event_at + interval '5 hours')::date = ?::date", params.Date)
		query = query.Where("(al.event_at + interval '5 hours')::date = ?::date", params.Date)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		s.log.Errorf("could not count attendance logs: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if total == 0 {
		return []domain.AttendanceLogListItem{}, 0, nil
	}

	query = query.Select(`
			al.id,
			al.store_id,
			COALESCE(s.name, '') AS store_name,
			al.employee_id,
			COALESCE(e.full_name, '') AS employee_name,
			al.event_type,
			al.event_at,
			al.created_at
		`).
		Order("al.event_at DESC")

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.AttendanceLogListItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get attendance log list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.AttendanceLogListItem{}
	}

	return results, total, nil
}
