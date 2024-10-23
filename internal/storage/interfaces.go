package storage

import (
	"context"
	"github.com/pharma-crm-backend/domain"
)

type (
	// Product Repo
	ProductRepo interface {
		Create(ctx context.Context, req *domain.Product) (*domain.Product, error)
		Get(ctx context.Context, Id string) (*domain.Product, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Product, error)
		Update(ctx context.Context, req *domain.Product) (*domain.Product, error)
		Delete(ctx context.Context, Id string) error
	}

	// Customer Repo
	CustomerRepo interface {
		Create(ctx context.Context, req *domain.Customer) (*domain.Customer, error)
		Get(ctx context.Context, Id string) (*domain.Customer, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Customer, error)
		Update(ctx context.Context, req *domain.Customer) (*domain.Customer, error)
		Delete(ctx context.Context, Id string) error
	}

	// Employee Repo
	EmployeeRepo interface {
		Create(ctx context.Context, req *domain.Employee) (*domain.Employee, error)
		Get(ctx context.Context, Id string) (*domain.Employee, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Employee, error)
		Update(ctx context.Context, req *domain.Employee) (*domain.Employee, error)
		Delete(ctx context.Context, Id string) error
	}
)
