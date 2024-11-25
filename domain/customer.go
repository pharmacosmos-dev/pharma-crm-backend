package domain

import "time"

type Customer struct {
	Id        string     `gorm:"id" json:"id"`
	Name      string     `gorm:"name" json:"name"`
	Email     string     `gorm:"email" json:"email"`
	Phone     string     `gorm:"phone" json:"phone"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type CustomerRequest struct {
	Id    string `gorm:"id" json:"-"`
	Name  string `gorm:"name" json:"name"`
	Email string `gorm:"email" json:"email"`
	Phone string `gorm:"phone" json:"phone"`
}
