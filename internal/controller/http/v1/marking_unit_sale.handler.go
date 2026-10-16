package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

type MarkingUnitSaleHandler struct {
	*Handler
}

func (h *Handler) NewMarkingUnitSaleHandler(r *gin.RouterGroup) {
	markingUnitSale := &MarkingUnitSaleHandler{h}
	markingUnitSale.MarkingUnitSaleRoutes(r)
}

func (h *MarkingUnitSaleHandler) MarkingUnitSaleRoutes(r *gin.RouterGroup) {
	markingUnitSale := r.Group("/marking-unit-sale")
	{
		markingUnitSale.GET("/list", h.List)
	}
}

// List godoc
// @Summary      List marking unit sales
// @Description  Har bir markirovka bo'yicha nechta dona sotilgan (sold_quantity) va nechta qolgan (stock_quantity).
// @Tags         marking-unit-sale
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        marking  query  string  false  "Markirovka bo'yicha qidiruv"
// @Param        limit    query  int     false  "Limit (default 10)"
// @Param        offset   query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /marking-unit-sale/list [get]
func (h *MarkingUnitSaleHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.MarkingUnitSaleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetMarkingUnitSales(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}
