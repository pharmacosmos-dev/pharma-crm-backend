package repo

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type productRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewProductRepository(db *sqlx.DB, log *logger.Logger) *productRepo {
	return &productRepo{db: db, log: log}
}

func (r *productRepo) Create(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	p := &domain.Product{}
	return p, nil
}

func (r *productRepo) Get(ctx context.Context, Id string) (*domain.Product, error) {
	p := &domain.Product{}
	return p, nil
}

func (r *productRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Product, error) {
	return nil, nil
}

func (r *productRepo) Update(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	return nil, nil
}

func (r *productRepo) Delete(ctx context.Context, id string) error {
	return nil
}
