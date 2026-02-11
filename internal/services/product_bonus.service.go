package services

import (
	"context"
	"time"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetProductBonuses(ctx context.Context, params *domain.QueryParam) ([]domain.ProductBonus, int64, error) {
	var tmpBonus []struct {
		Id           int64      `gorm:"id"`
		ProductId    string     `gorm:"product_id"`
		StoreId      string     `gorm:"store_id"`
		BonusAmount  float64    `gorm:"bonus_amount"`
		Status       int        `gorm:"status"`
		StartDate    string     `gorm:"start_date"`
		EndDate      string     `gorm:"end_date"`
		CreatedAt    *time.Time `gorm:"created_at"`
		UpdatedAt    *time.Time `gorm:"updated_at"`
		ProductName  string     `gorm:"product_name"`
		MaterialCode int        `gorm:"material_code"`
		StoreName    string     `gorm:"store_name"`
	}

	// get all product bonuses
	qb := s.db.
		WithContext(ctx).
		Joins("JOIN products p ON pb.product_id = p.id").
		Joins("LEFT JOIN stores s ON s.id = pb.store_id").
		Table("product_bonuses pb")

	// // filter with store id
	// if params.StoreID != "" {
	// 	query = query.Where("store_id = ?", params.StoreID)
	// }
	// if search is received it joins with products table and add search condtion
	if params.Search != "" {
		qb = qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.CompanyId != "" {
		qb = qb.Where("pb.company_id = ?", params.CompanyId)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get product_bonuses total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// complete query
	err := qb.
		Select(
			"pb.id",
			"pb.product_id",
			"pb.store_id",
			"pb.bonus_amount",
			"pb.status",
			"pb.start_date",
			"pb.end_date",
			"pb.created_at",
			"pb.updated_at",
			"p.name AS product_name",
			"p.material_code",
			"s.name AS store_name",
		).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("created_at DESC").
		Find(&tmpBonus).Error
	if err != nil {
		s.log.Errorf("could not get product_bonuses list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.ProductBonus
	for _, item := range tmpBonus {
		res = append(res, domain.ProductBonus{
			Id:          item.Id,
			ProductId:   item.ProductId,
			Status:      item.Status,
			BonusAmount: item.BonusAmount,
			StartDate:   item.StartDate,
			EndDate:     item.EndDate,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			Product: domain.NewNullStruct(domain.ProductForBonus{
				Id:           item.ProductId,
				Name:         item.ProductName,
				MaterialCode: item.MaterialCode,
			}, item.ProductId != ""),
			Store: domain.NewNullStruct(domain.StoreForBonus{
				Id:   item.StoreId,
				Name: item.StoreName,
			}, item.StoreId != ""),
		})
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

func (s *Services) SoldProductBonusList(ctx context.Context, params *domain.QueryParam) ([]domain.SoldProductBonus, int64, error) {
	var (
		totalCount int64
		res        []domain.SoldProductBonus
	)

	query := s.db.
		WithContext(ctx).
		Table("employee_bonus eb").
		Select(`
			eb.id,
			eb.employee_id,
			e.full_name as employee_name,
			eb.product_id,
			p.name as product_name,
			eb.bonus_amount,
			eb.quantity,
			eb.unit_quantity,
			eb.created_at
		`).
		Joins("LEFT JOIN products p ON p.id = eb.product_id").
		Joins("LEFT JOIN employees e ON e.id = eb.employee_id")

	// filter by store
	if params.StoreID != "" {
		query = query.Joins("LEFT JOIN sales s ON s.id = eb.sale_id").
			Where("s.store_id = ?", params.StoreID)
	}

	// search by product name
	if params.Search != "" {
		query = query.Where("p.name ILIKE ?", "%"+params.Search+"%")
	}

	// filter by company
	if params.CompanyId != "" {
		query = query.
			Where("e.company_id = ?", params.CompanyId)
	}

	// filter by employee
	if params.EmployeeId != "" {
		query = query.Where("eb.employee_id = ?", params.EmployeeId)
	}

	err := query.Count(&totalCount).
		Limit(params.Limit).Offset(params.Offset).
		Order("eb.created_at desc").
		Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get sold product bonus list: %v", err)
		return res, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
