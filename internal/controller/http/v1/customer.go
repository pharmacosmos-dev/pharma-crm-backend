package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CustomerHandler struct {
	c *services.CustomerService
	l logger.Interface
}

func NewCustomerHandler(handler *gin.RouterGroup, c *services.CustomerService, l logger.Interface) {
	r := &CustomerHandler{c, l}
	handler.POST("", r.Create)
	handler.GET("/:id", r.Get)
	handler.GET("", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("/:id", r.Delete)
}

func (h *CustomerHandler) Create(c *gin.Context) {}

func (h *CustomerHandler) Get(c *gin.Context) {}

func (h *CustomerHandler) List(c *gin.Context) {}

func (h *CustomerHandler) Update(c *gin.Context) {}

func (h *CustomerHandler) Delete(c *gin.Context) {}
