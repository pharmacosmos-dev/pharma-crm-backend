package services

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
