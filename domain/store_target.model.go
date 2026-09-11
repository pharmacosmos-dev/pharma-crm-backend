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
	// TotalStores — filtrga mos do'konlar soni. Target qo'yilmaganlari ham
	// kiradi, shuning uchun bu son GetStoreTargetList'ning _meta.total_count'i
	// bilan bir xil bo'ladi.
	TotalStores int64   `json:"total_stores"`
	TotalAmount float64 `json:"total_target_amount"`
	TotalSales  float64 `json:"total_target_sales"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
}

// StoreTargetStatistics — bitta do'konga yangi target qo'yishdan oldin kerak
// bo'ladigan tarixiy ko'rsatkichlar. Hammasi bitta do'kon va bitta oy (year,
// month) uchun, ya'ni "shu oyga qancha qo'yilgan va o'tmishda qancha sotilgan".
//
// Savdo raqamlari store_targets.sales dan emas, sales jadvalidan hisoblanadi:
// target qatori yo'q oylar ham hisobga kirishi kerak.
type StoreTargetStatistics struct {
	StoreId   string `json:"store_id"`
	StoreName string `json:"store_name"`

	// So'ralgan davr (berilmasa joriy oy).
	Year  int `json:"year"`
	Month int `json:"month"`

	// CurrentTargetAmount — shu oyga qo'yilgan target. Target yo'q bo'lsa 0.
	CurrentTargetAmount float64 `json:"current_target_amount"`

	// Oldingi oy: yanvar so'ralsa bu o'tgan yilning dekabri bo'ladi.
	PreviousYear              int     `json:"previous_year"`
	PreviousMonth             int     `json:"previous_month"`
	PreviousMonthTargetAmount float64 `json:"previous_month_target_amount"`
	PreviousMonthSales        float64 `json:"previous_month_sales"`

	// LastYearSameMonthSales — o'tgan yilning AYNAN shu oyidagi savdo.
	LastYearSameMonthSales float64 `json:"last_year_same_month_sales"`

	// Oxirgi 12 to'liq oy: so'ralgan oyning O'ZI kirmaydi. Masalan 2026-09
	// so'ralsa oraliq 2025-09 dan 2026-08 gacha (ikkala chegara ham kiradi).
	Last12MonthsFrom       string  `json:"last_12_months_from"`
	Last12MonthsTo         string  `json:"last_12_months_to"`
	Last12MonthsTotalSales float64 `json:"last_12_months_total_sales"`
	// Last12MonthsAvgSales — yuqoridagi yig'indi 12 ga bo'linadi. Do'kon
	// oraliqning bir qismida ishlamagan bo'lsa ham maxraj 12 bo'lib qoladi.
	Last12MonthsAvgSales float64 `json:"last_12_months_avg_sales"`
}

type StoreTargetExcelRow struct {
	StoreId string  // A ustun
	Amount  float64 // B ustun
	Month   int     // C ustun
	Year    int     // D ustun
}

type StoreTargetUpsertResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// StoreTargetStoreCodeRow — excel qatori: A=store_code, B=store_name, C=amount.
// store_name faqat hisobotda ko'rinadi, do'kon store_code orqali topiladi.
type StoreTargetStoreCodeRow struct {
	RowNumber int
	StoreCode int
	StoreName string
	Amount    float64
}

// StoreTargetSkippedRow — o'tkazib yuborilgan qator: qaysi qator, nima sababdan.
// Excel yuklashda bitta xato qator butun faylni to'xtatmaydi, shuning uchun
// javobda aynan qaysi qatorlar tushib qolgani ko'rinib turishi kerak.
type StoreTargetSkippedRow struct {
	Row       int     `json:"row"`
	StoreCode int     `json:"store_code"`
	StoreName string  `json:"store_name"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
}

type StoreTargetCodeUpsertResult struct {
	Total   int                     `json:"total"`
	Created int                     `json:"created"`
	Updated int                     `json:"updated"`
	Skipped []StoreTargetSkippedRow `json:"skipped"`
	Year    int                     `json:"year"`
	Month   int                     `json:"month"`

	// Summa qaysi ustundan o'qilgani. Yashirilgan ustunlari bor faylda summa
	// C da emas, masalan J da turadi, shuning uchun bu javobda ko'rinib turishi kerak.
	AmountColumn string `json:"amount_column"`
	AmountHeader string `json:"amount_header,omitempty"`
}