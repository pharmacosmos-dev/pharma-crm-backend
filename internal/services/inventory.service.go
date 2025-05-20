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
			import_id, product_id, store_product_id,  received_count, supply_price_vat, retail_price_vat, expire_date, series_number
				)
		SELECT
			?,
			sp.product_id,
			sp.id,
			COALESCE(sp.pack_quantity::numeric + (sp.unit_quantity::numeric%p.unit_per_pack)/p.unit_per_pack, 0.00) AS quantity,
			sp.supply_price,
			sp.retail_price,
			expire_date,
			sp.serial_number
		FROM
			products p
		LEFT JOIN
			store_products sp ON sp.product_id = p.id and sp.store_id = ?
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
	var (
		res        []domain.InventoryDetail
		totalCount int64
		args       = []any{}
		filter     = " WHERE imd.import_id = ? "
		orderBy    = ""
	)
	args = append(args, param.InventoryId)
	//
	query := `
	SELECT
        imd.id,
        imd.import_id AS inventory_id,
        imd.product_id,
        p.material_code,
        p.name,
        p.unit_per_pack,
        imd.received_count AS current_quantity,
        (imd.received_count - FLOOR(imd.received_count))*p.unit_per_pack AS current_unit,
        imd.scanned_count AS fact_quantity,
        (imd.scanned_count - FLOOR(imd.scanned_count))*p.unit_per_pack AS fact_unit,
        imd.scanned_count - imd.received_count AS difference_quantity,
        ((imd.scanned_count - imd.received_count) - FLOOR(imd.scanned_count - imd.received_count))*p.unit_per_pack AS difference_unit,
        (imd.retail_price_vat*imd.received_count) AS current_sum,
        (imd.retail_price_vat*imd.scanned_count) AS fact_sum,
        imd.retail_price_vat*(imd.scanned_count - imd.received_count) AS difference_sum
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
	`
	tquery := `
	SELECT
		COUNT(*) AS total_count
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
	`

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+param.Search+"%")
		case "name/category":
			filter += " AND p.name ILIKE ?"
			args = append(args, "%"+param.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		}
	}
	// filter with inventory stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			filter += " AND imd.received_count > imd.scanned_count"
		case "scanned":
			filter += " AND imd.scanned_count > 0"
		case "surplus":
			filter += " AND imd.scanned_count > imd.received_count"
		}
	}

	// order by
	switch param.Order {
	case "+name":
		orderBy = " ORDER BY p.name ASC "
	case "-name":
		orderBy = " ORDER BY p.name DESC "
	case "+current_sum":
		orderBy = " ORDER BY current_sum ASC "
	case "-current_sum":
		orderBy = " ORDER BY current_sum DESC "
	case "+fact_sum":
		orderBy = " ORDER BY fact_sum ASC "
	case "-fact_sum":
		orderBy = " ORDER BY fact_sum DESC "
	case "+difference_sum":
		orderBy = " ORDER BY difference_sum ASC "
	case "-difference_sum":
		orderBy = " ORDER BY difference_sum DESC "
	default:
		orderBy = " ORDER BY current_quantity DESC "
	}
	// execute total count query
	tquery += filter
	// get total count
	err := s.db.Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting total count: %v", err)
		return res, 0, err
	}

	// complete query
	query += filter + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	// execute query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory detail list: %v", err)
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
	
	// clean import details if fact and received count is equal
	query = `DELETE FROM import_details WHERE import_id = ? AND scanned_count = received_count`
	err = tx.Exec(query, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on deleting import details %v", err)
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

	
	// delta < 0 holat uchun query
	query := `
	UPDATE store_products 
	SET pack_quantity =  pack_quantity - qoldiq WHERE id = ? AND store_id = ?`

	
	err := s.db.Exec(query, req.Id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on updating store_products: %v", err)
		tx.Rollback()
		return err
	}
	// delta > 0 holat uchun query
	query = `
	UPDATE store_products
	SET pack_quantity = pack_quantity + qoldiq WHERE id = ? AND store_id = ?`	


	err = tx.Exec(query, req.Id, req.StoreId).Error
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
func (s *Services) CancelInventory(inventoryId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? AND status = ?`
	err := tx.Exec(query, config.CANCELED, userId, inventoryId, config.PENDING).Error
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
