package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type OnlineStoreProductHandler struct {
	*Handler
}

func (h *Handler) NewOnlineStoreProductHandler(r *gin.RouterGroup) {
	osp := &OnlineStoreProductHandler{h}
	g := r.Group("/online-store-products")
	g.POST("/upsert", osp.Upsert)
	g.GET("", osp.List)
}

// Upsert godoc
// @Summary		Upsert online store product prices (bulk)
// @Tags		Online Store Products
// @Accept		json
// @Produce		json
// @Param		body body domain.UpsertOnlineStoreProductsRequest true "Upsert request"
// @Success		200 {object} v1.Response
// @Failure		400 {object} v1.Response
// @Failure		500 {object} v1.Response
// @Router		/v1/online-store-products/upsert [post]
func (h *OnlineStoreProductHandler) Upsert(c *gin.Context) {
	var req domain.UpsertOnlineStoreProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if req.StoreId == "" || req.Type == "" {
		handleResponse(c, BadRequest, "store_id and type are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.UpsertOnlineStoreProducts(ctx, &req); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "upserted successfully")
}

// List godoc
// @Summary		List online store products
// @Tags		Online Store Products
// @Produce		json
// @Param		store_id query string false "Store ID"
// @Param		type     query string false "Platform type (uzum, yandex_eda, ...)"
// @Success		200 {object} v1.Response
// @Failure		500 {object} v1.Response
// @Router		/v1/online-store-products [get]
func (h *OnlineStoreProductHandler) List(c *gin.Context) {
	var params domain.OnlineStoreProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetOnlineStoreProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, result, int64(len(result)))
}
