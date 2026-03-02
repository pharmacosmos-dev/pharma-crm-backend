package domain

type LoyaltyCardCreateRequest struct {
	CustomerID               string  `gorm:"customer_id" json:"customer_id"`
	LoyaltyCardBarcode       *string `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
	VirtualLoyaltyCardNeeded bool    `gorm:"virtual_loyalty_card_needed" json:"virtual_loyalty_card_needed"`
	LoyaltyCardCreatedBy     string  `gorm:"-" json:"-"`
}

type LoyaltyCardDashboard struct {
	TotalCashbackGiven float64              `json:"total_cashback_given"`
	TotalCards         int64                `json:"total_cards"`
	NewCardsInPeriod   int64                `json:"new_cards_in_period"`
	CardsByLevel       []LoyaltyCardByLevel `json:"cards_by_level"`
	Customers          any                  `json:"customers"`
	CustomerCount      int64                `json:"-"`
}

type LoyaltyCardByLevel struct {
	LevelID   string `json:"level_id"`
	LevelName string `json:"level_name"`
	Percent   int    `json:"percent"`
	Count     int64  `json:"count"`
}

type LoyaltyCardTopCustomer struct {
	CustomerID             string  `json:"customer_id"`
	PublicID			   string  `json:"public_id"`
	FullName               string  `json:"full_name"`
	Phone                  string  `json:"phone"`
	LoyaltyCardBarcode     string  `json:"loyalty_card_barcode"`
	LoyaltyCardLevelName   string  `json:"loyalty_card_level_name"`
	LoyaltyCardPercent     int     `json:"loyalty_card_percent"`
	TotalSpent             float64 `json:"total_spent"`
	TotalCashbackEarned    float64 `json:"total_cashback_earned"`
}

type LoyaltyCardDashboardRequest struct {
	FromDate  *string `form:"from_date" json:"from_date" example:"2024-01-01"`
	ToDate    *string `form:"to_date" json:"to_date" example:"2024-12-31"`
	IsLoyalty *bool   `form:"is_loyalty" json:"is_loyalty"`
	Limit     int     `form:"limit" json:"limit"`
	Offset    int     `form:"offset" json:"offset"`
}

type LoyaltyCardTopRequest struct {
	Limit  int     `form:"limit" json:"limit" example:"10"`
	Offset int     `form:"offset" json:"offset" example:"0"`
	FromDate *string `form:"from_date" json:"from_date" example:"2024-01-01"`
	ToDate   *string `form:"to_date" json:"to_date" example:"2024-12-31"`
}
