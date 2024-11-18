package domain

import "time"

type Role struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	Name      string     `gorm:"name" json:"name" db:"name"`
	Desc      string     `gorm:"description" json:"description" db:"description"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
