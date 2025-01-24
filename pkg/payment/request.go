package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/pharma-crm-backend/domain"
)

// DoRequest for Click pass
func (h *PaymentAction) ClickPassDoRequest(ctx context.Context, url string, data interface{}, token string) (*domain.ClickPassResponse, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", h.cfg.ClickEndpointUrl+url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Auth", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result domain.ClickPassResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DoRequest for Payme Go
func (h *PaymentAction) PaymeGoDoRequest(ctx context.Context, data interface{}) (*domain.ClickPassResponse, error) {
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

// DoRequest for Uzum Fast Pay
func (h *PaymentAction) UzumFastPayDoRequest(ctx context.Context, data interface{}) (*domain.ClickPassResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}
