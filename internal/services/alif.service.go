package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Alif Pay
func (s *Services) AlifPay(ctx context.Context, paymentService *domain.PaymentService, sale *domain.Sale) (*domain.AlifPayResponse, error) {
	payload := domain.AlifPaymentRequest{
		ID:     sale.Id,
		Amount: int64(sale.Alif * constants.SumsToTiyns),
		Method: domain.AlifMethod{
			Type:  constants.AlifPaymentTypeQrShow,
			Token: sale.OtpCode,
		},
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	id, err := s.createAlifRequestInDb(ctx, payload, constants.AlifPay, sale.Id)
	if err != nil {
		s.log.Errorf("could not save alif request: %v", err)
		return nil, domain.InternalServerError
	}
	// Send request to create receipt
	var response *http.Response
	if err = s.AlifRequest(
		&response,
		s.cfg.AlifApiUrl+constants.AlifPayCreatePath,
		jsonBytes,
		paymentService.CashboxId,
	); err != nil {
		s.log.Errorf("could not send create alifPay request: %v", err)
		return nil, domain.InternalServerError
	}

	defer utils.Close(response.Body, s.log)

	// Decode response
	result, bytes, err := DecodeAlifResponse[domain.AlifPayResponse](response.Body)
	_ = s.updateAlifRequestInDb(ctx, id, bytes, constants.ActionCreateReceipt)

	if err != nil {
		return result.Result, err
	}

	return result.Result, nil
}

func (s *Services) createAlifRequestInDb(ctx context.Context, payload domain.AlifPaymentRequest, method string, saleId string) (int, error) {
	var seqId int

	// Prepare payload
	payloadDb, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal payme payload: %v", err)
		return seqId, err
	}
	query := `
	INSERT INTO payment_requests (method, payload, transaction_id, payment_provider) VALUES (?, ?, ?, ?) RETURNING seq_id`
	err = s.db.WithContext(ctx).Raw(
		query,
		method,
		payloadDb,
		saleId,
		constants.PaymentTypeAlif,
	).Scan(&seqId).Error
	if err != nil {
		s.log.Errorf("could not create click_pass request(%d) in db: %v", payload.Amount, err)
		return seqId, err
	}

	return seqId, nil
}

func (s *Services) updateAlifRequestInDb(ctx context.Context, id int, response []byte, method string) error {
	err := s.db.WithContext(ctx).Exec(
		"UPDATE payment_requests SET response = ? WHERE seq_id = ? AND method = ?",
		response, id, method,
	).Error
	if err != nil {
		s.log.Errorf("could not update click request in db: %v", err)
		return err
	}

	return nil
}

func DecodeAlifResponse[T any](r io.Reader) (domain.AlifResponseWrapper[T], []byte, error) {
	var result domain.AlifResponseWrapper[T]

	response, err := io.ReadAll(r)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return result, response, err
	}

	// Agar Alif xatolik yuborgan bo‘lsa
	if result.Error != nil {
		alifErr := fmt.Errorf(
			"alif error: %d, message: %s",
			result.Error.Code, result.Error.Message,
		)

		switch result.Error.Code {
		case constants.AlifInvalidRequestBodyErrorCode:
			return result, response, domain.AlifInvalidRequestBodyError
		case constants.AlifUnauthorizedRequestErrorCode:
			return result, response, domain.AlifUnauthorizedError
		case constants.AlifInvalidParametersErrorCode:
			return result, response, domain.AlifInvalidParametersError
		case constants.AlifInvalidOtpErrorCode:
			return result, response, domain.IncorrectOTPError
		case constants.AlifOtpExpiredErrorCode:
			return result, response, domain.IncorrectCardExpiryDateError
		case constants.AlifInvalidCardDataErrorCode:
			return result, response, domain.IncorrectCardError
		case constants.AlifInsufficientFundsErrorCode:
			return result, response, domain.InsufficientFunds
		default:
			return result, response, alifErr
		}
	}

	return result, response, nil
}
