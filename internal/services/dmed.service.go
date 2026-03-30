package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

func (s *Services) GetPrescriptionsFromDMED(patientId, safeCode string) ([]domain.Prescription, error) {
	url := fmt.Sprintf("/prescriptions?patient_id=%s&safe_code=%s", patientId, safeCode)

	// request payload for logging
	payload := domain.DmedGeneral[domain.DmedGerPrescripReq]{
		Request: domain.DmedGerPrescripReq{
			PatientId: patientId,
			SafeCode:  safeCode,
			Url:       url,
		},
	}

	id, _ := SaveDmedRequest(context.Background(), s.db, s.log, "GET-prescriptions", payload)

	// Send request dmed receipt
	var response *http.Response
	if err := s.DmedRequest(
		&response,
		http.MethodGet,
		s.cfg.DmedApiUrl+url,
		nil,
	); err != nil {
		var dmedErr *domain.DmedError
		if errors.As(err, &dmedErr) {
			// ✅ Save real API error response
			_ = s.SaveDmedResponse(
				context.Background(),
				id,
				dmedErr.Body,
				0,
			)

			s.log.Errorf(
				"dmed error: status=%d body=%s",
				dmedErr.StatusCode,
				string(dmedErr.Body),
			)
			err = FormatDmedErrorResponse(dmedErr.Body)
			if err != nil {
				return nil, err
			}
		} else {
			s.log.Errorf("unexpected dmed error: %v", err)
		}
		return nil, err
	}

	defer utils.Close(response.Body, s.log)

	result, bytes, err := DecodeDmedResponse[[]domain.Prescription](response.Body)
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

			payload.Url = url

			reqPayload := domain.DmedGeneral[domain.DmedGiveReceiptReq]{
				Request: payload,
			}

			id, _ := SaveDmedRequest(context.Background(), s.db, s.log, method+"-"+action, reqPayload)

			// Send request dmed receipt
			var response *http.Response
			if err := s.DmedRequest(
				&response,
				method,
				s.cfg.DmedApiUrl+url,
				jsonBytes,
			); err != nil {
				var dmedErr *domain.DmedError
				if errors.As(err, &dmedErr) {
					// ✅ Save real API error response
					_ = s.SaveDmedResponse(
						context.Background(),
						id,
						dmedErr.Body,
						0,
					)

					s.log.Errorf(
						"dmed error: status=%d body=%s",
						dmedErr.StatusCode,
						string(dmedErr.Body),
					)
					err = FormatDmedErrorResponse(dmedErr.Body)
					if err != nil {
						return err
					}
				} else {
					s.log.Errorf("unexpected dmed error: %v", err)
				}
				return err
			}

			defer utils.Close(response.Body, s.log)

			resBytes, err := io.ReadAll(response.Body)
			// save response payload
			_ = s.SaveDmedResponse(context.Background(), id, resBytes, 1)
			if err != nil {
				s.log.Errorf("could not decode get prescriptions response: %v", err)
				return err
			}
			j++
		}
	}
	return nil
}

func SaveDmedRequest[T any](
	ctx context.Context,
	db *gorm.DB,
	log *logger.Logger,
	method string,
	payload domain.DmedGeneral[T],
) (int64, error) {

	payloadDb, err := json.Marshal(payload.Request)
	if err != nil {
		log.Errorf("could not marshal dmed payload: %v", err)
		return 0, err
	}
	var id int64
	err = db.WithContext(ctx).
		Raw(
			"INSERT INTO dmed_requests(payload, method) VALUES(?, ?) RETURNING id;",
			string(payloadDb), method,
		).
		Scan(&id).Error

	if err != nil {
		log.Errorf("could not save dmed request payload: %v", err)
		return 0, domain.InternalServerError
	}

	return id, nil
}

func (s *Services) SaveDmedResponse(ctx context.Context, reqId int64, response []byte, status int) error {
	err := s.db.WithContext(ctx).
		Exec("UPDATE dmed_requests SET response = ?, status = ?, updated_at = NOW() WHERE id = ?;",
			response, status, reqId,
		).Error
	if err != nil {
		s.log.Errorf("could not save dmed response payload: %v -> %s", err, string(response))
		return domain.InternalServerError
	}

	return nil
}

func DecodeDmedResponse[T any](r io.Reader) (domain.DmedResponseWrapper[T], []byte, error) {
	var result domain.DmedResponseWrapper[T]

	response, err := io.ReadAll(r)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return result, response, err
	}

	if result.Message != "" {
		error := fmt.Errorf(
			"dmed error message: %s",
			result.Message,
		)
		// Convert payme errors to out own
		switch result.Message {
		case constants.DmedPrescriptionsNotFound:
			return result, response, domain.PrescriptionNotFound
		case constants.DmedPrescriptionsExpired:
			return result, response, domain.PrescriptionExpiredError
		case constants.DmedPrescriptionsAlreadyIssued:
			return result, response, domain.PrescriptionsAlreadyIssued
		default:
			return result, response, error
		}
	}

	return result, response, nil
}

func FormatDmedErrorResponse(resp []byte) error {
	var result domain.DmedResponseWrapper[any]

	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if result.Message != "" {
		error := fmt.Errorf(
			"dmed error message: %s",
			result.Message,
		)
		// Convert payme errors to out own
		switch result.Message {
		case constants.DmedPrescriptionsNotFound:
			return domain.PrescriptionNotFound
		case constants.DmedPrescriptionsExpired:
			return domain.PrescriptionExpiredError
		case constants.DmedPrescriptionsAlreadyIssued:
			return domain.PrescriptionsAlreadyIssued
		default:
			return error
		}
	}

	return nil
}
