package domain

import "time"

// Reserved - reserved document model
type Reserved struct {
	Id                string     `gorm:"id" json:"id"`
	StoreId           string     `gorm:"store_id" json:"store_id"`
	DocumentNumber    string     `gorm:"document_number" json:"document_number"`
	CreatedBy         *string    `gorm:"created_by" json:"created_by"`
	TotalQuantity     float64    `gorm:"total_quantity" json:"total_quantity"`
	TotalProductCount int        `gorm:"total_product_count" json:"total_product_count"`
	Status            string     `gorm:"status" json:"status"` // 'new', 'checking', 'done'
	Comment           *string    `gorm:"comment" json:"comment"`
	CreatedAt         *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `gorm:"updated_at" json:"updated_at"`
}

func (Reserved) TableName() string {
	return "reserved"
}

// ReservedDetails - reserved document details
type ReservedDetails struct {
	Id         string     `gorm:"id" json:"id"`
	ReservedId string     `gorm:"reserved_id" json:"reserved_id"`
	ProductId  string     `gorm:"product_id" json:"product_id"`
	Quantity   float64    `gorm:"quantity" json:"quantity"`
	CreatedBy  *string    `gorm:"created_by" json:"created_by"`
	UpdatedBy  *string    `gorm:"updated_by" json:"updated_by"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}

func (ReservedDetails) TableName() string {
	return "reserved_details"
}

// CreateReservedDocumentRequest - request to create reserved document
type CreateReservedDocumentRequest struct {
	DocumentNumber string  `json:"document_number" binding:"required"`
	Comment        *string `json:"comment"`
}

// AddReservedDetailsRequest - request to add/update reserved details
type AddReservedDetailsRequest struct {
	ProductId string  `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
}

// InsertReservedDetailsDirectRequest - request to insert reserved details directly (frontend can use this)
type InsertReservedDetailsDirectRequest struct {
	StoreId   string  `json:"store_id" binding:"required"`
	ProductId string  `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
}

// ListReservedDetailsRequest - query parameters for listing reserved details.
// Store ID is read from the signed user's token.
type ListReservedDetailsRequest struct {
	ReservedId string `form:"reserved_id" json:"reserved_id"`
}

// ReservedDetailsWithProduct - response model with product info
type ReservedDetailsWithProduct struct {
	ReservedDetails
	ProductName string `json:"product_name"`
	ProductCode string `json:"product_code"`
}
