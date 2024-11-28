package domain

import "time"

// Category structure
type Category struct {
	Id         string     `gorm:"id" json:"id"`
	Name       string     `gorm:"name" json:"name"`
	CategoryId *string    `gorm:"category_id" json:"category_id"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
	Category   *Category  `gorm:"foreignKey:CategoryId" json:"category"`
}

// Category create request
type CategoryRequest struct {
	Id         string `gorm:"id" json:"-"`
	Name       string `gorm:"name" json:"name"`
	CategoryId string `gorm:"category_id" json:"category_id"`
}
