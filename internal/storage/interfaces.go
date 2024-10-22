package storage

import "github.com/pharma-crm-backend/domain"

type (
	// Product Repo
	ProductRepo interface {
		Create(req *domain.Product) (*domain.Product, error)
		Get(Id string) (*domain.Product, error)
		GetList(params *domain.Params) (*domain.Product, error)
		Update(req *domain.Product) (*domain.Product, error)
		Delete(Id string) error
	}

	// Customer Repo
	CustomerRepo interface {
		Create(req *domain.Customer) (*domain.Customer, error)
		Get(Id string) (*domain.Customer, error)
		GetList(params *domain.Params) ([]*domain.Customer, error)
		Update(req *domain.Customer) (*domain.Customer, error)
		Delete(Id string) error
	}

	// Employee Repo
	EmployeeRepo interface {
		Create(req *domain.Employee) (*domain.Employee, error)
		Get(Id string) (*domain.Employee, error)
		GetList(params *domain.Params) ([]*domain.Employee, error)
		Update(req *domain.Employee) (*domain.Employee, error)
		Delete(Id string) error
	}
)
