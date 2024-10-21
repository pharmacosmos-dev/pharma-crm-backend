package usecase

import "github.com/pharma-crm-backend/domain"

type (
	// ProductRepo
	ProductRepo interface {
		Create(req *domain.Product) (*domain.Product, error)
		Get(Id string) (*domain.Product, error)
		GetList(params *domain.Params) (*domain.Product, error)
		Update(req *domain.Product) (*domain.Product, error)
		Delete(Id string) error
	}

	Product interface {
	}
)
