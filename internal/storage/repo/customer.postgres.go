package repo

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type customerRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewCustomerRepository(db *gorm.DB, log *logger.Logger) *customerRepo {
	return &customerRepo{db: db, log: log}
}

func (r *customerRepo) Create(ctx context.Context, req *domain.Customer) (*domain.Customer, error) {
	c := &domain.Customer{}
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		r.log.Error("Failed to create customer: ", err)
		return nil, err
	}
	return c, nil
}

func (r *customerRepo) Get(ctx context.Context, Id string) (*domain.Customer, error) {
	c := &domain.Customer{}
	if err := r.db.WithContext(ctx).First(c, "id = ?", Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Customer not found:", Id)
			return nil, nil // Return nil if not found
		}
		r.log.Error("Failed to get customer:", err)
		return nil, err
	}
	return c, nil
}

func (r *customerRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Customer, error) {
	customers := []*domain.Customer{}
	if err := r.db.WithContext(ctx).Limit(req.Limit).Offset(req.Offset).Find(&customers).Error; err != nil {
		r.log.Error("Failed to list customers:", err)
		return nil, err
	}
	return customers, nil
}

func (r *customerRepo) Update(ctx context.Context, req *domain.Customer) (*domain.Customer, error) {
	c := &domain.Customer{}
	return c, nil
}

func (r *customerRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
