package domain

import "time"

type Country struct {
	Id        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}
