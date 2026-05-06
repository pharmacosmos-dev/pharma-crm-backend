package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// payme request
func (s *Services) PaymeRequest(res **http.Response, url string, data []byte, token string) error {
	headers := map[string]string{
		constants.HeaderContentType: constants.ContentTypeJson,
		constants.HeaderXAuth:       token,
		constants.HeaderHost:        strings.TrimPrefix(s.cfg.PaymeApiUrl, "https://"),
	}

	return s.DoRequest(res, http.MethodPost, url, data, headers)
}

// receipts.pay uchun context timeout bilan (developer tavsiyasi)
func (s *Services) PaymePayReceiptRequest(ctx context.Context, res **http.Response, url string, data []byte, token string) error {
	payCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(payCtx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("could not construct payme pay receipt request to %s: %v", url, err)
		return err
	}
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJson)
	req.Header.Set(constants.HeaderXAuth, token)
	req.Header.Set(constants.HeaderHost, strings.TrimPrefix(s.cfg.PaymeApiUrl, "https://"))
	req.Header.Set("Accept", constants.ContentTypeJson)

	client := http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("payme pay receipt request to %s failed: %v", url, err)
		return err
	}
	*res = response

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(response.Body)
		response.Body.Close()
		return fmt.Errorf("payme pay receipt: status %d, body: %s", response.StatusCode, string(errBody))
	}
	return nil
}

// click request
func (s *Services) ClickRequest(res **http.Response, url string, data []byte, token string) error {
	headers := map[string]string{
		constants.HeaderContentType: constants.ContentTypeJson,
		constants.HeaderAuth:        token,
	}

	return s.DoRequest(res, http.MethodPost, url, data, headers)
}

// alif request
func (s *Services) AlifRequest(res **http.Response, url string, data []byte, token string) error {
	headers := map[string]string{
		constants.HeaderContentType: constants.ContentTypeJson,
		constants.HeaderStoreToken:  token,
	}

	return s.DoRequest(res, http.MethodPost, url, data, headers)
}

// dmed request
func (s *Services) DmedRequest(
	res **http.Response,
	method, url string,
	data []byte,
) error {

	newToken := strings.Trim(s.cfg.DmedApiToken, `"'`)
	auth := fmt.Sprintf("Bearer %s", newToken)

	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	req.Header.Set("Lang", constants.LanguageUz)
	req.Header.Set("Accept", constants.ContentTypeJson)

	client := http.Client{Timeout: 20 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	*res = resp

	// Check response status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return &domain.DmedError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	return nil
}

// noor request
func (s *Services) NoorRequest(res **http.Response, method string, url string, data []byte) error {
	headers := map[string]string{
		constants.HeaderContentType: constants.ContentTypeJson,
		"Authorization":             fmt.Sprintf("Bearer %s", s.cfg.NoorApiToken),
	}
	return s.DoRequest(res, method, url, data, headers)
}

// do request
func (s *Services) DoRequest(
	res **http.Response,
	method string,
	url string,
	data []byte,
	headers map[string]string,
) error {
	var client http.Client
	// Create new HTTP request
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("could not construct new request to %s: %v", url, err)
		return err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Accept", constants.ContentTypeJson)

	client = http.Client{
		Timeout: time.Second * 20,
	}

	// Send the request using http.Client
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("request to %s failed: %v", url, err)
		return err
	}

	*res = response

	// Check response status code
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errBody, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("call to %s resulted in %d status code, body: %e", url, response.StatusCode, err)
		}
		err = fmt.Errorf(
			"call to %s resulted in %d status code, body: %s",
			url,
			response.StatusCode,
			string(errBody),
		)

		return err
	}

	return nil
}
