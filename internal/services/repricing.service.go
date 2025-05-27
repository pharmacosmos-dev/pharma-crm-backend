package services

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
)

// create price_revalutions
func (s *Services) CreateRepricing(req *domain.RepricingRequest) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := s.db.Raw(`INSERT INTO price_revalutions(store_id, name, type, created_by) VALUES(?, ?, ?, ?) RETURNING *`,
		req.StoreId, req.Name, req.Type, req.CreatedBy).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on creating price_revalution: %v", err)
		return &res, err
	}
	return &res, nil
}

// get repricing by id
func (s *Services) GetRepricingByID(repricingID string) (*domain.PriceRevalution, error) {
	var res domain.PriceRevalution
	err := s.db.Model(&domain.PriceRevalution{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			price_revalutions.*
			`).
		First(&res, "imports.id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// repricing get list
func (s *Services) RepricingList(param *domain.QueryParam) ([]domain.PriceRevalution, int64, error) {
	var (
		res        []domain.PriceRevalution
		totalCount int64
	)

	query := s.db.Model(&domain.PriceRevalution{}).
		Preload("Store").Preload("CreatedBy").Preload("UpdatedBy").
		Select(`price_revalutions.*, SUM(prd.scanned_count) AS count
		`).Joins("LEFT JOIN price_revalution_details prd ON price_revalutions.id = prd.price_revalution_id").Group("price_revalutions.id")

	if param.StoreID != "" {
		query = query.Where("price_revalutions.store_id = ?", param.StoreID)
	}
	if param.Search != "" {
		query = query.
			Joins("JOIN stores s ON s.id = price_revalutions.store_id").
			Where("CAST(price_revalutions.id AS TEXT) like ? OR s.name ilike ?", "%"+param.Search+"%", "%"+param.Search+"%")
	}
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("price_revalutions.created_at::date BETWEEN ? AND ?", param.StartDate, param.EndDate)
	}

	if param.Status != "" {
		query = query.Where("price_revalutions.status = ?", param.Status)
	}

	err := query.Count(&totalCount).Limit(param.Limit).Offset(param.Offset).Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting price revalution: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}

// confirm repricing
func (s *Services) ConfirmRepricing(repricingID string, updatedBy string) error {
	var res domain.PriceRevalution
	err := s.db.First(&res, "id = ?", repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on getting price_revalution: %v", err)
		return err
	}

	return nil
}

// cancel repricing
func (s *Services) CancelRepricing(repricingID string, updatedBy string) error {
	err := s.db.Exec(`UPDATE price_revalutions SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`,
		config.CANCELED, updatedBy, repricingID).Error
	if err != nil {
		s.log.Warn("ERROR on canceling repricing: %v", err)
		return err
	}
	return nil
}
