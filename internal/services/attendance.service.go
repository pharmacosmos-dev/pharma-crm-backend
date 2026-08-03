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

	if params.Name != "" {
		countQuery = countQuery.Where("e.full_name ILIKE ?", "%"+params.Name+"%")
		query = query.Where("e.full_name ILIKE ?", "%"+params.Name+"%")
	}

	if params.Phone != "" {
		countQuery = countQuery.Where("e.phone ILIKE ?", "%"+params.Phone+"%")
		query = query.Where("e.phone ILIKE ?", "%"+params.Phone+"%")
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
			COALESCE(e.phone, '') AS employee_phone,
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

// AggregateEmployeeAttendanceDays — har kuni cron orqali chaqiriladi. Kechagi kun (Toshkent
// vaqti bo'yicha) uchun attendance_logs'dagi xom check-in/check-out voqealarini xodim
// bo'yicha yig'ib, employee_attendance_days'ga yozadi (mavjud qator bo'lsa yangilaydi).
// Faqat "Кассир" va "Заведующий" rolidagi xodimlar hisobga olinadi —
// rol nomi roles.name bo'yicha solishtiriladi, shu rollar keyinchalik qayta nomlansa
// shu yerdagi ro'yxatni ham yangilash kerak. planned_start_at va late_minutes faqat
// employees.start_date to'ldirilgan xodimlar uchun hisoblanadi — bo'lmasa NULL/0 qoladi.
// sales_amount — sales jadvalidan shu xodimning shu kundagi yakunlangan savdolari yig'indisi
// (sale_type='SALE' AND stage=9, boshqa servislarda ham shu filtr ishlatiladi).
// is_manual_override=true bo'lgan kunlarga HR tuzatishini yo'qotmaslik uchun tegilmaydi.
func (s *Services) AggregateEmployeeAttendanceDays() {
	err := s.db.Exec(`
		INSERT INTO employee_attendance_days
		    (id, employee_id, store_id, work_date, planned_start_at, first_check_in, last_check_out,
		     worked_minutes, late_minutes, sales_amount, is_absent, is_manual_override, created_at, updated_at)
		SELECT
		    uuid_generate_v4(),
		    l.employee_id,
		    l.store_id,
		    l.work_date,
		    CASE WHEN e.start_date IS NOT NULL
		         THEN (l.work_date + e.start_date) AT TIME ZONE 'Asia/Tashkent'
		         ELSE NULL END,
		    l.first_check_in,
		    l.last_check_out,
		    CASE WHEN l.first_check_in IS NOT NULL AND l.last_check_out IS NOT NULL
		         THEN GREATEST(0, ROUND(EXTRACT(EPOCH FROM (l.last_check_out - l.first_check_in)) / 60))::int
		         ELSE 0 END,
		    CASE WHEN e.start_date IS NOT NULL AND l.first_check_in IS NOT NULL
		         THEN GREATEST(0, ROUND(EXTRACT(EPOCH FROM (l.first_check_in - ((l.work_date + e.start_date) AT TIME ZONE 'Asia/Tashkent'))) / 60))::int
		         ELSE 0 END,
		    COALESCE(ds.sales_amount, 0),
		    FALSE,
		    FALSE,
		    NOW(),
		    NOW()
		FROM (
		    SELECT
		        employee_id,
		        (array_agg(store_id ORDER BY event_at))[1]            AS store_id,
		        (event_at + interval '5 hours')::date                 AS work_date,
		        MIN(event_at) FILTER (WHERE event_type = 'check-in')  AS first_check_in,
		        MAX(event_at) FILTER (WHERE event_type = 'check-out') AS last_check_out
		    FROM attendance_logs
		    WHERE (event_at + interval '5 hours')::date = (NOW() + interval '5 hours')::date - 1
		    GROUP BY employee_id, (event_at + interval '5 hours')::date
		) l
		JOIN employees e ON e.id = l.employee_id
		    AND e.is_active = TRUE
		    AND e.status = 'active'
		    AND e.deleted_at IS NULL
		    AND EXISTS (
		        SELECT 1 FROM employee_roles er
		        JOIN roles r ON r.id = er.role_id
		        WHERE er.employee_id = e.id
		          AND r.name IN ('Кассир', 'Заведующий')
		    )
		LEFT JOIN (
		    SELECT employee_id, SUM(total_amount) AS sales_amount
		    FROM sales
		    WHERE sale_type = 'SALE'
		      AND stage = 9
		      AND (created_at + interval '5 hours')::date = (NOW() + interval '5 hours')::date - 1
		    GROUP BY employee_id
		) ds ON ds.employee_id = l.employee_id
		ON CONFLICT (employee_id, work_date) DO UPDATE SET
		    store_id         = EXCLUDED.store_id,
		    planned_start_at = EXCLUDED.planned_start_at,
		    first_check_in   = EXCLUDED.first_check_in,
		    last_check_out   = EXCLUDED.last_check_out,
		    worked_minutes   = EXCLUDED.worked_minutes,
		    late_minutes     = EXCLUDED.late_minutes,
		    sales_amount     = EXCLUDED.sales_amount,
		    updated_at       = NOW()
		WHERE employee_attendance_days.is_manual_override = FALSE
	`).Error

	if err != nil {
		s.log.Errorf("cron AggregateEmployeeAttendanceDays: %v", err)
		return
	}

	// Kechagi kunga hech qanday attendance_logs yozuvi bo'lmagan faol xodimlar uchun
	// is_absent=true bilan qator yaratadi. Dam olish kuni yoki ruxsatli holatlarni HR
	// keyinroq is_manual_override orqali tuzatadi (ushbu so'rov manual tuzatilgan yoki
	// allaqachon check-in qilingan qatorlarga tegmaydi).
	err = s.db.Exec(`
		INSERT INTO employee_attendance_days
		    (id, employee_id, store_id, work_date, planned_start_at, first_check_in, last_check_out,
		     worked_minutes, late_minutes, sales_amount, is_absent, is_manual_override, created_at, updated_at)
		SELECT
		    uuid_generate_v4(),
		    e.id,
		    e.store_id,
		    (NOW() + interval '5 hours')::date - 1,
		    CASE WHEN e.start_date IS NOT NULL
		         THEN (((NOW() + interval '5 hours')::date - 1) + e.start_date) AT TIME ZONE 'Asia/Tashkent'
		         ELSE NULL END,
		    NULL,
		    NULL,
		    0,
		    0,
		    COALESCE(ds.sales_amount, 0),
		    TRUE,
		    FALSE,
		    NOW(),
		    NOW()
		FROM employees e
		LEFT JOIN (
		    SELECT employee_id, SUM(total_amount) AS sales_amount
		    FROM sales
		    WHERE sale_type = 'SALE'
		      AND stage = 9
		      AND (created_at + interval '5 hours')::date = (NOW() + interval '5 hours')::date - 1
		    GROUP BY employee_id
		) ds ON ds.employee_id = e.id
		WHERE e.is_active = TRUE
		  AND e.status = 'active'
		  AND e.deleted_at IS NULL
		  AND EXISTS (
		      SELECT 1 FROM employee_roles er
		      JOIN roles r ON r.id = er.role_id
		      WHERE er.employee_id = e.id
		        AND r.name IN ('Кассир', 'Заведующий')
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM attendance_logs al
		      WHERE al.employee_id = e.id
		        AND (al.event_at + interval '5 hours')::date = (NOW() + interval '5 hours')::date - 1
		  )
		ON CONFLICT (employee_id, work_date) DO UPDATE SET
		    is_absent    = TRUE,
		    sales_amount = EXCLUDED.sales_amount,
		    updated_at   = NOW()
		WHERE employee_attendance_days.is_manual_override = FALSE
		  AND employee_attendance_days.first_check_in IS NULL
	`).Error

	if err != nil {
		s.log.Errorf("cron AggregateEmployeeAttendanceDays (absent marking): %v", err)
		return
	}
	s.log.Infof("AggregateEmployeeAttendanceDays completed")
}

// GetEmployeeAttendanceDayList — kunlik davomat yig'indisi ro'yxati (employee_attendance_days),
// store_id, xodim ismi/telefoni va work_date oraliqi (start_date/end_date) filtrlari bilan.
func (s *Services) GetEmployeeAttendanceDayList(ctx context.Context, params *domain.EmployeeAttendanceDayQueryParams) ([]domain.EmployeeAttendanceDayListItem, int64, error) {
	countQuery := s.db.WithContext(ctx).Table("employee_attendance_days ead").
		Joins("LEFT JOIN employees e ON e.id = ead.employee_id")
	query := s.db.WithContext(ctx).Table("employee_attendance_days ead").
		Joins("LEFT JOIN employees e ON e.id = ead.employee_id").
		Joins("LEFT JOIN stores s ON s.id = ead.store_id")

	if params.StoreId != "" {
		countQuery = countQuery.Where("ead.store_id = ?", params.StoreId)
		query = query.Where("ead.store_id = ?", params.StoreId)
	}

	if params.Name != "" {
		countQuery = countQuery.Where("e.full_name ILIKE ?", "%"+params.Name+"%")
		query = query.Where("e.full_name ILIKE ?", "%"+params.Name+"%")
	}

	if params.Phone != "" {
		countQuery = countQuery.Where("e.phone ILIKE ?", "%"+params.Phone+"%")
		query = query.Where("e.phone ILIKE ?", "%"+params.Phone+"%")
	}

	if params.StartDate != "" {
		if _, err := time.Parse(constants.TimeOnlyDateFormat, params.StartDate); err != nil {
			return nil, 0, domain.InvalidTimeFormatError
		}
		countQuery = countQuery.Where("ead.work_date >= ?::date", params.StartDate)
		query = query.Where("ead.work_date >= ?::date", params.StartDate)
	}

	if params.EndDate != "" {
		if _, err := time.Parse(constants.TimeOnlyDateFormat, params.EndDate); err != nil {
			return nil, 0, domain.InvalidTimeFormatError
		}
		countQuery = countQuery.Where("ead.work_date <= ?::date", params.EndDate)
		query = query.Where("ead.work_date <= ?::date", params.EndDate)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		s.log.Errorf("could not count employee attendance days: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if total == 0 {
		return []domain.EmployeeAttendanceDayListItem{}, 0, nil
	}

	query = query.Select(`
			ead.id,
			ead.employee_id,
			COALESCE(e.full_name, '') AS employee_name,
			COALESCE(e.phone, '') AS employee_phone,
			ead.store_id,
			COALESCE(s.name, '') AS store_name,
			ead.work_date,
			ead.planned_start_at,
			ead.first_check_in,
			ead.last_check_out,
			ead.worked_minutes,
			ead.late_minutes,
			ead.sales_amount,
			ead.is_absent,
			ead.is_manual_override,
			ead.comment,
			ead.created_at,
			ead.updated_at
		`).
		Order("ead.work_date DESC, e.full_name ASC")

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.EmployeeAttendanceDayListItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get employee attendance day list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.EmployeeAttendanceDayListItem{}
	}

	return results, total, nil
}
