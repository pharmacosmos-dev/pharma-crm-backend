package domain

import (
	"time"
)

type Customer struct {
	Id                   string     `gorm:"id" json:"id"`
	StoreId              string     `gorm:"store_id" json:"store_id"`
	FirstName            string     `gorm:"first_name" json:"first_name"`
	LastName             string     `gorm:"last_name" json:"last_name"`
	FullName             string     `gorm:"full_name" json:"full_name"`
	PublicId             int        `gorm:"public_id" json:"public_id"`
	Phone                string     `gorm:"phone" json:"phone"`
	Birthday             string     `gorm:"birthday" json:"birthday" example:"2006-01-02"`
	Gender               string     `gorm:"gender" json:"gender" example:"male/female"`
	Balance              float64    `gorm:"balance" json:"balance"`
	SpendingFromBalance  float64    `gorm:"spending_from_balance" json:"spending_from_balance"`
	TagId                string     `gorm:"tag_id" json:"tag_id"`
	DiscountCard         string     `gorm:"discount_card" json:"discount_card"`
	DiscountPercent      int        `gorm:"discount_percent" json:"discount_percent"`
	CreatedBy            string     `gorm:"-" json:"created_by"`
	UpdatedBy            string     `gorm:"-" json:"updated_by"`
	DeletedBy            string     `gorm:"-" json:"deleted_by"`
	CreatedAt            *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt            *time.Time `gorm:"updated_at" json:"updated_at"`
	SaleDate             *time.Time `gorm:"sale_date" json:"sale_date"`
	SaleAmount           float64    `gorm:"sale_amount" json:"sale_amount"`
	DebtAmount           float64    `gorm:"debt_amount" json:"debt_amount"`
	Tag                  *Tag       `gorm:"-" json:"tag"`
	Store                *Store     `gorm:"-" json:"store"`
	LoyaltyCardBarcode   string     `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	LoyaltyCardPercent   int        `gorm:"loyalty_card_percent" json:"loyalty_card_percent"`
	LoyaltyCardLevelId   string     `gorm:"loyalty_card_level_id" json:"loyalty_card_level_id"`
	LoyaltyCardType      string     `gorm:"loyalty_card_type" json:"loyalty_card_type"`
	LoyaltyCardCreatedBy string     `gorm:"loyalty_card_created_by" json:"loyalty_card_created_by"`
	LoyaltyCardCreatedAt *time.Time `gorm:"loyalty_card_created_at" json:"loyalty_card_created_at"`
	TelegramChatId       int64      `gorm:"telegram_chat_id" json:"telegram_chat_id"`
	SalesCount24h        int64      `gorm:"sales_count_24h" json:"sales_count_24h"`
	MonthlySalesSum      float64    `gorm:"monthly_sales_sum" json:"monthly_sales_sum"`
	MonthlySalesCount    int64      `gorm:"monthly_sales_count" json:"monthly_sales_count"`
	IsActive             bool       `gorm:"is_active" json:"is_active"`
	IsBlocked            bool       `gorm:"is_blocked" json:"is_blocked"`
}

type UpdateCustomerBlockRequest struct {
	Id        string `json:"id" binding:"required"`
	IsBlocked bool   `json:"is_blocked"`
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
	Id                       string  `gorm:"id" json:"-"`
	StoreId                  *string `gorm:"store_id" json:"store_id"`
	FirstName                string  `gorm:"first_name" json:"first_name"`
	LastName                 string  `gorm:"last_name" json:"last_name,omitempty"`
	FullName                 string  `gorm:"full_name" json:"full_name,omitempty"`
	Phone                    string  `gorm:"phone" json:"phone"`
	Birthday                 *string `gorm:"birthday" json:"birthday,omitempty" example:"2006-01-02"`
	Gender                   string  `gorm:"gender" json:"gender,omitempty" example:"male/female"`
	TagId                    *string `gorm:"tag_id" json:"tag_id"`
	DiscountCard             *string `gorm:"discount_card" json:"discount_card"`
	DiscountPercent          int     `gorm:"discount_percent" json:"discount_percent"`
	LoyaltyCardBarcode       *string `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	VirtualLoyaltyCardNeeded bool    `gorm:"virtual_loyalty_card_needed" json:"virtual_loyalty_card_needed"`
	CreatedBy                string  `gorm:"created_by" json:"created_by"`
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
