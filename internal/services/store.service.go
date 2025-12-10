package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// create store on importing products to branch
func (s *Services) CreateStoreOnImport(ctx context.Context, tx *gorm.DB, req *domain.StoreRequest) (domain.Store, error) {
	var res domain.Store
	query := `INSERT INTO stores(name, detailed_name, store_code, company_id) VALUES(?, ?, ?, ?) RETURNING *`
	err := tx.WithContext(ctx).Debug().Raw(query, req.Name, req.Name, req.StoreCode, req.CompanyId).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not create new store on importing: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

// get store info by import id
func (s *Services) GetStoreByImportId(ctx context.Context, tx *gorm.DB, importId string) (*domain.Store, error) {
	var store domain.Store
	err := tx.WithContext(ctx).Raw(`SELECT stores.* FROM imports JOIN stores ON stores.id = imports.store_id WHERE imports.id = ?`, importId).Scan(&store).Error
	if err != nil {
		s.log.Errorf("could not get store by import id: %v", err)
		return nil, domain.InternalServerError
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
		return nil, domain.NotFoundError
	}
	// handle error
	if err != nil {
		s.log.Errorf("could not get store info by field: %v", err)
		return nil, domain.InternalServerError
	}
	return &store, nil
}
