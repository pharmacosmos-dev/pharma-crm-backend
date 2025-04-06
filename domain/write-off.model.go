package domain

import "time"

// write off structure
type WriteOff struct {
	Id        string     `gorm:"id" json:"id"`
	PublicId  int64      `gorm:"public_id" json:"public_id"`
	StoreId   string     `gorm:"store_id" json:"store_id"`
	Status    string     `gorm:"status" json:"status"`
	Comment   string     `gorm:"comment" json:"comment"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

// write off create request
type WriteOffRequest struct {
	PublicId int64  `gorm:"public_id" json:"public_id"`
	StoreId  string `gorm:"store_id" json:"store_id"`
	Status   string `gorm:"status" json:"status"`
	Comment  string `gorm:"comment" json:"comment"`
}
