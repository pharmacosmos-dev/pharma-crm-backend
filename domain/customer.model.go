package domain

import (
	"time"
)

type Customer struct {
	Id              string     `gorm:"id" json:"id"`
	StoreId         string     `gorm:"store_id" json:"store_id"`
	FirstName       string     `gorm:"first_name" json:"first_name"`
	LastName        string     `gorm:"last_name" json:"last_name"`
	FullName        string     `gorm:"full_name" json:"full_name"`
	PublicId        int        `gorm:"public_id" json:"public_id"`
	Phone           string     `gorm:"phone" json:"phone"`
	Birthday        string     `gorm:"birthday" json:"birthday" example:"2006-01-02"`
	Gender          string     `gorm:"gender" json:"gender" example:"male/female"`
	Balance         float64    `gorm:"balance" json:"balance"`
	TagId           string     `gorm:"tag_id" json:"tag_id"`
	DiscountCard    string     `gorm:"discount_card" json:"discount_card"`
	DiscountPercent int        `gorm:"discount_percent" json:"discount_percent"`
	CreatedBy       string     `gorm:"-" json:"created_by"`
	UpdatedBy       string     `gorm:"-" json:"updated_by"`
	DeletedBy       string     `gorm:"-" json:"deleted_by"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
	SaleDate        *time.Time `gorm:"sale_date" json:"sale_date"`
	SaleAmount      float64    `gorm:"sale_amount" json:"sale_amount"`
	DebtAmount      float64    `gorm:"debt_amount" json:"debt_amount"`
	Tag             *Tag       `gorm:"-" json:"tag"`
	Store           *Store     `gorm:"-" json:"store"`
}

type DiscountCardWithCustomer struct {
	Id         string `json:"id"`
	CustomerId string `json:"customer_id"`
	Barcode    string `json:"barcode"`
	Percent    int    `json:"percent"`

	// Customer info
	FullName string    `json:"full_name"`
	Phone    string    `json:"phone"`
	Birthday time.Time `json:"birthday"`
	Gender   string    `json:"gender"`
	Balance  float64   `json:"balance"`

	// Store/Tag info
	StoreId   string `json:"store_id"`
	StoreName string `json:"store_name"`
	TagId     string `json:"tag_id"`
	TagName   string `json:"tag_name"`

	CreatedAt *time.Time `json:"created_at"`
}

type CustomerRequest struct {
	Id              string  `gorm:"id" json:"-"`
	StoreId         *string `gorm:"store_id" json:"store_id"`
	FirstName       string  `gorm:"first_name" json:"first_name"`
	LastName        string  `gorm:"last_name" json:"last_name,omitempty"`
	FullName        string  `gorm:"full_name" json:"full_name,omitempty"`
	Phone           string  `gorm:"phone" json:"phone"`
	Birthday        *string `gorm:"birthday" json:"birthday,omitempty" example:"2006-01-02"`
	Gender          string  `gorm:"gender" json:"gender,omitempty" example:"male/female"`
	TagId           *string `gorm:"tag_id" json:"tag_id"`
	DiscountCard    *string `gorm:"discount_card" json:"discount_card"`
	DiscountPercent int     `gorm:"discount_percent" json:"discount_percent"`
	CreatedBy       string  `gorm:"created_by" json:"created_by"`
}

type Tag struct {
	Id   string `gorm:"id" json:"id"`
	Name string `gorm:"name" json:"name"`
}

type CustomerForSale struct {
	Id        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
}
