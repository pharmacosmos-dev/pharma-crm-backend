package repo

import (
	"context"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type UnitRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewUnitRepository(db *sqlx.DB, log *logger.Logger) *UnitRepo {
	return &UnitRepo{db: db, log: log}
}

func (r *UnitRepo) Create(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	c := &domain.Unit{}
	Id := uuid.New()
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		&Id, &c.Unit).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *UnitRepo) Get(ctx context.Context, Id string) (*domain.Unit, error) {
	c := &domain.Unit{}
	return c, nil
}

func (r *UnitRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Unit, error) {
	Units := []*domain.Unit{}
	return Units, nil
}

func (r *UnitRepo) Update(ctx context.Context, req *domain.Unit) (*domain.Unit, error) {
	c := &domain.Unit{}
	return c, nil
}

func (r *UnitRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
