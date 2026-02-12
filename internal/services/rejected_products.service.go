package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) CreateRejectedProduct(req *domain.RejectedProductRequest) error {
	if err := s.db.Table("rejected_products").Create(req).Error; err != nil {
		s.log.Errorf("could not create rejected product: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) GetRejectedProductsSearch(ctx context.Context, params *domain.RejectedProductQueryParam) ([]domain.Product, error) {
	var products []domain.Product

	query := s.db.
		Select(`
			p.id, 
			p.name, 
			p.barcode, 
			p.unit_per_pack, 
			pr.id AS producer_id,
			pr.name AS manufacturer
		`).
		Table("products p").
		Joins("LEFT JOIN producers AS pr ON pr.id = p.producer_id").
		Where("p.deleted_at IS NULL")

	if params.Search != "" {
		query = query.Where("p.name ILIKE ?", fmt.Sprintf("%%%s%%", params.Search))
	}
	err := query.
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&products).Error
	if err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, domain.InternalServerError
	}

	for i := range products {
		products[i].Producer = domain.NewNullStruct(domain.Producer{
			Id:   &products[i].ProducerID,
			Name: products[i].Manufacturer,
		}, products[i].ProducerID != "")
	}

	return products, nil
}

func (s *Services) ListRejectedProducts(ctx context.Context, params *domain.RejectedProductQueryParam) ([]domain.RejectedProduct, int64, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"rp.id AS id",
			"rp.store_id AS store_id",
			"rp.product_id AS product_id",
			"COALESCE(p.name, rp.product_name) AS product_name",
			"s.name AS store_name",
			"rp.rejected_times AS rejected_times",
			"rp.count AS count",
			"e.full_name AS created_by",
			"rp.created_at AS created_at",
		).
		Table("rejected_products rp").
		Joins("LEFT JOIN products p ON rp.product_id = p.id").
		Joins("LEFT JOIN stores s ON rp.store_id = s.id").
		Joins("LEFT JOIN employees e ON rp.created_by = e.id")

	if params.Search != "" {
		qb = qb.Where("p.name ILIKE ? OR rp.product_name ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	if params.StoreId != "" {
		qb = qb.Where("rp.store_id = ?", params.StoreId)
	}

	if params.ProductId != "" {
		qb = qb.Where("rp.product_id = ?", params.ProductId)
	}

	order := " created_at DESC"

	switch params.Order {
	case "+count":
		order = " count DESC"
	case "-count":
		order = " count ASC"
	case "+created_at":
		order = " created_at DESC"
	case "-created_at":
		order = " created_at ASC"
	default:
		order = " created_at DESC"
	}

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, 0, domain.InternalServerError
	}

	qb = qb.Order(order)

	var res []domain.RejectedProduct
	err := qb.Limit(params.Limit).Offset(params.Offset).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
