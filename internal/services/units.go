package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type UnitService struct {
	cusRepo storage.UnitRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewUnitService(cusRepo storage.UnitRepo, cfg *config.Config, log *logger.Logger) *UnitService {
	return &UnitService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *UnitService) Create(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	return s.cusRepo.Create(ctx, req)
}

func (s *UnitService) Get(ctx context.Context, Id string) (*domain.Unit, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *UnitService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Unit, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *UnitService) Update(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	return s.cusRepo.Update(ctx, req)
}

func (s *UnitService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
