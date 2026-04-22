package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

// CreateOnlineStoreProducts — store_products dan ko'chirib online_store_products ga yozadi.
// Allaqachon mavjud bo'lganlarni o'zgartirmaydi.
func (s *Services) CreateOnlineStoreProducts(ctx context.Context, storeId, platformType string, createdBy *string) error {
	err := s.db.WithContext(ctx).Exec(`
		INSERT INTO online_store_products (store_id, product_id, type, retail_price, supply_price, old_supply_price, created_by)
		SELECT DISTINCT ON (product_id)
			store_id,
			product_id,
			?,
			retail_price,
			supply_price,
			supply_price,
			?
		FROM store_products
		WHERE store_id = ?
		  AND (pack_quantity > 0 OR unit_quantity > 0)
		ORDER BY product_id, unit_quantity DESC
		ON CONFLICT (store_id, product_id, type) DO NOTHING
	`, platformType, createdBy, storeId).Error
	if err != nil {
		s.log.Errorf("failed to create online_store_products for store=%s type=%s: %v", storeId, platformType, err)
		return err
	}
	return nil
}

func (s *Services) GetOnlineStoreProducts(ctx context.Context, params *domain.OnlineStoreProductQueryParam) ([]domain.OnlineStoreProduct, error) {
	var result []domain.OnlineStoreProduct

	q := s.db.WithContext(ctx).Table("online_store_products")
	if params.StoreId != "" {
		q = q.Where("store_id = ?", params.StoreId)
	}
	if params.Type != "" {
		q = q.Where("type = ?", params.Type)
	}

	if err := q.Find(&result).Error; err != nil {
		s.log.Errorf("failed to get online_store_products: %v", err)
		return nil, err
	}

	return result, nil
}
