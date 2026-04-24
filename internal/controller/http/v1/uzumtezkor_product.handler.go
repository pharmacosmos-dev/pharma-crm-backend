package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type UzumTezkorProductHandler struct {
	*Handler
}

func (h *Handler) NewUzumTezkorProductHandler(r *gin.RouterGroup) {
	umtkproduct := &UzumTezkorProductHandler{h}
	umtkproduct.UzumTezkorProductRoutes(r)
}

func (h *UzumTezkorProductHandler) UzumTezkorProductRoutes(r *gin.RouterGroup) {
	umtkproduct := r.Group("/uzumtezkor-products")
	{
		umtkproduct.GET("/list", h.List)
	}
}


// List godoc
// @Summary		List UzumTezkor product price history (CRM)
// @Tags		UzumTezkor Products
// @Security	BearerAuth
// @Produce		json
// @Param		type          query string false "Platform type (uzum, yandex_eda)"
// @Param		product_id    query string false "Product ID"
// @Param		material_code query string false "Material code"
// @Param		limit         query int    false "Limit"
// @Param		offset        query int    false "Offset"
// @Success		200 {object} v1.Response
// @Router		/v1/uzumtezkor-products/list [get]
func (h *UzumTezkorProductHandler) List(c *gin.Context) {
	var params domain.UzumTezkorProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, total, err := h.service.GetOnlineProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, result, total)
}
