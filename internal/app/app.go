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
	"github.com/pharma-crm-backend/pkg/builder"
	"github.com/pharma-crm-backend/pkg/db"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/httpserver"
	"github.com/pharma-crm-backend/pkg/logger"
)

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	// set gin release mode
	gin.SetMode(gin.ReleaseMode)

	// logger
	l := logger.New(cfg.Log.Level)

	// database connection functio
	connDB, err := db.NewConnDB(cfg)
	if err != nil {
		l.Error(err)
	}
	// call to query builder
	builder := builder.NewQueryBuilder()

	// New storage
	storage := services.NewService(connDB, l, cfg, builder)

	// HTTP Server
	handler := gin.New()
	// call to new http router function
	v1.NewRouter(v1.Options{
		Gin:  handler,
		Db:   connDB,
		Log:  l,
		Cfg:  cfg,
		Strg: storage,
	})
	// // i18n localization
	// err = helper.InitI18n()
	// if err != nil {
	// 	l.Error(err)
	// }
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
