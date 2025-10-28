package domain

type LoyaltyCardCreateRequest struct {
	CustomerID               string  `gorm:"customer_id" json:"customer_id"`
	LoyaltyCardBarcode       *string `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	VirtualLoyaltyCardNeeded bool    `gorm:"virtual_loyalty_card_needed" json:"virtual_loyalty_card_needed"`
	LoyaltyCardCreatedBy     string  `gorm:"-" json:"-"`
}
