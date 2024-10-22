package customer

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CustomerHandler struct {
	c storage.CustomerRepo
	l logger.Interface
}

func NewCustomerHandler(handler *gin.RouterGroup, c storage.CustomerRepo, l logger.Interface) {
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
