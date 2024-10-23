package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type RoleService struct {
	cusRepo storage.RoleRepo
	cfg     *config.Config
	log     *logger.Logger
}

func NewRoleService(cusRepo storage.RoleRepo, cfg *config.Config, log *logger.Logger) *RoleService {
	return &RoleService{cusRepo: cusRepo, cfg: cfg, log: log}
}

func (s *RoleService) Create(ctx context.Context, Role *domain.Role) (*domain.Role, error) {
	return s.cusRepo.Create(ctx, Role)
}

func (s *RoleService) Get(ctx context.Context, Id string) (*domain.Role, error) {
	return s.cusRepo.Get(ctx, Id)
}

func (s *RoleService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Role, error) {
	return s.cusRepo.GetList(ctx, param)
}

func (s *RoleService) Update(ctx context.Context, Role *domain.Role) (*domain.Role, error) {
	return s.cusRepo.Update(ctx, Role)
}

func (s *RoleService) Delete(ctx context.Context, Id string) error {
	return s.cusRepo.Delete(ctx, Id)
}
