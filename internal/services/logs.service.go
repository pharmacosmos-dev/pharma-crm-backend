package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

func (s *Services) GetLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	switch {
	case params.ProviderType == "epos" || params.ProviderType == "":
		return s.GetEposLogs(ctx, params)

	case utils.In(params.ProviderType, constants.PaymentTypePayme, constants.PaymentTypeClick, constants.PaymentTypeAlif):
		return s.GetPaymentLogs(ctx, params)

	case params.ProviderType == constants.ServiceTypeDmed:
		return s.GetDmedLogs(ctx, params)

	case params.ProviderType == constants.ServiceTypeUzum:
		return s.GetUzumLogs(ctx, params)

	case params.ProviderType == "revolution":
		return s.GetRevolutionLogs(ctx, params)

	case params.ProviderType == "max_price":
		return s.GetMaxPriceLogs(ctx, params)

	case params.ProviderType == "transfers":
		return s.GetTransferOnecLogs(ctx, params)

	case params.ProviderType == "return":
		return s.GetReturnLogs(ctx, params)
	}

	return nil, 0, domain.BadRequestError
}

// --- Payment logs (payme / click / alif) ---

func (s *Services) GetPaymentLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select("id", "payment_provider AS provider_type", "method", "payload", "response", "created_at").
		Table("payment_requests")

	if params.ProviderType != "" {
		qb = qb.Where("payment_provider = ?", params.ProviderType)
	}
	qb = s.applyDateFilter(qb, params)

	return s.fetchLogs(qb, params, "payment logs")
}

// --- Epos logs ---

func (s *Services) GetEposLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select("id", "'epos' AS provider_type", "'ofd' AS method", "NULL AS payload", "response", "created_at").
		Table("epos_responses")

	qb = s.applyDateFilter(qb, params)

	return s.fetchLogs(qb, params, "epos logs")
}

// --- Dmed logs ---

func (s *Services) GetDmedLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select("id::TEXT AS id", "'dmed' AS provider_type", "method", "payload", "response", "created_at").
		Table("dmed_requests")

	qb = s.applyDateFilter(qb, params)

	return s.fetchLogs(qb, params, "dmed logs")
}

// --- Uzum order logs ---

func (s *Services) GetUzumLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select("id::TEXT AS id", "'uzum' AS provider_type", "method", "payload", "response", "created_at").
		Table("uzum_order_logs")

	qb = s.applyDateFilter(qb, params)

	return s.fetchLogs(qb, params, "uzum logs")
}

// --- onec_requests based logs ---

func (s *Services) GetRevolutionLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	return s.getOnecRequestLogs(ctx, params, "revolution",
		"method IN (?, ?)", "POST /product1c/repricing", "POST /product1c/multi-repricing",
	)
}

func (s *Services) GetMaxPriceLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	return s.getOnecRequestLogs(ctx, params, "max_price",
		"method = ?", "POST /product1c/max-price-changing",
	)
}

func (s *Services) GetTransferOnecLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	return s.getOnecRequestLogs(ctx, params, "transfers",
		"method = ?", "POST "+constants.OnecPathPerekit,
	)
}

func (s *Services) GetReturnLogs(ctx context.Context, params *domain.LogParams) ([]domain.Log, int64, error) {
	return s.getOnecRequestLogs(ctx, params, "return",
		"method = ?", "POST "+constants.OnecPathVozvrat,
	)
}

// --- helpers ---

func (s *Services) getOnecRequestLogs(ctx context.Context, params *domain.LogParams, providerType string, where string, args ...any) ([]domain.Log, int64, error) {
	qb := s.db.WithContext(ctx).
		Select("id::TEXT AS id", "? AS provider_type", "method", "payload", "response", "created_at", providerType).
		Table("onec_requests").
		Where(where, args...)

	qb = s.applyDateFilter(qb, params)

	return s.fetchLogs(qb, params, providerType+" logs")
}

func (s *Services) applyDateFilter(qb *gorm.DB, params *domain.LogParams) *gorm.DB {
	if params.StartDate == "" {
		return qb
	}
	date, err := s.FormatDatetimeParams(params.StartDate, params.EndDate)
	if err != nil {
		return qb
	}
	return qb.Where("created_at BETWEEN ? AND ?", date.StartTime, date.EndTime)
}

func (s *Services) fetchLogs(qb *gorm.DB, params *domain.LogParams, logName string) ([]domain.Log, int64, error) {
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get %s total_count: %v", logName, err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Log
	if err := qb.Order("created_at DESC").Limit(params.Limit).Offset(params.Offset).Find(&res).Error; err != nil {
		s.log.Errorf("could not get %s: %v", logName, err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
