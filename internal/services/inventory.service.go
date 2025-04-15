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
		s.log.Error("ERROR on creating inventory: ", err)
		tx.Rollback()
		return err
	}
	if len(req.Products) > 0 {
		for _, product := range req.Products {
			err = tx.Exec(`
			INSERT INTO import_details (
				import_id, product_id, received_count, supply_price_vat, retail_price_vat
			) SELECT ?, product_id, SUM(pack_quantity), MIN(supply_price), MIN(retail_price)
			FROM store_products
			WHERE product_id = ? GROUP BY product_id;
			`, id, product.ProductId).Error
			if err != nil {
				s.log.Error("ERROR on creating inventory: ", err)
				tx.Rollback()
				return err
			}
		}
	} else {
		// if no products provided, get all products from store_products
		// and insert them into inventory_details
		err = tx.Exec(
			`INSERT INTO import_details(import_id, product_id, received_count, supply_price_vat, retail_price_vat
			) SELECT ?, product_id, SUM(pack_quantity), MIN(supply_price), MIN(retail_price)
			FROM store_products
			WHERE store_id = ? AND pack_quantity > 0 GROUP BY product_id;`,
			id, req.StoreId).Error
		if err != nil {
			s.log.Error("ERROR on creating inventory details: ", err)
			tx.Rollback()
			return err
		}
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error("ERROR on committing transaction: ", err)
		tx.Rollback()
		return err
	}
	return nil
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
		Model(&domain.ImportDetail{}).
		Select(`
		import_details.*,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON import_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("import_details.import_id = ?", param.InventoryId)

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
func (s *Services) ConfirmInventory(inventoryId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.COMPLETED, userId, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	// update confirm inventory details
	query1 := `UPDATE import_details SET accepted_count = scanned_count, updated_at = NOW() WHERE import_id = ?`
	err = tx.Exec(query1, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
		tx.Rollback()
		return err
	}
	query2 := `DELETE FROM import_details WHERE scanned_count = 0 AND import_id = ?;`
	err = tx.Exec(query2, inventoryId).Error
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
