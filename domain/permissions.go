package domain

import "time"

type Permission struct {
	Id         string     `gorm:"id" json:"id"`
	EntityName string     `gorm:"entity_name" json:"entity_name"`
	Action     string     `gorm:"action" json:"action"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}
