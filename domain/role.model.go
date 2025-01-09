package domain

import "time"

type Role struct {
	Id              string     `gorm:"id" json:"id"`
	PublicID        int        `gorm:"public_id" json:"public_id"`
	Name            string     `gorm:"name" json:"name"`
	PermissionCount int        `gorm:"permission_count" json:"permission_count"`
	Description     string     `gorm:"description" json:"description"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
}

// RoleRequest structure for create, update
type RoleRequest struct {
	Id          string              `gorm:"id" json:"-"`
	PublicId    int                 `gorm:"public_id" json:"-"`
	Name        string              `gorm:"name" json:"name"`
	Description string              `gorm:"description" json:"description"`
	Permissions []RolePermissionReq `json:"permissions"`
}

// RoleUpdateRequest structure for update
type RoleUpdateRequest struct {
	Name        string              `gorm:"name" json:"name"`
	Description string              `gorm:"description" json:"description"`
	Permissions []RolePermissionReq `gorm:"-" json:"permissions"`
}

// RolePermissionRequest structure for create, update
type RolePermissionReq struct {
	RoleID       string   `gorm:"role_id" json:"-"`
	PermissionId string   `gorm:"permission_id" json:"parent_id"`
	IsActive     bool     `gorm:"is_active" json:"is_active"`
	ChildIds     []string `json:"children_ids"`
}
