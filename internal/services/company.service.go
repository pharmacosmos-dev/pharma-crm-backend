package services

import (
	"context"
	"encoding/json"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetCompanyIds(ctx context.Context, isFranchise bool) ([]string, error) {
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

func (s *Services) GetCompaniesWithStores(ctx context.Context) (*domain.CompanyWithStoresResponse, error) {
	// Final parsed response
	result := domain.CompanyWithStoresResponse{}

	query := `
WITH pharmaCosmos AS (
    SELECT
        json_build_object(
                'id', c.id,
                'company', c.name,
                'is_franchise', c.is_franchise,
                'stores', COALESCE(
                                json_agg(
                                json_build_object(
                                        'id', s.id,
                                        'name', s.name,
                                        'is_franchise', c.is_franchise
                                )
                                        ) FILTER (WHERE s.id IS NOT NULL), '[]'::json
                          )
        ) AS data
    FROM companies c
             LEFT JOIN stores s ON c.id = s.company_id
    WHERE c.is_franchise = false
    GROUP BY c.id
    LIMIT 1
),

     franchises AS (
         SELECT
             json_build_object(
                     'id', c.id,
                     'company', c.name,
                     'is_franchise', c.is_franchise,
                     'stores', COALESCE(
                                     json_agg(
                                     json_build_object(
                                             'id', s.id,
                                             'name', s.name,
                                             'is_franchise', c.is_franchise
                                     )
                                             ) FILTER (WHERE s.id IS NOT NULL), '[]'::json
                               )
             ) AS data
         FROM companies c
                  LEFT JOIN stores s ON c.id = s.company_id
         WHERE c.is_franchise = true
         GROUP BY c.id
     ),

     franchise_list AS (
         SELECT json_agg(data) AS data
         FROM franchises
     )

SELECT json_build_object(
               'pharma_cosmos', p.data,
               'franchises', fl.data
       )
FROM pharmaCosmos p
         CROSS JOIN franchise_list fl`

	var jsonResult string
	if err := s.db.WithContext(ctx).Raw(query).Scan(&jsonResult).Error; err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
