package services

import (
	"errors"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// get store info by import id
func (s *Services) GetStoreByImportId(importId string) (*domain.Store, error) {
	var store domain.Store
	err := s.db.Raw(`SELECT stores.* FROM imports JOIN stores ON stores.id = imports.store_id WHERE imports.id = ?`, importId).Scan(&store).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return &store, nil
}

// get store info by field and value
func (s *Services) GetStoreByField(field string, value string) (*domain.Store, error) {
	var store domain.Store
	query := fmt.Sprintf("SELECT * FROM WHERE %s = ?", field)
	err := s.db.Raw(query, value).Scan(&store).Error
	// check if store is found
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("store not found")
	}
	// handle error
	if err != nil {
		s.log.Error("Error on getting store info: %w", err)
		return nil, err
	}
	return &store, nil
}
