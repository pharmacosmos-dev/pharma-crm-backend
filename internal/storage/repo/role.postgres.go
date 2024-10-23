package repo

import (
	"context"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type RoleRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewRoleRepository(db *sqlx.DB, log *logger.Logger) *RoleRepo {
	return &RoleRepo{db: db, log: log}
}

func (r *RoleRepo) Create(ctx context.Context, req *domain.Role) (*domain.Role, error) {
	c := &domain.Role{}
	Id := uuid.New()
	query := `INSERT INTO roles(id, name, description) VALUES($1, $2, $3) RETURNING id, name, description, created_at, updated_at`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		Id, &req.Name, &req.Desc).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *RoleRepo) Get(ctx context.Context, Id string) (*domain.Role, error) {
	c := &domain.Role{}
	query := `SELECT id, name, description, created_at, updated_at FROM roles WHERE id=$1`
	if err := r.db.QueryRowxContext(
		ctx,
		query, Id).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *RoleRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Role, error) {
	roles := []*domain.Role{}
	query := `SELECT id, name, description, created_at, updated_at FROM roles LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryxContext(ctx, query, &req.Limit, &req.Offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t := domain.Role{}
		err = rows.StructScan(&t)
		if err != nil {
			return nil, err
		}
		roles = append(roles, &t)
	}
	return roles, nil
}

func (r *RoleRepo) Update(ctx context.Context, req *domain.Role) error {
	query := `UPDATE roles SET name=$1, description=$2 WHERE id=$3`
	_, err := r.db.ExecContext(ctx, query, &req.Name, &req.Desc, &req.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, Id string) error {
	query := `DELETE FROM roles WHERE id=$1`
	_, err := r.db.ExecContext(ctx, query, Id)
	if err != nil {
		return err
	}
	return nil
}
