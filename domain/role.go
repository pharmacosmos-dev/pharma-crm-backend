package domain

import "time"

type Role struct {
	Id          string     `gorm:"id" json:"id" db:"id"`
	Name        string     `gorm:"name" json:"name" db:"name"`
	NameUz      string     `gorm:"name_uz" json:"name_uz" db:"name_uz"`
	NameEn      string     `gorm:"name_en" json:"name_en" db:"name_en"`
	NameRu      string     `gorm:"name_ru" json:"name_ru" db:"name_ru"`
	Description string     `gorm:"description" json:"description" db:"description"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
