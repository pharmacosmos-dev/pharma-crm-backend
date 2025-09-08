package domain

import "time"

type AsilBelgiTokenRequest struct {
	Token     string `json:"token" binding:"required"`
	ExpiresAt string `json:"expires_at"`
}

type AsilBelgiToken struct {
	ID        string    `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"not null" json:"token"`
	IssuedAt  time.Time `gorm:"default:now()" json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
