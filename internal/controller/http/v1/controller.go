package v1

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type Controller struct {
	Brand    *BrandHandler
	Employee *EmployeeHandler
	Product  *ProductHandler
	Category *CategoryHandler
	Unit     *UnitHandler
	Role     *RoleHandler
	Store    *StoreHandler
}

func NewController(db *gorm.DB, cfg *config.Config, log *logger.Logger) *Controller {
	return &Controller{
		Brand:    NewBrandHandler(cfg, db, log),
		Category: NewCategoryHandler(cfg, db, log),
		Employee: NewEmployeeHandler(cfg, db, log),
	}
}
