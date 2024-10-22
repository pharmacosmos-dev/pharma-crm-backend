package employee

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type EmployeeHandler struct {
	c storage.EmployeeRepo
	l logger.Interface
}

func NewEmployeeHandler(handler *gin.RouterGroup, c storage.EmployeeRepo, l logger.Interface) {
	r := &EmployeeHandler{c, l}

	handler.POST("", r.Create)
	handler.GET("/:id", r.Get)
	handler.GET("", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("/:id", r.Delete)
}

func (h *EmployeeHandler) Create(c *gin.Context) {}

func (h *EmployeeHandler) Get(c *gin.Context) {}

func (h *EmployeeHandler) List(c *gin.Context) {}

func (h *EmployeeHandler) Update(c *gin.Context) {}

func (h *EmployeeHandler) Delete(c *gin.Context) {}
