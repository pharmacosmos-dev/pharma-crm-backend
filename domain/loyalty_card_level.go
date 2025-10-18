package domain

import "time"

type LoyaltyCardLevel struct {
	Id              string     `gorm:"id" json:"id"`
	Name            string     `gorm:"name" json:"name"`
	MinSpent        float64    `gorm:"min_spent" json:"min_spent"`
	CashbackPercent int        `gorm:"cashback_percent" json:"cashback_percent"`
	Position        int        `gorm:"position" json:"position"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `gorm:"deleted_at" json:"deleted_at"`
}
