package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CategoryRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewCategoryRepository(db *gorm.DB, log *logger.Logger) *CategoryRepo {
	return &CategoryRepo{db: db, log: log}
}

// Create a new category
func (r *CategoryRepo) Create(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	req.Id = uuid.New().String() // Generate a new UUID for the category
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		r.log.Error("Failed to create category: ", err)
		return nil, err
	}
	return req, nil
}

// Get a category by ID
func (r *CategoryRepo) Get(ctx context.Context, Id string) (*domain.Category, error) {
	c := &domain.Category{}
	if err := r.db.WithContext(ctx).First(c, "id = ?", Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Category not found:", Id)
			return nil, nil // Return nil if not found
		}
		r.log.Error("Failed to get category:", err)
		return nil, err
	}
	return c, nil
}

// GetList fetches multiple categories with pagination
func (r *CategoryRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Category, error) {
	var categories []*domain.Category
	if err := r.db.WithContext(ctx).Limit(req.Limit).Offset(req.Offset).Find(&categories).Order("created_at DESC").Error; err != nil {
		r.log.Error("Failed to list categories:", err)
		return nil, err
	}
	return categories, nil
}

// Update a category
func (r *CategoryRepo) Update(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	// `Updates` method will only update non-zero fields
	if err := r.db.WithContext(ctx).Model(&domain.Category{}).Where("id = ?", req.Id).Updates(req).Error; err != nil {
		r.log.Error("Failed to update category:", err)
		return nil, err
	}
	return req, nil
}

// Delete a category by ID
func (r *CategoryRepo) Delete(ctx context.Context, Id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Category{}, "id = ?", Id).Error; err != nil {
		r.log.Error("Failed to delete category:", err)
		return err
	}
	return nil
}
