package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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
		dashboard.GET("/count-stats", h.TotalCountStats)
		dashboard.GET("/chart", h.ChartStats)
		dashboard.GET("/top-stores", h.TopStores)
		dashboard.GET("/top-products", h.TopProducts)
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/count-stats [GET]
func (h *DashboardHandler) TotalCountStats(c *gin.Context) {
	var (
		param domain.DashboardQueryParam
		res   *domain.TotalCountStats
	)

	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
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
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		param.StoreId = employee.StoreId
		res.BonusAmount, err = h.service.GetEmployeeBonusAmount(&param, employee.Id)
		if err != nil {
			handleResponse(c, InternalError, "Can't get employee bonus amount")
			return
		}
	}
	// get dashboard data
	res, err = h.service.DashboardTotalCountStats(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
		return
	}
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/chart [GET]
func (h *DashboardHandler) ChartStats(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
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
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		param.StoreId = employee.StoreId
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
// @Router /dashboard/top-stores [GET]
func (h *DashboardHandler) TopStores(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
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
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		param.StoreId = employee.StoreId
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
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-products [GET]
func (h *DashboardHandler) TopProducts(c *gin.Context) {
	var param domain.DashboardQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
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
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		param.StoreId = employee.StoreId
	}
	// get dashboard data
	res, err := h.service.DashboardTopProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
		return
	}
	handleResponse(c, OK, res)
}
