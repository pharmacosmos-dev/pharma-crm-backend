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
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

// ClickPass implements PaymentService
func (s *Services) ClickPass(ctx context.Context, click *domain.PaymentService, sale *domain.Sale) (*domain.ClickPassResponse, error) {
	// Click Pass request body
	payload := domain.ClickPassRequest{
		ServiceId:     click.ServiceID,
		OtpData:       sale.OtpCode,
		CashboxCode:   sale.CashboxId,
		Amount:        sale.Click,
		TransactionId: sale.Id,
	}

	reqId, err := s.createClickRequestInDb(ctx, payload, constants.ActionClickPassCreate, sale.Id)
	if err != nil {
		return nil, err
	}
	// generate click pass auth token
	token := s.generateClickToken(click.SecretKey, click.MerchantUserID)

	// Prepare request body
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal create receipt payload: %v", err)
		return nil, domain.InternalServerError
	}
	// Send request to create receipt
	var response *http.Response
	if err := s.ClickRequest(
		&response,
		s.cfg.ClickApiUrl+constants.ClickPassCreatePath,
		jsonBytes,
		token,
	); err != nil {
		s.log.Errorf("could not send click_pass create request: %v", err)
		return nil, domain.InternalServerError
	}

	defer utils.Close(response.Body, s.log)

	result, bytes, err := DecodeClickResponse[domain.ClickPassResponse](response.Body)
	_ = s.updateClickRequestInDb(ctx, reqId, bytes, constants.ActionClickPassCreate)
	if err != nil {
		s.log.Errorf("could not decode create click_pass response: %v", err)
		return &result, err
	}

	return &result, nil
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
func (h *Services) generateClickToken(secretKey string, merchantUserId int) string {
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
	fullURL := h.cfg.ClickApiUrl + url
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

func (s *Services) createClickRequestInDb(ctx context.Context, payload domain.ClickPassRequest, method string, saleId string) (int, error) {
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
		constants.PaymentTypeClick,
	).Scan(&seqId).Error
	if err != nil {
		s.log.Errorf("could not create click_pass request(%f) in db: %v", payload.Amount, err)
		return seqId, err
	}

	return seqId, nil
}

func (s *Services) updateClickRequestInDb(ctx context.Context, id int, response []byte, method string) error {
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

func DecodeClickResponse[T any](r io.Reader) (domain.ClickPassResponse, []byte, error) {
	var result domain.ClickPassResponse

	response, err := io.ReadAll(r)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return result, response, err
	}

	if result.ErrorCode < 0 {
		fmt.Printf("click payment failed code(%d) error_note: %s", result.ErrorCode, result.ErrorNote)
		return result, response, domain.ClickNotOperationalError
	}

	return result, response, nil
}
