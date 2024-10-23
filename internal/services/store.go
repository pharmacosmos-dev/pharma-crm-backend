package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type StoreService struct {
	cusRepo storage.StoreRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewStoreService(cusRepo storage.StoreRepo, cfg *config.Config, log *logger.Logger) *StoreService {
	return &StoreService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *StoreService) Create(ctx context.Context, Store *domain.Store) (*domain.Store, error) {
	return s.cusRepo.Create(ctx, Store)
}

func (s *StoreService) Get(ctx context.Context, Id string) (*domain.Store, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *StoreService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Store, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *StoreService) Update(ctx context.Context, Store *domain.Store) (*domain.Store, error) {
	return s.cusRepo.Update(ctx, Store)
}

func (s *StoreService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
