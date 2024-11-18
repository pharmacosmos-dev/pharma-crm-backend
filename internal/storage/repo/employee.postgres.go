package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type EmployeeRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewEmployeeRepository(db *gorm.DB, log *logger.Logger) *EmployeeRepo {
	return &EmployeeRepo{db: db, log: log}
}

func (r *EmployeeRepo) Create(ctx context.Context, req *domain.Employee) (*domain.Employee, error) {
	req.Id = uuid.New().String()
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		r.log.Error("Failed to create employee: ", err)
		return nil, err
	}
	return req, nil
}

func (r *EmployeeRepo) Get(ctx context.Context, id string) (*domain.Employee, error) {
	e := &domain.Employee{}
	if err := r.db.WithContext(ctx).First(e, "id=?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Employee not found:", id)
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return e, nil
}

func (r *EmployeeRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	if err := r.db.WithContext(ctx).Limit(param.Limit).Offset(param.Offset).Find(&employees).Error; err != nil {
		r.log.Error("Failed to list employee:", err)
		return nil, err
	}
	return employees, nil
}

func (r *EmployeeRepo) Update(ctx context.Context, req *domain.Employee) (*domain.Employee, error) {
	res := &domain.Employee{}
	if err := r.db.WithContext(ctx).Model(res).Where("id = ?", req.Id).Updates(req).Error; err != nil {
		r.log.Error("Failed to update employee: ", err)
		return nil, err
	}
	return res, nil
}

func (r *EmployeeRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete("id = ?", id).Error; err != nil {
		r.log.Error("Failed to delete employee: ", err)
		return err
	}
	return nil
}

func (r *EmployeeRepo) CheckField(ctx context.Context, field, value string) (bool, error) {
	var count int64
	// Use GORM's `Where` clause to build the query dynamically
	if err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where(fmt.Sprintf("%s = ?", field), value).
		Count(&count).Error; err != nil {
		r.log.Error("Failed to check field:", err)
		return false, err
	}

	return count > 0, nil
}

func (r *EmployeeRepo) GetRoles(ctx context.Context, req *domain.Login) (*domain.LoginResponse, error) {
	e := &domain.Employee{}
	res := &domain.LoginResponse{}
	if err := r.db.WithContext(ctx).First(e, "phone=?", req.Phone).Preload("roles").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Employee not found:", req.Phone)
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	res.Employee = *e
	return res, nil
}
