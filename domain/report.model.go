package domain

type ReportQueryParam struct {
	StoreId    string   `form:"store_id"`
	StartDate  string   `form:"start_date"`
	EndDate    string   `form:"end_date"`
	Limit      int      `form:"limit"`
	Offset     int      `form:"offset"`
	Search     string   `form:"search"`
	ProductIds []string `form:"product_ids"`
}
