package domain

import "time"

type Store struct {
	Id        string     `gorm:"id" json:"id" db:"id"`
	Name      string     `gorm:"name" json:"name" db:"name"`
	Location  string     `gorm:"location" json:"location" db:"location"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at" db:"updated_at"`
}
