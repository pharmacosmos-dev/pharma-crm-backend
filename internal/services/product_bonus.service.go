package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetProductBonuses(ctx context.Context, params *domain.QueryParam) ([]domain.ProductBonus, int64, error) {
	var (
		totalCount int64
		res        []domain.ProductBonus
	)
	// get all product bonuses
	query := s.db.
		Model(&domain.ProductBonus{}).
		Table("product_bonuses").
		Preload("Product").
		Preload("Store")

	// filter with store id
	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}
	// if search is received it joins with products table and add search condtion
	if params.Search != "" {
		query = query.Joins("JOIN products ON product_bonuses.product_id = products.id").
			Where("products.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.CompanyId != "" {
		query = query.Where("company_id = ?", params.CompanyId)
	}
	// complete query
	err := query.
		Count(&totalCount).
		Limit(params.Limit).Offset(params.Offset).
		Order("created_at desc").
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}
	return res, totalCount, nil
}

func (s *Services) GetStoreProductsByIds(ctx context.Context, ids []string) ([]domain.StoreProduct, error) {
	var res []domain.StoreProduct
	err := s.db.WithContext(ctx).Where("id IN(?)", ids).Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get store_products by ids: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}
