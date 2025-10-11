package services

import (
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

func (s *Services) CreateOrUpdateFinanceCategory(req *domain.FinanceCategoryRequest) (*domain.FinanceCategory, error) {
	var res domain.FinanceCategory
	query := `
	INSERT INTO finance_categories(
		id, parent_id, name, description, account_group, status)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT (id)
	DO UPDATE SET 
		parent_id = EXCLUDED.parent_id, 
		name = EXCLUDED.name, 
		description = EXCLUDED.description, 
		account_group = EXCLUDED.account_group, 
		status = EXCLUDED.status 
	RETURNING *`

	err := s.db.Raw(query, req.Id, req.ParentId, req.Name, req.Description, req.AccountGroup, constants.GeneralStatusActive).Scan(&res).Error

	if err != nil {
		return nil, err
	}

	return &res, nil
}
