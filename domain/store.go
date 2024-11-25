package domain

import "time"

// Store structure
type Store struct {
	Id        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	Location  string     `gorm:"location" json:"location"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Store Create Request
type StoreRequest struct {
	Id       string `gorm:"id" json:"-"`
	Name     string `gorm:"name" json:"name"`
	Location string `gorm:"location" json:"location"`
}
