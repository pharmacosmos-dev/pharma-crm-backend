package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type OnlineRepricingHandler struct {
	*Handler
}

func (h *Handler) NewOnlineRepricingHandler(r *gin.RouterGroup) {
	orh := &OnlineRepricingHandler{h}
	orh.OnlineRepricingRoutes(r)
}



func (h *OnlineRepricingHandler) OnlineRepricingRoutes(r *gin.RouterGroup) {
	onlineRepricing := r.Group("/online-repricing")
	{
		onlineRepricing.POST("", h.Create)
		onlineRepricing.GET("/list", h.List)
		onlineRepricing.POST("/confirm/:id", h.Confirm)
		onlineRepricing.POST("/cancel/:id", h.Cancel)
	}
	detail := r.Group("/repricing-detail")
	{
		detail.GET("/:id/details", h.DetailList)
		detail.PUT("/:detail_id", h.UpdateDetailPrice)
		
	}
}


// Create godoc
// @Summary		Create online repricing session
// @Tags		Online Repricing
// @Accept		json
// @Produce		json
// @Param		body body domain.OnlineRepricingRequest true "Request body"
// @Success		201 {object} v1.Response
// @Router		/v1/online-repricing [post]
func (h *OnlineRepricingHandler) Create(c *gin.Context) {
	var req domain.OnlineRepricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if req.StoreId == "" || req.PlatformType == "" {
		handleResponse(c, BadRequest, "store_id and platform_type are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateOnlineRepricing(ctx, &req)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// List godoc
// @Summary		List online repricing sessions
// @Tags		Online Repricing
// @Produce		json
// @Param		store_id      query string false "Store ID"
// @Param		platform_type query string false "Platform type"
// @Param		status        query string false "Status"
// @Param		limit         query int    false "Limit"
// @Param		offset        query int    false "Offset"
// @Success		200 {object} v1.Response
// @Router		/v1/online-repricing [get]
func (h *OnlineRepricingHandler) List(c *gin.Context) {
	var params domain.OnlineRepricingQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, total, err := h.service.GetOnlineRepricingList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res, total)
}

// DetailList godoc
// @Summary		List product details of a repricing session
// @Tags		Online Repricing
// @Produce		json
// @Param		id     path  int    true  "Repricing ID"
// @Param		search query string false "Search"
// @Param		limit  query int    false "Limit"
// @Param		offset query int    false "Offset"
// @Success		200 {object} v1.Response
// @Router		/v1/online-repricing/{id}/details [get]
func (h *OnlineRepricingHandler) DetailList(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handleResponse(c, BadRequest, "invalid id")
		return
	}

	var params domain.OnlineRepricingQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, total, err := h.service.GetOnlineRepricingDetailList(ctx, id, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res, total)
}

// UpdateDetailPrice godoc
// @Summary		Update new_retail_price for a detail row
// @Tags		Online Repricing
// @Accept		json
// @Produce		json
// @Param		detail_id path string true "Detail UUID"
// @Param		body body domain.UpdateOnlineDetailPrice true "New price"
// @Success		200 {object} v1.Response
// @Router		/v1/online-repricing/detail/{detail_id} [put]
func (h *OnlineRepricingHandler) UpdateDetailPrice(c *gin.Context) {
	var req domain.UpdateOnlineDetailPrice
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	req.Id = c.Param("detail_id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.UpdateOnlineRepricingDetailPrice(ctx, &req); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "updated")
}

// Confirm godoc
// @Summary		Confirm online repricing — applies prices to online_store_products
// @Tags		Online Repricing
// @Produce		json
// @Param		id path int true "Repricing ID"
// @Success		200 {object} v1.Response
// @Router		/v1/online-repricing/{id}/confirm [post]
func (h *OnlineRepricingHandler) Confirm(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handleResponse(c, BadRequest, "invalid id")
		return
	}

	updatedBy, _ := c.Get("user_id")
	updatedByStr, _ := updatedBy.(string)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.ConfirmOnlineRepricing(ctx, id, updatedByStr); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Cancel godoc
// @Summary		Cancel online repricing session
// @Tags		Online Repricing
// @Produce		json
// @Param		id path int true "Repricing ID"
// @Success		200 {object} v1.Response
// @Router		/v1/online-repricing/{id}/cancel [post]
func (h *OnlineRepricingHandler) Cancel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handleResponse(c, BadRequest, "invalid id")
		return
	}

	updatedBy, _ := c.Get("user_id")
	updatedByStr, _ := updatedBy.(string)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.CancelOnlineRepricing(ctx, id, updatedByStr); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
