package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

func (s *Services) GetLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	if params.ProviderType == "epos" || params.ProviderType == "" {
		res, totalCount, err := s.GetEposLogs(ctx, params)
		if err != nil {
			return nil, 0, err
		}
		return res, totalCount, nil
	}

	if utils.In(params.ProviderType, []string{constants.PaymentTypePayme, constants.PaymentTypeClick, constants.PaymentTypeAlif}...) {
		res, totalCount, err := s.GetPaymentLogs(ctx, params)
		if err != nil {
			return nil, 0, err
		}
		return res, totalCount, nil
	}

	if params.ProviderType == constants.ServiceTypeDmed {
		res, totalCount, err := s.GetDmedLogs(ctx, params)
		if err != nil {
			return nil, 0, err
		}
		return res, totalCount, nil
	}

	return nil, 0, domain.BadRequestError
}

func (s *Services) GetPaymentLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"id",
			"payment_provider AS provider_type",
			"method",
			"payload",
			"response",
			"created_at",
		).Table("payment_requests")
	if params.ProviderType != "" {
		qb = qb.Where("payment_provider = ?", params.ProviderType)
	}
	if params.StartDate != "" {
		date, err := s.FormatDatetimeParams(params.StartDate, params.EndDate)
		if err != nil {
			return nil, 0, err
		}
		qb = qb.Where("created_at BETWEEN ? AND ?", date.StartTime, date.EndTime)
	}

	var totalCount int64
	err := qb.Count(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get logs total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Log
	err = qb.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get logs: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetEposLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"id",
			"'epos' AS provider_type",
			"'ofd' AS method",
			"NULL AS payload",
			"response",
			"created_at",
		).Table("epos_responses")
	if params.StartDate != "" {
		date, err := s.FormatDatetimeParams(params.StartDate, params.EndDate)
		if err != nil {
			return nil, 0, err
		}
		qb = qb.Where("created_at BETWEEN ? AND ?", date.StartTime, date.EndTime)
	}

	var totalCount int64
	err := qb.Count(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get epos logs total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Log
	err = qb.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get epos logs: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetDmedLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"id::TEXT AS id",
			"'dmed' AS provider_type",
			"method",
			"payload",
			"response",
			"created_at",
		).Table("dmed_requests")
	if params.StartDate != "" {
		date, err := s.FormatDatetimeParams(params.StartDate, params.EndDate)
		if err != nil {
			return nil, 0, err
		}
		qb = qb.Where("created_at BETWEEN ? AND ?", date.StartTime, date.EndTime)
	}

	var totalCount int64
	err := qb.Count(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get dmed logs total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Log
	err = qb.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get dmed logs: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
