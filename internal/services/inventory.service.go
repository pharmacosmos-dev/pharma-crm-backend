package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Create inventory creates a new inventory
func (s *Services) CreateInventory(req *domain.InventoryRequest) error {
	req.PublicId = utils.GenerateCode()
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
	INSERT INTO inventories (public_id, store_id, name, type, created_by)
	VALUES (?, ?, ?, ?, ?) RETURNING id`,
		req.PublicId, req.StoreId, req.Name, req.Type, req.CreatedBy,
	).Scan(&id).Error
	if err != nil {
		s.log.Error("ERROR on creating inventory: ", err)
		tx.Rollback()
		return err
	}
	if len(req.Products) > 0 {
		for _, product := range req.Products {
			err = tx.Exec(`
			INSERT INTO inventory_details (
				inventory_id, product_id, stock_count
			) SELECT ?, product_id, SUM(pack_quantity)
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
			`INSERT INTO inventory_details(inventory_id, product_id, stock_count)
			SELECT ?, product_id, SUM(pack_quantity) 
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
	query := s.db.Model(&domain.Inventory{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			inventories.*, 
			SUM(ind.stock_count) AS measurement_count,
			SUM(ind.stock_count) AS shortage,
			SUM(CASE WHEN ind.accepted_count > ind.stock_count THEN ind.accepted_count - ind.stock_count ELSE 0 END) AS surplus,
			SUM(ind.accepted_count*sp.retail_price) - SUM(ind.stock_count*sp.retail_price) AS difference_sum`).
		Joins("LEFT JOIN inventory_details ind ON inventories.id = ind.inventory_id").
		Joins("JOIN store_products sp ON sp.product_id = ind.product_id")
	if param.StoreId != "" {
		query = query.Where("inventories.store_id = ? ", param.StoreId)
	}
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("inventories.public_id LIKE ? OR inventories.name ILIKE ?", param.Search, param.Search)
	}
	if param.Type != "" {
		query = query.Where("inventories.type = ?", param.Type)
	}
	if param.Status != "" {
		query = query.Where("inventories.status = ?", param.Status)
	}
	err := query.
		Group("inventories.id").
		Order("inventories.created_at DESC").
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
		Model(&domain.InventoryDetail{}).
		Select(`
		inventory_details.*,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON inventory_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("inventory_details.inventory_id = ?", param.InventoryId)

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
	err := query.
		Order("inventory_details.updated_at DESC").
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

// get inventory detail status count
func (s *Services) InventoryDetailStatsCount(param *domain.InventoryDetailParam) (domain.InventoryDetailStatus, error) {
	var res domain.InventoryDetailStatus

	query := `
	SELECT
		SUM(scanned_count) AS scanned,
		SUM(stock_count - scanned_count) AS shortage,
		SUM(stock_count) AS "all",
		SUM(CASE WHEN scanned_count > stock_count THEN scanned_count - stock_count ELSE 0 END) AS surplus,
		SUM(accepted_count) AS accepted,
		SUM((stock_count - scanned_count)*sp.supply_price) AS shortage_supply_sum,
		SUM((stock_count - scanned_count)*sp.retail_price) AS shortage_retail_sum,
		SUM((CASE WHEN scanned_count > stock_count THEN scanned_count - stock_count ELSE 0 END)*sp.supply_price) AS surplus_supply_sum,
		SUM((CASE WHEN scanned_count > stock_count THEN scanned_count - stock_count ELSE 0 END)*sp.retail_price) AS surplus_retail_sum
	FROM inventory_details
	JOIN products p ON inventory_details.product_id = p.id
	LEFT JOIN store_products sp ON p.id = sp.product_id
	WHERE inventory_id = ?;
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
	query := `UPDATE inventories SET status = 2, updated_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, userId, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	// update confirm inventory details
	query1 := `UPDATE inventory_details SET accepted_count = scanned_count, updated_at = NOW() WHERE inventory_id = ?`
	err = tx.Exec(query1, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
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
