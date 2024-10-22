package repo

import (
	"github.com/pharma-crm-backend/pkg/postgres"
)

type ProductRepo struct {
	*postgres.Postgres
}

func New(pg *postgres.Postgres) *ProductRepo {
	return &ProductRepo{
		pg,
	}
}
