package domain

import "time"

type Brand struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	Name      string     `gorm:"name" json:"name" db:"name"`
	Logo      string     `gorm:"logo" json:"logo" db:"logo"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
