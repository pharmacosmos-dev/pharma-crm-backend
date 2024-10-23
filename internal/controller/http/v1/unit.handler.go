package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
)

type UnitHandler struct {
	c *services.UnitService
	l logger.Interface
}

func NewUnitHandler(handler *gin.RouterGroup, c *services.UnitService, l logger.Interface) {
	r := &UnitHandler{c, l}
	handler.POST("", r.Create)
	handler.GET("/:id", r.Get)
	handler.GET("", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("/:id", r.Delete)
}

func (h *UnitHandler) Create(c *gin.Context) {}

func (h *UnitHandler) Get(c *gin.Context) {}

func (h *UnitHandler) List(c *gin.Context) {}

func (h *UnitHandler) Update(c *gin.Context) {}

func (h *UnitHandler) Delete(c *gin.Context) {}
