package domain

import "time"

type Role struct {
	Id          string     `gorm:"id" json:"id"`
	Name        string     `gorm:"name" json:"name"`
	Description string     `gorm:"description" json:"description"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at"`
}

type RoleRequest struct {
	Id          string `gorm:"id" json:"-"`
	Name        string `gorm:"name" json:"name"`
	Description string `gorm:"description" json:"description"`
}
