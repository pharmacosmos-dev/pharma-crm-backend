package domain

import "time"

type StoreTarget struct {
	Id        string     `json:"id" gorm:"column:id"`
	StoreId   string     `json:"store_id" gorm:"column:store_id"`
	CompanyId string     `json:"company_id" gorm:"column:company_id"`
	Amount    float64    `json:"amount" gorm:"column:amount"`
	Year      int        `json:"year" gorm:"column:year"`
	Month     int        `json:"month" gorm:"column:month"`
	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`

	Store *Store `json:"store,omitempty" gorm:"foreignKey:StoreId"`
}

func (StoreTarget) TableName() string {
	return "store_targets"
}

// CREATE uchun request
type StoreTargetRequest struct {
	StoreId   string  `json:"store_id" binding:"required"`
	CompanyId string  `json:"company_id"`
	Amount    float64 `json:"amount" binding:"required"`
	Year      int     `json:"year" binding:"required"`
	Month     int     `json:"month" binding:"required"`
}

// UPDATE uchun request (faqat keyingi oy)
type StoreTargetUpdateRequest struct {
	Amount float64 `json:"amount" binding:"required"`
}

// Store history + sotuvlar bilan response
type StoreTargetHistoryItem struct {
	Id           string  `json:"id"`
	StoreId      string  `json:"store_id"`
	Amount       float64 `json:"amount"`
	ActualAmount float64 `json:"actual_amount"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// Barcha store target list + sotuvlar bilan response
type StoreTargetListItem struct {
	Id           string  `json:"id"`
	StoreId      string  `json:"store_id"`
	StoreName    string  `json:"store_name"`
	Amount       float64 `json:"amount"`
	ActualAmount float64 `json:"actual_amount"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// Query params
type StoreTargetQueryParams struct {
	StoreId   string `form:"store_id"`
	CompanyId string `form:"company_id"`
	Year      int    `form:"year"`
	Month     int    `form:"month"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}
