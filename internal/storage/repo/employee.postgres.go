package repo

import (
	"context"
	"fmt"
	"github.com/google/uuid"

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
	e := &domain.Employee{}
	Id := uuid.New()
	query := `
INSERT INTO 
    employees(id, role_id, first_name, last_name, phone, email, password) 
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING 
	id, role_id, first_name, last_name, phone, email, password, created_at, updated_at
`
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		Id,
		&req.RoleId, &req.FirstName, &req.LastName, &req.Phone, &req.Email, &req.Password,
	).StructScan(e); err != nil {
		return nil, err
	}

	return e, nil
}

func (r *employeeRepo) Get(ctx context.Context, id string) (*domain.Employee, error) {
	e := &domain.Employee{}
	query := `SELECT id, role_id, first_name, last_name, phone, email, password, created_at, updated_at FROM employees WHERE id=$1`
	if err := r.db.QueryRowxContext(
		ctx,
		query, id,
	).StructScan(e); err != nil {
		return nil, err
	}
	return e, nil
}

func (r *employeeRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := `SELECT id, role_id, first_name, last_name, phone, email, password, created_at, updated_at FROM employees LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryxContext(ctx, query, &param.Limit, &param.Offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t := domain.Employee{}
		err = rows.StructScan(&t)
		if err != nil {
			return nil, err
		}
		employees = append(employees, &t)
	}
	return employees, nil
}

func (r *employeeRepo) Update(ctx context.Context, req *domain.Employee) (*domain.Employee, error) {
	e := &domain.Employee{}
	query := `UPDATE employees SET role_id=$1, first_name=$2, last_name=$3, phone=$4, email=$5, password=$6 WHERE id=$7
	RETURNING id, role_id, first_name, last_name, phone, email, password, created_at, updated_at`
	if err := r.db.QueryRowxContext(ctx,
		query,
		&req.RoleId,
		&req.FirstName,
		&req.LastName,
		&req.Phone,
		&req.Email,
		&req.Password, &req.Id).StructScan(e); err != nil {
		return nil, err
	}

	return e, nil
}

func (r *employeeRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM employees WHERE id=$1`, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *employeeRepo) CheckField(ctx context.Context, field, value string) (bool, error) {
	query := fmt.Sprintf("SELECT 1 FROM employees WHERE %s=$1", field)
	var temp = 0
	if err := r.db.QueryRowxContext(
		ctx,
		query, value).Scan(&temp); err != nil {
		return false, err
	}
	if temp == 1 {
		return true, nil
	}
	return false, nil
}
