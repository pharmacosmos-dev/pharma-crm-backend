package repo

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type StoreRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewStoreRepository(db *sqlx.DB, log *logger.Logger) *StoreRepo {
	return &StoreRepo{db: db, log: log}
}

func (r *StoreRepo) Create(ctx context.Context, req *domain.Store) (*domain.Store, error) {
	c := &domain.Store{}
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		&c.Id, &c.Name).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *StoreRepo) Get(ctx context.Context, Id string) (*domain.Store, error) {
	c := &domain.Store{}
	return c, nil
}

func (r *StoreRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Store, error) {
	Stores := []*domain.Store{}
	return Stores, nil
}

func (r *StoreRepo) Update(ctx context.Context, req *domain.Store) (*domain.Store, error) {
	c := &domain.Store{}
	return c, nil
}

func (r *StoreRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
