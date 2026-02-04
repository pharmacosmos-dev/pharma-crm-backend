package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain/constants"
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

	// check product mxiks
	go s.changeByMxik()

	// update import totals arguments
	go s.updateImportTotalsLoop()

	// send remaining quantity to OsonApteka
	go s.performSendOsonApteka()

	// update cart items
	go s.performUpdateCartItems()

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

func (s *Services) updateImportTotalsLoop() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)

		s.performImportTotals(ctx)

		cancel()

		time.Sleep(time.Minute * 2)
	}
}

func (s *Services) performUpdateCartItems() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.log.Info("Starting cart items update job")
		if err := s.updateCartItems(); err != nil {
			s.log.Error("Cart items update failed", "error", err)
		} else {
			s.log.Info("Finished cart items update job")
		}
	}
}

func (s *Services) updateCartItems() error {
	query := `
	WITH batch AS (
		SELECT ci.ctid, ci.store_product_id
		FROM cart_items ci
		WHERE ci.product_id IS NULL
		LIMIT 50000
	)
	UPDATE cart_items ci
	SET product_id = sp.product_id
	FROM batch b
	JOIN store_products sp ON sp.id = b.store_product_id
	WHERE ci.ctid = b.ctid;
	`

	if err := s.db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to update cart items: %w", err)
	}

	return nil
}
