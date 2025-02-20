package domain

import (
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Permission structure for using save permissions data
type Permission struct {
	Id          string            `gorm:"id" json:"id"`
	Route       string            `gorm:"route" json:"route"`
	Type        string            `gorm:"type" json:"type"`
	Name        string            `gorm:"name" json:"name"`
	Description string            `gorm:"description" json:"description"`
	ParentId    string            `gorm:"parent_id" json:"parent_id"`
	IsActive    bool              `gorm:"is_active" json:"is_active"`
	Method      utils.StringArray `gorm:"type:text[]" json:"method"`
	CreatedAt   *time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time        `gorm:"updated_at" json:"updated_at"`
	Children    []Permission      `gorm:"foreignKey:ParentId" json:"children"`
}

// PermissionRequest structure for create, update
type PermissionRequest struct {
	Id          string            `gorm:"id" json:"-"`
	Route       string            `gorm:"route" json:"route"`
	Type        string            `gorm:"type" json:"type"`
	Name        string            `gorm:"name" json:"name"`
	Description string            `gorm:"description" json:"description"`
	Method      utils.StringArray `gorm:"type:text[]" json:"method"`
	ParentId    *string           `gorm:"parent_id" json:"parent_id,omitempty"`
	Key         string            `gorm:"key" json:"key"`
}

// RolePermission structure for using attech permission to role
type RolePermission struct {
	ID           string     `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	PermissionID string     `gorm:"permission_id" json:"permission_id"`
	RoleID       string     `gorm:"role_id" json:"role_id"`
	IsActive     bool       `gorm:"is_active" json:"is_active"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// RolePermissionRequest for create, update
type RolePermissionRequest struct {
	ID           string `gorm:"id" json:"-"`
	PermissionID string `gorm:"permission_id" json:"permission_id"`
	RoleID       string `gorm:"role_id" json:"role_id"`
}

// Menu Permissions structure
type MainPermission struct {
	ID          string       `gorm:"id" json:"id"`
	Key         string       `gorm:"key" json:"key"`
	Permissions []Permission `gorm:"foreignKey:ParentId" json:"permissions"`
}
