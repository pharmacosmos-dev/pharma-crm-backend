package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// ClickPass implements PaymentService
func (h *Services) ClickPass(ctx context.Context, tx *gorm.DB, click *domain.PaymentService, data *domain.FinalPaymentType, cashboxID string, transactionID string, saleID string) (map[string]any, error) {
	// Click Pass request body
	clickData := domain.ClickPassRequest{
		ServiceID:     click.ServiceID,
		OtpData:       data.OtpData,
		CashboxCode:   cashboxID,
		Amount:        data.Amount,
		TransactionID: transactionID,
	}
	// Marshal click pass request
	t, _ := json.Marshal(clickData)
	// Save request of one click pass data
	err := h.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       time.Now().Unix(),
		Method:          "click_pass",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "click",
	})
	if err != nil {
		return nil, err
	}
	// generate click pass auth token
	token := h.generateClickAndUzumAuthToken(click.SecretKey, click.MerchantUserID)
	// send request to click pass
	res, err := h.ClickPassDoRequest(ctx, "/click_pass/payment", clickData, token)
	if err != nil {
		h.log.Info("ClickPassDoRequest error: %v", err.Error())
		return nil, err
	}
	// convert struct response to map
	var result map[string]any
	temp, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      temp,
		Method:        "click_pass",
	})
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(temp, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal into map: %w", err)
	}
	// checking error code
	if res.ErrorCode != 0 {
		return result, errors.New(res.ErrorNote)
	}

	return result, nil
}

// Check click pass payment status
func (h *Services) ClickCheckPaymentStatus(ctx context.Context, data map[string]any, token string) (*domain.ClickPassResponse, error) {
	url := fmt.Sprintf("/payment/status/%v/%v", data["service_id"], data["payment_id"])
	res, err := h.ClickPassDoRequest(ctx, url, data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Cancel click pass payment
func (h *Services) ClickPassCancelPayment(ctx context.Context, data map[string]any, token string) (*domain.ClickPassResponse, error) {
	url := fmt.Sprintf("/payment/reversal/%v/%v", data["service_id"], data["payment_id"])
	res, err := h.ClickPassDoRequest(ctx, url, data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Generate click pass and uzum fast pay auth token
func (h *Services) generateClickAndUzumAuthToken(secretKey string, merchantUserId int) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	digest := sha1.Sum([]byte(timestamp + secretKey))
	digestStr := fmt.Sprintf("%x", digest)
	return fmt.Sprintf("%d:%s:%s", merchantUserId, digestStr, timestamp)
}

// DoRequest for Click Pass
func (h *Services) ClickPassDoRequest(ctx context.Context, url string, data any, token string) (*domain.ClickPassResponse, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	// Construct request
	fullURL := h.cfg.ClickEndpointUrl + url
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Auth", token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response body
	var clickResponse domain.ClickPassResponse
	err = json.NewDecoder(resp.Body).Decode(&clickResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &clickResponse, nil
}

// Payme Go Handler functon
// Improved PaymeGo Handler function
func (s *Services) PaymeGo(ctx context.Context, tx *gorm.DB, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
	// Method receipt create
	createRes, err := s.PaymeGoReceiptCreate(ctx, paymentService, data, transactionID, saleID)

	if err != nil {
		s.log.Error("Failed to create receipt: %v", err)
		return nil, fmt.Errorf("receipt creation failed: %w", err)
	}

	// Validate receipt creation response
	if createRes.Error.Message != "" || createRes.Error.Code != 0 {
		s.log.Error("PaymeGo receipt create error: code=%d, message=%s", createRes.Error.Code, createRes.Error.Message)
		return nil, fmt.Errorf("receipt create error: %s (code: %d)", createRes.Error.Message, createRes.Error.Code)
	}

	if createRes.Result.Receipt.ID == "" {
		s.log.Error("Receipt creation failed: empty receipt ID")
		return nil, errors.New("receipt creation failed: empty receipt ID")
	}

	receiptID := createRes.Result.Receipt.ID

	// Set receipt id to sale_payments
	if err := s.SetReceiptId(tx, receiptID, transactionID); err != nil {
		s.log.Error("Failed to set receipt ID: %v", err)
		// Try to cancel the created receipt
		s.cancelReceiptWithLog(ctx, paymentService, transactionID, saleID, receiptID)
		return nil, fmt.Errorf("failed to set receipt ID: %w", err)
	}

	// Method receipt pay
	payRes, err := s.PaymeGoReceiptPay(ctx, paymentService, data, transactionID, saleID, receiptID)
	if err != nil {
		s.log.Error("Failed to pay receipt: %v", err)
		s.cancelReceiptWithLog(ctx, paymentService, transactionID, saleID, receiptID)
		return nil, fmt.Errorf("receipt payment failed: %w", err)
	}

	// Validate payment response
	if payRes.Error.Message != "" || payRes.Error.Code != 0 {
		s.log.Warn("PaymeGo receipt pay error: code=%d, message=%s", payRes.Error.Code, payRes.Error.Message)
		s.cancelReceiptWithLog(ctx, paymentService, transactionID, saleID, receiptID)
		return nil, errors.New(payRes.Error.Message)
	}

	return map[string]any{
		"error_code":    0,
		"error_message": "success",
		"receipt_id":    receiptID,
	}, nil
}

// Helper function to cancel receipt with logging
func (s *Services) cancelReceiptWithLog(ctx context.Context, paymentService *domain.PaymentService, transactionID, saleID, receiptID string) {
	if _, err := s.PaymeGoReceiptCancel(ctx, paymentService, transactionID, saleID, receiptID); err != nil {
		s.log.Error("Failed to cancel receipt %s: %v", receiptID, err)
	} else {
		s.log.Info("Successfully cancelled receipt %s", receiptID)
	}
}

// Improved PaymeGo Receipt Create with proper response handling
func (s *Services) PaymeGoReceiptCreate(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, transactionID string, saleID string) (*domain.PaymeGoResponse, error) {
	requestID := time.Now().Unix()

	reqData := domain.PaymeGoReceiptCreate{
		Id:     requestID,
		Method: "receipts.create",
		Params: domain.PaymeGoParams{
			Amount: data.Amount * 100,
			Account: struct {
				OrderId string `json:"order_id"`
			}{
				OrderId: saleID,
			},
			Detail: nil,
		},
	}

	// Save request
	if reqJSON, err := json.Marshal(reqData); err == nil {
		if err := s.SaveRequest(ctx, &domain.PaymentRequest{
			RequestId:       requestID,
			Method:          "receipts.create",
			Payload:         reqJSON,
			TransactionID:   transactionID,
			PaymentProvider: "payme",
		}); err != nil {
			s.log.Warn("Failed to save payme go request: %v", err)
		}
	}

	// Send request
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("Failed to send receipt create request: %v", err)
		return nil, fmt.Errorf("receipt create request failed: %w", err)
	}

	// Save response
	if resJSON, err := json.Marshal(res); err == nil {
		if err := s.SaveResponse(ctx, &domain.PaymentRequest{
			TransactionID: transactionID,
			Response:      resJSON,
			Method:        "receipts.create",
		}); err != nil {
			s.log.Warn("Failed to save payme go response: %v", err)
		}
	}

	return res, nil
}

// Payme Go Receipt Pay
func (s *Services) PaymeGoReceiptPay(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, transactionID string, saleID string, receiptID string) (*domain.PaymeGoResponse, error) {
	requestID := time.Now().Unix()
	reqData := domain.PaymeGoReceiptPay{
		Id:     requestID,
		Method: "receipts.pay",
		Params: domain.PaymeGoPayParams{
			Id:    receiptID,
			Token: data.OtpData,
		},
	}
	t, _ := json.Marshal(reqData)
	// save request body
	err := s.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       requestID,
		Method:          "receipts.pay",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "payme",
	})
	if err != nil {
		s.log.Error("ERROR on saving receipt pay request: ", err)
		return nil, err
	}
	// send do request to payme go
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("ERROR on receipt pay: %v", err)
		return nil, err
	}
	// response to json
	r, _ := json.Marshal(res)
	// save response
	err = s.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      r,
		Method:        "receipts.pay",
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return nil, err
	}
	return res, nil
}

// Payme Go Receipt Cancel
func (s *Services) PaymeGoReceiptCancel(ctx context.Context, paymentService *domain.PaymentService, transactionID string, saleID string, receiptID string) (*domain.PaymeGoResponse, error) {
	requestID := time.Now().Unix()
	reqData := domain.PaymeGoReceiptCancel{
		Id:     requestID,
		Method: "receipts.cancel",
		Params: domain.PaymeGoCancelParams{
			Id: receiptID,
		},
	}
	t, _ := json.Marshal(reqData)
	// save request body
	err := s.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       requestID,
		Method:          "receipts.cancel",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "payme",
	})
	if err != nil {
		s.log.Error("ERROR on saving receipt cancel request: ", err)
		return nil, err
	}
	// send do request to payme go
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("ERROR on receipt cancel: %v", err)
		return nil, err
	}
	// response to json
	r, _ := json.Marshal(res)
	// save response
	err = s.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      r,
		Method:        "receipts.cancel",
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return nil, err
	}
	return res, nil
}

// Payme Go Set fiscal data
func (s *Services) PaymeGoSetFiscalData(ctx context.Context, fiscal *domain.FiscalData, salePayment *domain.SalePayment, paymentService *domain.PaymentService) error {

	requestID := time.Now().Unix()
	reqData := domain.FiscalDataRequest{
		Id:     requestID,
		Method: "receipts.set_fiscal_data",
		Params: domain.FiscalDataParams{
			Id:         salePayment.ReceiptId,
			FiscalData: *fiscal,
		},
	}
	t, _ := json.Marshal(reqData)
	// save request body
	err := s.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       requestID,
		Method:          "receipts.set_fiscal_data",
		Payload:         t,
		TransactionID:   salePayment.ID,
		PaymentProvider: "payme",
	})
	if err != nil {
		s.log.Error("ERROR on saving set fiscal data request: ", err)
		return err
	}
	// send do request to payme go
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("ERROR on set fiscal data: %v", err)
		return err
	}
	// response to json
	r, _ := json.Marshal(res)
	// save response
	err = s.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: salePayment.ID,
		Response:      r,
		Method:        "receipts.set_fiscal_data",
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return err
	}
	return nil
}

// DoRequest for Payme Go
func (s *Services) PaymeGoDoRequest(ctx context.Context, data any, paymentService *domain.PaymentService) (*domain.PaymeGoResponse, error) {
	// Validate input parameters
	if paymentService == nil {
		return nil, errors.New("payment service is nil")
	}
	if paymentService.CashboxId == "" || paymentService.SecretKey == "" {
		return nil, errors.New("missing cashbox ID or secret key")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode request data: %w", err)
	}
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.Payment.PaymeGoEndpointUrl, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	// Set headers
	req.Header.Set("X-Auth", paymentService.CashboxId+":"+paymentService.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var paymeResponse domain.PaymeGoResponse
	err = json.NewDecoder(resp.Body).Decode(&paymeResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &paymeResponse, nil
}

// Save payment request to database
func (h *Services) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	query := `
	INSERT INTO payment_requests (
		request_id, 
		method, 
		payload, 
		transaction_id, 
		payment_provider
		)
		VALUES (?, ?, ?, ?, ?)`
	err := h.db.WithContext(ctx).Exec(
		query,
		req.RequestId,
		req.Method,
		req.Payload,
		req.TransactionID,
		req.PaymentProvider).Error
	if err != nil {
		h.log.Warn("ERROR on saving payment request: %v", err)
		return err
	}
	return nil
}

// Save payment response to database
func (h *Services) SaveResponse(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.Exec(
		`UPDATE 
			payment_requests 
		SET 
			response = ? 
		WHERE 
			transaction_id = ? AND 
			method = ?`,
		req.Response,
		req.TransactionID,
		req.Method,
	).Error
	if err != nil {
		h.log.Warn("ERROR on saving payment response: %v", err)
		return err
	}
	return nil
}

// save receipt id to sale_payments
func (h *Services) SetReceiptId(tx *gorm.DB, receiptId, salePayId string) error {
	query := `UPDATE sale_payments SET receipt_id = ? WHERE id = ?`
	err := tx.Exec(query, receiptId, salePayId).Error
	if err != nil {
		h.log.Warn("ERROR on setting receipt_id: %v", err)
		tx.Rollback()
		return err
	}
	return nil
}

// get sale_payments by sale_id with receipt_id
func (s *Services) GetSalePaymentsWithReceipt(saleId string) (*domain.SalePayment, error) {
	var res domain.SalePayment
	query := `SELECT * FROM sale_payments where sale_id = ? AND (receipt_id is not null OR receipt_id <> '')`
	err := s.db.Raw(query, saleId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting sale_payment with receipt_id: %v", err)
		return &res, err
	}
	return &res, nil
}

// Uzum fast pay handler function
func (h *Services) UzumFastPay(ctx context.Context, tx *gorm.DB, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
	var cashBoxId string
	err := h.db.Raw(`SELECT cash_box_id FROM cashbox_operations WHERE id = ?`, CashOperationID).Scan(&cashBoxId).Error
	if err != nil {
		return nil, err
	}
	uzumData := domain.UzumRequest{
		OrderId:       saleID,
		TransactionID: transactionID,
		CashboxCode:   cashBoxId,
		ServiceID:     paymentService.ServiceID,
		Amount:        data.Amount,
		OtpData:       data.OtpData,
	}
	t, err := json.Marshal(uzumData)
	if err != nil {
		return nil, err
	}
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       time.Now().Unix(),
		Method:          "uzum_fast_pay",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "uzum",
	})
	if err != nil {
		h.log.Info("Error on saving uzum fast pay request: %v", err.Error())
		return nil, err
	}

	// Generate Uzum Fast Pay auth token
	token := h.generateClickAndUzumAuthToken(paymentService.SecretKey, paymentService.MerchantUserID)

	res, err := h.UzumFastPayDoRequest(ctx, "/v2/payment", uzumData, token)
	if err != nil {
		return nil, err
	}
	// convert to json response of click pass
	t, _ = json.Marshal(res)
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      t,
		Method:        "uzum_fast_pay",
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Uzum Fast Pay Check payment status
func (h *Services) UzumFastPayCheckPaymentStatus(ctx context.Context, paymentService domain.PaymentService, paymentId string) (map[string]any, error) {
	data := map[string]any{
		"service_id": paymentService.ServiceID,
		"payment_id": paymentId,
	}
	token := h.generateClickAndUzumAuthToken(paymentService.SecretKey, paymentService.MerchantUserID)

	res, err := h.UzumFastPayDoRequest(ctx, "/payment/status", data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DoRequest for Uzum Fast Pay
func (h *Services) UzumFastPayDoRequest(ctx context.Context, url string, data any, token string) (map[string]any, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	// Construct request
	fullURL := h.cfg.UzumEndpointUrl + url
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response body
	var result map[string]any
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return result, nil
}

// Alif Pay
func (h *Services) AlifPay(ctx context.Context, tx *gorm.DB, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
	alifData := domain.AlifPaymentRequest{
		ID:     transactionID,
		Amount: data.Amount,
		Method: domain.AlifMethod{
			Type:  "MOBI_SHOW_QR",
			Token: data.OtpData,
		},
	}
	t, err := json.Marshal(alifData)
	if err != nil {
		return nil, err
	}
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       time.Now().Unix(),
		Method:          "alif_pay",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "alif",
	})
	if err != nil {
		h.log.Info("Error on saving alif pay request: %v", err.Error())
		return nil, err
	}

	res, err := h.AlifPayDoRequest(ctx, "/v2/pay", alifData, paymentService.SecretKey)
	if err != nil {
		return nil, err
	}
	t, _ = json.Marshal(res)
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      t,
		Method:        "alif_pay",
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// alif confirm payment
func (h *Services) AlifConfirmPayment(ctx context.Context, data1 *domain.FinalPaymentType, paymentId string) (map[string]any, error) {
	data := map[string]any{
		"id":  paymentId,
		"otp": data1.OtpData,
	}
	res, err := h.AlifPayDoRequest(ctx, "/v2/confirmPayment", data, "TODO")
	if err != nil {
		return nil, err
	}
	return res, nil
}

// alif pay do request function
func (h *Services) AlifPayDoRequest(ctx context.Context, url string, data any, token string) (map[string]any, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	// Construct request
	fullURL := h.cfg.AlifBaseUrl + url
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response body
	var result map[string]any
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return result, nil
}

func (s *Services) UpdatePaymentType(salePaymentID string, newPaymentTypeID string) error {
	query := `
		UPDATE sale_payments
		SET payment_type_id = $1,
		    updated_at = now()
		WHERE id = $2
	`

	result := s.db.Exec(query, newPaymentTypeID, salePaymentID)
	rowsAffected := result.RowsAffected
	if rowsAffected == 0 {
		return fmt.Errorf("sale_payment_id not found")
	}

	return nil
}
