package services

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/builder"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type Services struct {
	db      *gorm.DB
	log     *logger.Logger
	cfg     *config.Config
	builder *builder.QueryBuilder
}

func NewService(db *gorm.DB, log *logger.Logger, cfg *config.Config, builder *builder.QueryBuilder) *Services {
	return &Services{
		db:      db,
		log:     log,
		cfg:     cfg,
		builder: builder,
	}
}
