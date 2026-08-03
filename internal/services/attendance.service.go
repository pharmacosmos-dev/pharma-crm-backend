package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
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
func (s *Services) CreateAttendanceLog(ctx context.Context, employeeId, storeId, eventType, faceIdUrl string) (*domain.AttendanceLog, error) {
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

	var faceIdUrlPtr *string
	if faceIdUrl != "" {
		faceIdUrlPtr = &faceIdUrl
	}

	log := domain.AttendanceLog{
		Id:         uuid.New().String(),
		StoreId:    storeIdPtr,
		EmployeeId: employeeId,
		EventType:  eventType,
		EventAt:    time.Now(),
		FaceIdUrl:  faceIdUrlPtr,
	}

	if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
		s.log.Errorf("could not create attendance log: %v", err)
		return nil, domain.InternalServerError
	}

	return &log, nil
}

// CreateManualAttendanceLog — admin tomonidan berilgan employee_id, event_type va
// event_at asosida attendance_logs yozuvini qo'lda yaratadi (face id orqali belgilash
// ishlamay qolgan hollar uchun). Xodimning store_id'si employees jadvalidan olinadi,
// ketma-ketlik (check-in/check-out navbati) tekshiruvi qo'llanilmaydi — bu qo'lda
// tuzatish uchun mo'ljallangan.
func (s *Services) CreateManualAttendanceLog(ctx context.Context, employeeId, eventType string, eventAt time.Time) (*domain.AttendanceLog, error) {
	if eventType != domain.AttendanceEventCheckIn && eventType != domain.AttendanceEventCheckOut {
		return nil, domain.InvalidEventTypeError
	}

	var employee domain.Employee
	if err := s.db.WithContext(ctx).Select("id", "store_id").Take(&employee, "id = ?", employeeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("could not get employee: %v", err)
		return nil, domain.InternalServerError
	}

	var storeIdPtr *string
	if employee.StoreId != "" {
		storeIdPtr = &employee.StoreId
	}

	log := domain.AttendanceLog{
		Id:         uuid.New().String(),
		StoreId:    storeIdPtr,
		EmployeeId: employeeId,
		EventType:  eventType,
		EventAt:    eventAt,
	}

	if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
		s.log.Errorf("could not create manual attendance log: %v", err)
		return nil, domain.InternalServerError
	}

	return &log, nil
}

// ClearAttendanceLogFaceIdUrl — attendance_logs yozuvining face_id_url maydonini
// NULL qiladi va eski qiymatni (fayl nomini) qaytaradi, handler shu nom bo'yicha
// faylni upload papkadan o'chiradi.
func (s *Services) ClearAttendanceLogFaceIdUrl(ctx context.Context, id string) (string, error) {
	var log domain.AttendanceLog
	if err := s.db.WithContext(ctx).Take(&log, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ResourceNotFoundError
		}
		s.log.Errorf("could not get attendance log: %v", err)
		return "", domain.InternalServerError
	}

	var oldFaceIdUrl string
	if log.FaceIdUrl != nil {
		oldFaceIdUrl = *log.FaceIdUrl
	}

	if err := s.db.WithContext(ctx).
		Model(&domain.AttendanceLog{}).
		Where("id = ?", id).
		Update("face_id_url", nil).Error; err != nil {
		s.log.Errorf("could not clear attendance log face_id_url: %v", err)
		return "", domain.InternalServerError
	}

	return oldFaceIdUrl, nil
}

// CleanupOldAttendanceFaceIds — keepDays kundan eski (Toshkent vaqti bo'yicha kun sanog'i)
// check-in/check-out rasmlarining face_id_url maydonini NULL qiladi va o'chirilgan fayl
// nomlarini qaytaradi (handler shu fayllarni ./app/uploads papkadan o'chiradi).
// Masalan keepDays=2 bo'lsa, bugungi va kechagi kun rasmlari saqlanib qoladi,
// undan oldingi barcha kunlar (masalan, 1-avgust, 31-iyul, ...) tozalanadi.
func (s *Services) CleanupOldAttendanceFaceIds(ctx context.Context, keepDays int) ([]string, error) {
	var fileNames []string
	err := s.db.WithContext(ctx).Raw(`
		UPDATE attendance_logs
		SET face_id_url = NULL
		WHERE face_id_url IS NOT NULL
		  AND (event_at + interval '5 hours')::date < (CURRENT_DATE - ?::int)
		RETURNING face_id_url
	`, keepDays-1).Scan(&fileNames).Error
	if err != nil {
		s.log.Errorf("could not cleanup old attendance face ids: %v", err)
		return nil, domain.InternalServerError
	}

	return fileNames, nil
}

// GetAttendanceLogList — check-in/check-out yozuvlari ro'yxati, store_id, employee_id
// va start_date/end_date (SaleStatistic bilan bir xil: end_date berilmasa start_date
// kuni yakunigacha qamrab olinadi) filtrlari bilan.
func (s *Services) GetAttendanceLogList(ctx context.Context, params *domain.AttendanceLogQueryParams) ([]domain.AttendanceLogListItem, int64, error) {
	var (
		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	countQuery := s.db.WithContext(ctx).Table("attendance_logs al").
		Where("al.event_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)
	query := s.db.WithContext(ctx).Table("attendance_logs al").
		Joins("LEFT JOIN employees e ON e.id = al.employee_id").
		Joins("LEFT JOIN stores s ON s.id = al.store_id").
		Where("al.event_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)

	if params.StoreId != "" {
		countQuery = countQuery.Where("al.store_id = ?", params.StoreId)
		query = query.Where("al.store_id = ?", params.StoreId)
	}

	if params.EmployeeId != "" {
		countQuery = countQuery.Where("al.employee_id = ?", params.EmployeeId)
		query = query.Where("al.employee_id = ?", params.EmployeeId)
	}

	if params.EventType != "" {
		countQuery = countQuery.Where("al.event_type = ?", params.EventType)
		query = query.Where("al.event_type = ?", params.EventType)
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
			al.face_id_url,
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
