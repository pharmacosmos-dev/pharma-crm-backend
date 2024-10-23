package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
)

type BrandHandler struct {
	c *services.BrandService
	l logger.Interface
}

func NewBrandHandler(handler *gin.RouterGroup, c *services.BrandService, l logger.Interface) {
	r := &BrandHandler{c, l}
	handler.POST("", r.Create)
	handler.GET("/:id", r.Get)
	handler.GET("", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("/:id", r.Delete)
}

func (h *BrandHandler) Create(c *gin.Context) {}

func (h *BrandHandler) Get(c *gin.Context) {}

func (h *BrandHandler) List(c *gin.Context) {}

func (h *BrandHandler) Update(c *gin.Context) {}

func (h *BrandHandler) Delete(c *gin.Context) {}
