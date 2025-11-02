package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) getCompanyIds(ctx context.Context, isFranchise bool) ([]string, error) {
	var companyIds []string
	err := s.db.WithContext(ctx).
		Model(&domain.Company{}).
		Where("is_franchise = ?", isFranchise).
		Pluck("id", &companyIds).Error
	if err != nil {
		s.log.Errorf("could not get company_ids: %v", err)
		return nil, domain.InternalServerError // return nil instead of empty slice
	}
	return companyIds, nil
}
