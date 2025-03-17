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
func (h *Services) ClickPass(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]any, error) {
	var cashBoxId string
	err := h.db.Raw(`SELECT cash_box_id FROM cashbox_operations WHERE id = ?`, CashOperationID).Scan(&cashBoxId).Error
	if err != nil {
		return nil, err
	}
	// Click Pass request body
	clickData := domain.ClickPassRequest{
		ServiceID:     click.ServiceID,
		OtpData:       data.OtpData,
		CashboxCode:   cashBoxId,
		Amount:        data.Amount,
		TransactionID: transactionID,
	}
	// Marshal click pass request
	t, _ := json.Marshal(clickData)
	// Save request of one click pass data
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
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
	fullUrl := h.cfg.ClickEndpointUrl + fmt.Sprintf("/payment/status/%v/%v", data["service_id"], data["payment_id"])
	res, err := h.ClickPassDoRequest(ctx, fullUrl, data, token)
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

// Payme Go Handler functon
func (h *Services) PaymeGo(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error) {
	return h.PaymeGoDoRequest(ctx, data)
}

// DoRequest for Payme Go
func (h *Services) PaymeGoDoRequest(ctx context.Context, data interface{}) (map[string]interface{}, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Auth", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

// Uzum fast pay handler function
func (h *Services) UzumFastPay(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error) {
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
func (h *Services) UzumFastPayCheckPaymentStatus(ctx context.Context, paymentService domain.PaymentService, paymentId string) (map[string]interface{}, error) {
	data := map[string]interface{}{
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
func (h *Services) UzumFastPayDoRequest(ctx context.Context, url string, data interface{}, token string) (map[string]interface{}, error) {
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

// Save payment request to database
func (h *Services) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Create(&req).Error
	if err != nil {
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
		return err
	}
	return nil
}
