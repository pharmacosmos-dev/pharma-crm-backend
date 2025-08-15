package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type RejectedProductsHandler struct {
	*Handler
}

func (h *Handler) NewRejectedProductsHandler(r *gin.RouterGroup) {
	rejectedProducts := &RejectedProductsHandler{h}
	rejectedProducts.RejectedProductsRoutes(r)
}

func (h *RejectedProductsHandler) RejectedProductsRoutes(r *gin.RouterGroup) {
	rejectedProducts := r.Group("/rejected-products")
	{
		rejectedProducts.POST("", h.Create)
		//rejectedProducts.GET("/:id", h.Get)
		//rejectedProducts.GET("/list", h.List)
		//rejectedProducts.PUT("/:id", h.Update)
		//rejectedProducts.POST("/excel-import", h.ImportRejectedProducts)
		//rejectedProducts.DELETE("", h.Delete)
	}
}

// godoc Create
// @Summary Create a rejected product
// @Description Create a rejected product
// @Tags rejected-products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.RejectedProductRequest true "Rejected product request"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /rejected-products [post]
func (h *RejectedProductsHandler) Create(c *gin.Context) {
	var body domain.RejectedProductRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}
	// get creator id from set header
	*body.CreatedBy = userId.(string)
	if body.StoreID == "" {
		handleResponse(c, BadRequest, "store_id required")
		return
	}

	if err := h.service.CreateOrUpdateRejectedProduct(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "CREATED")
}
