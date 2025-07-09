package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pharma-crm-backend/config"
)

// do request
func DoRequest(
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
	req.Header.Set("Accept", config.ContentTypeJson)

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
		fmt.Println(err)

		return err
	}

	return nil
}
