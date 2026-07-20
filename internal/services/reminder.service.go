package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
)

// CreateReminder - admin tomonidan bir yoki bir nechta aptekaga matnli eslatma yuboradi.
// Yozuv DB ga saqlanadi (created_by - token orqali olingan user_id), so'ngra belgilangan
// har bir do'kon uchun websocket orqali real vaqtda xabar yuboriladi.
func (s *Services) CreateReminder(ctx context.Context, req *domain.CreateReminderRequest, createdBy string) (*domain.Reminder, error) {
	fromDate, err := time.Parse(time.RFC3339, req.FromDate)
	if err != nil {
		s.log.Errorf("could not parse reminder from_date: %v", err)
		return nil, domain.InvalidTimeFormatError
	}

	toDate, err := time.Parse(time.RFC3339, req.ToDate)
	if err != nil {
		s.log.Errorf("could not parse reminder to_date: %v", err)
		return nil, domain.InvalidTimeFormatError
	}

	if !toDate.After(fromDate) {
		return nil, domain.ReminderDateRangeError
	}

	if toDate.Before(time.Now()) {
		return nil, domain.ReminderExpiredDateError
	}

	storeIds := uniqueNonEmptyIds(req.StoreIds)
	if len(storeIds) == 0 {
		return nil, domain.StoreIdsRequiredError
	}

	var existingCount int64
	if err := s.db.WithContext(ctx).
		Table("stores").
		Where("id IN ?", storeIds).
		Count(&existingCount).Error; err != nil {
		s.log.Errorf("could not validate stores for reminder: %v", err)
		return nil, domain.InternalServerError
	}
	if int(existingCount) != len(storeIds) {
		return nil, domain.NotFoundError
	}

	reminder := domain.Reminder{
		Id:        uuid.New().String(),
		Text:      strings.TrimSpace(req.Text),
		FromDate:  fromDate,
		ToDate:    toDate,
		StoreIds:  pq.StringArray(storeIds),
		CreatedBy: createdBy,
		IsActive:  true,
	}

	if err := s.db.WithContext(ctx).Create(&reminder).Error; err != nil {
		s.log.Errorf("could not create reminder: %v", err)
		return nil, domain.InternalServerError
	}

	// tanlangan har bir apteka uchun real vaqtda websocket orqali xabar yuborish
	s.NotifyReminderCreated(&reminder)

	return &reminder, nil
}

// uniqueNonEmptyIds - bo'sh va takrorlanuvchi id larni olib tashlaydi
func uniqueNonEmptyIds(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

// GetReminderList - eslatmalar ro'yxati.
// params.StoreId berilsa faqat shu aptekaga tegishli eslatmalar qaytariladi.
// params.IsActive=true bo'lsa faqat hozirgi vaqt from_date-to_date oralig'ida bo'lgan
// eslatmalar, params.IsActive=false bo'lsa hozirgi vaqt shu oraliqda bo'lmagan (hali
// boshlanmagan yoki muddati o'tgan) eslatmalar qaytariladi. Berilmasa (nil) sana bo'yicha
// filtr qo'llanmaydi. Delete qilingan (soft delete) eslatmalar bu holatlarning barchasida
// hech qachon qaytarilmaydi.
func (s *Services) GetReminderList(ctx context.Context, params *domain.ReminderQueryParams) ([]domain.ReminderListItem, int64, error) {
	countQuery := s.db.WithContext(ctx).Table("reminders r").Where("r.deleted_at IS NULL")
	query := s.db.WithContext(ctx).Table("reminders r").
		Joins("LEFT JOIN employees e ON e.id = r.created_by").
		Where("r.deleted_at IS NULL")

	if params.StoreId != "" {
		countQuery = countQuery.Where("? = ANY(r.store_ids)", params.StoreId)
		query = query.Where("? = ANY(r.store_ids)", params.StoreId)
	}

	if params.IsActive != nil {
		now := time.Now()
		if *params.IsActive {
			countQuery = countQuery.Where("r.from_date <= ? AND r.to_date >= ?", now, now)
			query = query.Where("r.from_date <= ? AND r.to_date >= ?", now, now)
		} else {
			countQuery = countQuery.Where("r.from_date > ? OR r.to_date < ?", now, now)
			query = query.Where("r.from_date > ? OR r.to_date < ?", now, now)
		}
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		s.log.Errorf("could not count reminders: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if total == 0 {
		return []domain.ReminderListItem{}, 0, nil
	}

	query = query.Select(`
			r.id,
			r.text,
			r.from_date,
			r.to_date,
			r.store_ids,
			r.created_by,
			e.full_name AS created_by_name,
			(r.from_date <= NOW() AND r.to_date >= NOW()) AS is_active,
			r.created_at,
			r.deleted_at
		`).
		Order("r.created_at DESC")

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.ReminderListItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get reminder list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.ReminderListItem{}
	}

	return results, total, nil
}

// DeleteReminder - eslatmani soft delete qiladi: is_active=false va deleted_at
// joriy vaqtga o'rnatiladi, qatordan jismonan o'chirilmaydi.
func (s *Services) DeleteReminder(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).
		Model(&domain.Reminder{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"is_active":  false,
			"deleted_at": time.Now(),
		})

	if result.Error != nil {
		s.log.Errorf("could not delete reminder: %v", result.Error)
		return domain.InternalServerError
	}

	if result.RowsAffected == 0 {
		return domain.NotFoundError
	}

	return nil
}
