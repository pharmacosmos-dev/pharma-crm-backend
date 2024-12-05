package domain

import (
	"time"

	"github.com/google/uuid"
)

// Permission structure for using save permissions data
type Permission struct {
	Id         string     `gorm:"id" json:"id"`
	Route      string     `gorm:"route" json:"route"`
	Type       string     `gorm:"type" json:"type"`
	EntityName string     `gorm:"entity_name" json:"entity_name"`
	Action     string     `gorm:"action" json:"action"`
	ParentId   string     `gorm:"parent_id" json:"parent_id"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}

// PermissionRequest structure for create, update
type PermissionRequest struct {
	Id         string     `gorm:"id" json:"-"`
	Route      string     `gorm:"route" json:"route"`
	Type       string     `gorm:"type" json:"type"`
	EntityName string     `gorm:"entity_name" json:"entity_name"`
	Action     string     `gorm:"action" json:"action"`
	ParentId   *uuid.UUID `gorm:"parent_id" json:"parent_id,omitempty"`
}

// RolePermission structure for using attech permission to role
type RolePermission struct {
	ID           string     `gorm:"id" json:"id"`
	PermissionID string     `gorm:"permission_id" json:"permission_id"`
	RoleID       string     `gorm:"role_id" json:"role_id"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
}

// RolePermissionRequest for create, update
type RolePermissionRequest struct {
	ID           string `gorm:"id" json:"-"`
	PermissionID string `gorm:"permission_id" json:"permission_id"`
	RoleID       string `gorm:"role_id" json:"role_id"`
}
