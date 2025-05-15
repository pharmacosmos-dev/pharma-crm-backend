package services

import (
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Create inventory creates a new inventory
func (s *Services) CreateInventory(req *domain.InventoryRequest) error {
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
	INSERT INTO imports (store_id, name, inventory_type, created_by, entry_type, import_date)
	VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		req.StoreId, req.Name, req.Type, req.CreatedBy, 2, time.Now(),
	).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory: %v", err)
		tx.Rollback()
		return err
	}
	// insert all products (including those not in store_products)
	err = tx.Exec(`
		INSERT INTO import_details (
			import_id, product_id, received_count, received_unit_count, supply_price_vat, retail_price_vat, expire_date, series_number, store_product_id
		)
		SELECT 
			?,
			p.id, 
			COALESCE(sp.pack_quantity, 0),
			COALESCE(sp.unit_quantity%p.unit_per_pack, 0),
			COALESCE(sp.supply_price, 0),
			COALESCE(sp.retail_price, 0),
			COALESCE(sp.expire_date, NULL),
			COALESCE(sp.serial_number, ''),
			COALESCE(sp.id, NULL)
		FROM
			products p
		LEFT JOIN
			store_products sp ON sp.product_id = p.id AND sp.store_id = ?
		`, id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory detail: %v", err)
		tx.Rollback()
		return err
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on committing transaction: %v", err)
		tx.Rollback()
		return err
	}
	return nil
}

// get inventory by id
func (s *Services) GetInventoryById(inventoryID string) (*domain.Inventory, error) {
	var res domain.Inventory
	err := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			SUM(imd.accepted_count) AS measurement_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum, 
			SUM((received_count - scanned_count)*imd.supply_price_vat) AS shortage_supply_sum,
			SUM((received_count - scanned_count)*imd.retail_price_vat) AS shortage_retail_sum,
			SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*imd.supply_price_vat) AS surplus_supply_sum,
			SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*imd.retail_price_vat) AS surplus_retail_sum
			`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").
		Group("imports.id").
		First(&res, "imports.id = ?", inventoryID).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// get inventory list
func (s *Services) InventoryList(param *domain.InventoryParam) ([]domain.Inventory, int64, error) {
	var res []domain.Inventory
	var totalCount int64
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*, 
			SUM(imd.received_count) AS measurement_count,
			SUM(imd.received_count-imd.scanned_count) AS shortage,
			SUM(CASE WHEN imd.accepted_count > imd.received_count THEN imd.accepted_count - imd.received_count ELSE 0 END) AS surplus,
			SUM(imd.scanned_count*imd.retail_price_vat) - SUM(imd.received_count*imd.retail_price_vat) AS difference_sum`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").
		Where("imports.entry_type = ?", 2)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("imports.store_id = ? ", param.StoreId)
	}
	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("imports.public_id LIKE ? OR imports.name ILIKE ?", param.Search, param.Search)
	}
	// filter by inventory type
	if param.Type != "" {
		query = query.Where("imports.inventory_type = ?", param.Type)
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

// get inventory detail list
func (s *Services) InventoryDetailList(param *domain.InventoryDetailParam) ([]domain.InventoryDetail, int64, error) {
	var res []domain.InventoryDetail
	var totalCount int64
	query := s.db.
		Table("import_details imd").
		Select(`
		imd.id, imd.import_id AS inventory_id,
		imd.product_id,
		ROUND(imd.received_count + (imd.received_unit_count/p.unit_per_pack), 4) AS received_count,
		imd.scanned_count,
		imd.scanned_count - imd.received_count AS difference_count,
		imd.supply_price_vat, imd.retail_price_vat,
		imd.expire_date, imd.series_number, imd.created_at, imd.updated_at,
		(imd.received_count*imd.retail_price_vat) AS stock_sum,
		(imd.scanned_count*imd.retail_price_vat) AS scanned_sum,
		((imd.scanned_count - imd.received_count)*imd.retail_price_vat) AS difference_sum,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON imd.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("imd.import_id = ?", param.InventoryId)

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
			query = query.Where("imd.received_count > imd.scanned_count")
		case "scanned":
			query = query.Where("imd.scanned_count > 0")
		case "surplus":
			query = query.Where("imd.scanned_count > imd.received_count")
		}
	}

	err := query.
		Order("imd.updated_at DESC").
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
func (s *Services) InventoryDetailStatsCount(param *domain.InventoryDetailParam) (domain.InventoryDetailStatus, error) {
	var res domain.InventoryDetailStatus

	query := `
	SELECT
		SUM(scanned_count) AS scanned,
		SUM(received_count - scanned_count) AS shortage,
		SUM(received_count) AS "all",
		SUM(CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END) AS surplus,
		SUM(accepted_count) AS accepted,
		SUM((received_count - scanned_count)*import_details.supply_price_vat) AS shortage_supply_sum,
		SUM((received_count - scanned_count)*import_details.retail_price_vat) AS shortage_retail_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*import_details.supply_price_vat) AS surplus_supply_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*import_details.retail_price_vat) AS surplus_retail_sum
	FROM import_details
	JOIN products p ON import_details.product_id = p.id
	WHERE import_id = ?;
	`
	err := s.db.Raw(query, param.InventoryId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}

	return res, nil
}

// confirm inventory
func (s *Services) ConfirmInventory(inventoryId string, userId string) (*domain.Inventory, error) {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var res domain.Inventory
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? RETURNING *`
	err := tx.Raw(query, config.COMPLETED, userId, inventoryId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return &res, err
	}

	// update confirm inventory details
	query1 := `UPDATE import_details SET accepted_count = scanned_count, updated_at = NOW() WHERE import_id = ?`
	err = tx.Exec(query1, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
		tx.Rollback()
		return &res, err
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		tx.Rollback()
		return &res, err
	}
	return &res, nil
}

// attach inventory products to store_products
func (s *Services) AttachInventoryToStoreProduct(req *domain.Inventory) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := `
	WITH UpdatedProducts AS (
		SELECT 
			sp.id AS sp_id,
			id.scanned_count::INT AS pack_quantity,
			FLOOR(id.scanned_count::INT) * p.unit_per_pack + 
			((id.scanned_count - FLOOR(id.scanned_count)) * p.unit_per_pack) AS unit_quantity,
			id.supply_price_vat,
			id.retail_price_vat,
			id.expire_date,
			ROW_NUMBER() OVER (PARTITION BY sp.product_id ORDER BY sp.id) AS row_num
		FROM 
			store_products sp
		JOIN import_details id ON sp.product_id = id.product_id
		JOIN products p ON id.product_id = p.id
		WHERE 
			id.scanned_count > 0 
			AND id.import_id = ? 
			AND sp.store_id = ?
	)
	UPDATE store_products sp
	SET 
		pack_quantity = up.pack_quantity,
		unit_quantity = up.unit_quantity,
		supply_price = up.supply_price_vat,
		retail_price = up.retail_price_vat,
		expire_date = up.expire_date
	FROM UpdatedProducts up
	WHERE 
		sp.id = up.sp_id
		AND up.row_num = 1;
	`
	err := s.db.Exec(query, req.Id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on updating store_products: %v", err)
		tx.Rollback()
		return err
	}

	// query1 := `
	// INSERT INTO store_products (
	// 		store_id, product_id, pack_quantity, unit_quantity, supply_price, retail_price, expire_date, serial_number)
	// 	SELECT
	// 		?, id.product_id, FLOOR(id.scanned_count),  
	// 		FLOOR(id.scanned_count)*p.unit_per_pack + (id.scanned_count - FLOOR(id.scanned_count))*p.unit_per_pack, 
	// 		id.supply_price_vat,id.retail_price_vat, id.expire_date, id.series_number
	// 	FROM import_details id
	// 	JOIN products p ON id.product_id = p.id
	// 	LEFT JOIN store_products sp ON sp.product_id = id.product_id AND sp.store_id = ?
	// 	WHERE id.import_id = ? AND sp.id IS NULL;
	// `
	// err = s.db.Exec(query1, req.StoreId, req.StoreId, req.Id).Error
	// if err != nil {
	// 	s.log.Warn("ERROR on inserting store_products: %v", err)
	// 	tx.Rollback()
	// 	return err
	// }

	query2 := `
	UPDATE store_products sp
	SET
		pack_quantity = 0,
		unit_quantity = 0
	WHERE
		sp.store_id = ?
		AND sp.product_id IN (
			SELECT product_id
			FROM import_details
			WHERE import_id = ?
			GROUP BY product_id
			HAVING SUM(scanned_count) = 0
	);`
	err = s.db.Exec(query2, req.StoreId, req.Id).Error
	if err != nil {
		s.log.Warn("ERROR on updating store_product to 0: %v", err)
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
func (s *Services) CancelInventory(inventoryId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.CANCELED, userId, inventoryId).Error
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
