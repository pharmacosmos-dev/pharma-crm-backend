package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// UpsertEmployeeFaceDescriptor — xodim uchun yuz descriptor(lar)ini yaratadi yoki,
// mavjud bo'lsa, yangilaydi. Har bir xodim uchun faqat bitta yozuv saqlanadi.
func (s *Services) UpsertEmployeeFaceDescriptor(ctx context.Context, employeeId string, descriptor [][]float64) (*domain.EmployeeFaceDescriptor, error) {
	var employeeExists bool
	if err := s.db.WithContext(ctx).
		Raw(`SELECT EXISTS(SELECT 1 FROM employees WHERE id = ? AND status != ?)`, employeeId, constants.GeneralStatusDeleted).
		Scan(&employeeExists).Error; err != nil {
		s.log.Errorf("could not check employee existence: %v", err)
		return nil, domain.InternalServerError
	}
	if !employeeExists {
		return nil, domain.NotFoundError
	}

	raw, err := json.Marshal(descriptor)
	if err != nil {
		s.log.Errorf("could not marshal face descriptor: %v", err)
		return nil, domain.InternalServerError
	}

	// employee_id ustunidagi UNIQUE constraint (migration 000230) tufayli
	// bu so'rov bitta xodim uchun bitta yozuvni atomik tarzda yaratadi yoki yangilaydi.
	var res domain.EmployeeFaceDescriptor
	err = s.db.WithContext(ctx).Raw(`
		INSERT INTO employee_face_descriptors (id, employee_id, descriptor, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
		ON CONFLICT (employee_id)
		DO UPDATE SET descriptor = EXCLUDED.descriptor, updated_at = NOW()
		RETURNING id, employee_id, descriptor, created_at, updated_at
	`, uuid.New().String(), employeeId, raw).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not upsert face descriptor: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// GetEmployeeFaceDescriptor — bitta xodimning saqlangan yuz descriptor yozuvini qaytaradi.
func (s *Services) GetEmployeeFaceDescriptor(ctx context.Context, employeeId string) (*domain.EmployeeFaceDescriptor, error) {
	var res domain.EmployeeFaceDescriptor
	err := s.db.WithContext(ctx).Take(&res, "employee_id = ?", employeeId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("could not get face descriptor: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

// DeleteEmployeeFaceDescriptor — xodimning saqlangan yuz descriptor yozuvini o'chiradi.
func (s *Services) DeleteEmployeeFaceDescriptor(ctx context.Context, employeeId string) error {
	result := s.db.WithContext(ctx).
		Where("employee_id = ?", employeeId).
		Delete(&domain.EmployeeFaceDescriptor{})
	if result.Error != nil {
		s.log.Errorf("could not delete face descriptor: %v", result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return domain.ResourceNotFoundError
	}
	return nil
}

// GetEmployeeFaceDescriptorList — filtrlar (store_id, company_id, employee_id) bo'yicha
// xodimlarning yuz descriptor yozuvlari ro'yxati (masalan, attendance qurilmasi tomonidan
// lokal solishtirish uchun oldindan yuklab olinadi).
func (s *Services) GetEmployeeFaceDescriptorList(ctx context.Context, params *domain.EmployeeFaceDescriptorQueryParams) ([]domain.EmployeeFaceDescriptorListItem, int64, error) {
	baseQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).
			Table("employee_face_descriptors efd").
			Joins("JOIN employees e ON e.id = efd.employee_id").
			Where("e.status != ?", constants.GeneralStatusDeleted)

		if params.StoreId != "" {
			q = q.Where("e.store_id = ?", params.StoreId)
		}
		if params.CompanyId != "" {
			q = q.Where("e.company_id = ?", params.CompanyId)
		}
		if params.EmployeeId != "" {
			q = q.Where("efd.employee_id = ?", params.EmployeeId)
		}
		return q
	}

	var total int64
	if err := baseQuery().Count(&total).Error; err != nil {
		s.log.Errorf("could not count face descriptors: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if total == 0 {
		return []domain.EmployeeFaceDescriptorListItem{}, 0, nil
	}

	query := baseQuery().Select(`
			efd.id,
			efd.employee_id,
			e.full_name AS employee_name,
			e.store_id,
			efd.descriptor,
			efd.updated_at
		`).
		Order("efd.updated_at DESC")

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.EmployeeFaceDescriptorListItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get face descriptor list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.EmployeeFaceDescriptorListItem{}
	}

	return results, total, nil
}
