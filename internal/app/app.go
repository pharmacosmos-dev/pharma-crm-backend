// Package app configures and runs application.
package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	v1 "github.com/pharma-crm-backend/internal/controller/http"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/db"
	"github.com/pharma-crm-backend/pkg/helper"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/httpserver"
	"github.com/pharma-crm-backend/pkg/logger"
)

// Run creates objects via constructors.
func Run(cfg *config.Config) {

	gin.SetMode(gin.ReleaseMode)

	l := logger.New(cfg.Log.Level)

	connDB, err := db.NewConnDB(cfg)
	if err != nil {
		l.Error(err)
	}
	// New storage
	storage := services.NewService(connDB, l, cfg)

	// HTTP Server
	handler := gin.New()

	v1.NewRouter(v1.Options{
		Gin:  handler,
		Db:   connDB,
		Log:  l,
		Cfg:  cfg,
		Strg: storage,
	})
	// i18n localization
	err = helper.InitI18n()
	if err != nil {
		l.Error(err)
	}
	// call to http server
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))

	// Start http server
	fmt.Println("Server is running on port:", cfg.HTTP.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: %s", s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}
}
