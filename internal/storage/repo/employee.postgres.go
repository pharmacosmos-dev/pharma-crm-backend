package repo

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type employeeRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewEmployeeRepository(db *sqlx.DB, log *logger.Logger) *employeeRepo {
	return &employeeRepo{db: db, log: log}
}

func (r *employeeRepo) Create(ctx context.Context, req *domain.Employee) (*domain.Employee, error) {
	return nil, nil
}

func (r *employeeRepo) Get(ctx context.Context, id string) (*domain.Employee, error) {
	return nil, nil
}

func (r *employeeRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Employee, error) {
	return nil, nil
}

func (r *employeeRepo) Update(ctx context.Context, req *domain.Employee) (*domain.Employee, error) {
	return nil, nil
}

func (r *employeeRepo) Delete(ctx context.Context, id string) error {
	return nil
}
