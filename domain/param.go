package domain

type Params struct {
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Order  map[string]any `json:"order"`
}

// QueryParam is used to store query parameters for all filter endpoints
type QueryParam struct {
	StoreID         string  `form:"store_id"`
	IsOpen          string  `form:"is_open"`
	Search          string  `form:"search"`
	StartDate       string  `form:"start_date"`
	EndDate         string  `form:"end_date"`
	VendorID        string  `form:"vendor_id"`
	PaymentTypeID   string  `form:"payment_type_id"`
	CashBoxID       string  `form:"cashbox_id"`
	Limit           int     `form:"limit"`
	Offset          int     `form:"offset"`
	TotalAmountTo   float64 `form:"total_amount_to"`
	TotalAmountFrom float64 `form:"total_amount_from"`
}
