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
	osp.NewOnlineStoreProductRouters(r)
}


func (h *OnlineStoreProductHandler) NewOnlineStoreProductRouters(r *gin.RouterGroup) {
	onlineStoreProducts:= r.Group("/online-store-products")
	{
		onlineStoreProducts.GET("", h.List)
	}
}

// List godoc
// @Summary		List online store products (current platform prices)
// @Tags		Online Store Products
// @Security     BearerAuth
// @Produce		json
// @Param		store_id query string false "Store ID"
// @Param		type     query string false "Platform type (uzum, yandex_eda, ...)"
// @Success		200 {object} v1.Response
// @Router		/online-store-products [get]
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
