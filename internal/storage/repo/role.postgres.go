package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type RoleRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewRoleRepository(db *gorm.DB, log *logger.Logger) *RoleRepo {
	return &RoleRepo{db: db, log: log}
}

// Create a new role
func (r *RoleRepo) Create(ctx context.Context, req *domain.Role) (*domain.Role, error) {
	req.Id = uuid.New().String() // Generate a new UUID for the role ID
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return req, nil
}

// Get a role by ID
func (r *RoleRepo) Get(ctx context.Context, id string) (*domain.Role, error) {
	role := &domain.Role{}
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.log.Error("Role not found:", id)
			return nil, nil // Return nil if not found
		}
		r.log.Error("Failed to get role:", err)
		return nil, err
	}
	return role, nil
}

// GetList returns a paginated list of roles
func (r *RoleRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Role, error) {
	var roles []*domain.Role
	if err := r.db.WithContext(ctx).
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// Update an existing role
func (r *RoleRepo) Update(ctx context.Context, req *domain.Role) error {
	// Ensure the record exists, then update
	if err := r.db.WithContext(ctx).Model(&domain.Role{}).Where("id = ?", req.Id).Updates(map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
	}).Error; err != nil {
		return err
	}
	return nil
}

// Delete a role by ID
func (r *RoleRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Role{}).Error; err != nil {
		return err
	}
	return nil
}
