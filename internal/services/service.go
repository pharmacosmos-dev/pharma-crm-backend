package services

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/controller/ws"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type Services struct {
	db  *gorm.DB
	log *logger.Logger
	cfg *config.Config
	hub *ws.Hub
}

func NewService(db *gorm.DB, log *logger.Logger, cfg *config.Config, hub *ws.Hub) *Services {
	return &Services{
		db:  db,
		log: log,
		cfg: cfg,
		hub: hub,
	}
}
