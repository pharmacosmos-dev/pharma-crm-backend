package repo

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type customerRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewCustomerRepository(db *sqlx.DB, log *logger.Logger) *customerRepo {
	return &customerRepo{db: db, log: log}
}

func (r *customerRepo) Create(ctx context.Context, req *domain.Customer) (*domain.Customer, error) {
	c := &domain.Customer{}
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		&c.Id, &c.Name).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *customerRepo) Get(ctx context.Context, Id string) (*domain.Customer, error) {
	c := &domain.Customer{}
	return c, nil
}

func (r *customerRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Customer, error) {
	customers := []*domain.Customer{}
	return customers, nil
}

func (r *customerRepo) Update(ctx context.Context, req *domain.Customer) (*domain.Customer, error) {
	c := &domain.Customer{}
	return c, nil
}

func (r *customerRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
