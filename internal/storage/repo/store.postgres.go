package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type StoreRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewStoreRepository(db *gorm.DB, log *logger.Logger) *StoreRepo {
	return &StoreRepo{db: db, log: log}
}

// Create a new store
func (r *StoreRepo) Create(ctx context.Context, req *domain.Store) (*domain.Store, error) {
	req.Id = uuid.New().String() // Generate a new UUID for the store ID
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return req, nil
}

// Get a store by ID
func (r *StoreRepo) Get(ctx context.Context, id string) (*domain.Store, error) {
	store := &domain.Store{}
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(store).Error; err != nil {
		return nil, err
	}
	return store, nil
}

// GetList returns a paginated list of stores
func (r *StoreRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Store, error) {
	var stores []*domain.Store
	if err := r.db.WithContext(ctx).
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&stores).Error; err != nil {
		return nil, err
	}
	return stores, nil
}

// Update an existing store
func (r *StoreRepo) Update(ctx context.Context, req *domain.Store) error {
	// Using `Updates` for partial updates
	if err := r.db.WithContext(ctx).Model(&domain.Store{}).Where("id = ?", req.Id).Updates(map[string]interface{}{
		"name":     req.Name,
		"location": req.Location,
	}).Error; err != nil {
		return err
	}
	return nil
}

// Delete a store by ID
func (r *StoreRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Store{}).Error; err != nil {
		return err
	}
	return nil
}
