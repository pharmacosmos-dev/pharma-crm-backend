package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) CreateRejectedProduct(req *domain.RejectedProductRequest) error {
	if err := s.db.Table("rejected_products").Create(req).Error; err != nil {
		return err
	}
	return nil
}

func (s *Services) SearchProductsByName(name string, limit, offset int) ([]domain.Product, error) {
	var products []domain.Product

	query := s.db.Preload("Producer").Table("products AS p").
		Select(`
			p.id, 
			p.name, 
			p.barcode, 
			p.unit_per_pack, 
			pr.name AS producer_name
		`).
		Joins("LEFT JOIN producers AS pr ON pr.id = p.producer_id").
		Where("p.deleted_at IS NULL")

	if name != "" {
		query = query.Where("p.name ILIKE ?", fmt.Sprintf("%%%s%%", name))
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func (s *Services) ListRejectedProducts(param *domain.RejectedProductQueryParam) ([]domain.RejectedProduct, int64, error) {
	var (
		res        []domain.RejectedProduct
		totalCount int64
		args       []any
		filter     = "WHERE 1=1"
		order      = " ORDER BY count DESC"
	)

	if param.Search != "" {
		filter += " AND (p.name ILIKE ? OR rp.product_name ILIKE ? OR p.name ILIKE ?)"
		args = append(args, "%"+param.Search+"%", "%"+param.Search+"%", "%"+param.Search+"%")
	}
	if param.StoreID != "" {
		filter += " AND rp.store_id = ?"
		args = append(args, param.StoreID)
	}
	if param.ProductID != "" {
		filter += " AND rp.product_id = ?"
		args = append(args, param.ProductID)
	}

	query := fmt.Sprintf(`
	SELECT 
		ROW_NUMBER() OVER(ORDER BY rp.store_id) as id, 
		rp.store_id, 
		rp.product_id,
		COALESCE(p.name, rp.product_name) AS product_name,
		s.name AS store_name,
		e.full_name AS created_by,
		MAX(rp.created_at) as created_at,
		COUNT(*) AS count,
		COUNT(*) OVER() AS total_count
	FROM rejected_products rp
	LEFT JOIN products AS p ON rp.product_id = p.id
	LEFT JOIN stores AS s ON rp.store_id = s.id
	LEFT JOIN employees AS e ON rp.created_by = e.id
	%s
	GROUP BY 
		rp.store_id, 
		rp.product_id, 
		p.name, 
		rp.product_name, 
		s.name, 
		e.full_name
	%s
	LIMIT ? OFFSET ?
`, filter, order)

	args = append(args, param.Limit, param.Offset)

	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		return nil, 0, err
	}

	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}
