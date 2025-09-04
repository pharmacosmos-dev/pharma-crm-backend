package services

import (
	"github.com/pharma-crm-backend/domain"
)

func (s *Services) ProductBonusList(param *domain.QueryParam) ([]domain.ProductBonus, int64, error) {
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
	if param.StoreID != "" {
		query = query.Where("store_id = ?", param.StoreID)
	}
	// if search is received it joins with products table and add search condtion
	if param.Search != "" {
		query = query.Joins("JOIN products ON product_bonuses.product_id = products.id").
			Where("products.name ILIKE ?", "%"+param.Search+"%")
	}
	if param.CompanyId != "" {
		query = query.Where("company_id = ?", param.CompanyId)
	}
	// complete query
	err := query.
		Count(&totalCount).
		Limit(param.Limit).Offset(param.Offset).
		Order("created_at desc").
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}
	return res, totalCount, nil
}
