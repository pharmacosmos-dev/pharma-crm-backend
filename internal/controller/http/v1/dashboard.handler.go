package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

type DashboardHandler struct {
	*Handler
}

func (h *Handler) NewDashboardHandler(r *gin.RouterGroup) {
	dashboard := &DashboardHandler{h}
	dashboard.DashboardRoutes(r)
}

func (h *DashboardHandler) DashboardRoutes(r *gin.RouterGroup) {
	dashboard := r.Group("/dashboard")
	{
		dashboard.POST("/count-stats", h.TotalCountStats)
		dashboard.POST("/chart", h.ChartStats)
		dashboard.POST("/top-stores", h.TopStores)
		dashboard.POST("/top-products", h.TopProducts)
		dashboard.POST("/bonus-products", h.BonusProducts)
		dashboard.POST("/top-seller", h.TopSeller)
		dashboard.POST("/payments", h.Payments)
		dashboard.POST("/transaction", h.Transaction)
		dashboard.POST("/old-import", h.OldImport)
	}
}

// TotalCountStats godoc
// @Summary Get total count stats
// @Description Get total count stats
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   type 		query string false "Type"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/count-stats [POST]
func (h *DashboardHandler) TotalCountStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get bonus amount
	bonus, err := h.service.GetEmployeeBonusAmount(ctx, &params, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTotalCountStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	// get employee bonus amount
	res.BonusAmount = bonus.BonusAmount
	res.BeforeBonusAmount = bonus.BeforeBonusAmount

	handleResponse(c, OK, res)
}

// ChartStats godoc
// @Summary Get total chart stats
// @Description Get total chart stats
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   type 		query string false "Type might be -> (HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY)"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/chart [POST]
func (h *DashboardHandler) ChartStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}

	// get dashboard data
	res, err := h.service.DashboardChartStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Top Stores godoc
// @Summary Get top stores
// @Description Get top stores
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-stores [POST]
func (h *DashboardHandler) TopStores(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopStores(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Top Products godoc
// @Summary Get top products
// @Description Get top products
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-products [POST]
func (h *DashboardHandler) TopProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Top Bonus Products godoc
// @Summary Get bonus products
// @Description Get bonus products
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   limit 		query int false "Limit"
// @Param 	offset 		query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/bonus-products [POST]
func (h *DashboardHandler) BonusProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardBonusProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	handleResponse(c, OK, res)
}

// Top Bonus Products godoc
// @Summary Get bonus products
// @Description Get bonus products
// @Tags dashboard
// @Security     BearerAuth
// @Produce json
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-seller [POST]
func (h *DashboardHandler) TopSeller(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopSeller(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Payments godoc
// @Summary Get all payments stats
// @Description Get all payments stats
// @Tags 	dashboard
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   type 		query string false "Type might be -> (HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY)"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/payments [POST]
func (h *DashboardHandler) Payments(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.DashboardPayments(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	handleResponse(c, OK, res)
}

// Payments godoc
// @Summary Get all transaction stats
// @Description Get all transaction stats
// @Tags 	dashboard
// @Security     BearerAuth
// @Produce json
// @Param   start_date 	query string true "Start Date"
// @Param   end_date 	query string true "End Date"
// @Param   type 		query string false "Type might be -> (HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY)"
// @Param   store_id 	query string false "Store ID"
// @Param   ids 		body  domain.DashboardBody false "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/transaction [POST]
func (h *DashboardHandler) Transaction(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var param domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&param); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	param.CompanyId = user.CompanyId
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	res, err := h.service.DashboardTransaction(ctx, &param)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	handleResponse(c, OK, res)
}

// OldImport godoc
// @Summary Get all import stats
// @Description Get all import stats
// @Tags 	dashboard
// @Security     BearerAuth
// @Produce json
// @Param 	limit 	query string false "Limit"
// @Param 	offset query string false 	"Offset"
// @Param   store_id 	query string false "Store ID"
// @Param   search 	query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/old-import [POST]
func (h *DashboardHandler) OldImport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.DashboardQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get limit and offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.DashboardOldImports(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}
