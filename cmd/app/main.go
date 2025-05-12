package main

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/app"
)

func main() {
	// Configuration
	cfg := config.Load()

	// Run
	app.Run(&cfg)
}
