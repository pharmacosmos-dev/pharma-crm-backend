package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetPrescriptionsFromDMED(patientID, safeCode string) ([]domain.Prescription, error) {
	url := fmt.Sprintf("/prescriptions?patient_id=%s&safe_code=%s", patientID, safeCode)

	// request payload for logging
	reqPayload := map[string]any{
		"patient_id": patientID,
		"safe_code":  safeCode,
		"url":        url,
	}
	id, _ := s.SaveDmedRequest(context.Background(), "GET-prescriptions", reqPayload)

	respBody, err := s.doRequestToDMED("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// save response payload
	_ = s.SaveDmedResponse(context.Background(), id, respBody, 1)

	var rawResp = domain.PrescriptionResponse{}
	if err = json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return rawResp.Data, nil
}

func (s *Services) DmedGiveReceipt(cartItems []domain.CartItemForDMED, markingData []domain.MarkingData, employeeName, prescriptionID, action string) error {
	for i, cartItem := range cartItems {
		q := cartItem.Quantity
		uq := cartItem.UnitQuantity
		j := 0

		for q > 0 || uq > 0 {
			var drugAmount int
			if q > 0 {
				drugAmount = cartItem.UnitPerPack
				q--
			} else if uq > 0 {
				drugAmount = uq
				uq = 0
			}

			payload := map[string]any{
				"drug_amount":         drugAmount,
				"price":               int(cartItem.UnitPrice),
				"issued_by_full_name": employeeName,
				// //   "pharmacy_id": 123,
			}
			if j < len(markingData[i].MarkingList) && markingData[i].MarkingList[j] != "" {
				payload["marking_code"] = markingData[i].MarkingList[j]
			} else if cartItem.SerialNumber != "" && cartItem.Barcode != "" {
				payload["serial_number"] = cartItem.SerialNumber
				payload["gtin"] = "010" + cartItem.Barcode
			} else {
				s.log.Error("could not find serial number or marking code for dmed")
				return domain.SerialOrMarkingRequiredError
			}

			url := fmt.Sprintf("/prescriptions/%d/%s", markingData[i].DmedId, action)
			method := http.MethodPost
			if action == "issue" {
				method = http.MethodPut
			}

			id, _ := s.SaveDmedRequest(context.Background(), method+action, payload)

			res, err := s.doRequestToDMED(method, url, payload)
			if err != nil {
				s.log.Error("could not send dmed %s request: %v", action, err)
				return fmt.Errorf("DMED %s failed: %w", action, err)
			}
			_ = s.SaveDmedResponse(context.Background(), id, res, 1)
			j++
		}
	}
	return nil
}

func (s *Services) doRequestToDMED(method, url string, data any) ([]byte, error) {
	var (
		body       []byte
		bodyReader io.Reader
		err        error
	)

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		fmt.Printf("Request body DMED: %s\n", string(body))
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, s.cfg.DmedApiUrl+url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	newToken := strings.Trim(s.cfg.DmedApiToken, `"'`)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	fmt.Println(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("dmed request failed: %s", string(respBody))
		return nil, fmt.Errorf("DMED API error: %s", string(respBody))
	}

	return respBody, nil
}

func (s *Services) SaveDmedRequest(ctx context.Context, method string, payload map[string]any) (int64, error) {
	payloadDb, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("could not marshal dmed request payload: %v", err)
		return 0, domain.InternalServerError
	}

	var id int64
	err = s.db.WithContext(ctx).
		Raw("INSERT INTO dmed_requests(payload, method) VALUES(?, ?) RETURNING id;",
			payloadDb, method,
		).Scan(&id).Error
	if err != nil {
		s.log.Errorf("could not save dmed request payload: %v", err)
		return 0, domain.InternalServerError
	}

	return id, nil
}

func (s *Services) SaveDmedResponse(ctx context.Context, reqId int64, response []byte, status int) error {
	err := s.db.WithContext(ctx).Raw("UPDATE dmed_requests SET response = ?, status = ?, updated_at = NOW() WHERE id = ?;",
		response, status, reqId,
	).Error
	if err != nil {
		s.log.Errorf("could not save dmed response payload: %v", err)
		return domain.InternalServerError
	}

	return nil
}
