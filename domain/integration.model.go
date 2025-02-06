package domain

import "time"

type Request1C struct {
	ID        *string    `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	Method    string     `gorm:"method" json:"method"`
	Payload   []byte     `gorm:"payload" json:"payload"`
	Response  []byte     `gorm:"response" json:"response"`
	Action    string     `gorm:"action" json:"action"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}
