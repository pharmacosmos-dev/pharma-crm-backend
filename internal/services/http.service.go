package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

func (s *Services) DmedRequest(res **http.Response, method string, url string, data []byte) error {
	newToken := strings.Trim(s.cfg.DmedApiToken, `"'`)
	auth := fmt.Sprintf("Bearer %s", newToken)
	headers := map[string]string{
		constants.HeaderContentType: constants.ContentTypeJson,
		"Authorization":             auth,
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
