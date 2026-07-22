package services

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (s *Services) Request1CCreate(req domain.InventoryHelper) error {
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	responseBytes, err := json.Marshal(req.Response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	err = s.db.Table("requests_1c").Create(&domain.RequestOnec{
		Method:   req.Method,
		Payload:  payloadBytes,
		Response: responseBytes,
		Action:   req.Action,
		DocDate:  req.DocDate,
		DocNum:   req.DocNum,
		Status:   req.Status,
	}).Error
	if err != nil {
		return fmt.Errorf("failed to create RequestOnec: %w", err)
	}

	return nil
}

// recoverTransaction handles panics and rolls back the transaction if necessary.
func recoverTransaction(tx *gorm.DB, log logger.Interface) {
	if r := recover(); r != nil {
		log.Error("panic recovered:", r)
		tx.Rollback()
	}
}

// RollbackIfError checks if the error pointer is not nil and if it contains an error.
func RollbackIfError(tx *gorm.DB, errPtr *error) {
	if errPtr != nil && *errPtr != nil {
		tx.Rollback()
	}
}

func (s *Services) GetSignedUser(c *gin.Context) *domain.EmployeeClaims {
	user := domain.EmployeeClaims{}

	if userId, ok := c.Get("user_id"); ok && userId != nil {
		user.UserId, _ = userId.(string)
	}

	if companyId, ok := c.Get("company_id"); ok && companyId != nil {
		user.CompanyId, _ = companyId.(string)
	}

	if storeId, ok := c.Get("store_id"); ok && storeId != nil {
		user.StoreId, _ = storeId.(string)
	}

	if storeIds, ok := c.Get("store_ids"); ok && storeIds != nil {
		switch v := storeIds.(type) {
		case []string:
			user.StoreIds = v
		case []interface{}:
			user.StoreIds = make([]string, 0, len(v))
			for _, id := range v {
				user.StoreIds = append(user.StoreIds, cast.ToString(id))
			}
		case string:
			if v != "" {
				user.StoreIds = []string{v}
			}
		default:
			// fallback: try to cast to string
			user.StoreIds = []string{cast.ToString(v)}
		}
	}

	if role, ok := c.Get("role"); ok && role != nil {
		user.Role, _ = role.(string)
	}

	return &user
}

func (s *Services) ConvertIntegerTo8DigitEquivalent(value int) (int, int) {
	digitDifference := 8 - len(strconv.Itoa(value))

	if digitDifference <= 0 {
		return value, value
	}

	zeroes := int(math.Pow(float64(10), float64(digitDifference)))

	return value * zeroes, (value + 1) * zeroes
}

func (s *Services) FormatDatetimeParams(startTime, endTime string) (domain.FilterDatetimeDto, error) {
	fromTime, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		s.log.Errorf("could not parse filter start_time: %v", err)
		return domain.FilterDatetimeDto{}, domain.InvalidTimeFormatError
	}

	var tillTime time.Time
	if endTime == "" {
		// if endTime will be empty, startTime + 24 soat
		tillTime = fromTime.Add(24 * time.Hour)
	} else {
		tillTime, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			s.log.Errorf("could not parse filter end_time: %v", err)
			return domain.FilterDatetimeDto{}, domain.InvalidTimeFormatError
		}
	}

	// Convert to UTC timezone
	fromTimeUTC := fromTime.UTC()
	tillTimeUTC := tillTime.UTC()

	diff := tillTimeUTC.Sub(fromTimeUTC)
	if diff == 0 {
		diff = 24 * time.Hour
	}

	res := domain.FilterDatetimeDto{
		StartTime:     fromTimeUTC.Format(time.RFC3339),
		EndTime:       tillTimeUTC.Format(time.RFC3339),
		PrevStartTime: fromTimeUTC.Add(-diff).Format(time.RFC3339),
		PrevEndTime:   tillTimeUTC.Add(-diff).Format(time.RFC3339),
	}

	return res, nil
}

func (s *Services) ConvenrtTimeAsiaTashkent(timeStr string) (time.Time, error) {
	dateTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		s.log.Errorf("could not parse filter start_time: %v", err)
		return dateTime, domain.InvalidTimeFormatError
	}
	return dateTime.Add(constants.DateTimeTashkent), nil
}

func (s *Services) normalizeEposResponse(req *domain.EposResponseRequest) (string, error) {
	var raw []byte
	switch v := req.ResponseData.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	case nil:
		return "", nil
	default:
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			return "", err
		}
	}

	if len(raw) == 0 {
		return "", nil
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		// If not JSON, return as is (could be a simple message)
		return string(raw), nil
	}

	// Update req.Error if found in data
	if isErr, ok := data["error"].(bool); ok && isErr {
		req.Error = true
	}

	// Case 1: Nested in data.receipt (Format 2)
	if dataMap, ok := data["data"].(map[string]any); ok {
		if receipt, ok := dataMap["receipt"].(map[string]any); ok {
			// Transform Format 2 to the expected EposSuccessResponse format
			normalized := map[string]any{
				"error": data["error"],
				"message": map[string]any{
					"dateTime":   receipt["date_time"],
					"qrCodeURL":  receipt["qr_code_url"],
					"fiscalSign": receipt["fiscal_sign"],
					"receiptSeq": cast.ToString(receipt["receipt_seq"]),
					"terminalId": receipt["terminal_id"],
					"cash":       receipt["cash"],
					"card":       receipt["card"],
					"amount":     receipt["eps_amount"],
				},
			}
			res, _ := json.Marshal(normalized)
			return string(res), nil
		}
	}

	// Case 2: Success in message as an object (Format 3)
	if msgObj, ok := data["message"].(map[string]any); ok {
		// Normalize fields if they are in snake_case and missing in camelCase
		fields := map[string]string{
			"date_time":   "dateTime",
			"qr_code_url": "qrCodeURL",
			"fiscal_sign": "fiscalSign",
			"receipt_seq": "receiptSeq",
			"terminal_id": "terminalId",
		}
		for snake, camel := range fields {
			if _, exists := msgObj[camel]; !exists && msgObj[snake] != nil {
				msgObj[camel] = msgObj[snake]
			}
		}
		// ensure receiptSeq is string for EposSuccessMessage
		if rs, ok := msgObj["receiptSeq"]; ok {
			msgObj["receiptSeq"] = cast.ToString(rs)
		}

		res, _ := json.Marshal(data)
		return string(res), nil
	}

	return string(raw), nil
}
