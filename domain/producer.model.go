package domain

import "time"

// Producer structure
type Producer struct {
	Id        string     `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// Shelf structure
type Shelf struct {
	Id        string     `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}
