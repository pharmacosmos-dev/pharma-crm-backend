package services

import (
	"fmt"
	"strings"

	"github.com/pharma-crm-backend/domain"
)

// get customer list data
func (s *Services) ListCustomer(param *domain.QueryParam) ([]domain.Customer, int64, error) {
	var (
		res        []domain.Customer
		totalCount int64
	)

	// Start building the query
	query := s.db.
		Model(&domain.Customer{}).
		Preload("Store").
		Preload("Tag").
		Select(`
		customers.*,
		(SELECT created_at
		FROM sales
		WHERE sales.customer_id = customers.id
		ORDER BY sales.created_at DESC LIMIT 1)
		AS sale_date,
		COALESCE(SUM(sales.total_amount), 0) AS sale_amount`).
		Joins("LEFT JOIN sales ON sales.customer_id = customers.id").
		Where("customers.is_active = ?", true)

	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("customers.full_name ILIKE ? OR customers.phone LIKE ? OR CAST(customers.public_id AS TEXT) LIKE ?",
			param.Search, param.Search, strings.Trim(param.Search, "%"))
	}
	if param.StoreID != "" {
		query = query.Where("customers.store_id = ?", param.StoreID)
	}
	err := query.
		Group("customers.id").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("customers.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	return res, totalCount, nil
}
