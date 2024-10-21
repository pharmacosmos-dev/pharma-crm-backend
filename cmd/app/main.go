package main

import (
	"log"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
