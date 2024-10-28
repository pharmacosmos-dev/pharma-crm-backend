package services

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CategoryService struct {
	categoryRepo storage.CategoryRepo
	cfg          *config.Config
	log          *logger.Logger
}

func NewCategoryService(cusRepo storage.CategoryRepo, cfg *config.Config, log *logger.Logger) *CategoryService {
	return &CategoryService{categoryRepo: cusRepo, cfg: cfg, log: log}
}

func (s *CategoryService) Create(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	return s.categoryRepo.Create(ctx, req)
}

func (s *CategoryService) Get(ctx context.Context, Id string) (*domain.Category, error) {
	return s.categoryRepo.Get(ctx, Id)
}

func (s *CategoryService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Category, error) {
	return s.categoryRepo.GetList(ctx, param)
}

func (s *CategoryService) Update(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	return s.categoryRepo.Update(ctx, req)
}

func (s *CategoryService) Delete(ctx context.Context, Id string) error {
	return s.categoryRepo.Delete(ctx, Id)
}
