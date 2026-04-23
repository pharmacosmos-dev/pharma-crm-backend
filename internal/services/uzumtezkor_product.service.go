package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

// InsertOnlinePricesFromOnec — 1C dan material_code + price oladi, product_id topib insert qiladi.
// type = 'uzum', created_by handler dan uzatiladi.
func (s *Services) InsertOnlinePricesFromOnec(ctx context.Context, req *domain.UzumTezkorProductRepriceFromOnecRequest, createdBy string) error {
	if len(req.Items) == 0 {
		return fmt.Errorf("items list is empty")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	for _, item := range req.Items {
		err := tx.Exec(`
			INSERT INTO online_store_products (product_id, material_code, type, retail_price, created_by)
			SELECT p.id, ?, 'uzum', ?, ?
			FROM products p
			WHERE p.material_code::text = ?
			LIMIT 1`,
			item.MaterialCode,
			item.RetailPrice,
			createdBy,
			item.MaterialCode,
		).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("failed to insert online price material_code=%s: %v", item.MaterialCode, err)
			return domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit InsertOnlinePricesFromOnec: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// GetOnlineStoreProducts — CRM uchun narx tarixi
func (s *Services) GetOnlineStoreProducts(ctx context.Context, params *domain.UzumTezkorProductQueryParam) ([]domain.OnlineStoreProduct, int64, error) {
	var result []domain.OnlineStoreProduct
	var total int64

	q := s.db.WithContext(ctx).Table("online_store_products")

	if params.Type != "" {
		q = q.Where("type = ?", params.Type)
	}
	if params.ProductId != "" {
		q = q.Where("product_id = ?", params.ProductId)
	}
	if params.MaterialCode != "" {
		q = q.Where("material_code = ?", params.MaterialCode)
	}

	if err := q.Count(&total).Error; err != nil {
		s.log.Errorf("failed to count online_store_products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if err := q.Order("created_at DESC").
		Limit(params.Limit).Offset(params.Offset).
		Find(&result).Error; err != nil {
		s.log.Errorf("failed to get online_store_products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return result, total, nil
}
