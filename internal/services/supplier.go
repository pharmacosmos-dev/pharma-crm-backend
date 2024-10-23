package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type SupplierService struct {
	cusRepo storage.SupplierRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewSupplierService(cusRepo storage.SupplierRepo, cfg *config.Config, log *logger.Logger) *SupplierService {
	return &SupplierService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *SupplierService) Create(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error) {
	return s.cusRepo.Create(ctx, req)
}

func (s *SupplierService) Get(ctx context.Context, Id string) (*domain.Supplier, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *SupplierService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Supplier, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *SupplierService) Update(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error) {
	return s.cusRepo.Update(ctx, req)
}

func (s *SupplierService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
