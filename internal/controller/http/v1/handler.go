package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	"gorm.io/gorm"
)

type Handler struct {
	Log        *logger.Logger
	Db         *gorm.DB
	Cfg        *config.Config
	JwtHandler *token.JWTHandler
}

func NewHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger, jwt *token.JWTHandler) *Handler {
	return &Handler{
		Cfg:        cfg,
		Db:         db,
		Log:        log,
		JwtHandler: jwt,
	}
}

func (h *Handler) InitRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	{
		h.NewBrandController(v1)
		h.NewCategoryController(v1)
		h.NewCustomerHandler(v1)
		h.NewProductHandler(v1)
		h.NewStoreHandler(v1)
		h.NewRoleHandler(v1)
		h.NewUnitHandler(v1)
		h.NewEmployeeHandler(v1)
	}
}
