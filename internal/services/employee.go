package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type EmployeeService struct {
	cusRepo storage.EmployeeRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewEmployeeService(cusRepo storage.EmployeeRepo, cfg *config.Config, log *logger.Logger) *EmployeeService {
	return &EmployeeService{
		cusRepo: cusRepo,
		cfg:     cfg,
		log:     log,
	}
}

func (s *EmployeeService) Create(ctx context.Context, employee *domain.Employee) (*domain.Employee, error) {
	return s.cusRepo.Create(ctx, employee)
}

func (s *EmployeeService) Get(ctx context.Context, id string) (*domain.Employee, error) {
	return s.cusRepo.Get(ctx, id)
}

func (s *EmployeeService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Employee, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *EmployeeService) Update(ctx context.Context, employee *domain.Employee) error {
	return s.cusRepo.Update(ctx, employee)
}

func (s *EmployeeService) Delete(ctx context.Context, id string) error {
	return s.cusRepo.Delete(ctx, id)
}
