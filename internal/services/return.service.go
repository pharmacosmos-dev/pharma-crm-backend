package services

import (
	"fmt"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Create inventory creates a new inventory
func (s *Services) CreateReturn(req *domain.ReturnRequest) error {
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
	INSERT INTO transfers (public_id, from_store_id, name,  created_by, entry_type)
	VALUES (?, ?, ?, ?, ?) RETURNING id`,
		req.PublicId, req.StoreId, req.Name, req.CreatedBy, 2,
	).Scan(&id).Error
	if err != nil {
		s.log.Error("ERROR on creating return: ", err)
		tx.Rollback()
		return err
	}

	// if no products provided, get all products from store_products
	// and insert them into inventory_details
	err = tx.Exec(
		`INSERT INTO transfer_details(transfer_id, product_id, received_count, supply_price, retail_price, expire_date, serial_number
			) SELECT ?, product_id, pack_quantity, supply_price, retail_price, expire_date, serial_number
			FROM store_products
			WHERE store_id = ? AND pack_quantity > 0;`,
		id, req.StoreId).Error
	if err != nil {
		s.log.Error("ERROR on creating inventory details: ", err)
		tx.Rollback()
		return err
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error("ERROR on committing transaction: ", err)
		tx.Rollback()
		return err
	}
	return nil
}

// get return by id
func (s *Services) GetReturnById(returnId string) (*domain.Return, error) {
	var res domain.Return
	err := s.db.Model(&domain.Transfer{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			transfers.*,
			SUM(td.accepted_count) AS return_count,
			SUM(td.received_count*td.retail_price) AS received_retail_sum,
			SUM(td.accepted_count*td.retail_price) AS accepted_retail_sum`).
		Joins("LEFT JOIN transfer_details td ON transfers.id = td.transfer_id").
		Group("transfers.id").
		First(&res, "transfers.id = ?", returnId).Error
	if err != nil {
		s.log.Warn("ERROR on getting return by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// get inventory list
func (s *Services) ReturnList(param *domain.ReturnParam) ([]domain.Return, int64, error) {
	var res []domain.Return
	var totalCount int64
	query := s.db.Model(&domain.Transfer{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Preload("AcceptedBy").
		Select(`
			transfers.*, 
			SUM(trd.scanned_count) AS return_count,
			SUM(trd.received_count-trd.scanned_count) AS shortage,
			SUM(CASE WHEN trd.accepted_count > trd.received_count THEN trd.accepted_count - trd.received_count ELSE 0 END) AS surplus,
			SUM(trd.scanned_count*trd.supply_price) AS received_supply_sum,
			SUM(trd.scanned_count*trd.retail_price) AS received_retail_sum,
			SUM(trd.accepted_count*trd.supply_price) AS accepted_supply_sum,
			SUM(trd.accepted_count*trd.retail_price) AS accepted_retail_sum
			`).
		Joins("LEFT JOIN transfer_details trd ON transfers.id = trd.transfer_id").
		Where("transfers.entry_type = ?", 2)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("transfers.from_store_id = ? ", param.StoreId)
	}

	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("transfers.public_id LIKE ? OR transfers.name ILIKE ?", param.Search, param.Search)
	}

	// filter by inventory status
	if param.Status != "" {
		query = query.Where("transfers.status = ?", param.Status)
	}
	// complete query
	err := query.
		Group("transfers.id").
		Order("transfers.created_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Debug().
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}
	return res, totalCount, nil
}

// get inventory detail list
func (s *Services) ReturnDetailList(param *domain.ReturnDetailParam) ([]domain.ReturnDetail, int64, error) {
	var res []domain.ReturnDetail
	var totalCount int64
	query := s.db.
		Model(&domain.TransferDetail{}).
		Select(`
		transfer_details.*,
		transfer_details.received_count*transfer_details.retail_price AS received_sum,
		transfer_details.scanned_count*transfer_details.retail_price AS scanned_sum,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON transfer_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("transfer_details.transfer_id = ?", param.ReturnId)

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
			query = query.Where("transfer_details.received_count > transfer_details.scanned_count")
		case "scanned":
			query = query.Where("transfer_details.scanned_count > 0")
		case "surplus":
			query = query.Where("transfer_details.scanned_count > transfer_details.received_count")

		}
	}

	err := query.
		Order("transfer_details.updated_at DESC").
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

// get inventory detail status count
func (s *Services) ReturnDetailStatsCount(param *domain.ReturnDetailParam) (domain.ReturnDetailStatus, error) {
	var res domain.ReturnDetailStatus

	query := `
	SELECT
		SUM(scanned_count) AS scanned,
		SUM(received_count - scanned_count) AS shortage,
		SUM(received_count) AS "all",
		SUM(CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END) AS surplus,
		SUM(accepted_count) AS accepted,
		SUM((received_count - scanned_count)*transfer_details.supply_price) AS shortage_supply_sum,
		SUM((received_count - scanned_count)*transfer_details.retail_price) AS shortage_retail_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*transfer_details.supply_price) AS surplus_supply_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*transfer_details.retail_price) AS surplus_retail_sum
	FROM transfer_details
	JOIN products p ON transfer_details.product_id = p.id
	WHERE transfer_id = ?;
	`
	err := s.db.Raw(query, param.ReturnId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}

	return res, nil
}

// confirm inventory
func (s *Services) SendReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`
	err := tx.Exec(query, config.SENT, userId, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}

	query2 := `DELETE FROM transfer_details WHERE scanned_count = 0 AND transfer_id = ?;`
	err = tx.Exec(query2, returnId).Error
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

// confirm inventory
func (s *Services) ConfirmReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE transfers SET status = ?, accepted_by = ?, accepted_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.COMPLETED, userId, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	// update confirm inventory details
	query1 := `UPDATE transfer_details SET accepted_count = scanned_count, updated_at = NOW() WHERE transfer_id = ?`
	err = tx.Exec(query1, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
		tx.Rollback()
		return err
	}
	// update store products
	storeproductQuery := `
	UPDATE store_products sp
	SET pack_quantity = pack_quantity - td.accepted_count
	FROM transfer_details td
	JOIN transfers t ON td.transfer_id = t.id
	WHERE sp.product_id = td.product_id
	AND sp.store_id = t.from_store_id
	AND sp.expire_date = td.expire_date
	AND t.status = 'completed'
	AND td.transfer_id = ?;`
	err = tx.Exec(storeproductQuery, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating store_products: %v", err)
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

// canceled inventory
func (s *Services) CancelReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE transfers SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.CANCELED, userId, returnId).Error
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
