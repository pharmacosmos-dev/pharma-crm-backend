package domain

import (
	"github.com/pharma-crm-backend/domain/constants"
)

// region Init

type PaymeAuth struct {
	Id    string
	Key   string
	Token string
}

// endregion

// region Wrapper

type PaymePayloadWrapper[T any] struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
	Params T      `json:"params"`
}

func NewPaymePayloadWrapper[T any](method string, params T) PaymePayloadWrapper[T] {
	return PaymePayloadWrapper[T]{Method: method, Params: params}
}

func NewPaymePayloadWrapperWithId[T any](id int, method string, params T) PaymePayloadWrapper[T] {
	return PaymePayloadWrapper[T]{Id: id, Method: method, Params: params}
}

type PaymeResponseError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data"`
}

type PaymeResponseWrapper[T any] struct {
	Id     int                            `json:"id"`
	Result T                              `json:"result"`
	Error  NullStruct[PaymeResponseError] `json:"error"`
}

func (p *PaymeResponseWrapper[T]) SetId(id int) {
	p.Id = id
}

func (p *PaymeResponseWrapper[T]) FormatResponseError(code int, message string) {
	p.Error = NewNullStruct(PaymeResponseError{
		Code:    code,
		Message: message,
	}, true)
}

// endregion

// region GetVerifyCode

type GetVerifyCodeResponseDto struct {
	Sent  bool   `json:"sent"`
	Phone string `json:"phone"`
	Wait  int    `json:"wait"`
}

// endregion

// region Verify

type VerifyRequestDto struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

type VerifyResponseDto struct {
	Card struct {
		Number    string `json:"number"`
		Expire    string `json:"expire"`
		Token     string `json:"token"`
		Recurrent bool   `json:"recurrent"`
		Verify    bool   `json:"verify"`
	} `json:"card"`
}

// endregion

// region Remove Card

type RemoveCardRequestDto struct {
	Token string `json:"token"`
}

type RemoveCardResponseDto struct {
	Success bool `json:"success"`
}

// endregion

// region Receipt Create

// NOTE: temporary struct

type Receiptable interface {
	GetTarget() string
}

type UserReceiptCreateRequestDto struct {
	Amount      int    `json:"amount"`
	Description string `json:"description,omitempty"`
	Account     struct {
		OrderId string `json:"order_id"`
	} `json:"account"`
	Target string `json:"-"`
}

func (d UserReceiptCreateRequestDto) GetTarget() string {
	return d.Target
}

func NewUserReceiptCreateRequestDto(amount int, orderId string) UserReceiptCreateRequestDto {
	return UserReceiptCreateRequestDto{
		Amount: amount * constants.SumsToTiyns,
		Account: struct {
			OrderId string `json:"order_id"`
		}{
			OrderId: orderId,
		},
	}

}

type ReceiptCreateRequestDto struct {
	Amount  int  `json:"amount"`
	Hold    bool `json:"hold,omitempty"`
	Account struct {
		OrderId string `json:"order_id"`
	} `json:"account"`
	Target string `json:"-"`
}

func (d ReceiptCreateRequestDto) GetTarget() string {
	return d.Target
}

func NewReceiptCreateRequestDto(amount int, orderId string) ReceiptCreateRequestDto {
	return ReceiptCreateRequestDto{
		Amount: amount * constants.SumsToTiyns,
		Account: struct {
			OrderId string `json:"order_id"`
		}{
			OrderId: orderId,
		},
	}

}

type ReceiptCreateResponseDto struct {
	Receipt struct {
		Id          string `json:"_id"`
		CreateTime  int    `json:"create_time"`
		PayTime     int    `json:"pay_time"`
		CancelTime  int    `json:"cancel_time"`
		Stage       int    `json:"stage"`
		State       int    `json:"state"`
		Type        int    `json:"type"`
		External    bool   `json:"external"`
		Operation   int    `json:"operation"`
		Error       string `json:"error"`
		Description string `json:"description"`
		Amount      int    `json:"amount"`
		Currency    int    `json:"currency"`
		Commission  int    `json:"commission"`
	} `json:"receipt"`
}

// region Receipt Check

type ReceiptCheckRequestDto struct {
	Id string `json:"id"`
}

// receipts.check response: {"result": {"state": 4}}
type ReceiptCheckResponseDto struct {
	State int `json:"state"`
}

// endregion

// region Receipt Pay

type ReceiptPayRequestDto struct {
	Id    string `json:"id"`
	Token string `json:"token"`
	Payer struct {
		Phone string `json:"phone"`
	} `json:"payer"`
	Hold bool `json:"hold,omitempty"`
}

func NewReceiptPayRequestDto(id, token string) ReceiptPayRequestDto {
	return ReceiptPayRequestDto{
		Id:    id,
		Token: token,
		Payer: struct {
			Phone string `json:"phone"`
		}{
			Phone: "",
		},
	}
}

// region Receipt SetFiscalData

type ReceiptSetFiscalRequestDto struct {
	Id         string     `json:"id"`
	FiscalData FiscalData `json:"fiscal_data"`
}

type FiscalData struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	TerminalId string `json:"terminal_id"`
	ReceiptId  int    `json:"receipt_id"`
	Date       string `json:"date"`
	FiscalSign string `json:"fiscal_sign"`
	QrCodeUrl  string `json:"qr_code_url"`
}

func NewReceiptSetFiscalRequestDto(id string, data FiscalData) ReceiptSetFiscalRequestDto {
	return ReceiptSetFiscalRequestDto{
		Id:         id,
		FiscalData: data,
	}
}

// endregion

// region Confirm Hold

type ReceiptConfirmHoldRequestDto struct {
	Id string `json:"id"`
}

// endregion

// region Cancel Hold

type ReceiptCancelRequestDto struct {
	Id string `json:"id"`
}

// endregion

// region Tmp Card

type TempRedisCardDto struct {
	Card         string `json:"card"`
	Token        string `json:"token"`
	IsBusiness   bool   `json:"is_business"`
	ForAfDriver  bool   `json:"for_af_driver"`
	ForIbox      bool   `json:"for_ibox"`
	IsCommission bool   `json:"is_commission"`
}

type CourierAccount struct {
	Phone     string `json:"phone"`
	CourierId string `json:"courier_id"`
}

// endregion
