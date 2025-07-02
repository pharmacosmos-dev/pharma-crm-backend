package services

import (
	"encoding/json"
	"fmt"
	"reflect"

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

func extractDocMeta(data any) (docDate, docNum string) {
	// Get the reflect.Value of the input
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", ""
	}

	// Find the field tagged with json:"Dok"
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("json") == "Dok" {
			// Get the nested struct (Document or ExpenseDok)
			dok := v.Field(i)
			if dok.Kind() != reflect.Struct {
				return "", ""
			}

			// Iterate through fields of the nested struct
			dokType := dok.Type()
			for j := 0; j < dokType.NumField(); j++ {
				dokField := dokType.Field(j)
				jsonTag := dokField.Tag.Get("json")
				fieldValue := dok.Field(j)

				// Ensure the field is a string before accessing
				if fieldValue.Kind() != reflect.String {
					continue
				}

				// Check JSON tags for data_dok and nomer_dok
				if jsonTag == "data_dok" {
					docDate = fieldValue.String()
				} else if jsonTag == "nomer_dok" {
					docNum = fieldValue.String()
				}
			}
			return docDate, docNum
		}
	}
	return "", ""
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
