package services

import (
	"encoding/json"
	"fmt"

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

	err = s.db.Table("requests_1c").Create(&domain.Request1C{
		Method:   req.Method,
		Payload:  payloadBytes,
		Response: responseBytes,
		Action:   req.Action,
		DocDate:  req.DocDate,
		DocNum:   req.DocNum,
		Status:   req.Status,
	}).Error
	if err != nil {
		return fmt.Errorf("failed to create Request1C: %w", err)
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
