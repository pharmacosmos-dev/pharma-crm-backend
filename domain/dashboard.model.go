package domain

import "time"

// Count and Amount stats structure
type DashboardCountStats struct {
	ImportAmount             float64 `json:"import_amount"`
	NotLast24HImportCount    float64 `json:"not_last_24h_import_count"`
	BeforeImportAmount       float64 `json:"before_import_amount"`
	NotLast24HImportAmount   float64 `json:"not_last_24h_import_amount"`
	TotalSaleCount           int64   `gorm:"total_sale_count" json:"total_sale_count"`
	BeforeSaleCount          int64   `gorm:"before_sale_count" json:"before_sale_count"`
	TotalSaleAmount          float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
	BeforeSaleAmount         float64 `gorm:"before_sale_amount" json:"before_sale_amount"`
	DiscountAmount           float64 `gorm:"discount_amount" json:"discount_amount"`
	BeforeDiscountAmount     float64 `gorm:"before_discount_amount" json:"before_discount_amount"`
	TotalProductCount        float64 `gorm:"total_product_count" json:"total_product_count"`
	BeforeProductCount       float64 `gorm:"before_product_count" json:"before_product_count"`
	StockTotalAmount         float64 `gorm:"stock_total_amount" json:"stock_total_amount"`
	BeforeStockAmount        float64 `gorm:"before_stock_amount" json:"before_stock_amount"`
	ExpiringSoonCount        float64 `gorm:"expiring_soon_count" json:"expiring_soon_count"`
	ExpiredSoonCount         float64 `gorm:"expired_soon_count" json:"expired_soon_count"`
	BeforeExpiringSoonCount  float64 `gorm:"before_expiring_soon_count" json:"before_expiring_soon_count"`
	ExpiringSoonAmount       float64 `gorm:"expiring_soon_amount" json:"expiring_soon_amount"`
	ExpiredSoonAmount        float64 `gorm:"expired_soon_amount" json:"expired_soon_amount"`
	BeforeExpiringSoonAmount float64 `gorm:"before_expiring_soon_amount" json:"before_expiring_soon_amount"`
	BeforeExpiredSoonAmount  float64 `gorm:"before_expired_soon_amount" json:"before_expired_soon_amount"`
	TotalNetIncome           float64 `gorm:"total_net_income" json:"total_net_income"`
	BeforeTotalNetIncome     float64 `gorm:"before_total_net_income" json:"before_total_net_income"`
	BonusAmount              float64 `gorm:"bonus_amount" json:"bonus_amount"`
	BeforeBonusAmount        float64 `gorm:"before_bonus_amount" json:"before_bonus_amount"`
}

// ChartResponse structure
type ChartResponse struct {
	ID          string     `gorm:"id" json:"id"`
	Count       int64      `gorm:"count" json:"count"`
	TotalAmount float64    `gorm:"total_amount" json:"total_amount"`
	CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
}

// Top Stores structure
type TopStores struct {
	Id                  string  `gorm:"id" json:"id"`
	Name                string  `gorm:"name" json:"name"`
	Count               int64   `gorm:"count" json:"count"`
	TotalCount          int64   `gorm:"total_count" json:"total_count"`
	Percent             float64 `gorm:"percent" json:"percent"`
	TotalAmount         float64 `gorm:"total_amount" json:"total_amount"`
	PreviousTotalAmount float64 `gorm:"previous_total_amount" json:"previous_total_amount"`
}

// Top Products structure
type TopProducts struct {
	Id                  string  `gorm:"id" json:"id"`
	Name                string  `gorm:"name" json:"name"`
	Count               int     `gorm:"count" json:"count"`
	UnitQuantity        int     `gorm:"unit_quantity" json:"unit_quantity"`
	ProducerName        string  `gorm:"producer_name" json:"producer_name"`
	UnitPerPack         int     `gorm:"unit_per_pack" json:"unit_per_pack"`
	TotalCount          int64   `gorm:"total_count" json:"total_count"`
	Percent             float64 `gorm:"percent" json:"percent"`
	TotalAmount         float64 `gorm:"total_amount" json:"total_amount"`
	PreviousTotalAmount float64 `gorm:"previous_total_amount" json:"previous_total_amount"`
}

// Bonus Products structure
type BonusProducts struct {
	Id                  string  `gorm:"id" json:"id"`
	Name                string  `gorm:"name" json:"name"`
	Count               float64 `gorm:"count" json:"count"`
	UnitQuantity        float64 `gorm:"unit_quantity" json:"unit_quantity"`
	UnitPerPack         float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
	TotalCount          int64   `gorm:"total_count" json:"total_count"`
	BonusAmount         float64 `gorm:"bonus_amount" json:"bonus_amount"`
	PreviousBonusAmount float64 `gorm:"previous_bonus_amount" json:"previous_bonus_amount"`
	Percent             float64 `gorm:"percent" json:"percent"`
}

type BonusProductsStats struct {
	DocumentsCount    int64   `json:"documents_count"`
	TotalCount        int64   `json:"total_count"`
	TotalUnitQuantity int64   `json:"total_unit_quantity"`
	TotalBonusAmount  float64 `json:"total_bonus_amount"`
	//PreviousBonusAmount float64 `json:"previous_bonus_amount"`
	//Percent             float64 `json:"percent"`
}

// Top Seller structure
type TopSeller struct {
	Id                  string  `gorm:"id" json:"id"`
	FullName            string  `gorm:"full_name" json:"full_name"`
	StoreName           string  `gorm:"store_name" json:"store_name"`
	Count               float64 `gorm:"count" json:"count"`
	TotalCount          int64   `gorm:"total_count" json:"total_count"`
	TotalAmount         float64 `gorm:"total_amount" json:"total_amount"`
	PreviousTotalAmount float64 `gorm:"previous_total_amount" json:"previous_total_amount"`
	Percent             float64 `gorm:"percent" json:"percent"`
}

// Dashboard query param
type DashboardQueryParam struct {
	StoreId   string   `form:"store_id"`
	StartDate string   `form:"start_date"`
	EndDate   string   `form:"end_date"`
	Type      string   `form:"type"`
	Limit     int      `form:"limit"`
	Offset    int      `form:"offset"`
	StoreIds  []string `form:"store_ids"`
	CompanyId string   `form:"company_id"`
	Search    string   `form:"search"`
}

// Dashboard payments structure
type DashboardPayment struct {
	Name                string  `gorm:"name" json:"name"`
	Count               int64   `gorm:"count" json:"count"`
	Amount              float64 `gorm:"amount" json:"amount"`
	Percent             float64 `gorm:"percent" json:"percent"`
	PreviousTotalAmount float64 `gorm:"previous_total_amount" json:"previous_total_amount"`
}

// Dashboard transactions structure
type DashboardTransaction struct {
	Name                string  `gorm:"name" json:"name"`
	Count               string  `gorm:"count" json:"count"`
	Amount              float64 `gorm:"amount" json:"amount"`
	Percent             float64 `gorm:"percent" json:"percent"`
	PreviousTotalAmount float64 `gorm:"previous_total_amount" json:"previous_total_amount"`
}

// Dashboard count stats sale
type DashboardCountStatsSale struct {
	SaleCount            int64   `gorm:"sale_count" json:"sale_count"`
	BeforeSaleCount      int64   `gorm:"before_sale_count" json:"before_sale_count"`
	SaleAmount           float64 `gorm:"sale_amount" json:"sale_amount"`
	BeforeSaleAmount     float64 `gorm:"before_sale_amount" json:"before_sale_amount"`
	DiscountAmount       float64 `gorm:"discount_amount" json:"discount_amount"`
	BeforeDiscountAmount float64 `gorm:"before_discount_amount" json:"before_discount_amount"`
}

// Dashboard count stats product
type DashboardCountStatsProduct struct {
	StockCount           float64 `gorm:"stock_count" json:"stock_count"`
	BeforeStockCount     float64 `gorm:"before_stock_count" json:"before_stock_count"`
	StockAmount          float64 `gorm:"stock_amount" json:"stock_amount"`
	BeforeStockAmount    float64 `gorm:"before_stock_amount" json:"before_stock_amount"`
	ExpiringCount        float64 `gorm:"expiring_count" json:"expiring_count"`
	ExpiringAmount       float64 `gorm:"expiring_amount" json:"expiring_amount"`
	BeforeExpiringAmount float64 `gorm:"before_expiring_amount" json:"before_expiring_amount"`
	ExpiredCount         float64 `gorm:"expired_count" json:"expired_count"`
	ExpiredAmount        float64 `gorm:"expired_amount" json:"expired_amount"`
	BeforeExpiredAmount  float64 `gorm:"before_expired_amount" json:"before_expired_amount"`
}

type DashboardCountStatsIncome struct {
	IncomeAmount       float64 `gorm:"income_amount" json:"income_amount"`
	BeforeIncomeAmount float64 `gorm:"before_income_amount" json:"before_income_amount"`
}
type DashboardImport struct {
	ImportAmount           float64 `gorm:"column:import_amount" json:"import_amount"`
	NotLast24hImportAmount float64 `gorm:"column:not_last_24h_import_amount" json:"not_last_24h_import_amount"`
}

type DashboardCountStatsBonus struct {
	BonusAmount       float64 `gorm:"bonus_amount" json:"bonus_amount"`
	BeforeBonusAmount float64 `gorm:"before_bonus_amount" json:"before_bonus_amount"`
}

type DashboardBody struct {
	StoreIds   []string `json:"store_ids"`
	CompanyIds []string `json:"company_ids"`
}
