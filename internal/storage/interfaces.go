package storage

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

type (

	// Store Repo
	StoreRepo interface {
		Create(ctx context.Context, req *domain.Store) (*domain.Store, error)
		Get(ctx context.Context, Id string) (*domain.Store, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Store, error)
		Update(ctx context.Context, req *domain.Store) (*domain.Store, error)
		Delete(ctx context.Context, Id string) error
	}

	// Brand Repo
	BrandRepo interface {
		Create(ctx context.Context, req *domain.Brand) (*domain.Brand, error)
		Get(ctx context.Context, Id string) (*domain.Brand, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Brand, error)
		Update(ctx context.Context, req *domain.Brand) (*domain.Brand, error)
		Delete(ctx context.Context, Id string) error
	}

	// Role Repo
	RoleRepo interface {
		Create(ctx context.Context, req *domain.Role) (*domain.Role, error)
		Get(ctx context.Context, Id string) (*domain.Role, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Role, error)
		Update(ctx context.Context, req *domain.Role) (*domain.Role, error)
		Delete(ctx context.Context, Id string) error
	}

	// Units Repo
	UnitRepo interface {
		Create(ctx context.Context, req *domain.Unit) (*domain.Unit, error)
		Get(ctx context.Context, Id string) (*domain.Unit, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Unit, error)
		Update(ctx context.Context, req *domain.Unit) (*domain.Unit, error)
		Delete(ctx context.Context, Id string) error
	}

	// Supplier Repo
	SupplierRepo interface {
		Create(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error)
		Get(ctx context.Context, Id string) (*domain.Supplier, error)
		GetList(ctx context.Context, params *domain.Params) ([]*domain.Supplier, error)
		Update(ctx context.Context, req *domain.Supplier) (*domain.Supplier, error)
		Delete(ctx context.Context, Id string) error
	}

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
