package domain

type Params struct {
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Order  map[string]any `json:"order"`
}

// QueryParam is used to store query parameters for all filter endpoints
type QueryParam struct {
	StoreID         string  `form:"store_id"`
	CompanyId       string  `form:"company_id"`
	RepricingID     int     `form:"repricing_id"`
	IsOpen          string  `form:"is_open"`
	Search          string  `form:"search"`
	StartDate       string  `form:"start_date"`
	EndDate         string  `form:"end_date"`
	VendorID        string  `form:"vendor_id"`
	PaymentTypeID   string  `form:"payment_type_id"`
	CashBoxID       string  `form:"cashbox_id"`
	EmployeeId      string  `form:"employee_id"`
	Limit           int     `form:"limit"`
	Offset          int     `form:"offset"`
	TotalAmountTo   float64 `form:"total_amount_to"`
	TotalAmountFrom float64 `form:"total_amount_from"`
	Status          string  `form:"status"`
	SaleType        string  `form:"sale_type"` // for sales
	MaxPrice        *bool   `form:"max_price"`
	IsBlocked       *bool   `form:"is_blocked"`
	Order           string  `form:"order"`
	Type            string  `form:"type"`
}
