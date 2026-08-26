package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// Bayram kunlari. Payroll ish kunlarini shu jadvaldan hisoblaydi, shuning uchun
// bu yerdagi har bir o'zgarish keyingi qayta hisobda KPI foiziga ta'sir qiladi
// (qarang: employee_payroll.service.go, workdays CTE).

// region Create

// CreateHoliday — yangi bayram kuni qo'shadi.
// Sana takrorlansa (jadvalda UNIQUE) tushunarli xato qaytariladi.
func (s *Services) CreateHoliday(ctx context.Context, req *domain.HolidayRequest) (*domain.Holiday, error) {
	date, err := normalizeHolidayDate(req.Date)
	if err != nil {
		s.log.Errorf("holiday: %v", err)
		return nil, domain.BadRequestError
	}

	holiday := domain.Holiday{
		Id:   uuid.New().String(),
		Date: date,
		Name: req.Name,
	}

	if err := s.db.WithContext(ctx).Create(&holiday).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.AlreadyExistsError
		}
		s.log.Errorf("holiday: could not create: %v", err)
		return nil, domain.InternalServerError
	}

	return &holiday, nil
}

// region Get

// GetHolidays — bayramlar ro'yxati. year berilsa o'sha yil, year+month berilsa
// o'sha oy bilan cheklanadi.
func (s *Services) GetHolidays(
	ctx context.Context, params *domain.HolidayQueryParams,
) ([]domain.Holiday, int64, error) {
	// Count va Find uchun so'rov qaytadan quriladi: GORM'da finisher (Count)
	// chaqirilgandan keyin o'sha *gorm.DB'ni qayta ishlatish statement'ni ifloslantiradi.
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&domain.Holiday{})
		if params.Year != 0 {
			q = q.Where("EXTRACT(YEAR FROM date) = ?", params.Year)
		}
		if params.Month != 0 {
			q = q.Where("EXTRACT(MONTH FROM date) = ?", params.Month)
		}
		if params.Search != "" {
			q = q.Where("name ILIKE ?", fmt.Sprintf("%%%s%%", params.Search))
		}
		return q
	}

	var totalCount int64
	if err := newQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("holiday: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var holidays []domain.Holiday
	err := newQuery().
		Order("date").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&holidays).Error
	if err != nil {
		s.log.Errorf("holiday: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return holidays, totalCount, nil
}

// region Update

// UpdateHoliday — bayram sanasi yoki nomini o'zgartiradi.
func (s *Services) UpdateHoliday(
	ctx context.Context, id string, req *domain.HolidayRequest,
) (*domain.Holiday, error) {
	date, err := normalizeHolidayDate(req.Date)
	if err != nil {
		s.log.Errorf("holiday: %v", err)
		return nil, domain.BadRequestError
	}

	result := s.db.WithContext(ctx).
		Model(&domain.Holiday{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"date":       date,
			"name":       req.Name,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return nil, domain.AlreadyExistsError
		}
		s.log.Errorf("holiday: could not update: %v", result.Error)
		return nil, domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return nil, domain.ResourceNotFoundError
	}

	var holiday domain.Holiday
	if err := s.db.WithContext(ctx).Take(&holiday, "id = ?", id).Error; err != nil {
		s.log.Errorf("holiday: could not read back: %v", err)
		return nil, domain.InternalServerError
	}

	return &holiday, nil
}

// region Delete

// DeleteHoliday — bayramni butunlay o'chiradi (soft delete emas: jadval kichik
// va o'chirilgan sana ish kunlari hisobiga qaytishi kerak).
func (s *Services) DeleteHoliday(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&domain.Holiday{}, "id = ?", id)
	if result.Error != nil {
		s.log.Errorf("holiday: could not delete: %v", result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return domain.ResourceNotFoundError
	}
	return nil
}

// region Helpers

// normalizeHolidayDate — "YYYY-MM-DD" ni tekshiradi. Vaqt bilan (RFC3339) kelsa
// ham qabul qilinadi, faqat sana qismi olinadi.
func normalizeHolidayDate(value string) (string, error) {
	if t, err := time.Parse(constants.TimeOnlyDateFormat, value); err == nil {
		return t.Format(constants.TimeOnlyDateFormat), nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format(constants.TimeOnlyDateFormat), nil
	}
	return "", fmt.Errorf("date must be YYYY-MM-DD: %s", value)
}

// isUniqueViolation — Postgres 23505 (unique_violation) xatosimi.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
