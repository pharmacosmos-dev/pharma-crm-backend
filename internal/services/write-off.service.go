package services

import (
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// create new write-off
func (s *Services) CreateWriteOff(req *domain.WriteOffRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// insert inventory into inventories table
	err := tx.Raw(`
	INSERT INTO imports (store_id, name, created_by, entry_type, import_date, comment)
	VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		req.StoreId, req.Name, req.CreatedBy, 3, time.Now(), req.Comment,
	).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory: %v", err)
		tx.Rollback()
		return err
	}
	// if no products provided, get all products from store_products
	// and insert them into write-off
	err = tx.Exec(
		`INSERT INTO import_details(import_id, product_id, received_count, supply_price_vat, retail_price_vat, expire_date, series_number
			) SELECT ?, product_id, pack_quantity, supply_price, retail_price, expire_date, serial_number
			FROM store_products
			WHERE store_id = ? AND pack_quantity > 0;`,
		id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory details: %v", err)
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		tx.Rollback()
		return err
	}

	return nil
}

// get write-off list
func (s *Services) WriteOffList(param *domain.WriteOffParam) ([]domain.WriteOff, int64, error) {
	var res []domain.WriteOff
	var totalCount int64
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			SUM(imd.accepted_count) AS writeoff_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").
		Where("imports.entry_type = ?", 3)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("imports.store_id = ? ", param.StoreId)
	}
	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("CAST(public_id AS TEXT) LIKE ? OR imports.name ILIKE ?", param.Search, param.Search)
	}

	// filter by inventory status
	if param.Status != "" {
		query = query.Where("imports.status = ?", param.Status)
	}
	// complete query
	err := query.
		Group("imports.id").
		Order("imports.created_at DESC").
		Count(&totalCount).
		Limit(param.Limit).Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}
	return res, totalCount, nil
}

// get write-off by id
func (s *Services) GetWriteOffById(writeOffId string) (*domain.WriteOff, error) {
	var res domain.WriteOff
	err := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			SUM(imd.accepted_count) AS writeoff_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").Group("imports.id").First(&res, "imports.id = ?", writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// write-off confirm
func (s *Services) ConfirmWriteOff(writeOffId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.COMPLETED, userId, writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	// update confirm inventory details
	query1 := `UPDATE import_details SET accepted_count = scanned_count, updated_at = NOW() WHERE import_id = ?`
	err = tx.Exec(query1, writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
		tx.Rollback()
		return err
	}
	query2 := `DELETE FROM import_details WHERE scanned_count = 0 AND import_id = ?;`
	err = tx.Exec(query2, writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on deleting scanned 0 inventory details: %v", err)
		tx.Rollback()
		return err
	}
	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		tx.Rollback()
		return err
	}
	return nil
}

// canceled write-off
func (s *Services) CancelWriteOff(writeOffId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.CANCELED, userId, writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction %v", err)
		tx.Rollback()
		return err
	}

	return nil
}

// get write-off detail list
func (s *Services) WriteOffDetailList(param *domain.WriteOffDetailParam) ([]domain.WriteOffDetail, int64, error) {
	var res []domain.WriteOffDetail
	var totalCount int64
	query := s.db.
		Model(&domain.ImportDetail{}).
		Select(`
		import_details.*,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON import_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("import_details.import_id = ?", param.WriteOffId)

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			query = query.Where("p.barcode = ?", param.Search)
		case "name/category":
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ?", param.Search)
		default:
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ? OR p.barcode LIKE ?", param.Search, param.Search)
		}
	}
	// filter with inventory stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			query = query.Where("import_details.received_count > import_details.scanned_count")
		case "scanned":
			query = query.Where("import_details.scanned_count > 0")
		case "surplus":
			query = query.Where("import_details.scanned_count > import_details.received_count")

		}
	}

	err := query.
		Order("import_details.updated_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}

	return res, totalCount, nil
}
