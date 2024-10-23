package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type BrandService struct {
	cusRepo storage.BrandRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewBrandService(cusRepo storage.BrandRepo, cfg *config.Config, log *logger.Logger) *BrandService {
	return &BrandService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *BrandService) Create(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	return s.cusRepo.Create(ctx, brand)
}

func (s *BrandService) Get(ctx context.Context, Id string) (*domain.Brand, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *BrandService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Brand, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *BrandService) Update(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	return s.cusRepo.Update(ctx, brand)
}

func (s *BrandService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
