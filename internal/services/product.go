package services

import (
	"context"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type ProductService struct {
	productRepo storage.ProductRepo
	cfg         *config.Config
	log         *logger.Logger
}

func NewProductService(productRepo storage.ProductRepo, cfg *config.Config, log *logger.Logger) *ProductService {
	return &ProductService{productRepo: productRepo, cfg: cfg, log: log}
}

func (s *ProductService) Create(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	return s.productRepo.Create(ctx, req)
}

func (s *ProductService) Get(ctx context.Context, id string) (*domain.Product, error) {
	return s.productRepo.Get(ctx, id)
}

func (s *ProductService) GetList(ctx context.Context, param *domain.Params) ([]*domain.Product, error) {
	return s.productRepo.GetList(ctx, param)
}

func (s *ProductService) Update(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	return s.productRepo.Update(ctx, req)
}

func (s *ProductService) Delete(ctx context.Context, id string) error {
	return s.productRepo.Delete(ctx, id)
}
