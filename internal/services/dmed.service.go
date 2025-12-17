package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

func (s *Services) GetPrescriptionsFromDMED(patientId, safeCode string) ([]domain.Prescription, error) {
	url := fmt.Sprintf("/prescriptions?patient_id=%s&safe_code=%s", patientId, safeCode)

	// request payload for logging
	reqPayload := domain.DmedGerPrescripReq{
		PatientId: patientId,
		SafeCode:  safeCode,
		Url:       url,
	}

	jsonBytes, err := json.Marshal(&reqPayload)
	if err != nil {
		return nil, err
	}

	id, _ := s.SaveDmedRequest(context.Background(), "GET-prescriptions", jsonBytes)

	// Send request dmed receipt
	var response *http.Response
	if err := s.DmedRequest(
		&response,
		http.MethodGet,
		s.cfg.DmedApiUrl+url,
		nil,
	); err != nil {
		s.log.Errorf("could not send dmed request: %v", err)
		return nil, err
	}

	defer utils.Close(response.Body, s.log)

	result, bytes, err := DecodeDmedResponse[domain.PrescriptionResponse](response.Body)
	// save response payload
	_ = s.SaveDmedResponse(context.Background(), id, bytes, 1)
	if err != nil {
		s.log.Errorf("could not decode get prescriptions response: %v", err)
		return result.Data, err
	}

	return result.Data, nil
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
			payload := domain.DmedGiveReceiptReq{
				DrugAmount:       drugAmount,
				Price:            int(cartItem.UnitPrice),
				IssuedByFullName: employeeName,
			}

			if j < len(markingData[i].MarkingList) && markingData[i].MarkingList[j] != "" {
				payload.MarkingCode = markingData[i].MarkingList[j]
			} else if cartItem.SerialNumber != "" && cartItem.Barcode != "" {
				payload.SerialNumber = cartItem.SerialNumber
				payload.Gtin = "010" + cartItem.Barcode
			} else {
				s.log.Error("could not find serial number or marking code for dmed")
				return domain.SerialOrMarkingRequiredError
			}

			url := fmt.Sprintf("/prescriptions/%d/%s", markingData[i].DmedId, action)
			method := http.MethodPost
			if action == "issue" {
				method = http.MethodPut
			}

			jsonBytes, err := json.Marshal(&payload)
			if err != nil {
				return err
			}

			id, _ := s.SaveDmedRequest(context.Background(), method+action, jsonBytes)

			// Send request dmed receipt
			var response *http.Response
			if err := s.DmedRequest(
				&response,
				method,
				s.cfg.DmedApiUrl+url,
				jsonBytes,
			); err != nil {
				s.log.Errorf("could not send dmed request: %v", err)
				return domain.InternalServerError
			}

			defer utils.Close(response.Body, s.log)

			dmedRes, err := io.ReadAll(response.Body)
			if err != nil {
				s.log.Errorf("could not decode dmed response: %v", err)
				return domain.InternalServerError
			}
			_ = s.SaveDmedResponse(context.Background(), id, dmedRes, 1)
			j++
		}
	}
	return nil
}

// func (s *Services) doRequestToDMED(method, url string, data any) ([]byte, error) {
// 	var (
// 		body       []byte
// 		bodyReader io.Reader
// 		err        error
// 	)

// 	if data != nil {
// 		body, err = json.Marshal(data)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to marshal data: %w", err)
// 		}
// 		fmt.Printf("Request body DMED: %s\n", string(body))
// 		bodyReader = bytes.NewReader(body)
// 	}

// 	req, err := http.NewRequest(method, s.cfg.DmedApiUrl+url, bodyReader)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}
// 	newToken := strings.Trim(s.cfg.DmedApiToken, `"'`)
// 	req.Header.Set("Accept", "application/json")
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+newToken)

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to send request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	respBody, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read response: %w", err)
// 	}
// 	fmt.Println(string(respBody))
// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		s.log.Errorf("dmed request failed: %s", string(respBody))
// 		return nil, fmt.Errorf("DMED API error: %s", string(respBody))
// 	}

// 	return respBody, nil
// }

func (s *Services) SaveDmedRequest(ctx context.Context, method string, payload []byte) (int64, error) {
	var id int64
	err := s.db.WithContext(ctx).Debug().
		Raw("INSERT INTO dmed_requests(payload, method) VALUES(?, ?) RETURNING id;",
			json.RawMessage(payload), method,
		).Scan(&id).Error
	if err != nil {
		s.log.Errorf("could not save dmed request payload: %v", err)
		return 0, domain.InternalServerError
	}

	return id, nil
}

func (s *Services) SaveDmedResponse(ctx context.Context, reqId int64, response []byte, status int) error {
	err := s.db.WithContext(ctx).Raw("UPDATE dmed_requests SET response = ?, status = ?, updated_at = NOW() WHERE id = ?;",
		json.RawMessage(response), status, reqId,
	).Error
	if err != nil {
		s.log.Errorf("could not save dmed response payload: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func DecodeDmedResponse[T any](r io.Reader) (domain.PrescriptionResponse, []byte, error) {
	var result domain.PrescriptionResponse

	response, err := io.ReadAll(r)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return result, response, err
	}

	return result, response, nil
}
