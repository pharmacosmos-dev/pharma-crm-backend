package repo

import (
	"context"

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
	if err := r.db.QueryRowxContext(
		ctx,
		"",
		&c.Id, &c.Name).StructScan(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *RoleRepo) Get(ctx context.Context, Id string) (*domain.Role, error) {
	c := &domain.Role{}
	return c, nil
}

func (r *RoleRepo) GetList(ctx context.Context, req *domain.Params) ([]*domain.Role, error) {
	Roles := []*domain.Role{}
	return Roles, nil
}

func (r *RoleRepo) Update(ctx context.Context, req *domain.Role) (*domain.Role, error) {
	c := &domain.Role{}
	return c, nil
}

func (r *RoleRepo) Delete(ctx context.Context, Id string) error {
	return nil
}
