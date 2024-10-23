package repo

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type BrandRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewBrandRepository(db *sqlx.DB, log *logger.Logger) *BrandRepo {
	return &BrandRepo{db: db, log: log}
}

func (r *BrandRepo) Create(ctx context.Context, req *domain.Brand) (*domain.Brand, error) {
	c := &domain.Brand{}
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		&c.Id, &c.Name).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *BrandRepo) Get(ctx context.Context, Id string) (*domain.Brand, error) {
	c := &domain.Brand{}
	return c, nil
}

func (r *BrandRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Brand, error) {
	Brands := []*domain.Brand{}
	return Brands, nil
}

func (r *BrandRepo) Update(ctx context.Context, req *domain.Brand) (*domain.Brand, error) {
	c := &domain.Brand{}
	return c, nil
}

func (r *BrandRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
