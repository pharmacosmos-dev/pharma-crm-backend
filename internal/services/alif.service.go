package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// Alif Pay
func (h *Services) AlifPay(ctx context.Context, paymentService *domain.PaymentService, sale *domain.Sale) error {
	alifData := domain.AlifPaymentRequest{
		ID:     sale.Id,
		Amount: int64(sale.TotalAmount * constants.SumsToTiyns),
		Method: domain.AlifMethod{
			Type:  "MOBI_SHOW_QR",
			Token: sale.OtpCode,
		},
	}
	t, err := json.Marshal(alifData)
	if err != nil {
		return err
	}
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
		RequestId:       time.Now().Unix(),
		Method:          "alif_pay",
		Payload:         t,
		TransactionID:   sale.Id,
		PaymentProvider: "alif",
	})
	if err != nil {
		h.log.Info("Error on saving alif pay request: %v", err.Error())
		return err
	}

	res, err := h.AlifPayDoRequest(ctx, "/v2/pay", alifData, paymentService.CashboxId)
	if err != nil {
		return err
	}
	t, _ = json.Marshal(res)
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: sale.Id,
		Response:      t,
		Method:        "alif_pay",
	})
	if err != nil {
		return err
	}

	if status, ok := res["status"].(string); ok && status == constants.GeneralStatusDeclined {
		h.log.Warn("Payment declined for transactionID=%s", sale.Id)
		return fmt.Errorf("payment declined by alif")
	}

	return nil
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
	fullURL := h.cfg.AlifApiUrl + url
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store-Token", token)

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
