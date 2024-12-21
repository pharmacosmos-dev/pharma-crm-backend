package domain

import "time"

// Import structure
type Import struct {
	Id             string     `gorm:"id" json:"id"`
	PublicID       int        `gorm:"public_id" json:"public_id"`
	StoreID        string     `gorm:"store_id" json:"store_id"`
	Status         string     `gorm:"status" json:"status"`
	ImportDate     *time.Time `gorm:"import_date" json:"import_date"`
	AcceptedAmount float64    `gorm:"accepted_amount" json:"accepted_amount"`
	ReceivedAmount float64    `gorm:"received_amount" json:"received_amount"`
	ReceivedCount  int        `gorm:"received_count" json:"received_count"`
	AcceptedCount  int        `gorm:"accepted_count" json:"accepted_count"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Stores         *Store     `gorm:"foreignKey:StoreID" json:"stores"`
}

// ImportRequest structure
type ImportRequest struct {
	StoreID    string `gorm:"store_id" json:"store_id"`
	PublicID   int    `gorm:"public_id" json:"-"`
	Status     string `gorm:"status" json:"status"`
	ImportDate string `gorm:"import_date" json:"import_date"`
}

type ImportDetail struct {
	Id             string     `gorm:"id" json:"id"`
	ImportID       string     `gorm:"import_id" json:"import_id"`
	ProductID      string     `gorm:"product_id" json:"product_id"`
	ReceivedCount  int        `gorm:"received_count" json:"received_count"`
	AcceptedCount  int        `gorm:"accepted_count" json:"accepted_count"`
	CanceledCount  int        `gorm:"canceled_count" json:"canceled_count"`
	ReceivedAmount float64    `gorm:"received_amount" json:"received_amount"`
	AcceptedAmount float64    `gorm:"accepted_amount" json:"accepted_amount"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
}

type ImportDetailRequest struct {
	ImportID      string `gorm:"import_id" json:"import_id"`
	ProductID     string `gorm:"product_id" json:"product_id"`
	ReceivedCount int    `gorm:"received_count" json:"received_count"`
	AcceptedCount int    `gorm:"accepted_count" json:"accepted_count"`
	CanceledCount int    `gorm:"canceled_count" json:"canceled_count"`
}
