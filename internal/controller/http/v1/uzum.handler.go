package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type UzumHandler struct {
	*Handler
}

func (h *Handler) NewUzumHandler(r *gin.RouterGroup) {
	uzum := &UzumHandler{h}
	uzum.UzumRoutes(r)
}

func (h *UzumHandler) UzumRoutes(r *gin.RouterGroup) {
	r.GET("/v1/nomenclature/:storeId/composition", h.GetNomenclature)
	r.GET("/v1/nomenclature/:storeId/availability", h.GetAvailability)
	r.POST("/v1/order", h.CreateOrder)
	r.GET("/v1/order/:orderId", h.GetOrder)
	r.GET("/v1/order/:orderId/status", h.GetOrderStatus)
	r.PUT("/v1/order/:orderId", h.UpdateOrder)
	r.DELETE("/v1/order/:orderId", h.CancelOrder)
	r.GET("/v1/restaurants", h.GetRestaurants)

}

// @Summary      Get Nomenclature Composition
// @Description  Returns the current product catalog with categories for a specific store
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        storeId path string true "Store ID (UUID)"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} domain.NomenclatureResponse
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/nomenclature/{storeId}/composition [get]
func (h *UzumHandler) GetNomenclature(c *gin.Context) {
	storeId := c.Param("storeId")

	if storeId == "" {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: "storeId is required"},
		})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetNomenclature(ctx, storeId, page, limit)
	if err != nil {
		h.log.Errorf("failed to get nomenclature: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	if result == nil || len(result.Items) == 0 {
		c.JSON(http.StatusNotFound, domain.UzumErrorList{
			{Code: 404, Description: "No products found for this store"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Get Product Availability
// @Description  Returns product stock levels for a specific store
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        storeId path string true "Store ID (UUID)"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} domain.AvailabilityResponse
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/nomenclature/{storeId}/availability [get]
func (h *UzumHandler) GetAvailability(c *gin.Context) {
	storeId := c.Param("storeId")

	if storeId == "" {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: "storeId is required"},
		})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetAvailability(ctx, storeId, page, limit)
	if err != nil {
		h.log.Errorf("failed to get availability: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Create Order
// @Description  Creates a new order from Uzum Tezkor
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.UzumCreateOrderRequest true "Order Request body"
// @Success      200 {object} domain.UzumCreateOrderResponse
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      409 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/order [post]
func (h *UzumHandler) CreateOrder(c *gin.Context) {
	var body domain.UzumCreateOrderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind uzum order create request body: %v", err)
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: err.Error()},
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateUzumOrder(ctx, &body)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			c.JSON(http.StatusBadRequest, domain.UzumErrorList{
				{Code: 400, Description: notAddErr.Data.(string)},
			})
			return
		}
		h.log.Errorf("failed to create uzum order: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Get Order
// @Description  Returns order details by ID
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        orderId path string true "Order ID (Sale UUID)"
// @Success      200 {object} domain.UzumGetOrderResponse
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/order/{orderId} [get]
func (h *UzumHandler) GetOrder(c *gin.Context) {
	orderId := c.Param("orderId")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetUzumOrder(ctx, orderId)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.UzumErrorList{
			{Code: 404, Description: "Order not found"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Get Order Status
// @Description  Returns the current status of an order
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        orderId path string true "Order ID (Sale UUID)"
// @Success      200 {object} domain.UzumOrderStatusResponse
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/order/{orderId}/status [get]
func (h *UzumHandler) GetOrderStatus(c *gin.Context) {
	orderId := c.Param("orderId")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetUzumOrderStatus(ctx, orderId)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.UzumErrorList{
			{Code: 404, Description: "Order not found"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Update Order
// @Description  Updates an existing order
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        orderId path string true "Order ID (Sale UUID)"
// @Param        body body domain.UzumCreateOrderRequest true "Updated Order body"
// @Success      200 {object} map[string]string
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/order/{orderId} [put]
func (h *UzumHandler) UpdateOrder(c *gin.Context) {
	orderId := c.Param("orderId")

	var body domain.UzumCreateOrderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: err.Error()},
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.UpdateUzumOrder(ctx, orderId, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

// @Summary      Cancel Order
// @Description  Cancels an existing order
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        orderId path string true "Order ID (Sale UUID)"
// @Param        body body domain.UzumCancelOrderRequest true "Cancel Order body"
// @Success      200 {object} map[string]string
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/order/{orderId} [delete]
func (h *UzumHandler) CancelOrder(c *gin.Context) {
	orderId := c.Param("orderId")

	var body domain.UzumCancelOrderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: err.Error()},
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.CancelUzumOrder(ctx, orderId, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

// @Summary      Get Restaurants
// @Description  Returns the list of restaurants
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Limit"
// @Param        page query int false "Page"
// @Success      200 {object} []domain.Restaurant
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /uzum/v1/restaurants [get]
func (h *UzumHandler) GetRestaurants(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	result, err := h.service.GetRestaurants(ctx, limit, page)
	if err != nil {
		h.log.Errorf("failed to get restaurants: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
