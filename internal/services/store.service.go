package services

import (
	"errors"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// create store on importing products to branch
func (s *Services) CreateStoreOnImport(req *domain.StoreRequest) (domain.Store, error) {
	var res domain.Store
	query := `INSERT INTO stores(name, detailed_name, store_code, company_id) VALUES(?, ?, ?, ?) RETURNING *`
	err := s.db.Raw(query, req.Name, req.Name, req.StoreCode, req.CompanyId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on creating new store: %v", err)
		return res, err
	}

	return res, nil
}

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
	query := fmt.Sprintf("SELECT * FROM stores WHERE %s = ?", field)
	err := s.db.Raw(query, value).Scan(&store).Error
	// check if store is found
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("store.not.found")
	}
	// handle error
	if err != nil {
		s.log.Error("Error on getting store info: %w", err)
		return nil, errors.New("internal.server.error")
	}
	return &store, nil
}
