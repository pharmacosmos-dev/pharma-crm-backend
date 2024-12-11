package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type Product1cHandler struct {
	*Handler
}

func (h *Handler) NewProduct1cHandler(r *gin.RouterGroup) {
	product1c := &Product1cHandler{h}
	product1c.Product1cRoutes(r)
}

func (h *Product1cHandler) Product1cRoutes(r *gin.RouterGroup) {
	product1c := r.Group("/product1c")
	{
		product1c.POST("", h.Create)
		// product1c.GET("/:id", h.Get)
		// 	product1c.GET("/list", h.List)
		// 	product1c.PUT("/:id", h.Update)
		// 	product1c.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a product
// @Description Create a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param product body []domain.ProductRequest1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c [post]
func (h *Product1cHandler) Create(c *gin.Context) {
	var (
		body []domain.ProductRequest1C
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Table("products").Create(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
