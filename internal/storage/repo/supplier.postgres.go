package repo

import (
	"context"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type SupplierRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewSupplierRepository(db *sqlx.DB, log *logger.Logger) *SupplierRepo {
	return &SupplierRepo{db: db, log: log}
}

func (r *SupplierRepo) Create(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error) {
	c := &domain.Supplier{}
	Id := uuid.New()
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		Id, &c.FirstName).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *SupplierRepo) Get(ctx context.Context, Id string) (*domain.Supplier, error) {
	c := &domain.Supplier{}
	return c, nil
}

func (r *SupplierRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Supplier, error) {
	Suppliers := []*domain.Supplier{}
	return Suppliers, nil
}

func (r *SupplierRepo) Update(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error) {
	c := &domain.Supplier{}
	return c, nil
}

func (r *SupplierRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
