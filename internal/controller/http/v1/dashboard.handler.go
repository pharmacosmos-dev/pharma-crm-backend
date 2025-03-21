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
// @Param   type 		query string false "Type"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/count-stats [GET]
func (h *DashboardHandler) TotalCountStats(c *gin.Context) {
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	var storeId string
	// check if employee is not admin or superadmin
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		storeId = employee.StoreId
	}
	// get dashboard data
	res, err := h.service.DashboardTotalCountStats(storeId, c.Query("start_date"), c.Query("end_date"))
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/chart [GET]
func (h *DashboardHandler) ChartStats(c *gin.Context) {
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	var storeId string
	// check if employee is not admin or superadmin
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		storeId = employee.StoreId
	}
	// get dashboard data
	res, err := h.service.DashboardChartStats(storeId, c.Query("start_date"), c.Query("end_date"), c.Query("type"))
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /dashboard/top-stores [GET]
func (h *DashboardHandler) TopStores(c *gin.Context) {
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	var storeId string
	// check if employee is not admin or superadmin
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		storeId = employee.StoreId
	}
	// get dashboard data
	res, err := h.service.DashboardTopStores(storeId, c.Query("start_date"), c.Query("end_date"))
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
func (h *DashboardHandler) TopProducts(c *gin.Context) {
	// get user id from header
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	var storeId string
	// check if employee is not admin or superadmin
	if employee.RoleType != config.ADMIN && employee.RoleType != config.SUPERADMIN {
		storeId = employee.StoreId
	}
	// get dashboard data
	res, err := h.service.DashboardTopProducts(storeId, c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		handleResponse(c, InternalError, "Can't get dashboard data")
		return
	}
	handleResponse(c, OK, res)
}
