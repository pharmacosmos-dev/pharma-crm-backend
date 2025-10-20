package domain

import "time"

// cashbox_closure structure
type CashboxClosure struct {
	Id                 int64      `gorm:"id" json:"id"`
	CashboxOperationId string     `gorm:"cashbox_operation_id" json:"cashbox_operation_id"`
	SenderId           string     `gorm:"sender_id" json:"sender_id"`
	ReceiverId         string     `gorm:"receiver_id" json:"receiver_id"`
	ReceivedAmount     float64    `gorm:"received_amount" json:"received_amount"`
	AcceptedAmount     float64    `gorm:"accepted_amount" json:"accepted_amount"`
	Status             string     `gorm:"status" json:"status"`
	Comment            string     `gorm:"comment" json:"comment"`
	CreatedAt          *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt          *time.Time `gorm:"updated_at" json:"updated_at"`
}

// cashbox_closure create request
type CashboxClosureRequest struct {
	CashboxOperationId string  `json:"cashbox_operation_id"`
	SenderId           string  `json:"sender_id"`
	ReceivedAmount     float64 `json:"received_amount"`
	Status             string  `json:"status"`
	Comment            string  `json:"comment"`
}

type CashboxOperationSummary struct {
	TotalSum       SalePaymentTotalAmount    `json:"total_data"`
	PaymentTypeSum []SalePaymentCloseCashBox `json:"data"`
}
