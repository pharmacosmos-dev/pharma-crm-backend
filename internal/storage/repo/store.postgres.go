package repo

import (
	"context"
	"github.com/google/uuid"
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
	Id := uuid.New()
	query := `INSERT INTO stores(id, name, location) VALUES($1, $2, $3) RETURNING id, name, location, created_at, updated_at`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		Id, &req.Name, &req.Location).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *StoreRepo) Get(ctx context.Context, Id string) (*domain.Store, error) {
	c := &domain.Store{}
	query := `SELECT id, name, location, created_at, updated_at FROM stores WHERE id=$1`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		&Id).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *StoreRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Store, error) {
	stores := []*domain.Store{}
	query := `SELECT id, name, location, created_at, updated_at FROM stores LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryxContext(
		ctx,
		query,
		&req.Limit, &req.Offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t := domain.Store{}
		err = rows.StructScan(&t)
		if err != nil {
			return nil, err
		}
		stores = append(stores, &t)
	}
	return stores, nil
}

func (r *StoreRepo) Update(ctx context.Context, req *domain.Store) error {
	query := `UPDATE stores SET name=$1, location=$2, updated_at=NOW() WHERE id=$3`
	_, err := r.db.ExecContext(
		ctx, query, &req.Name, &req.Location, &req.Id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *StoreRepo) Delete(ctx context.Context, Id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM stores WHERE id=$1`, Id)
	if err != nil {
		return err
	}
	return nil
}
