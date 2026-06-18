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
	"github.com/pharma-crm-backend/internal/controller/ws"
	"github.com/pharma-crm-backend/internal/services"
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
	l := logger.New(cfg.App.Level)

	// database connection functio
	connDB, err := db.NewConnDB(cfg)
	if err != nil {
		l.Error(err)
	}

	// 🧠 WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// New storage
	service := services.NewService(connDB, l, cfg, hub)

	// HTTP Server
	handler := gin.New()
	// call to new http router function
	v1.NewRouter(v1.Options{
		Gin:     handler,
		Db:      connDB,
		Log:     l,
		Cfg:     cfg,
		Service: service,
	}, hub)

	// call to http server
	httpServer := httpserver.New(handler, httpserver.Port(cfg.App.Port))

	// Start http server
	fmt.Println("Server is running on port:", cfg.App.Port)

	c, err := RegisterCronJobs(service)
	if err != nil {
		l.Error(err)
	}
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

func RegisterCronJobs(service *services.Services) (*cron.Cron, error) {

	c := cron.New(
		cron.WithLocation(time.UTC), // important: sets cron to UTC
	)
	c.AddFunc("00 23 * * *", func() {
		log.Println("Starting send expense to 1C...")
		service.SendReportsSequentially()
	})
	// c.AddFunc("0 * * * *", func() {
	// 	log.Println("Staring checking customers' loyalty leveling up...")
	// 	service.LoyaltyCardLevelingUp()
	// })
	c.AddFunc("59 23 * * *", func() {
		//service.DistributeMonthlyTargets()
		log.Println("Starting update store target sales...")
		service.UpdateStoreTargetSales()
		log.Println("Starting update employee target sales...")
		service.UpdateEmployeeTargetSales()
		log.Println("Starting update average target sales for stores...")
		service.UpdateAverateStoreTargetSales()
	})
	
	// Set store and employee goals for the new month at 00:00 (UTC) on the 1st of each month
	c.AddFunc("0 0 1 * *", func() {
		log.Println("Starting auto create monthly store targets...")
		service.AutoCreateMonthlyStoreTargets()
	})

	return c, nil
}
