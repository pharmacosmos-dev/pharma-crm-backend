package services

import (
	"context"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CustomerService struct {
	cusRepo storage.CustomerRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewCustomerService(cusRepo storage.CustomerRepo, cfg *config.Config, log *logger.Logger) *CustomerService {
	return &CustomerService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *CustomerService) Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	return s.cusRepo.Create(ctx, customer)
}

func (s *CustomerService) Get(ctx context.Context, Id string) (*domain.Customer, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *CustomerService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Customer, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *CustomerService) Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	return s.cusRepo.Update(ctx, customer)
}

func (s *CustomerService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
