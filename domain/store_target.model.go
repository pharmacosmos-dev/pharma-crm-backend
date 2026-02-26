package domain

import "time"

type StoreTarget struct {
	Id        string     `json:"id" gorm:"column:id;primaryKey"`
	StoreId   string     `json:"store_id" gorm:"column:store_id"`
	CompanyId string     `json:"company_id" gorm:"column:company_id"`
	Amount    float64    `json:"amount" gorm:"column:amount"`
	Sales     float64    `json:"sales" gorm:"column:sales"`
	Year      int        `json:"year" gorm:"column:year"`
	Month     int        `json:"month" gorm:"column:month"`
	SyncedAt  *time.Time `json:"synced_at" gorm:"column:synced_at"`
	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`

	Store *Store `json:"store,omitempty" gorm:"foreignKey:StoreId"`
}


// CREATE uchun request
type StoreTargetRequest struct {
	StoreId string  `json:"store_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
	Year    int     `json:"year" binding:"required"`
	Month   int     `json:"month" binding:"required"`
}

// UPDATE for request(only next month)
type StoreTargetUpdateRequest struct {
	StoreId string `json:"store_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

// Store history + response with sales
type StoreTargetHistoryItem struct {
	Id           string  `json:"id"`
	StoreId      string  `json:"store_id"`
	Amount       float64 `json:"amount"`
	Sales        float64 `json:"sales"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// Response with all store target list + sales
type StoreTargetListItem struct {
	Id           string  `json:"id"`
	StoreId      string  `json:"store_id"`
	CompanyId    string  `json:"company_id"`
	StoreName    string  `json:"store_name"`
	Amount       float64 `json:"amount"`
    Sales 		 float64 `json:"sales"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// Query params
type StoreTargetQueryParams struct {
	StoreId     string   `form:"store_id"`
	CompanyId   string   `form:"company_id"`
	CompanyIds  []string `form:"-"`
	SearchField string   `form:"search"`
	IsFranchise *bool    `form:"is_franchise"`
	IsPharma    *bool    `form:"is_pharma"`
	Year        int      `form:"year"`
	Month       int      `form:"month"`
	Limit       int      `form:"limit"`
	Offset      int      `form:"offset"`
	Order       string      `form:"order"`
}

type StoreTargetSummary struct {
	TotalAmount float64 `json:"total_target_amount"`
	TotalSales  float64 `json:"total_target_sales"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
}