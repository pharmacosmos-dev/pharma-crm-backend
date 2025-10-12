package domain

// Click Pass request body
type ClickPassRequest struct {
	ServiceId     int     `json:"service_id"`
	OtpData       string  `json:"otp_data"`
	Amount        float64 `json:"amount"`
	CashboxCode   string  `json:"cashbox_code"`
	TransactionId string  `json:"transaction_id"`
}

// Click Pass response body
type ClickPassResponse struct {
	ErrorCode      int    `json:"error_code"`
	ErrorNote      string `json:"error_note"`
	PaymentId      string `json:"payment_id"`
	PaymentStatus  int    `json:"payment_status"`
	ConfirmMode    bool   `json:"confirm_mode"`
	CardType       string `json:"card_type"`
	ProcessingType string `json:"processing_type"`
	CardNumber     string `json:"card_number"`
	PhoneNumber    string `json:"phone_number"`
}
