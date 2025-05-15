// Package app configures and runs application.
package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/pharma-crm-backend/internal/controller/http"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/builder"
	"github.com/pharma-crm-backend/pkg/db"
	"github.com/robfig/cron/v3"

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
	service := services.NewService(connDB, l, cfg, builder)

	// HTTP Server
	handler := gin.New()
	// call to new http router function
	v1.NewRouter(v1.Options{
		Gin:     handler,
		Db:      connDB,
		Log:     l,
		Cfg:     cfg,
		Service: service,
	})

	// call to http server
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))

	// ✅ Automatically run backlog report before starting HTTP server
	// start, _ := time.Parse("2006-01-02", "2025-05-14")
	// end, _ := time.Parse("2006-01-02", "2025-05-14")
	// fmt.Println("Starting backlog report processing...")
	// service.SendBacklogReportsSequentially(start, end)
	// fmt.Println("Backlog report processing completed.")

	// Start http server
	fmt.Println("Server is running on port:", cfg.HTTP.Port)
	// load location
	location, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		log.Printf("Failed to load location Asia/Tashkent: %v. Using UTC instead.", err)
		location = time.UTC
	}

	// add cronjob runner with load location
	c := cron.New(cron.WithLocation(location))
	// The time is set to 23:45 in UTC -> .
	c.AddFunc("45 23 * * *", func() {
		log.Println("Starting send expense to 1C...")
		service.SendReportsSequentially()
	})

	c.Start()
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
