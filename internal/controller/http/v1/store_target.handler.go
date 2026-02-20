package v1

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type StoreTargetHandler struct {
	*Handler
}

func (h *Handler) NewStoreTargetHandler(r *gin.RouterGroup) {
	handler := &StoreTargetHandler{h}
	handler.StoreTargetRoutes(r)
}

func (h *StoreTargetHandler) StoreTargetRoutes(r *gin.RouterGroup) {
	target := r.Group("/store-target")
	{
		target.POST("", h.Create)
		//target.PUT("/:id", h.Update)
		target.GET("/history/:store_id", h.StoreHistory)
		target.GET("/list", h.List)
		// target.GET("/my", h.GetMyTarget)
		target.GET("/employee/history", h.EmployeeHistory)
		target.GET("/summary", h.Summary)
	}
}

// Create godoc
// @Summary      Create store target
// @Description  Creates a monthly target for the store and is equal to all active employees
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input body domain.StoreTargetRequest true "Store target data"
// @Success      201 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target [post]
func (h *StoreTargetHandler) Create(c *gin.Context) {
	var body domain.StoreTargetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	user := h.service.GetSignedUser(c)
	body.CompanyId = user.CompanyId

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateStoreTarget(ctx, &body)
	if err != nil {
		if err.Error() == "target already exists for this store and month" {
			handleResponse(c, BadRequest, err.Error())
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}                      

	handleResponse(c, CREATED, result)
}

// Update godoc
// @Summary      Update store target
// @Description  Store target amount will be updated. Only allowed for next month
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path  string                       true  "Store target ID"
// @Param        input body domain.StoreTargetUpdateRequest true "New target amount"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      403 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/{id} [put]
// func (h *StoreTargetHandler) Update(c *gin.Context) {
// 	id := c.Param("id")

// 	var body domain.StoreTargetUpdateRequest
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		handleResponse(c, BadRequest, err.Error())
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
// 	defer cancel()

// 	result, err := h.service.UpdateStoreTarget(ctx, id, &body)
// 	if err != nil {
// 		if err.Error() == "store target not found" {
// 			handleResponse(c, NotFound, err.Error())
// 			return
// 		}
// 		if err.Error() == "permission denied: can only update next month or future targets" {
// 			handleResponse(c, Forbidden, err.Error())
// 			return
// 		}
// 		handleServiceResponse(c, nil, err)
// 		return
// 	}

// 	handleResponse(c, OK, result)
// }

// StoreHistory godoc
// @Summary      Store target history
// @Description  Sum of all monthly targets and actual sales by Store ID
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  path   string  true   "Store ID"
// @Param        year      query  int     false  "Year (example: 2026)"
// @Param        month     query  int     false  "Month (1-12)"
// @Success      200 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/history/{store_id} [get]
func (h *StoreTargetHandler) StoreHistory(c *gin.Context) {
	storeId := c.Param("store_id")

	year := 0
	month := 0
	if y := c.Query("year"); y != "" {
		fmt.Sscanf(y, "%d", &year)
	}
	if m := c.Query("month"); m != "" {
		fmt.Sscanf(m, "%d", &month)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, err := h.service.GetStoreTargetHistory(ctx, storeId, year, month)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, results)
}

// List godoc
// @Summary      All store target list
// @Description  All store targets, with sales to date for the current month
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id   query  string  false  "Store ID"
// @Param        company_id   query  string  false  "Company ID"
// @Param        year       query  int     false  "Year (exmaple: 2026)"
// @Param        month      query  int     false  "Month (1-12)"
// @Param        limit      query  int     false  "Limit"
// @Param        offset     query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/list [get]
func (h *StoreTargetHandler) List(c *gin.Context) {
	var params domain.StoreTargetQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	user := h.service.GetSignedUser(c)
	params.CompanyId = user.CompanyId
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetStoreTargetList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, results, count)
}

// GetMyTarget godoc
// @Summary      Employee joriy oy target
// @Description  Login qilgan xodimning joriy oy uchun oylik va kunlik target summasi, haqiqiy sotuvlar bilan
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/my [get]
// func (h *StoreTargetHandler) GetMyTarget(c *gin.Context) {
// 	user := h.service.GetSignedUser(c)

// 	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
// 	defer cancel()

// 	result, err := h.service.GetEmployeeTargetWithSales(ctx, user.Id)
// 	if err != nil {
// 		handleServiceResponse(c, nil, err)
// 		return
// 	}

// 	if result == nil {
// 		handleResponse(c, OK, "No target assigned for this month")
// 		return
// 	}

// 	handleResponse(c, OK, result)
// }

// EmployeeHistory godoc
// @Summary      Store cashier target history
// @Description  Monthly target history for everyone in a given store
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id    query  string  true   "Store ID (required)"
// @Param        employee_id query  string  false  "Employee ID (optional, for one employee)"
// @Param        year        query  int     false  "Year (example: 2026)"
// @Param        month       query  int     false  "Moth (1-12)"
// @Param        limit       query  int     false  "Limit"
// @Param        offset      query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/employee/history [get]
func (h *StoreTargetHandler) EmployeeHistory(c *gin.Context) {
	var params domain.EmployeeTargetQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if params.StoreId == "" {
		handleResponse(c, BadRequest, "store_id is required")
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetEmployeeTargetHistoryByStore(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, results, count)
}



// Summary godoc
// @Summary      Current Month Store Targets Summary
// @Description  Get total amount, total sales, and store count for current month
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200 {object} domain.StoreTargetSummary
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/summary [get]
func (h *StoreTargetHandler) Summary(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	// if user.CompanyId == "" {
	// 	handleResponse(c, BadRequest, "company_id not found for user")
	// 	return
	// }

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	summary, err := h.service.GetCurrentMonthStoreTargetsSummary(ctx, user.CompanyId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, summary)
}