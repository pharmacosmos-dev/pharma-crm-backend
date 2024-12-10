package domain

import "time"

type Brand struct {
	Id        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	Logo      string     `gorm:"logo" json:"logo"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type BrandRequest struct {
	Id   string `gorm:"id" json:"-"`
	Name string `gorm:"name" json:"name"`
	Logo string `gorm:"logo" json:"logo"`
}
