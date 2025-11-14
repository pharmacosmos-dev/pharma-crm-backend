package domain

import "time"

type LoyaltyCardLevelupHistory struct {
	Id                 string     `gorm:"id" json:"id"`
	CustomerId         string     `gorm:"customer_id" json:"customer_id"`
	LoyaltyCardLevelId string     `gorm:"loyalty_card_level_id" json:"loyalty_card_level_id"`
	TotalSpent         float64    `gorm:"total_spent" json:"total_spent"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
}

func (LoyaltyCardLevelupHistory) TableName() string {
	return "loyalty_card_levelup_history"
}
