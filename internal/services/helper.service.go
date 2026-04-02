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


// normalizeEposResponse ikki xil formatni bitta JSON string ga keltiradi.
// Format 1: {"error":false,"message":{...fiscal...}}
// Format 2: {"data":{"exists":true,"receipt":{...fiscal...}},"error":false}
// req.Error ni ham shu yerda set qiladi (outer yoki inner error).
func (s *Services) normalizeEposResponse(req *domain.EposResponseRequest) (string, error) {
    // response_data -> []byte
    var raw []byte
    if len(req.ResponseData) > 0 {
        raw = []byte(req.ResponseData)
        // Agar bu string bo'lsa (qo'shtirnoq ichida bo'lsa), unmarshal qilib ichini olamiz
        var strVal string
        if json.Unmarshal(raw, &strVal) == nil {
            raw = []byte(strVal)
        }
    }
    req.Response = raw

    // Umumiy wrapper parse
    var wrapper struct {
        Error   bool            `json:"error"`
        Message json.RawMessage `json:"message"` // Format 1
        Data    *struct {
            Exists  bool            `json:"exists"`
            Receipt json.RawMessage `json:"receipt"`
        } `json:"data"` // Format 2
    }

    if err := json.Unmarshal(raw, &wrapper); err != nil {
        s.log.Errorf("normalizeEposResponse: unmarshal failed: %v", err)
        return "", domain.BadRequestError
    }

    // Outer error flag
    if wrapper.Error {
        req.Error = true
        return "", nil
    }

    // Format 2: {"data":{"exists":true,"receipt":{...}}}
    if wrapper.Data != nil {
        if !wrapper.Data.Exists || wrapper.Data.Receipt == nil {
            req.Error = true
            return "", nil
        }
        // receipt field nomlari snake_case — DecodeFiscalData uchun normalize
        normalizedReceipt, err := s.normalizeReceiptFields(wrapper.Data.Receipt)
        if err != nil {
            return "", err
        }
        return fmt.Sprintf(`{"error":false,"message":%s}`, normalizedReceipt), nil
    }

    // Format 1: {"message":{...}} yoki {"message":"ERROR_STRING"}
    if wrapper.Message != nil {
        // message string bo'lsa (xato holat)
        var msgStr string
        if json.Unmarshal(wrapper.Message, &msgStr) == nil {
            s.log.Warnf("epos message is error string: %s", msgStr)
            req.Error = true
            return "", nil
        }
        return string(raw), nil
    }

    s.log.Error("normalizeEposResponse: unrecognized format")
    return "", domain.BadRequestError
}

// normalizeReceiptFields Format-2 receipt (snake_case) -> Format-1 ga o'xshash camelCase
func (s *Services) normalizeReceiptFields(raw json.RawMessage) (string, error) {
    var r struct {
        FiscalSign string `json:"fiscal_sign"`
        QrCodeUrl  string `json:"qr_code_url"`
        DateTime   string `json:"date_time"`
        ReceiptSeq any    `json:"receipt_seq"`
        TerminalId string `json:"terminal_id"`
        Cash       any    `json:"cash"`
        Card       any    `json:"card"`
    }
    if err := json.Unmarshal(raw, &r); err != nil {
        return "", domain.BadRequestError
    }

    normalized := map[string]any{
        "fiscalSign": r.FiscalSign,
        "qrCodeURL":  r.QrCodeUrl,
        "dateTime":   r.DateTime,
        "receiptSeq": r.ReceiptSeq,
        "terminalId": r.TerminalId,
        "cash":       r.Cash,
        "card":       r.Card,
    }
    b, err := json.Marshal(normalized)
    if err != nil {
        return "", domain.InternalServerError
    }
    return string(b), nil
}
