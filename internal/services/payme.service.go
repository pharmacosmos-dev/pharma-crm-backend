package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

func (s *Services) CreateReceiptAndPay(
	ctx context.Context,
	sale *domain.Sale,
	fiscal *domain.FiscalData,
	token string,
) (domain.ReceiptCreateResponseDto, error) {
	var response domain.ReceiptCreateResponseDto

	var createReceiptBody domain.Receiptable = domain.NewReceiptCreateRequestDto(
		utils.Abs(int(sale.Payme)),
		strconv.Itoa(sale.SaleNumber),
	)

	// Create receipt and pay
	receipt, err := s.CreateReceipt(
		ctx,
		createReceiptBody,
		*sale,
		token,
	)
	if err != nil {
		return response, err
	}

	response, err = s.PayReceipt(
		ctx,
		domain.NewReceiptPayRequestDto(
			receipt.Receipt.Id,
			sale.OtpCode,
		),
		*sale,
		token,
	)
	if err != nil {
		return response, err
	}

	// set fiscal data
	if fiscal != nil {
		go s.SetFiscalDataReceipt(
			ctx, domain.NewReceiptSetFiscalRequestDto(
				receipt.Receipt.Id,
				*fiscal,
			),
			*sale,
			token,
		)
	}

	return response, nil
}

func (s *Services) CreateReceipt(
	ctx context.Context,
	receipt domain.Receiptable,
	sale domain.Sale,
	token string,
) (domain.ReceiptCreateResponseDto, error) {
	var result domain.PaymeResponseWrapper[domain.ReceiptCreateResponseDto]

	// Prepare payload
	payload, err := s.newPaymePayloadWrapper(
		ctx,
		constants.ActionCreateReceipt,
		sale.Id,
		receipt,
	)
	if err != nil {
		return result.Result, domain.InternalServerError
	}

	// Prepare request body
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal create receipt payload: %v", err)
		return result.Result, domain.InternalServerError
	}

	// Send request to create receipt
	var response *http.Response
	if err := s.PaymeRequest(
		&response, s.cfg.PaymeApiUrl, jsonBytes, token,
	); err != nil {
		s.log.Errorf("could not send create receipt request: %v", err)
		return result.Result, domain.InternalServerError
	}
	defer utils.Close(response.Body, s.log)

	// Decode response
	result, bytes, err := DecodePaymeResponse[domain.ReceiptCreateResponseDto](response.Body)
	_ = s.updatePaymeRequestInDb(ctx, payload.Id, bytes, constants.ActionCreateReceipt)
	if err != nil {
		s.log.Errorf("could not decode create receipt response: %v", err)
		return result.Result, err
	}

	return result.Result, nil
}

func (s *Services) PayReceipt(
	ctx context.Context,
	data domain.ReceiptPayRequestDto,
	sale domain.Sale,
	token string,
) (domain.ReceiptCreateResponseDto, error) {
	var result domain.PaymeResponseWrapper[domain.ReceiptCreateResponseDto]
	payload, err := s.newPaymePayloadWrapper(
		ctx,
		constants.ActionPayReceipt,
		sale.Id,
		data,
	)
	if err != nil {
		return result.Result, domain.InternalServerError
	}

	// Prepare request body
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal pay receipt payload: %v", err)
		return result.Result, domain.InternalServerError
	}

	var response *http.Response
	payErr := s.PaymePayReceiptRequest(ctx, &response, s.cfg.PaymeApiUrl, jsonBytes, token)
	if payErr != nil {
		s.log.Errorf("could not send pay receipt request: %v", payErr)
		// Javob umuman kelmadi (tarmoq/timeout): natija noma'lum, holat tekshiriladi
		return s.checkReceiptAfterPayFailure(ctx, data.Id, sale, token, payErr, false)
	}
	defer utils.Close(response.Body, s.log)

	// Decode response
	result, bytes, err := DecodePaymeResponse[domain.ReceiptCreateResponseDto](response.Body)
	_ = s.updatePaymeRequestInDb(ctx, payload.Id, bytes, constants.ActionPayReceipt)
	if err != nil {
		s.log.Errorf("could not decode pay receipt response: %v", err)
		if isPaymeBusinessError(err) {
			return result.Result, err
		}
		// result.Error.Valid — Payme javob berdi va rad etdi (masalan -32999, -31260):
		// chek state=0 bo'lsa bu yakuniy rad, pending emas
		return s.checkReceiptAfterPayFailure(ctx, data.Id, sale, token, err, result.Error.Valid)
	}

	return result.Result, nil
}

func isPaymeBusinessError(err error) bool {
	switch err {
	case domain.IncorrectOTPError,
		domain.OTPExpiredError,
		domain.CardNotFoundError,
		domain.IncorrectCardExpiryDateError,
		domain.InsufficientFunds,
		domain.PaymeNotOperationalError:
		return true
	}
	return false
}

// paymeRejected — receipts.pay ga Payme aniq `error` javob qaytargan (javob kelgan, u rad etgan).
// false bo'lsa javob umuman kelmagan (tarmoq/timeout) va natija noma'lum.
func (s *Services) checkReceiptAfterPayFailure(
	ctx context.Context,
	receiptId string,
	sale domain.Sale,
	token string,
	originalErr error,
	paymeRejected bool,
) (domain.ReceiptCreateResponseDto, error) {
	var empty domain.ReceiptCreateResponseDto
	s.log.Infof("receipts.pay failed for sale %s, checking via receipts.check", sale.Id)

	state, err := s.CheckReceipt(ctx, receiptId, sale, token)
	if err != nil {
		s.log.Errorf("receipts.check also failed for sale %s: %v", sale.Id, err)
		return empty, domain.PaymePendingError
	}

	switch state {
	case constants.PaymeReceiptStatePaid:
		// to'lov tasdiqlandi
		s.log.Infof("receipts.check confirmed payment for sale %s (receipt: %s, state: %d)", sale.Id, receiptId, state)
		return empty, nil
	case constants.PaymeReceiptStateCancelled, 21:
		// payme tomonidan bekor qilindi — originalErr qaytariladi, tx rollback bo'ladi
		s.log.Warnf("receipts.check: receipt cancelled for sale %s, state=%d", sale.Id, state)
		return empty, originalErr
	case constants.PaymeReceiptStateCreated:
		if paymeRejected {
			// Payme aniq rad etdi va chek hali to'lanmagan (state=0): bu yakuniy xato,
			// hech narsa jarayonda emas — Payme xabari qaytadi (424), kassir to'lov turini
			// almashtira oladi. Tarmoq xatosida esa (paymeRejected=false) pending qoladi.
			s.log.Warnf("receipts.check: receipt not paid after payme rejection for sale %s, returning error: %v", sale.Id, originalErr)
			return empty, originalErr
		}
		s.log.Warnf("receipts.check: receipt still processing for sale %s, state=%d, marking as pending", sale.Id, state)
		return empty, domain.PaymePendingError
	default:
		// state 1,2,3,5... — hali jarayonda (hold bo'lishi mumkin), pending sifatida saqlash
		s.log.Warnf("receipts.check: receipt still processing for sale %s, state=%d, marking as pending", sale.Id, state)
		return empty, domain.PaymePendingError
	}
}

func (s *Services) CheckReceipt(
	ctx context.Context,
	receiptId string,
	sale domain.Sale,
	token string,
) (int, error) {
	var result domain.PaymeResponseWrapper[domain.ReceiptCheckResponseDto]

	payload, err := s.newPaymePayloadWrapper(
		ctx,
		constants.ActionCheckReceipt,
		sale.Id,
		domain.ReceiptCheckRequestDto{Id: receiptId},
	)
	if err != nil {
		return 0, domain.InternalServerError
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal check receipt payload: %v", err)
		return 0, domain.InternalServerError
	}

	var response *http.Response
	if err := s.PaymeRequest(&response, s.cfg.PaymeApiUrl, jsonBytes, token); err != nil {
		s.log.Errorf("could not send check receipt request: %v", err)
		return 0, domain.InternalServerError
	}
	defer utils.Close(response.Body, s.log)

	result, bytes, err := DecodePaymeResponse[domain.ReceiptCheckResponseDto](response.Body)
	_ = s.updatePaymeRequestInDb(ctx, payload.Id, bytes, constants.ActionCheckReceipt)
	if err != nil {
		s.log.Errorf("could not decode check receipt response: %v", err)
		return 0, err
	}

	return result.Result.State, nil
}

func (s *Services) CancelReceipt(
	ctx context.Context,
	data domain.ReceiptCancelRequestDto,
	sale domain.Sale,
	token string,
) (domain.ReceiptCreateResponseDto, error) {
	var result domain.PaymeResponseWrapper[domain.ReceiptCreateResponseDto]
	payload, err := s.newPaymePayloadWrapper(
		ctx,
		constants.ActionCancelReceipt,
		sale.Id,
		data,
	)
	if err != nil {
		return result.Result, domain.InternalServerError
	}

	// Prepare request body
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal cancel receipt payload: %v", err)
		return result.Result, domain.InternalServerError
	}

	// Send request to hold receipt
	var response *http.Response
	if err := s.PaymeRequest(
		&response, s.cfg.PaymeApiUrl, jsonBytes, token,
	); err != nil {
		s.log.Errorf("could not send cancel receipt request: %v", err)
		return result.Result, domain.InternalServerError
	}
	defer utils.Close(response.Body, s.log)

	// Decode response
	result, bytes, err := DecodePaymeResponse[domain.ReceiptCreateResponseDto](response.Body)
	_ = s.updatePaymeRequestInDb(ctx, payload.Id, bytes, constants.ActionCancelReceipt)
	if err != nil {
		s.log.Errorf("could not decode cancel receipt response: %v", err)
		return result.Result, err
	}

	return result.Result, nil
}

func (s *Services) SetFiscalDataReceipt(
	ctx context.Context,
	data domain.ReceiptSetFiscalRequestDto,
	sale domain.Sale,
	token string,
) (domain.ReceiptCreateResponseDto, error) {
	var result domain.PaymeResponseWrapper[domain.ReceiptCreateResponseDto]
	payload, err := s.newPaymePayloadWrapper(
		ctx,
		constants.ActionSetFiscalData,
		sale.Id,
		data,
	)
	if err != nil {
		return result.Result, domain.InternalServerError
	}

	// Prepare request body
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal pay receipt payload: %v", err)
		return result.Result, domain.InternalServerError
	}

	var response *http.Response
	if err := s.PaymeRequest(
		&response, s.cfg.PaymeApiUrl, jsonBytes, token,
	); err != nil {
		s.log.Errorf("could not send pay receipt request: %v", err)
		return result.Result, domain.InternalServerError
	}
	defer utils.Close(response.Body, s.log)

	// Decode response
	result, bytes, err := DecodePaymeResponse[domain.ReceiptCreateResponseDto](response.Body)
	_ = s.updatePaymeRequestInDb(ctx, payload.Id, bytes, constants.ActionSetFiscalData)
	if err != nil {
		s.log.Errorf("could not decode pay receipt response: %v", err)
		return result.Result, err
	}

	return result.Result, nil
}

func (s *Services) newPaymePayloadWrapper(ctx context.Context, method, saleId string, params any) (domain.PaymePayloadWrapper[any], error) {
	// Create payload
	payload := domain.PaymePayloadWrapper[any]{Method: method, Params: params}

	id, err := s.createPaymeRequestInDb(payload, saleId)
	if err != nil {
		return payload, err
	}

	payload.Id = id

	return payload, nil
}

func (s *Services) createPaymeRequestInDb(payload domain.PaymePayloadWrapper[any], saleId string) (int, error) {
	var seqId int

	// Prepare payload
	payloadDb, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal payme payload: %v", err)
		return seqId, err
	}
	query := `
	INSERT INTO payment_requests (method, payload, transaction_id, payment_provider) VALUES (?, ?, ?, ?) RETURNING seq_id`
	err = s.db.Raw(
		query,
		payload.Method,
		payloadDb,
		saleId,
		constants.PaymentTypePayme,
	).Scan(&seqId).Error
	if err != nil {
		s.log.Errorf("could not create payme request(%d) in db: %v", payload.Id, err)
		return seqId, err
	}

	return seqId, nil
}

func (s *Services) updatePaymeRequestInDb(ctx context.Context, id int, response []byte, method string) error {
	err := s.db.Exec(
		"UPDATE payment_requests SET response = ? WHERE seq_id = ? AND method = ?",
		response, id, method,
	).Error
	if err != nil {
		s.log.Errorf("could not update payme request in db: %v", err)
		return err
	}

	return nil
}

func DecodePaymeResponse[T any](r io.Reader) (domain.PaymeResponseWrapper[T], []byte, error) {
	var result domain.PaymeResponseWrapper[T]

	response, err := io.ReadAll(r)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return result, response, err
	}

	if result.Error.Valid {
		// Ro'yxatda yo'q Payme xatolari (masalan -32999 "Сервис EPOS временно недоступен",
		// -31260 "ИНН не предоставлен") 424 bilan, Payme xabari o'zgarmagan holda qaytadi —
		// kassir aniq sababni ko'radi (avval 500 yoki 202 pending bo'lib ketar edi).
		paymeErr := domain.NewError(
			http.StatusFailedDependency,
			fmt.Sprintf("%s (payme %d)", result.Error.Value.Message, result.Error.Value.Code),
		)

		// Convert payme errors to out own
		switch result.Error.Value.Code {
		case constants.PaymeServerNotOperationalErrorCode, constants.PaymeTemporarilyUnavailableErrorCode:
			return result, response, domain.PaymeNotOperationalError
		case constants.PaymeIncorrectOTPError:
			return result, response, domain.IncorrectOTPError
		case constants.PaymeCardNotFoundError:
			return result, response, domain.CardNotFoundError
		case constants.PaymeInvalidExpiryDateErrorCode:
			return result, response, domain.IncorrectCardExpiryDateError
		case constants.PaymeOTPExpiredErrorCode:
			return result, response, domain.OTPExpiredError
		case constants.PaymeInsufficientFundsErrorCode:
			return result, response, domain.InsufficientFunds
		default:
			return result, response, paymeErr
		}
	}

	return result, response, nil
}

func (s *Services) generatePaymeToken(payService *domain.PaymentService) string {
	return payService.CashboxId + ":" + payService.SecretKey
}
