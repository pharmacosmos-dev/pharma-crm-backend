package main

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/app"
)

func main() {
	// Configuration
	cfg := config.Load()

	// Add http server for profiling
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// Run
	app.Run(&cfg)
}
