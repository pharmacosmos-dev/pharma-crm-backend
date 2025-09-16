package services

import (
	"time"

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
	s := &Services{
		db:  db,
		log: log,
		cfg: cfg,
		hub: hub,
	}
	go s.changeByMxik()
	return s
}

func (s *Services) changeByMxik() {
	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.log.Info("Starting MXIK unit_code sync job")

		if err := s.syncUnitCodes(); err != nil {
			s.log.Error("MXIK sync failed", "error", err)
		} else {
			s.log.Info("Finished MXIK unit_code sync job")
		}
	}
}
