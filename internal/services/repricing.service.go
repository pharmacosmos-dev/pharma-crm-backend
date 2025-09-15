package services

import (
	"fmt"
	"strings"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
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
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// if no products provided, get all products from store_products
	// and insert them into inventory_details
	err = tx.Exec(
		`INSERT INTO price_revalution_details(
			price_revalution_id,
			store_product_id,
			product_id,
			old_supply_price,
			old_retail_price,
			old_expire_date,
			serial_number)
		SELECT  
			?, 
			sp.id,
			sp.product_id, 
			sp.supply_price, 
			sp.retail_price, 
			sp.expire_date, 
			sp.serial_number
		FROM store_products sp
		JOIN
			products p ON sp.product_id = p.id
		WHERE 
			sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0);`,
		res.Id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory details: %v", err)
		tx.Rollback()
		return &res, err
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on creating repricing details: %v", err)
		tx.Rollback()
		return &res, err
	}

	return &res, nil
}

// create price_revalution by 1C
func (s *Services) CreateRepricingBy1C(tx *gorm.DB, req *domain.RepricingRequest) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := tx.Raw(`INSERT INTO price_revalutions(store_id, name, type, created_by, status) VALUES(?, ?, ?, ?, ?) RETURNING *`,
		req.StoreId, req.Name, req.Type, req.CreatedBy, req.Status).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on creating price_revalution: %v", err)
		return &res, err
	}

	return &res, nil
}

// create price_revalution detail
func (s *Services) CreatePriceRevalutionDetail(tx *gorm.DB, req []domain.PriceRevalutionDetailRequest) error {
	query := `
	INSERT INTO price_revalution_details(
		price_revalution_id,
		store_product_id,
		product_id,
		old_supply_price,
		old_retail_price,
		new_retail_price,
		old_expire_date, 
		serial_number
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (store_product_id, price_revalution_id) 
	DO UPDATE SET
		new_retail_price = EXCLUDED.new_retail_price
	`
	for _, v := range req {
		err := tx.Exec(query, v.PriceRevalutionId, v.StoreProductID, v.ProductID, v.OldSupplyPrice, v.OldRetailPrice, v.NewRetailPrice, v.OldExpireDate, v.SerialNumber).Error
		if err != nil {
			s.log.Warn("ERROR on updating price_revalution_details: %v", err)
			tx.Rollback()
			return err
		}
	}

	return nil
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
		First(&res, "price_revalutions.id = ?", repricingID).Error
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
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`price_revalutions.*, 
						COUNT(prd.store_product_id) AS count,
						SUM(prd.old_retail_price) AS total_old_retail_price,
						SUM(prd.new_retail_price) AS total_new_retail_price
		`).Joins("LEFT JOIN price_revalution_details prd ON price_revalutions.id = prd.price_revalution_id").
		Group("price_revalutions.id")

	if param.StoreID != "" {
		query = query.Where("price_revalutions.store_id = ?", param.StoreID)
	}
	if param.CompanyId != "" {
		query = query.Where("st.company_id = ?", param.CompanyId).Joins("LEFT JOIN stores st ON price_revalutions.store_id = st.id")
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
		Order("price_revalutions.created_at DESC").
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting price revalution: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}

func (s *Services) RepricingStatus(param *domain.QueryParam) (*domain.RepricingStatusSummary, error) {
	query := `
		SELECT
			COALESCE(COUNT(prd.store_product_id), 0) AS count,
			COALESCE(SUM(prd.old_retail_price), 0) AS total_old_retail_price,
			COALESCE(SUM(prd.new_retail_price), 0) AS total_new_retail_price
		FROM price_revalutions
		LEFT JOIN price_revalution_details prd ON price_revalutions.id = prd.price_revalution_id
		LEFT JOIN stores st ON price_revalutions.store_id = st.id
	`

	var conditions []string
	var args []any

	if param.StoreID != "" {
		conditions = append(conditions, "price_revalutions.store_id = ?")
		args = append(args, param.StoreID)
	}
	if param.Search != "" {
		conditions = append(conditions, `(CAST(price_revalutions.id AS TEXT) ILIKE ? OR EXISTS (
			SELECT 1 FROM stores s WHERE s.id = price_revalutions.store_id AND s.name ILIKE ?
		))`)
		search := "%" + param.Search + "%"
		args = append(args, search, search)
	}
	if param.CompanyId != "" {
		conditions = append(conditions, "st.company_id = ?")
		args = append(args, param.CompanyId)
	}
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	if param.StartDate != "" && param.EndDate != "" {
		conditions = append(conditions, "price_revalutions.created_at::date BETWEEN ? AND ?")
		args = append(args, param.StartDate, param.EndDate)
	}
	if param.Status != "" {
		conditions = append(conditions, "price_revalutions.status = ?")
		args = append(args, param.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var res domain.RepricingStatusSummary
	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Error("Failed to get repricing summary: %v", err)
		return nil, err
	}

	return &res, nil
}

// repricing get detail list
func (s *Services) RepricingDetailList(repricingID int, param *domain.QueryParam) ([]domain.PriceRevalutionDetail, int64, error) {
	var (
		res        []domain.PriceRevalutionDetail
		query      = ""
		totalCount int64
		search     = ""
	)

	// filter products by search key
	if param.Search != "" {
		search = fmt.Sprintf(" AND (p.name ILIKE %s OR p.barcode LIKE %s) ", "%"+param.Search+"%", "%"+param.Search+"%")
	}

	query = `
	SELECT 
		prd.id, 
		prd.store_product_id,
		prd.product_id,
		prd.price_revalution_id,
		prd.old_supply_price, 
		prd.new_supply_price,
		prd.old_retail_price, 
		prd.new_retail_price,
		prd.old_expire_date, 
		prd.new_expire_date,
		prd.serial_number,
		ROUND(
          CASE
            WHEN prd.old_supply_price = 0 THEN 0
            ELSE ((prd.old_retail_price - prd.old_supply_price) / prd.old_supply_price) * 100
          END, 0
        ) AS old_markup,
        ROUND(
          CASE
            WHEN prd.old_supply_price = 0 THEN 0
            ELSE ((prd.new_retail_price - prd.old_supply_price) / prd.old_supply_price) * 100
          END, 0
        ) AS new_markup,
		p.name, p.barcode,
		COUNT(*) OVER() AS total_count
	FROM price_revalution_details prd
	JOIN products p ON prd.product_id = p.id
	WHERE prd.price_revalution_id = ?
	`
	// collect query
	query += search + " ORDER BY prd.updated_at DESC LIMIT ? OFFSET ?;" // add search condition and limit, offset
	// execute query
	err := s.db.Raw(query, repricingID, param.Limit, param.Offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting price revalution details: %v", err)
		return res, 0, err
	}

	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

func (s *Services) RepricingDetailStatus(repricingID int, param *domain.QueryParam) (*domain.RepricingDetailStatusSummary, error) {
	query := `
		SELECT
			COALESCE(COUNT(prd.id), 0) AS count,
			COALESCE(SUM(prd.old_retail_price), 0) AS total_old_retail_price,
			COALESCE(SUM(prd.new_retail_price), 0) AS total_new_retail_price,
			COALESCE(SUM(prd.old_supply_price), 0) AS total_old_supply_price,
-- 			COALESCE(SUM(prd.new_supply_price), 0) AS total_new_supply_price,
			ROUND(AVG(
				CASE WHEN prd.old_supply_price = 0 THEN 0
					 ELSE ((prd.old_retail_price - prd.old_supply_price) / prd.old_supply_price) * 100
				END
			), 2) AS avg_old_markup,
			ROUND(AVG(
				CASE WHEN prd.old_supply_price = 0 THEN 0
					 ELSE ((prd.new_retail_price - prd.old_supply_price) / prd.old_supply_price) * 100
				END
			), 2) AS avg_new_markup
		FROM price_revalution_details prd
		JOIN products p ON p.id = prd.product_id
		WHERE prd.price_revalution_id = ? and new_retail_price > 0
	`

	var args []any
	args = append(args, repricingID)

	if param.Search != "" {
		query += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
		search := "%" + param.Search + "%"
		args = append(args, search, search)
	}

	var res domain.RepricingDetailStatusSummary
	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Error("Failed to get repricing detail status: %v", err)
		return nil, err
	}

	return &res, nil
}

// confirm repricing
func (s *Services) ConfirmRepricing(repricingID int, updatedBy string) error {
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

	err = tx.Exec(`DELETE FROM price_revalution_details WHERE new_retail_price = 0 and price_revalution_id = ?`, repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on deleting price_revalution_details if price will be zero: %v", err)
		tx.Rollback()
		return err
	}

	err = tx.Find(&details, "price_revalution_id = ? AND new_retail_price > 0", repricingID).Error
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
		s.log.Warn("ERROR on commiting transaction: %v", err)
		tx.Rollback()
		return err
	}

	return nil
}

// cancel repricing
func (s *Services) CancelRepricing(repricingID string, updatedBy string) error {
	// check if repricing exists
	err := s.db.Exec(`UPDATE price_revalutions SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`,
		config.CANCELED, updatedBy, repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on canceling repricing: %v", err)
		return err
	}
	// get all details for the repricing
	var details []domain.PriceRevalutionDetail
	err = s.db.Find(&details, "price_revalution_id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting price_revalution_detail list: %v", err)
		return err
	}
	// update store_products retail_price with old retail price
	for _, v := range details {
		err = s.db.Exec(`UPDATE store_products SET retail_price = ? WHERE id = ?`,
			v.OldRetailPrice, v.StoreProductID).Error
		if err != nil {
			s.log.Warn("ERROR on updating store_product price: %v", err)
			return err
		}
	}

	return nil
}
