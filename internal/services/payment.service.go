package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pharma-crm-backend/domain"
)

// ClickPass implements PaymentService
func (h *Services) ClickPass(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, cashboxID string, transactionID string, saleID string) (map[string]any, error) {
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
	// convert to json response of click pass
	t, _ = json.Marshal(res)
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      t,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Check click pass payment status
func (h *Services) ClickCheckPaymentStatus(ctx context.Context, data map[string]any, token string) (map[string]any, error) {
	url := fmt.Sprintf("/payment/status/%v/%v", data["service_id"], data["payment_id"])
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
func (h *Services) ClickPassDoRequest(ctx context.Context, url string, data any, token string) (map[string]any, error) {
	client := &http.Client{
		Timeout: 7 * time.Second,
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

// Uzum fast pay handler function
func (h *Services) UzumFastPay(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
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
	var result map[string]interface{}
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

// Payme Go Handler functon
func (s *Services) PaymeGo(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
	// method receipt create
	res, err := s.PaymeGoReceiptCreate(ctx, paymentService, data, transactionID, saleID)
	if err != nil {
		return nil, err
	}
	// set receipt id to sale_payments
	err = s.SetReceiptId(ctx, res.Result.Receipt.ID, transactionID)
	if err != nil {
		return nil, err
	}
	// method receipt pay
	res, err = s.PaymeGoReceiptPay(ctx, paymentService, data, transactionID, saleID, res.Result.Receipt.ID)
	if err != nil {
		s.log.Warn("ERROR on receipt pay: %v", err)
		// method receipt cancel
		_, err = s.PaymeGoReceiptCancel(ctx, paymentService, transactionID, saleID, res.Result.Receipt.ID)
		if err != nil {
			s.log.Warn("ERROR on receipt cancel: %v", err)
			return nil, err
		}
		return nil, err
	}

	return map[string]any{
		"error_code":    0,
		"error_message": "success",
	}, nil
}

// Payme Go Receipt Create
func (s *Services) PaymeGoReceiptCreate(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, transactionID string, saleID string) (*domain.PaymeGoResponse, error) {
	requestID := time.Now().Unix()
	// get current time
	reqData := domain.PaymeGoReceiptCreate{
		Id:     time.Now().Unix(),
		Method: "receipts.create",
		Params: domain.PaymeGoParams{
			Amount: data.Amount,
			Account: struct {
				OrderId string `json:"order_id"`
			}{
				OrderId: saleID, // Assign your order ID here
			},
			Detail: nil,
		},
	}
	// convert to json for saving payment requests
	t, _ := json.Marshal(reqData)
	// save request payme go request
	err := s.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       requestID,
		Method:          "receipts.create",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "payme",
	})
	if err != nil {
		s.log.Info("Error on saving payme go request: %v", err.Error())
		return nil, err
	}
	// send do request for receipt create
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("ERROR on receipt create: %v", err)
		return nil, err
	}
	// response to json
	r, _ := json.Marshal(res)
	// save response
	err = s.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      r,
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return nil, err
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
			Payer: nil,
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
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return nil, err
	}
	return res, nil
}

// Payme Go Set fiscal data
func (s *Services) PaymeGoSetFiscalData(ctx context.Context, paymentService *domain.PaymentService, fiscal *domain.FiscalData, transactionID string, receiptID string) (*domain.PaymeGoResponse, error) {
	requestID := time.Now().Unix()
	reqData := domain.FiscalDataRequest{
		Id:     requestID,
		Method: "receipts.set_fiscal_data",
		Params: domain.FiscalDataParams{
			Id:         receiptID,
			FiscalData: *fiscal,
		},
	}
	t, _ := json.Marshal(reqData)
	// save request body
	err := s.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       requestID,
		Method:          "receipts.set_fiscal_data",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "payme",
	})
	if err != nil {
		s.log.Error("ERROR on saving set fiscal data request: ", err)
		return nil, err
	}
	// send do request to payme go
	res, err := s.PaymeGoDoRequest(ctx, reqData, paymentService)
	if err != nil {
		s.log.Error("ERROR on set fiscal data: %v", err)
		return nil, err
	}
	// response to json
	r, _ := json.Marshal(res)
	// save response
	err = s.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      r,
	})
	if err != nil {
		s.log.Info("Error on saving payme go response: %v", err.Error())
		return nil, err
	}
	return res, nil
}

// DoRequest for Payme Go
func (s *Services) PaymeGoDoRequest(ctx context.Context, data any, paymentServe *domain.PaymentService) (*domain.PaymeGoResponse, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}
	// get url
	url := s.cfg.Payment.PaymeGoEndpointUrl
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, err
	}
	// add headers
	req.Header.Add("X-Auth", paymentServe.CashboxId+":"+paymentServe.SecretKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res domain.PaymeGoResponse
	// read response body
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		s.log.Error("ERROR on decoding response: %w", err)
		return nil, err
	}
	return &res, nil
}

// Save payment request to database
func (h *Services) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	query := `
	INSERT INTO payment_requests (
		request_id, method, payload, transaction_id, payment_provider
		)
		VALUES (?, ?, ?, ?, ?)`
	err := h.db.
		WithContext(ctx).
		Exec(query,
			req.RequestId, req.Method, req.Payload, req.TransactionID, req.PaymentProvider,
		).Error
	if err != nil {
		h.log.Error("ERROR on saving payment request: %w", err)
		return err
	}
	return nil
}

// Save payment response to database
func (h *Services) SaveResponse(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Exec(
		`UPDATE payment_requests SET response = ? WHERE transaction_id = ?`,
		req.Response, req.TransactionID,
	).Error
	if err != nil {
		h.log.Error("ERROR on saving payment response: %w", err)
		return err
	}
	return nil
}

// save receipt id to sale_payments
func (h *Services) SetReceiptId(ctx context.Context, receiptId, salePayId string) error {
	query := `UPDATE sale_payments SET receipt_id = ? WHERE id = ?`
	err := h.db.Exec(query, receiptId, salePayId).Error
	if err != nil {
		h.log.Warn("ERROR on setting receipt_id: %v", err)
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
