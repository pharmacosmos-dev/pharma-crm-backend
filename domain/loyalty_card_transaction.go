package domain

import (
	"time"
)

// LoyaltyCardTransaction is a single loyalty card balance movement (in|out) of a customer
type LoyaltyCardTransaction struct {
	Id               string     `gorm:"id" json:"id"`
	SaleId           string     `gorm:"sale_id" json:"sale_id"`
	CustomerId       string     `gorm:"customer_id" json:"customer_id"`
	Type             string     `gorm:"type" json:"type" example:"in/out"`
	Percent          int        `gorm:"percent" json:"percent"`
	TotalSaleAmount  float64    `gorm:"total_sale_amount" json:"total_sale_amount"`
	OldBalanceAmount float64    `gorm:"old_balance_amount" json:"old_balance_amount"`
	BonusInAmount    float64    `gorm:"bonus_in_amount" json:"bonus_in_amount"`
	BonusOutAmount   float64    `gorm:"bonus_out_amount" json:"bonus_out_amount"`
	NewBalanceAmount float64    `gorm:"new_balance_amount" json:"new_balance_amount"`
	CreatedAt        *time.Time `gorm:"created_at" json:"created_at"`
}

type LoyaltyCardTransactionListItem struct {
	Id                 string     `gorm:"id" json:"id"`
	SaleId             string     `gorm:"sale_id" json:"sale_id"`
	SaleNumber         int        `gorm:"sale_number" json:"sale_number"`
	CustomerId         string     `gorm:"customer_id" json:"customer_id"`
	CustomerPublicId   int        `gorm:"customer_public_id" json:"customer_public_id"`
	CustomerName       string     `gorm:"customer_name" json:"customer_name"`
	CustomerPhone      string     `gorm:"customer_phone" json:"customer_phone"`
	LoyaltyCardBarcode string     `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	Type               string     `gorm:"type" json:"type" example:"in/out"`
	Percent            int        `gorm:"percent" json:"percent"`
	TotalSaleAmount    float64    `gorm:"total_sale_amount" json:"total_sale_amount"`
	OldBalanceAmount   float64    `gorm:"old_balance_amount" json:"old_balance_amount"`
	BonusInAmount      float64    `gorm:"bonus_in_amount" json:"bonus_in_amount"`
	BonusOutAmount     float64    `gorm:"bonus_out_amount" json:"bonus_out_amount"`
	NewBalanceAmount   float64    `gorm:"new_balance_amount" json:"new_balance_amount"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
}

type LoyaltyCardTransactionDashboard struct {
	TotalCount          int64   `gorm:"total_count" json:"total_count"`
	TotalInCount        int64   `gorm:"total_in_count" json:"total_in_count"`
	TotalOutCount       int64   `gorm:"total_out_count" json:"total_out_count"`
	TotalSaleAmountSum  float64 `gorm:"total_sale_amount_sum" json:"total_sale_amount_sum"`
	TotalBonusInAmount  float64 `gorm:"total_bonus_in_amount" json:"total_bonus_in_amount"`
	TotalBonusOutAmount float64 `gorm:"total_bonus_out_amount" json:"total_bonus_out_amount"`
}

type LoyaltyCardTransactionListRequest struct {
	Limit      int         `form:"limit" json:"limit" example:"10"`
	Offset     int         `form:"offset" json:"offset" example:"0"`
	StartDate  *CustomTime `form:"start_date" json:"start_date" example:"2026-01-01T00:00:00+05:00"`
	EndDate    *CustomTime `form:"end_date" json:"end_date" example:"2026-12-31T23:59:59+05:00"`
	Search     string      `form:"search" json:"search"`
	Type       string      `form:"type" json:"type" example:"in/out"`
	CustomerId string      `form:"customer_id" json:"customer_id"`
	SaleId     string      `form:"sale_id" json:"sale_id"`
}
