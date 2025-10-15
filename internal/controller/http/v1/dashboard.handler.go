package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/count-stats [POST]
func (h *DashboardHandler) TotalCountStats(c *gin.Context) {
	var param domain.DashboardQueryParam

	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	// get user id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	var bonus domain.DashboardCountStatsBonus
	// get bonus amount
	bonus, err = h.service.GetEmployeeBonusAmount(&param, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Can't get employee bonus amount")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTotalCountStats(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/chart [POST]
func (h *DashboardHandler) ChartStats(c *gin.Context) {
	var param domain.DashboardQueryParam

	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}

	// get dashboard data
	res, err := h.service.DashboardChartStats(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
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
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// get limit offset with checking default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopStores(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-products [POST]
func (h *DashboardHandler) TopProducts(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	// get limit offset with checking default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/bonus-products [POST]
func (h *DashboardHandler) BonusProducts(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	// get limit offset with checking default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardBonusProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-seller [POST]
func (h *DashboardHandler) TopSeller(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	// get limit offset with checking default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreIds = []string{employee.StoreId}
		}
		param.CompanyId = employee.CompanyId
	}
	// get dashboard data
	res, err := h.service.DashboardTopSeller(&param)
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/payments [POST]
func (h *DashboardHandler) Payments(c *gin.Context) {
	var param domain.DashboardQueryParam
	// company_id from header
	companyId, ok := c.Get("company_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "Company ID not found")
		return
	}
	param.CompanyId = companyId.(string)
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	res, err := h.service.DashboardPayments(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get payment type stats")
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
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/transaction [POST]
func (h *DashboardHandler) Transaction(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// company_id from header
	companyId, ok := c.Get("company_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "Company ID not found")
		return
	}
	param.CompanyId = companyId.(string)
	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	res, err := h.service.DashboardTransaction(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get transaction")
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
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res, totalCount, err := h.service.DashboardOldImports(c, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get import")
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}
