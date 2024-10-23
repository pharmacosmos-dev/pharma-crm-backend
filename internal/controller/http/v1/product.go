package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
)

type ProductHandler struct {
	p storage.ProductRepo
	l *logger.Logger
}

func NewProductRoutes(hander *gin.RouterGroup, p storage.ProductRepo, l *logger.Logger) {
	r := ProductHandler{p, l}
	hander.GET("/hello", func(c *gin.Context) { c.JSON(200, gin.H{"msg": "HELLO!!!"}) })
	hander.POST("", r.Create)
	hander.GET("", r.Get)
	hander.GET("/list", r.List)
	hander.PUT("", r.Update)
	hander.DELETE("", r.Delete)
}

func (h ProductHandler) Create(c *gin.Context) {

}

func (h ProductHandler) Get(c *gin.Context) {

}

func (h ProductHandler) List(c *gin.Context) {

}

func (h ProductHandler) Update(c *gin.Context) {

}

func (h ProductHandler) Delete(c *gin.Context) {

}
