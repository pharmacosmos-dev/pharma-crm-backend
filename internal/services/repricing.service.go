package services

import (
	"fmt"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
)

// create price_revalutions
func (s *Services) CreateRepricing(req *domain.RepricingRequest) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := s.db.Raw(`INSERT INTO price_revalutions(store_id, name, type, created_by) VALUES(?, ?, ?, ?) RETURNING *`,
		req.StoreId, req.Name, req.Type, req.CreatedBy).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on creating price_revalution: %v", err)
		return &res, err
	}
	return &res, nil
}

// get repricing by id
func (s *Services) GetRepricingByID(repricingID string) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := s.db.Model(&domain.PriceRevalution{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			price_revalutions.*
			`).
		First(&res, "imports.id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// repricing get list
func (s *Services) RepricingList(param *domain.QueryParam) ([]domain.PriceRevalution, int64, error) {
	var (
		res        []domain.PriceRevalution
		totalCount int64
	)

	query := s.db.Model(&domain.PriceRevalution{}).
		Preload("Store").Preload("CreatedBy").Preload("UpdatedBy").
		Select(`price_revalutions.*, SUM(prd.scanned_count) AS count
		`).Joins("LEFT JOIN price_revalution_details prd ON price_revalutions.id = prd.price_revalution_id").Group("price_revalutions.id")

	if param.StoreID != "" {
		query = query.Where("price_revalutions.store_id = ?", param.StoreID)
	}
	if param.Search != "" {
		query = query.
			Joins("JOIN stores s ON s.id = price_revalutions.store_id").
			Where("CAST(price_revalutions.id AS TEXT) like ? OR s.name ilike ?", "%"+param.Search+"%", "%"+param.Search+"%")
	}
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("price_revalutions.created_at::date BETWEEN ? AND ?", param.StartDate, param.EndDate)
	}

	if param.Status != "" {
		query = query.Where("price_revalutions.status = ?", param.Status)
	}

	err := query.
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Debug().
		Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting price revalution: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}

// repricing get detail list
func (s *Services) RepricingDetailList(repricingID int, param *domain.QueryParam) ([]domain.PriceRevalutionDetail, int64, error) {
	var (
		res        []domain.PriceRevalutionDetail
		reprice    domain.PriceRevalution
		query      = ""
		totalCount int64
		search     = ""
	)
	err := s.db.First(&reprice, "id = ? ", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting proce revalution: %v", err)
		return res, 0, err
	}

	// filter products by search key
	if param.Search != "" {
		search = fmt.Sprintf(" AND (p.name ILIKE %s OR p.barcode LIKE %s) ", "%"+param.Search+"%", "%"+param.Search+"%")
	}

	// get store_products info if price_revalution status would be new otherwise we get price_revalution details
	if reprice.Status == "new" {
		query = `
		SELECT
			gen_random_uuid() AS id, 
			?::int AS price_revalution_id,
			sp.id AS store_product_id,
			sp.product_id,
			sp.supply_price AS old_supply_price,
			sp.retail_price AS old_retail_price,
			sp.expire_date AS old_expire_date,
			sp.serial_number,
			ROUND(((sp.retail_price - sp.supply_price)/sp.supply_price)*100, 0) AS old_markup,
			p.name, p.barcode,
			COUNT(*) OVER() AS total_count
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		WHERE sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0)
		`
		// collect query
		query += search + " LIMIT ? OFFSET ?;" // add search condition and limit, offset
		// execute query
		err = s.db.Raw(query, repricingID, reprice.StoreID, param.Limit, param.Offset).Scan(&res).Error
		if err != nil {
			s.log.Warn("ERROR on getting price revalution details: %v", err)
			return res, 0, err
		}
	} else {
		query = `
		SELECT 
			prd.id, prd.store_product_id,
			prd.product_id,
			prd.price_revalution_id,
			prd.old_supply_price, 
			prd.new_supply_price,
			prd.old_retail_price, 
			prd.new_retail_price,
			prd.old_expire_date, 
			prd.new_expire_date,
			prd.serial_number,
			ROUND(((prd.old_retail_price - prd.old_supply_price)/prd.old_supply_price)*100, 0) AS old_markup,
			ROUND(((prd.new_retail_price - prd.old_supply_price)/prd.old_supply_price)*100, 0) AS new_markup,
			p.name, p.barcode,
			COUNT(*) OVER() AS total_count
		FROM price_revalution_details prd
		JOIN products p ON prd.product_id = p.id
		WHERE prd.price_revalution_id = ?
		`
		// collect query
		query += search + " LIMIT ? OFFSET ?;" // add search condition and limit, offset
		// execute query
		err = s.db.Raw(query, repricingID, param.Limit, param.Offset).Scan(&res).Error
		if err != nil {
			s.log.Warn("ERROR on getting price revalution details: %v", err)
			return res, 0, err
		}
	}

	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// confirm repricing
func (s *Services) ConfirmRepricing(repricingID string, updatedBy string) error {
	var (
		res     domain.PriceRevalution
		details []domain.PriceRevalutionDetail
	)

	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.First(&res, "id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting price_revalution: %v", err)
		tx.Rollback()
		return err
	}

	err = tx.Exec(`UPDATE price_revalutions SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`, config.COMPLETED, updatedBy, repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on updating price_revalution status: %v", err)
		tx.Rollback()
		return err
	}

	err = tx.Find(&details, "price_revalution_id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting price_revalution_detail list: %v", err)
		tx.Rollback()
		return err
	}

	for _, v := range details {
		err = tx.Exec(`
		UPDATE store_products SET retail_price = ? WHERE id = ?
		`, v.NewRetailPrice, v.StoreProductID).Error
		if err != nil {
			s.log.Warn("ERROR on updating store_product price: %v", err)
			tx.Rollback()
			return err
		}
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

// cancel repricing
func (s *Services) CancelRepricing(repricingID string, updatedBy string) error {
	err := s.db.Exec(`UPDATE price_revalutions SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`,
		config.CANCELED, updatedBy, repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on canceling repricing: %v", err)
		return err
	}
	return nil
}
