package domain

import (
	"time"
)

type Customer struct {
	Id           string     `gorm:"id" json:"id"`
	StoreId      string     `gorm:"store_id" json:"store_id"`
	FirstName    string     `gorm:"first_name" json:"first_name"`
	LastName     string     `gorm:"last_name" json:"last_name"`
	FullName     string     `gorm:"full_name" json:"full_name"`
	PublicId     int        `gorm:"public_id" json:"public_id"`
	Phone        string     `gorm:"phone" json:"phone"`
	Birthday     string     `gorm:"birthday" json:"birthday" example:"2006-01-02"`
	Gender       string     `gorm:"gender" json:"gender" example:"male/female"`
	Balance      float64    `gorm:"balance" json:"balance"`
	TagId        string     `gorm:"tag_id" json:"tag_id"`
	Tag          *Tag       `gorm:"foreignKey:TagId" json:"tag"`
	DiscountCard string     `gorm:"discount_card" json:"discount_card"`
	CreatedBy    string     `gorm:"-" json:"created_by"`
	UpdatedBy    string     `gorm:"-" json:"updated_by"`
	DeletedBy    string     `gorm:"-" json:"deleted_by"`
	CreatedAt    *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `gorm:"updated_at" json:"updated_at"`
	SaleDate     *time.Time `gorm:"sale_date" json:"sale_date"`
	SaleAmount   float64    `gorm:"sale_amount" json:"sale_amount"`
	DebtAmount   float64    `gorm:"debt_amount" json:"debt_amount"`
	Store        *Store     `gorm:"foreignKey:StoreId" json:"store"`
}

type DiscountCardWithCustomer struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Barcode    string `json:"barcode"`
	Percent    int    `json:"percent"`

	// Customer info
	FullName string    `json:"full_name"`
	Phone    string    `json:"phone"`
	Birthday time.Time `json:"birthday"`
	Gender   string    `json:"gender"`
	Balance  float64   `json:"balance"`

	// Store/Tag info
	StoreID   string `json:"store_id"`
	StoreName string `json:"store_name"`
	TagID     string `json:"tag_id"`
	TagName   string `json:"tag_name"`

	CreatedAt time.Time `json:"created_at"`
}

type CustomerRequest struct {
	Id        string  `gorm:"id" json:"-"`
	StoreId   string  `gorm:"store_id" json:"store_id"`
	FirstName string  `gorm:"first_name" json:"first_name"`
	LastName  string  `gorm:"last_name" json:"last_name,omitempty"`
	FullName  string  `gorm:"full_name" json:"full_name,omitempty"`
	Phone     string  `gorm:"phone" json:"phone"`
	Birthday  *string `gorm:"birthday" json:"birthday,omitempty" example:"2006-01-02"`
	Gender    string  `gorm:"gender" json:"gender,omitempty" example:"male/female"`
	TagId     *string `gorm:"tag_id" json:"tag_id"`
	CreatedBy string  `gorm:"created_by" json:"created_by"`
	// Email          string            `gorm:"email" json:"email"`
	// MaritalStatus  string            `gorm:"marital_status" json:"marital_status"`
	// PrimaryLang    string            `gorm:"primary_lang" json:"primary_lang"`
	// TgUsername     string            `gorm:"tg_username" json:"tg_username"`
	// Facebook       string            `gorm:"facebook" json:"facebook"`
	// Instagram      string            `gorm:"instagram" json:"instagram"`
	// IsSmsNotify    bool              `gorm:"is_sms_notify" json:"is_sms_notify"`
	// IsPhoneNotify  bool              `gorm:"is_phone_notify" json:"is_phone_notify"`
	// IsSocialNotify bool              `gorm:"is_social_notify" json:"is_social_notify"`
	// IsEmailNotify  bool              `gorm:"is_email_notify" json:"is_email_notify"`
}

type Tag struct {
	Id   string `gorm:"id" json:"id"`
	Name string `gorm:"name" json:"name"`
}
