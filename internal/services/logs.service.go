package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	query := `
	SELECT * FROM (
	SELECT
		pr.id::TEXT AS id,
		pr.payment_provider AS provider_type,
		pr.method,
		pr.payload,
		pr.response,
		pr.created_at
	FROM payment_requests pr
	UNION ALL
	SELECT
		ep.id::TEXT AS id,
		'epos' AS provider_type,
		'ofd' AS method,
		null AS payload,
		ep.response,
		ep.created_at
	FROM epos_responses ep
	UNION ALL
	SELECT
		dr.id::TEXT AS id,
		'dmed' AS provider_type,
		dr.method,
		dr.payload,
		dr.response,
		dr.created_at
	FROM dmed_requests dr) combined
	`
	countQuery := `
	SELECT COUNT(*) as total_count FROM (
	SELECT
		pr.id::TEXT AS id,
		pr.payment_provider AS provider_type,
		pr.method,
		pr.payload,
		pr.response,
		pr.created_at
	FROM payment_requests pr
	UNION ALL
	SELECT
		ep.id::TEXT AS id,
		'epos' AS provider_type,
		'ofd' AS method,
		null AS payload,
		ep.response,
		ep.created_at
	FROM epos_responses ep
	UNION ALL
	SELECT
		dr.id::TEXT AS id,
		'dmed' AS provider_type,
		dr.method,
		dr.payload,
		dr.response,
		dr.created_at
	FROM dmed_requests dr) combined
	`
	var (
		filter = "WHERE 1 = 1"
		args   = []any{}
	)
	if params.ProviderType != "" {
		filter += " AND provider_type = ?"
		args = append(args, params.ProviderType)
	}
	if params.StartDate != "" {
		date, err := s.FormatDatetimeParams(params.StartDate, params.EndDate)
		if err != nil {
			return nil, 0, err
		}
		filter += " AND created_at BETWEEN ? AND ?"
		args = append(args, date.StartTime, date.EndTime)
	}

	var totalCount int64
	// execute count query
	err := s.db.WithContext(ctx).Raw(countQuery+filter, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get logs total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// add order by
	filter += " ORDER BY created_at DESC"

	// add limit, offset
	filter += " LIMIT ? OFFSET ?;"
	args = append(args, params.Limit, params.Offset)
	var res []domain.Log
	err = s.db.WithContext(ctx).Raw(query+filter, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get logs: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
