package services

import "github.com/pharma-crm-backend/domain"

// get store info by import id
func (s *Storage) GetStoreByImportId(importId string) (*domain.Store, error) {
	var store domain.Store
	err := s.db.Raw(`SELECT stores.* FROM imports JOIN stores ON stores.id = imports.store_id WHERE imports.id = ?`, importId).Scan(&store).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return &store, nil
}
