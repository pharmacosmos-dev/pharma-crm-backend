package repo

import (
	"context"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CategoryRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewCategoryRepository(db *sqlx.DB, log *logger.Logger) *CategoryRepo {
	return &CategoryRepo{db: db, log: log}
}

func (r *CategoryRepo) Create(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	c := &domain.Category{}
	Id := uuid.New()
	query := `INSERT INTO categories(id, name) VALUES($1, $2) RETURNING id, name, created_at, updated_at`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		Id, &req.Name).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CategoryRepo) Get(ctx context.Context, Id string) (*domain.Category, error) {
	c := &domain.Category{}
	query := `SELECT id, name, created_at, updated_at FROM categories WHERE id=$1`
	if err := r.db.QueryRowxContext(
		ctx,
		query, Id).StructScan(c); err != nil {
		return nil, err
	}

	return c, nil
}

func (r *CategoryRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Category, error) {
	categorys := []*domain.Category{}
	query := `SELECT id, name, created_at, updated_at FROM categories LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryxContext(ctx, query, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t := &domain.Category{}
		if err = rows.StructScan(t); err != nil {
			return nil, err
		}
		categorys = append(categorys, t)
	}
	return categorys, nil
}

func (r *CategoryRepo) Update(ctx context.Context, req *domain.Category) (*domain.Category, error) {
	c := &domain.Category{}
	query := `UPDATE categories SET name=$1 WHERE id=$2 RETURNING id, name, created_at, updated_at`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		&req.Name, &req.Id).StructScan(c); err != nil {
		return nil, err
	}

	return c, nil
}

func (r *CategoryRepo) Delete(ctx context.Context, Id string) error {
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM categories WHERE id=$1`,
		Id); err != nil {
		return err
	}
	return nil
}
