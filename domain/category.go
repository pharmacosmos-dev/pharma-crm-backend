package domain

import "time"

type Category struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	Name      string     `gorm:"id" json:"name" db:"name"`
	CreatedAt *time.Time `gorm:"id" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"id" json:"updated_at" db:"updated_at"`
}
