package domain

import "time"

// Category structure
type Category struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	Name      string     `gorm:"name" json:"name" db:"name"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}

// Category create request
type CategoryRequest struct {
	Id   string `gorm:"id" json:"-"`
	Name string `gorm:"name" json:"name"`
}

// Category update request
type CategoryUpdateRequest struct {
	Id   string `gorm:"id" json:"id"`
	Name string `gorm:"name" json:"name"`
}
