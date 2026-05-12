package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
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
		target.DELETE("/:id", h.Delete)
		target.PUT("/:id", h.Update)
		target.GET("/:store_id", h.StoreHistory)
		target.GET("/list", h.List)
		// target.GET("/my", h.GetMyTarget)
		target.GET("/employee/list/:store_id", h.EmployeeHistory)
		target.GET("/employee/my", h.GetMyEmployeeTarget)
		target.PUT("/employee/:employee_id", h.UpdateEmployeeTarget)
		target.GET("/summary", h.Summary)
		target.POST("/import-excel", h.ImportFromExcel)
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

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.StoreTargetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateStoreTarget(ctx, &body)
	if err != nil {
		if err.Error() == "permission denied: can only update current or future month targets" {
			handleResponse(c, FORBIDDEN, err.Error())
			return
		}

		// if err.Error() == "target already exists for this store and month" {
		// 	handleResponse(c, BadRequest, err.Error())
		// 	return
		// }
		handleServiceResponse(c, nil, err)
		return
	}                      

	handleResponse(c, CREATED, result)
}

// Delete godoc
// @Summary      Delete store target
// @Description  Deletes store target and all its employee targets
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Store target ID"
// @Success      200 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/{id} [delete]
func (h *StoreTargetHandler) Delete(c *gin.Context) {

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")

	tx := h.db.WithContext(c.Request.Context()).Begin()

	// 1️⃣ Avval employee targetlarni o‘chiramiz
	if err := tx.
		Where("store_target_id = ?", id).
		Delete(&domain.EmployeeTarget{}).Error; err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}

	// 2️⃣ Keyin store targetni o‘chiramiz
	result := tx.
		Where("id = ?", id).
		Delete(&domain.StoreTarget{})

	if result.Error != nil {
		tx.Rollback()
		handleResponse(c, InternalError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		handleResponse(c, NotFound, "store target not found")
		return
	}

	tx.Commit()
	handleResponse(c, OK, "DELETED")
}


// Update godoc
// @Summary      Update store target
// @Description  Store target amount will be updated. Only allowed for current or future months
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
func (h *StoreTargetHandler) Update(c *gin.Context) {

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")

	var body domain.StoreTargetUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.UpdateStoreTarget(ctx, id, &body)
	if err != nil {
		if err.Error() == "store target not found" {
			handleResponse(c, NotFound, err.Error())
			return
		}
		if err.Error() == "permission denied: can only update current or future month targets" {
			handleResponse(c, FORBIDDEN, err.Error())
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, result)
}


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
// @Router       /store-target/{store_id} [get]
func (h *StoreTargetHandler) StoreHistory(c *gin.Context) {
	user := h.service.GetSignedUser(c)

	pathStoreId := c.Param("store_id")

	var storeId string
	if !utils.In(user.Role, constants.StoreTargetViewAll...) {
		if user.StoreId == "" {
			handleResponse(c, BadRequest, "store_id not found for user")
			return
		}
		storeId = user.StoreId
	} else {
		if pathStoreId == "" {
			handleResponse(c, BadRequest, "store_id is required")
			return
		}
		storeId = pathStoreId
	}

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
	
	results, err := h.service.GetStoreTargetHistory(ctx, storeId, user.CompanyId, year, month)
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
// @Param 		 search query string false "Search"
// @Param        year       query  int     false  "Year (exmaple: 2026)"
// @Param        month      query  int     false  "Month (1-12)"
// @Param        limit      query  int     false  "Limit"
// @Param        offset     query  int     false  "Offset"
// @Param        order query string false "Order by (+store_name || -store_name || +target || -target || -sales || +sales)"
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
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var isAdmin bool
	if !utils.In(user.Role, constants.StoreTargetViewAll...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyIds = []string{user.CompanyId}
	} else {
		isAdmin = true
		params.CompanyIds = []string{user.CompanyId}
	}

	if isAdmin && params.IsFranchise != nil && *params.IsFranchise {
		params.CompanyIds, _ = h.service.GetCompanyIds(ctx, true)
		params.StoreId = ""
	} else if isAdmin && params.IsPharma != nil && *params.IsPharma {
		params.CompanyIds, _ = h.service.GetCompanyIds(ctx, false)
		params.StoreId = ""
	}

	results, count, err := h.service.GetStoreTargetList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, map[string]interface{}{
		"_meta": utils.Meta{
			TotalCount:  count,
			PerPage:     params.Limit,
			CurrentPage: (params.Offset / params.Limit) + 1,
			PageCount:   int((count + int64(params.Limit) - 1) / int64(params.Limit)),
		},
		"data": results,
	})
}


// EmployeeHistory godoc
// @Summary      Store cashier target history
// @Description  Monthly target history for everyone in a given store
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id    path   string  true   "Store ID"
// @Param        employee_id query  string  false  "Employee ID (optional, for one employee)"
// @Param        year        query  int     false  "Year (example: 2026)"
// @Param        month       query  int     false  "Moth (1-12)"
// @Param        limit       query  int     false  "Limit"
// @Param        offset      query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/employee/list/{store_id} [get]
func (h *StoreTargetHandler) EmployeeHistory(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeeTargetQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	pathStoreId := c.Param("store_id")

	if !utils.In(user.Role, constants.StoreTargetViewAll...) {
		if user.StoreId == "" {
			handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
			return
		}
		params.StoreId = user.StoreId
	} else {
		if pathStoreId == "" {
			handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
			return
		}
		params.StoreId = pathStoreId
	}

	params.CompanyId = user.CompanyId
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	results, count, err := h.service.GetEmployeeTargetHistoryByStore(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, results, count)
}


// GetMyEmployeeTarget godoc
// @Summary      Employee current month target and sales
// @Description  Target amount and actual sales of the logged-in employee for the current month
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/employee/my [get]
func (h *StoreTargetHandler) GetMyEmployeeTarget(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleResponse(c, BadRequest, "user not found")
		return
	}
	if user.StoreId == "" {
		handleResponse(c, BadRequest, "store not found for user")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetDailySalesStoreTargetEmployee(ctx, user.UserId, user.StoreId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	if result == nil {
		handleResponse(c, OK, gin.H{})
		return
	}

	handleResponse(c, OK, result)
}


// UpdateEmployeeTarget godoc
// @Summary      Update employee target amount
// @Description  Updates the target amount for a specific employee. The remaining amount is distributed equally among other employees in the same store target. Store target total is not affected.
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        employee_id  path  string                            true  "Employee ID"
// @Param        input        body  domain.EmployeeTargetUpdateRequest  true  "New target amount"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      403 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/employee/{employee_id} [put]
func (h *StoreTargetHandler) UpdateEmployeeTarget(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	employeeId := c.Param("employee_id")
	if employeeId == "" {
		handleResponse(c, BadRequest, "employee_id is required")
		return
	}

	var body domain.EmployeeTargetUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.UpdateEmployeeTargetAmount(ctx, body.StoreTargetId, employeeId, body.Amount)
	if err != nil {
		if err == domain.NotFoundError {
			handleResponse(c, NotFound, err.Error())
			return
		}
		if err.Error() == "permission denied: can only update current or future month targets" {
			handleResponse(c, FORBIDDEN, err.Error())
			return
		}
		if err.Error() == "employee target amount cannot be greater than or equal to store target amount" {
			handleResponse(c, BadRequest, err.Error())
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "updated")
}

// Summary godoc
// @Summary      Store Targets Summary
// @Description  Get total amount and total sales for given month/year (defaults to current month/year)
// @Tags         store-target
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        year         query  int     false  "Year (example: 2026)"
// @Param        month        query  int     false  "Month (1-12)"
// @Param        is_franchise query  bool    false  "Filter by franchise stores"
// @Param        is_pharma    query  bool    false  "Filter by pharma stores"
// @Success      200 {object} domain.StoreTargetSummary
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router /store-target/summary [GET]
func (h *StoreTargetHandler) Summary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.StoreTargetQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var isAdmin bool
	storeId := ""

	if !utils.In(user.Role, constants.StoreTargetViewAll...) {
		if user.StoreId != "" {
			storeId = user.StoreId
		}
		params.CompanyIds = []string{user.CompanyId}
	} else {
		isAdmin = true
		params.CompanyIds = []string{user.CompanyId}
	}

	if isAdmin && params.IsFranchise != nil && *params.IsFranchise {
		params.CompanyIds, _ = h.service.GetCompanyIds(ctx, true)
		storeId = ""
	} else if isAdmin && params.IsPharma != nil && *params.IsPharma {
		params.CompanyIds, _ = h.service.GetCompanyIds(ctx, false)
		storeId = ""
	}

	summary, err := h.service.GetCurrentMonthStoreTargetsSummary(ctx, params.CompanyIds, storeId, params.Year, params.Month)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, summary)
}


// ImportFromExcel godoc
// @Summary      Import store targets from Excel
// @Description  Reads store_id, amount, month, year from uploaded Excel file and upserts store_target records in a single transaction
// @Tags         store-target
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Excel file (.xlsx). Columns: A=store_id, B=amount, C=month, D=year"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /store-target/import-excel [post]
func (h *StoreTargetHandler) ImportFromExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		handleResponse(c, BadRequest, "file field is required (multipart/form-data, key: file)")
		return
	}

	src, err := file.Open()
	if err != nil {
		handleResponse(c, InternalError, "Could not open the file: "+err.Error())
		return
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		handleResponse(c, BadRequest, "The Excel file is invalid or corrupted: "+err.Error())
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		handleResponse(c, BadRequest, "No sheet was found in the Excel file.")
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		handleResponse(c, BadRequest, "Could not read the sheet: "+err.Error())
		return
	}

	if len(rows) < 2 {
		handleResponse(c, BadRequest, "The uploaded file is empty or contains only a header row.")
		return
	}

	var excelRows []domain.StoreTargetExcelRow
	for i, row := range rows {
		if i == 0 {
			continue // header
		}
		if len(row) < 4 {
			continue // not enough columns
		}

		storeId := strings.TrimSpace(row[0])
		amount, _ := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		month, _ := strconv.Atoi(strings.TrimSpace(row[2]))
		year, _ := strconv.Atoi(strings.TrimSpace(row[3]))

		excelRows = append(excelRows, domain.StoreTargetExcelRow{
			StoreId: storeId,
			Amount:  amount,
			Month:   month,
			Year:    year,
		})
	}

	if len(excelRows) == 0 {
		handleResponse(c, BadRequest, "No data was found in the Excel file.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := h.service.UpsertStoreTargetsFromExcel(ctx, excelRows)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}