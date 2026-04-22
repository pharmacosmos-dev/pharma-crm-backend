package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// CreateOnlineRepricing creates an online repricing session and auto-populates details.
// For each store_product: uses online_store_products price if exists, else store_products price.
func (s *Services)  CreateOnlineRepricing(ctx context.Context, req *domain.OnlineRepricingRequest) (*domain.OnlinePriceRevaluation, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var res domain.OnlinePriceRevaluation
	err := tx.Raw(`
		INSERT INTO online_price_revaluations (store_id, platform_type, name, created_by)
		VALUES (?, ?, ?, ?)
		RETURNING *`,
		req.StoreId, req.PlatformType, req.Name, req.CreatedBy,
	).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create online_price_revaluation: %v", err)
		return nil, domain.InternalServerError
	}

	// Auto-populate details: all products from store_products.
	// old_retail_price = online price if exists, else store_products price.
	// Detail ga store_products dan oladi, online_store_products da mavjud bo'lsa old_price sifatida ishlatadi
	err = tx.Exec(`
		INSERT INTO online_price_revaluation_details
			(online_price_revaluation_id, store_id, product_id, old_retail_price, new_retail_price, old_supply_price)
		SELECT
			?,
			sp.store_id,
			sp.product_id,
			COALESCE(osp.retail_price, sp.retail_price),
			0,
			COALESCE(osp.supply_price, sp.supply_price)
		FROM (
			SELECT DISTINCT ON (product_id)
				product_id, store_id, retail_price, supply_price
			FROM store_products
			WHERE store_id = ?
			  AND (pack_quantity > 0 OR unit_quantity > 0)
			ORDER BY product_id, unit_quantity DESC
		) sp
		LEFT JOIN online_store_products osp
			ON osp.product_id = sp.product_id
			AND osp.store_id = sp.store_id
			AND osp.type = ?
		ON CONFLICT (online_price_revaluation_id, product_id) DO NOTHING`,
		res.Id, req.StoreId, req.PlatformType,
	).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not populate online_price_revaluation_details: %v", err)
		return nil, domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit create online repricing: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// GetOnlineRepricingList returns list of online repricings with filters.
func (s *Services) GetOnlineRepricingList(ctx context.Context, params *domain.OnlineRepricingQueryParam) ([]domain.OnlinePriceRevaluation, int64, error) {
	var (
		res        []domain.OnlinePriceRevaluation
		totalCount int64
	)

	q := s.db.WithContext(ctx).
		Model(&domain.OnlinePriceRevaluation{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`online_price_revaluations.*, COUNT(d.id) AS count`).
		Joins(`LEFT JOIN online_price_revaluation_details d ON d.online_price_revaluation_id = online_price_revaluations.id`).
		Group(`online_price_revaluations.id`)

	if params.StoreId != "" {
		q = q.Where("online_price_revaluations.store_id = ?", params.StoreId)
	}
	if params.PlatformType != "" {
		q = q.Where("online_price_revaluations.platform_type = ?", params.PlatformType)
	}
	if params.Status != "" {
		q = q.Where("online_price_revaluations.status = ?", params.Status)
	}

	if err := q.Count(&totalCount).Error; err != nil {
		return nil, 0, domain.InternalServerError
	}

	if err := q.Order("online_price_revaluations.created_at DESC").
		Limit(params.Limit).Offset(params.Offset).
		Find(&res).Error; err != nil {
		s.log.Errorf("could not get online repricing list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// GetOnlineRepricingDetailList returns product details for a given repricing session.
func (s *Services) GetOnlineRepricingDetailList(ctx context.Context, repricingId int, params *domain.OnlineRepricingQueryParam) ([]domain.OnlinePriceRevalutionDetail, int64, error) {
	var res []domain.OnlinePriceRevalutionDetail

	args := []any{repricingId}
	searchClause := ""
	if params.Search != "" {
		searchClause = " AND (p.name ILIKE ? OR p.barcode ILIKE ?)"
		s := "%" + params.Search + "%"
		args = append(args, s, s)
	}
	args = append(args, params.Limit, params.Offset)

	query := `
		SELECT
			d.id,
			d.online_price_revaluation_id,
			d.store_id,
			d.product_id,
			d.old_retail_price,
			d.new_retail_price,
			d.old_supply_price,
			d.created_at,
			d.updated_at,
			p.name,
			p.barcode,
			COUNT(*) OVER() AS total_count
		FROM online_price_revaluation_details d
		JOIN products p ON p.id = d.product_id
		WHERE d.online_price_revaluation_id = ?` + searchClause + `
		ORDER BY d.updated_at DESC, p.name ASC
		LIMIT ? OFFSET ?`

	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get online repricing details: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var totalCount int64
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// UpdateOnlineRepricingDetailPrice updates new_retail_price for a single detail row.
func (s *Services) UpdateOnlineRepricingDetailPrice(ctx context.Context, req *domain.UpdateOnlineDetailPrice) error {
	err := s.db.WithContext(ctx).
		Exec(`UPDATE online_price_revaluation_details SET new_retail_price = ?, updated_at = NOW() WHERE id = ?`,
			req.NewRetailPrice, req.Id).Error
	if err != nil {
		s.log.Errorf("could not update online repricing detail price: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// ConfirmOnlineRepricing applies new prices to online_store_products and marks session completed.
func (s *Services) ConfirmOnlineRepricing(ctx context.Context, repricingId int, updatedBy string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var storeId, platformType string
	if err := tx.Raw(`SELECT store_id, platform_type FROM online_price_revaluations WHERE id = ?`, repricingId).
		Row().Scan(&storeId, &platformType); err != nil {
		_ = tx.Rollback()
		s.log.Errorf("online repricing %d not found: %v", repricingId, err)
		return domain.InternalServerError
	}

	// Mark completed
	if err := tx.Exec(`
		UPDATE online_price_revaluations
		SET status = ?, updated_by = ?, updated_at = NOW()
		WHERE id = ?`,
		constants.GeneralStatusCompleted, updatedBy, repricingId,
	).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update online repricing status: %v", err)
		return domain.InternalServerError
	}

	// Delete details where price was not changed (new_retail_price = 0)
	if err := tx.Exec(
		`DELETE FROM online_price_revaluation_details WHERE new_retail_price = 0 AND online_price_revaluation_id = ?`,
		repricingId,
	).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not delete unchanged online repricing details: %v", err)
		return domain.InternalServerError
	}

	// Upsert online_store_products from detail rows where new_retail_price > 0
	if err := tx.Exec(`
		INSERT INTO online_store_products (store_id, product_id, type, retail_price, supply_price, old_supply_price, updated_at)
		SELECT
			?,
			d.product_id,
			?,
			d.new_retail_price,
			d.old_supply_price,
			d.old_supply_price,
			NOW()
		FROM online_price_revaluation_details d
		WHERE d.online_price_revaluation_id = ? AND d.new_retail_price > 0
		ON CONFLICT (store_id, product_id, type)
		DO UPDATE SET
			retail_price     = EXCLUDED.retail_price,
			old_supply_price = online_store_products.supply_price,
			updated_at       = NOW()`,
		storeId, platformType, repricingId,
	).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not upsert online_store_products on confirm: %v", err)
		return domain.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit confirm online repricing: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// CancelOnlineRepricing marks the session canceled.
func (s *Services) CancelOnlineRepricing(ctx context.Context, repricingId int, updatedBy string) error {
	
	err := s.db.WithContext(ctx).Exec(`
		UPDATE online_price_revaluations
		SET status = ?, updated_by = ?, updated_at = NOW()
		WHERE id = ?`,
		constants.GeneralStatusCanceled, updatedBy, repricingId,
	).Error
	if err != nil {
		s.log.Errorf("could not cancel online repricing: %v", err)
		return domain.InternalServerError
	}
	return nil
}
