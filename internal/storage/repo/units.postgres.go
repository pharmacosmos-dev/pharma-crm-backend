package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type UnitRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewUnitRepository(db *gorm.DB, log *logger.Logger) *UnitRepo {
	return &UnitRepo{db: db, log: log}
}

// Create a new unit
func (r *UnitRepo) Create(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	req.Id = uuid.New().String() // Generate a new UUID for the unit ID
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return req, nil
}

// Get a unit by ID
func (r *UnitRepo) Get(ctx context.Context, id string) (*domain.Unit, error) {
	unit := &domain.Unit{}
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(unit).Error; err != nil {
		return nil, err
	}
	return unit, nil
}

// GetList returns a paginated list of units
func (r *UnitRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Unit, error) {
	var units []*domain.Unit
	if err := r.db.WithContext(ctx).
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&units).Error; err != nil {
		return nil, err
	}
	return units, nil
}

// Update an existing unit
func (r *UnitRepo) Update(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	if err := r.db.WithContext(ctx).Model(&domain.Unit{}).Where("id = ?", req.Id).Updates(map[string]interface{}{
		"unit": req.Unit,
	}).Error; err != nil {
		return nil, err
	}
	// Return the updated unit
	updatedUnit := &domain.Unit{}
	if err := r.db.WithContext(ctx).Where("id = ?", req.Id).First(updatedUnit).Error; err != nil {
		return nil, err
	}
	return updatedUnit, nil
}

// Delete a unit by ID
func (r *UnitRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Unit{}).Error; err != nil {
		return err
	}
	return nil
}
