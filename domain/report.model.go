package domain

type ReportQueryParam struct {
	StoreId    string   `form:"store_id"`
	StartDate  string   `form:"start_date"`
	EndDate    string   `form:"end_date"`
	Limit      int      `form:"limit"`
	Offset     int      `form:"offset"`
	Search     string   `form:"search"`
	Order      string   `form:"order"`
	StoreIds   []string `form:"store_ids"`
	ProductIds []string `form:"product_ids"`
}

// Bonus report structure
type BonusReport struct {
	EmployeeId string  `gorm:"employee_id" json:"employee_id"`
	PublicId   int     `gorm:"public_id" json:"public_id"`
	FullName   string  `gorm:"full_name" json:"full_name"`
	Phone      string  `gorm:"phone" json:"phone"`
	StoreName  string  `gorm:"store_name" json:"store_name"`
	Role       string  `gorm:"role" json:"role"`
	Amount     float64 `gorm:"amount" json:"amount"`
	Count      float64 `gorm:"count" json:"count"`
	TotalCount int64   `gorm:"total_count" json:"-"`
}
