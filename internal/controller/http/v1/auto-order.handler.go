package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type AutoOrderHandler struct {
	*Handler
}

func (h *Handler) NewAutoOrderHandler(r *gin.RouterGroup) {
	autoOrder := &AutoOrderHandler{h}
	autoOrder.AutoOrderRoutes(r)
}

func (h *AutoOrderHandler) AutoOrderRoutes(r *gin.RouterGroup) {
	autoOrder := r.Group("/auto-order")
	{
		autoOrder.POST("/confirm", h.Confirm)
		// autoOrder.GET("/:id", h.Get)
		autoOrder.GET("/list", h.List)
		// autoOrder.PUT("/:id", h.Update)
		// autoOrder.DELETE("/:id", h.Delete)
	}
}

// ConfirmAutoOrder godoc
// @Summary Confirm auto order
// @Description Confirm auto order
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body 	[]domain.AutoOrderConfirm true "Auto order information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/confirm [post]
func (h *AutoOrderHandler) Confirm(c *gin.Context) {
	var (
		body       []domain.AutoOrderConfirm
		autoOrders []domain.AutoOrder
		err        error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	autoOrders, _, err = h.storage.ListAutoOrder(c.Request.Context(), 1000, 0)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, autoOrders)
}

// ListAutoOrder godoc
// @Summary List auto orders
// @Description List auto orders
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/list [get]
func (h *AutoOrderHandler) List(c *gin.Context) {
	var (
		autoOrders []domain.AutoOrder
		err        error
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	autoOrders, totalCount, err = h.storage.ListAutoOrder(c.Request.Context(), limit, offset)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
	}
	result := utils.ListResponse(autoOrders, totalCount, limit, offset)
	handleResponse(c, OK, result)
}
