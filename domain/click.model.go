package domain

type ClickPassRequest struct {
	ServiceID     int     `json:"service_id"`
	OtpData       string  `json:"otp_data"`
	Amount        float64 `json:"amount"`
	CashboxCode   string  `json:"cashbox_code"`
	TransactionID string  `json:"transaction_id"`
}

type ClickPassResponse struct {
	ErrorCode      int    `json:"error_code"`
	ErrorNote      string `json:"error_note"`
	PaymentID      int    `json:"payment_id"`
	PaymentStatus  string `json:"payment_status"`
	ConfirmMode    int    `json:"confirm_mode"`
	CardType       string `json:"card_type"`
	ProcessingType int    `json:"processing_type"`
	CardNumber     string `json:"card_number"`
	PhoneNumber    string `json:"phone_number"`
}
