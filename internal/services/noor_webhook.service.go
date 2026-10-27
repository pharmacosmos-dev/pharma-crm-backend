package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

const (
	noorWebhookAttempts       = 3
	noorWebhookAttemptTimeout = 8 * time.Second
)

// SendNoorOrderStatus sends the order status to the noor webhook.
// 429, 5xx and network errors are retried: noor says a repeat is safe.
// 400/401/404/413 are not retried, the request itself has to be fixed.
func (s *Services) SendNoorOrderStatus(ctx context.Context, req *domain.NoorOrderStatusRequest) error {
	if s.cfg.NoorWebhookUrl == "" || s.cfg.NoorWebhookKey == "" {
		s.log.Errorf("noor webhook is not configured (NOOR_WEBHOOK_URL, NOOR_WEBHOOK_KEY), %s status of order %d is not sent",
			req.Status, req.OrderId)
		return domain.NoorNotOperationalError
	}

	// noor finds the order by its own id
	if req.ExternalId == "" {
		s.log.Errorf("noor order %d has no external_id, %s status is not sent", req.OrderId, req.Status)
		return domain.NoorNotOperationalError
	}

	data, err := json.Marshal(req)
	if err != nil {
		s.log.Errorf("could not marshal noor order status request: %v", err)
		return domain.InternalServerError
	}

	var lastErr error
	attempts:
	for attempt := 1; attempt <= noorWebhookAttempts; attempt++ {
		if attempt > 1 {
			// pause: 1s, then 2s
			select {
			case <-ctx.Done():
				lastErr = fmt.Errorf("%v (last error: %v)", ctx.Err(), lastErr)
				break attempts
			case <-time.After(time.Duration(attempt-1) * time.Second):
			}
		}

		statusCode, body, err := s.doNoorWebhookRequest(ctx, data)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			return nil
		}

		lastErr = fmt.Errorf("status %d, body: %s", statusCode, string(body))
		if statusCode != http.StatusTooManyRequests && statusCode < http.StatusInternalServerError {
			break
		}
	}

	s.log.Errorf("could not send %s status of noor order %d (external_id %s): %v",
		req.Status, req.OrderId, req.ExternalId, lastErr)
	return domain.NoorNotOperationalError
}

// one webhook call: returns status code and (limited) response body
func (s *Services) doNoorWebhookRequest(ctx context.Context, data []byte) (int, []byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, noorWebhookAttemptTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, s.cfg.NoorWebhookUrl, bytes.NewReader(data))
	if err != nil {
		return 0, nil, fmt.Errorf("could not construct request: %w", err)
	}

	// server env'da qiymat qo'shtirnoq bilan yozilib qolsa ham kalit buzilmasin
	key := strings.Trim(s.cfg.NoorWebhookKey, `"'`)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJson)
	req.Header.Set("Accept", constants.ContentTypeJson)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, body, nil
}
