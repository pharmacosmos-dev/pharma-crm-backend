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

	// 00 05 UTC = 10:00 Toshkent, har oyning 1-kuni — tugagan o'tgan oyning yakuniy
	// oylik hisobini employee_payrolls'ga snapshot qiladi.
	// Ataylab kech: o'tgan oyning oxirgi kuni employee_attendance_days'ga
	// AggregateEmployeeAttendanceDays (30 19 UTC = 00:30 Toshkent) orqali tushadi,
	// shuning uchun snapshot undan keyin olinishi shart.
	// c.AddFunc("00 05 1 * *", func() {
	// 	log.Println("Starting auto create monthly employee payrolls...")
	// 	service.AutoCreateMonthlyPayrolls()
	// })

	// 58 18 UTC = 23:58 Tashkent — bugungi kun tugashidan oldin check-out qilinmagan
	// check-in'larni avtomatik yopadi (event_at=bugun 23:59:59, is_auto_closed=true)
	c.AddFunc("59 18 * * *", func() {
		log.Println("Starting auto-close unclosed attendance logs...")
		service.AutoCloseUnclosedAttendanceLogs()
	})

	// 30 19 UTC = 00:30 Tashkent — kechagi Toshkent kuni to'liq yakunlangach ishlaydi
	// (kechagi kunning check-out'lari yuqoridagi auto-close orqali bir kun oldin yopilgan bo'ladi)
	c.AddFunc("30 19 * * *", func() { //30 19 * * *
		log.Println("Starting aggregate employee attendance days...")
		service.AggregateEmployeeAttendanceDays()
	})

	// 00 20 UTC = 01:00 Toshkent — kechagi auto-close'larni o'sha kungi oxirgi check-in
	// vaqtiga qarab tuzatadi va employee_attendance_days'ni qayta hisoblaydi.
	// TEST: vaqtincha har 5 daqiqada ishlaydi.
	// !!! PRODGA CHIQARISHDAN OLDIN "00 20 * * *" ga QAYTARILSIN !!!
	c.AddFunc("00 20 * * *", func() {
		log.Println("Update employee attendance days...")
		service.UpdateEmployeeAttendanceDays()
	})

	return c, nil
}
