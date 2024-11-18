package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type BrandRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewBrandRepository(db *gorm.DB, log *logger.Logger) *BrandRepo {
	return &BrandRepo{db: db, log: log}
}

func (r *BrandRepo) Create(ctx context.Context, req *domain.Brand) (*domain.Brand, error) {
	req.Id = uuid.New().String()
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		r.log.Error("Failed to create brand: ", err)
		return nil, err
	}

	return req, nil
}

func (r *BrandRepo) Get(ctx context.Context, Id string) (*domain.Brand, error) {
	c := &domain.Brand{}
	if err := r.db.First(c, "id = ?", Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Brand not found:", Id)
			return nil, nil // Return nil if not found
		}
		r.log.Error("Failed to get brand:", err)
		return nil, err
	}
	return c, nil
}

func (r *BrandRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Brand, error) {
	brands := []*domain.Brand{}
	if err := r.db.WithContext(ctx).Limit(req.Limit).Offset(req.Offset).Find(&brands).Error; err != nil {
		r.log.Error("Failed to list brands:", err)
		return nil, err
	}
	return brands, nil
}

func (r *BrandRepo) Update(ctx context.Context, req *domain.Brand) (*domain.Brand, error) {
	if err := r.db.WithContext(ctx).Model(&domain.Brand{}).Where("id = ?").Updates(req).Error; err != nil {
		r.log.Error("Failed to update brand:", err)
		return nil, err
	}
	return req, nil
}

func (r *BrandRepo) Delete(ctx context.Context, Id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Brand{}, "id = ?", Id).Error; err != nil {
		r.log.Error("Failed to delete brand: ", err)
		return err
	}
	return nil
}
