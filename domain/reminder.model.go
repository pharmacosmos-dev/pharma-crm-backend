package domain

import (
	"time"

	"github.com/lib/pq"
)

// Reminder — admin tomonidan bir yoki bir nechta aptekaga yuboriladigan,
// belgilangan vaqt oralig'ida (from_date - to_date) frontendda ovozli eslatma
// sifatida ko'rsatiladigan matnli xabar.
type Reminder struct {
	Id        string         `gorm:"column:id" json:"id"`
	Text      string         `gorm:"column:text" json:"text"`
	FromDate  time.Time      `gorm:"column:from_date" json:"from_date"`
	ToDate    time.Time      `gorm:"column:to_date" json:"to_date"`
	StoreIds  pq.StringArray `gorm:"type:text[];column:store_ids" json:"store_ids"`
	CreatedBy string         `gorm:"column:created_by" json:"created_by"`
	CreatedAt *time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt *time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

func (Reminder) TableName() string {
	return "reminders"
}

// CreateReminderRequest — eslatma yaratish uchun so'rov.
// from_date/to_date RFC3339 formatida bo'lishi kerak (masalan: 2026-07-20T09:00:00+05:00)
type CreateReminderRequest struct {
	Text     string   `json:"text" binding:"required"`
	FromDate string   `json:"from_date" binding:"required" example:"2026-07-20T09:00:00+05:00"`
	ToDate   string   `json:"to_date" binding:"required" example:"2026-07-20T18:00:00+05:00"`
	StoreIds []string `json:"store_ids" binding:"required,min=1"`
}

// ReminderListItem — GET list javobi uchun
type ReminderListItem struct {
	Id            string         `json:"id"`
	Text          string         `json:"text"`
	FromDate      time.Time      `json:"from_date"`
	ToDate        time.Time      `json:"to_date"`
	StoreIds      pq.StringArray `gorm:"type:text[]" json:"store_ids"`
	CreatedBy     string         `json:"created_by"`
	CreatedByName string         `json:"created_by_name"`
	CreatedAt     *time.Time     `json:"created_at"`
}

// ReminderQueryParams — GET list uchun filter parametrlari.
// Active=true bo'lsa faqat muddati (to_date) hali o'tmagan eslatmalar qaytariladi.
type ReminderQueryParams struct {
	StoreId string `form:"store_id"`
	Active  *bool  `form:"active"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}
