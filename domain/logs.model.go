package domain

import "time"

type Log struct {
	Id           string     `json:"id" gorm:"id"`
	ProviderType string     `json:"provider_type" gorm:"provider_type"`
	Method       string     `json:"method" gorm:"method"`
	Payload      string     `json:"payload" gorm:"payload"`
	Response     string     `json:"response" gorm:"response"`
	CreatedAt    *time.Time `json:"created_at" gorm:"created_at"`
}

type LogParams struct {
	ProviderType string `form:"provider_type"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}
