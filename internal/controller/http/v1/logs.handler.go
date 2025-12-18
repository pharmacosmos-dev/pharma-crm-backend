package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

type LogHandler struct {
	*Handler
}

func (h *Handler) NewLogHandler(r *gin.RouterGroup) {
	helper := &LogHandler{h}
	helper.LogRoutes(r)
}

func (h *LogHandler) LogRoutes(r *gin.RouterGroup) {
	logs := r.Group("/logs")
	{
		logs.GET("", h.FetchLogs)
	}
}

// FetLogs godoc
// @Summary Get all transaction stats
// @Description Get all transaction stats
// @Tags 	Logs
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   provider_type query string false "Type might be -> (payme, click, epos, dmed)"
// @Param   limit  query int false "Limit"
// @Param   offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /logs [GET]
func (h *LogHandler) FetchLogs(c *gin.Context) {
	var params domain.LogParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetLogs(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}
