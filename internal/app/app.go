// Package app configures and runs application.
package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	v1 "github.com/pharma-crm-backend/internal/controller/http"
	"github.com/pharma-crm-backend/pkg/db"

	"github.com/gin-gonic/gin"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/httpserver"
	"github.com/pharma-crm-backend/pkg/logger"
)

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	l := logger.New(cfg.Log.Level)

	// Postgres connect
	// pgConn, err := db.NewPsqlDB(cfg)
	// if err != nil {
	// 	l.Error(err)
	// }

	connDB, err := db.NewConnDB(cfg)
	if err != nil {
		l.Error(err)
	}

	// HTTP Server
	handler := gin.New()
	v1.NewRouter(handler, connDB, l, cfg)
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

}
