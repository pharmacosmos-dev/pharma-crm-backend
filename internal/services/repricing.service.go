package services

import (
	"context"
	"strings"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// create price_revalutions
func (s *Services) CreateRepricing(ctx context.Context, req *domain.RepricingRequest) (*domain.PriceRevalution, error) {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var res domain.PriceRevalution
	err := tx.WithContext(ctx).
		Raw(`
		INSERT INTO price_revalutions(
			store_id, 
			name, 
			type, 
			created_by
			) 
		VALUES(?, ?, ?, ?) 
		RETURNING *`,
			req.StoreId,
			req.Name,
			req.Type,
			req.CreatedBy,
		).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create price_revalution: %v", err)
		return &res, domain.InternalServerError
	}
	switch req.Type {
	case "IMPORT":
		if req.ImportId != "" {
			err = tx.WithContext(ctx).
				Exec(`
				INSERT INTO price_revalution_details(
					price_revalution_id,
					store_product_id,
					product_id,
					old_supply_price,
					old_retail_price,
					old_expire_date,
					serial_number
				)
				SELECT
					?,
					sp.id,
					sp.product_id,
					sp.supply_price,
					sp.retail_price,
					sp.expire_date,
					sp.serial_number
				FROM store_products sp
				JOIN import_details id ON id.id = sp.import_detail_id
				WHERE id.import_id = ?;`,
					res.Id, req.ImportId).Error
		}

	case "MEDICINE":
		if req.StoreProductId != "" {
			err = tx.WithContext(ctx).
				Exec(`
				INSERT INTO price_revalution_details(
					price_revalution_id,
					store_product_id,
					product_id,
					old_supply_price,
					old_retail_price,
					old_expire_date,
					serial_number
				)
				SELECT
					?,
					sp.id,
					sp.product_id,
					sp.supply_price,
					sp.retail_price,
					sp.expire_date,
					sp.serial_number
				FROM store_products sp
				WHERE sp.id = ?;`,
					res.Id, req.StoreProductId).Error
		}

	default: // FULL repricing (store_id bo‘yicha)
		err = tx.WithContext(ctx).
			Exec(`
			INSERT INTO price_revalution_details(
				price_revalution_id,
				store_product_id,
				product_id,
				old_supply_price,
				old_retail_price,
				old_expire_date,
				serial_number
			)
			SELECT  
				?, 
				sp.id,
				sp.product_id, 
				sp.supply_price, 
				sp.retail_price, 
				sp.expire_date, 
				sp.serial_number
			FROM store_products sp
			WHERE sp.store_id = ? 
			AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0);`,
				res.Id,
				req.StoreId,
			).Error
	}

	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create repricing details: %v", err)
		return &res, domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit create repricing transaction: %v", err)
		return &res, domain.InternalServerError
	}

	return &res, nil
}

// create price_revalution by 1C
func (s *Services) CreateRepricingByOnec(ctx context.Context, tx *gorm.DB, req *domain.RepricingRequest) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := tx.WithContext(ctx).Raw(`INSERT INTO price_revalutions(store_id, name, type, created_by, status) VALUES(?, ?, ?, ?, ?) RETURNING *`,
		req.StoreId, req.Name, req.Type, req.CreatedBy, req.Status).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not create price_revalution: %v", err)
		return &res, domain.InternalServerError
	}

	return &res, nil
}

// create price_revalution detail
func (s *Services) CreatePriceRevalutionDetail(ctx context.Context, tx *gorm.DB, req []domain.PriceRevalutionDetailRequest) error {
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
		err := tx.WithContext(ctx).
			Exec(query,
				v.PriceRevalutionId,
				v.StoreProductId,
				v.ProductId,
				v.OldSupplyPrice,
				v.OldRetailPrice,
				v.NewRetailPrice,
				v.OldExpireDate,
				v.SerialNumber,
			).Error
		if err != nil {
			s.log.Errorf("could not update price_revalution_details: %v", err)
			return domain.InternalServerError
		}
	}

	return nil
}

// get repricing by id
func (s *Services) GetRepricingByID(ctx context.Context, repricingId string) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := s.db.WithContext(ctx).
		Model(&domain.PriceRevalution{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			price_revalutions.*
			`).
		First(&res, "price_revalutions.id = ?", repricingId).Error
	if err != nil {
		s.log.Errorf("could not get repricing by id: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

// repricing get list
func (s *Services) GetRepricingList(ctx context.Context, params *domain.QueryParam) ([]domain.PriceRevalution, int64, error) {
	var (
		res        []domain.PriceRevalution
		totalCount int64
	)

	query := s.db.WithContext(ctx).
		Model(&domain.PriceRevalution{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`price_revalutions.*, 
						COUNT(prd.store_product_id) AS count,
						SUM(prd.old_retail_price) AS total_old_retail_price,
						SUM(prd.new_retail_price) AS total_new_retail_price
		`).Joins("LEFT JOIN price_revalution_details prd ON price_revalutions.id = prd.price_revalution_id").
		Group("price_revalutions.id")

	if params.StoreID != "" {
		query = query.Where("price_revalutions.store_id = ?", params.StoreID)
	}
	if params.CompanyId != "" {
		query = query.Where("st.company_id = ?", params.CompanyId).Joins("LEFT JOIN stores st ON price_revalutions.store_id = st.id")
	}
	if params.Search != "" {
		query = query.
			Joins("JOIN stores s ON s.id = price_revalutions.store_id").
			Where("CAST(price_revalutions.id AS TEXT) like ? OR s.name ilike ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	if params.EndDate == "" {
		params.EndDate = params.StartDate
	}
	if params.StartDate != "" && params.EndDate != "" {
		query = query.Where("price_revalutions.created_at::date BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}

	if params.Status != "" {
		query = query.Where("price_revalutions.status = ?", params.Status)
	}

	err := query.
		Count(&totalCount).
		Order("price_revalutions.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get price revalution: %v", err)
		return res, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetRepricingStats(ctx context.Context, params *domain.QueryParam) (*domain.RepricingStatusSummary, error) {
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

	if params.StoreID != "" {
		conditions = append(conditions, "price_revalutions.store_id = ?")
		args = append(args, params.StoreID)
	}
	if params.Search != "" {
		conditions = append(conditions, `(CAST(price_revalutions.id AS TEXT) ILIKE ? OR EXISTS (
			SELECT 1 FROM stores s WHERE s.id = price_revalutions.store_id AND s.name ILIKE ?
		))`)
		search := "%" + params.Search + "%"
		args = append(args, search, search)
	}
	if params.CompanyId != "" {
		conditions = append(conditions, "st.company_id = ?")
		args = append(args, params.CompanyId)
	}
	if params.EndDate == "" {
		params.EndDate = params.StartDate
	}
	if params.StartDate != "" && params.EndDate != "" {
		conditions = append(conditions, "price_revalutions.created_at::date BETWEEN ? AND ?")
		args = append(args, params.StartDate, params.EndDate)
	}
	if params.Status != "" {
		conditions = append(conditions, "price_revalutions.status = ?")
		args = append(args, params.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var res domain.RepricingStatusSummary
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get repricing summary: %v", err)
		return nil, domain.InternalServerError
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

	args := []any{repricingID}
	// filter products by search key
	if param.Search != "" {
		search = " AND (p.name ILIKE ? OR p.barcode LIKE ?) "
		searchKey := "%" + param.Search + "%"
		args = append(args, searchKey, searchKey)
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
		COALESCE(p.max_price, 0) AS max_price,
		COUNT(*) OVER() AS total_count
	FROM price_revalution_details prd
	JOIN products p ON prd.product_id = p.id
	WHERE prd.price_revalution_id = ?
	`
	args = append(args, param.Limit, param.Offset)
	// collect query
	query += search + " ORDER BY  prd.updated_at DESC, p.name ASC LIMIT ? OFFSET ?;" // add search condition and limit, offset
	// execute query
	err := s.db.Raw(query, args...).Scan(&res).Error
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

func (s *Services) RepricingDetailStatus(ctx context.Context, repricingID int, params *domain.QueryParam) (*domain.RepricingDetailStatusSummary, error) {
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
		WHERE prd.price_revalution_id = ?
	`

	var args []any
	args = append(args, repricingID)

	if params.Search != "" {
		query += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
		search := "%" + params.Search + "%"
		args = append(args, search, search)
	}

	var res domain.RepricingDetailStatusSummary
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get repricing detail status: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// confirm repricing
func (s *Services) ConfirmRepricing(ctx context.Context, repricingID int, updatedBy string) error {
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

	err := tx.WithContext(ctx).First(&res, "id = ?", repricingID).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get price_revalution: %v", err)
		return domain.InternalServerError
	}

	err = tx.WithContext(ctx).
		Exec(`
		UPDATE price_revalutions 
		SET 
			status = ?, 
			updated_by = ?, 
			updated_at = NOW() 
		WHERE id = ?`,
			constants.GeneralStatusCompleted,
			updatedBy,
			repricingID,
		).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update price_revalution status: %v", err)
		return domain.InternalServerError
	}

	err = tx.WithContext(ctx).
		Exec(`DELETE FROM price_revalution_details WHERE new_retail_price = 0 and price_revalution_id = ?`, repricingID).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not delete price_revalution_details if price will be zero: %v", err)
		return domain.InternalServerError
	}

	err = tx.WithContext(ctx).Find(&details, "price_revalution_id = ? AND new_retail_price > 0", repricingID).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get price_revalution_detail list: %v", err)
		return domain.InternalServerError
	}

	for _, v := range details {
		err = tx.WithContext(ctx).Exec(`
		UPDATE store_products SET retail_price = ? WHERE id = ?
		`, v.NewRetailPrice, v.StoreProductID).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store_product price: %v", err)
			return err
		}
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit confirm repricing transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// cancel repricing
func (s *Services) CancelRepricing(repricingID string, updatedBy string) error {
	// check if repricing exists
	err := s.db.Exec(`UPDATE price_revalutions SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`,
		constants.GeneralStatusCanceled, updatedBy, repricingID).Error
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
