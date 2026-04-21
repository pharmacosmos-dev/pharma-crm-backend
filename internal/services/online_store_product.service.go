package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) UpsertOnlineStoreProducts(ctx context.Context, req *domain.UpsertOnlineStoreProductsRequest) error {
	if len(req.Products) == 0 {
		return fmt.Errorf("products list is empty")
	}

	for _, item := range req.Products {
		err := s.db.WithContext(ctx).Exec(`
			INSERT INTO online_store_products (store_id, product_id, type, retail_price, supply_price, old_supply_price, created_by, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
			ON CONFLICT (store_id, product_id, type)
			DO UPDATE SET
				retail_price    = EXCLUDED.retail_price,
				supply_price    = EXCLUDED.supply_price,
				old_supply_price = EXCLUDED.old_supply_price,
				updated_at      = NOW()
		`, req.StoreId, item.ProductId, req.Type,
			item.RetailPrice, item.SupplyPrice, item.OldSupplyPrice, req.CreatedBy,
		).Error
		if err != nil {
			s.log.Errorf("failed to upsert online_store_product product_id=%s: %v", item.ProductId, err)
			return err
		}
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
