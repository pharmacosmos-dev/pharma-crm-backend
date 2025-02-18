package services

import (
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type Storage struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewStorage(db *gorm.DB, log *logger.Logger) *Storage {
	return &Storage{
		db:  db,
		log: log,
	}
}
