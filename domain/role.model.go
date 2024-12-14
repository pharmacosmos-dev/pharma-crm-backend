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
	Id          string `gorm:"id" json:"-"`
	PublicId    int    `gorm:"public_id" json:"-"`
	Name        string `gorm:"name" json:"name"`
	Description string `gorm:"description" json:"description"`
}
