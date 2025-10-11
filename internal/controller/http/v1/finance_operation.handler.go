package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

type FinanceOperationHandler struct {
	*Handler
}

func (h *Handler) NewFinanceOperationHandler(r *gin.RouterGroup) {
	finance := &FinanceOperationHandler{h}
	finance.FinanceOperationRoutes(r)
}

func (h *FinanceOperationHandler) FinanceOperationRoutes(r *gin.RouterGroup) {
	financeOperation := r.Group("/finance-operation")
	{
		financeOperation.POST("", h.Create)
		financeOperation.GET("/:id", h.Get)
		financeOperation.GET("/list", h.List)
		financeOperation.PUT("/:id", h.Update)
	}
}

// Create godoc
// @Summary Create a finance operation
// @Description Create a new finance operation from the request body
// @Tags finance operations
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.FinanceOperationRequest true "Finance operation information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-operation [post]
func (h *FinanceOperationHandler) Create(c *gin.Context) {
	var (
		body domain.FinanceOperationRequest
		err  error
	)
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, BadRequest, "User ID not found")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.EmployeeId = userId.(string)
	body.Status = constants.GeneralStatusConfirmed
	// create finance operation
	err = h.db.WithContext(c.Request.Context()).Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a finance operation
// @Description Get a finance operation by ID
// @Tags finance operations
// @Security     BearerAuth
// @Produce json
// @Param id path int true "Finance operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-operation/{id} [get]
func (h *FinanceOperationHandler) Get(c *gin.Context) {
	var (
		res domain.FinanceOperation
		id  = c.Param("id")
	)
	// validate finance id
	if id == "" {
		handleResponse(c, BadRequest, "Finance ID is required")
		return
	}
	// get one finance operation
	err := h.db.First(&res, id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary List all finance operations
// @Description List all finance operations
// @Tags finance operations
// @Security     BearerAuth
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param cashbox_id query string false "Cashbox ID"
// @Param employee_id query string false "Employee ID"
// @Param finance_category_id query int false "Finance Category ID"
// @Param report_type query string false "Report Type"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param from_amount query float64 false "From Amount"
// @Param to_amount query float64 false "To Amount"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-operation/list [get]
func (h *FinanceOperationHandler) List(c *gin.Context) {
	var (
		res        []domain.FinanceOperation
		totalCount int64
		param      domain.FinanceQueryParams
		err        error
	)
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// build query
	query := h.db.
		Model(&domain.FinanceOperation{}).
		Preload("Employee").Preload("Cashbox")

	// filter by search
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("comment LIKE ? OR CAST(id AS TEXT) LIKE ?", param.Search, param.Search)
	}

	// filter by cashbox_id
	if param.CashboxId != "" {
		query = query.Where("cashbox_id = ?", param.CashboxId)
	}

	// filter by employee_id
	if param.EmployeeId != "" {
		query = query.Where("employee_id = ?", param.EmployeeId)
	}

	// filter by finance_category_id
	if param.FinanceCategoryId != 0 {
		query = query.Where("finance_category_id = ?", param.FinanceCategoryId)
	}

	// filter by report_type
	if param.ReportType != "" {
		query = query.Where("report_type = ?", param.ReportType)
	}

	// filter by start_date and end_date
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", param.StartDate, param.EndDate)
	}

	if param.StartDate != "" && param.EndDate == "" {
		query = query.Where("created_at >= ?", param.StartDate)
	}

	// filter by from_amount and to_amount
	if param.FromAmount != 0 {
		query = query.Where("amount >= ?", param.FromAmount)
	}
	if param.ToAmount != 0 {
		query = query.Where("amount <= ?", param.ToAmount)
	}

	// get all finance operations
	err = query.Count(&totalCount).Limit(param.Limit).Offset(param.Offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Stats godoc
// @Summary Get finance operation stats
// @Description Get finance operation stats
// @Tags finance operations
// @Security     BearerAuth
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param cashbox_id query string false "Cashbox ID"
// @Param employee_id query string false "Employee ID"
// @Param finance_category_id query int false "Finance Category ID"
// @Param report_type query string false "Report Type"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param from_amount query float64 false "From Amount"
// @Param to_amount query float64 false "To Amount"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-operation/stats [get]
func (h *FinanceOperationHandler) Stats(c *gin.Context) {
	var (
		res   domain.FinanceOperationStats
		param domain.FinanceQueryParams
		err   error
	)
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// build query
	query := h.db.
		Model(&domain.FinanceOperation{}).
		Select("SUM(finance_operations.amount) AS total_amount")

	// filter by search
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("comment LIKE ? OR CAST(id AS TEXT) LIKE ?", param.Search, param.Search)
	}

	// filter by cashbox_id
	if param.CashboxId != "" {
		query = query.Where("cashbox_id = ?", param.CashboxId)
	}

	// filter by employee_id
	if param.EmployeeId != "" {
		query = query.Where("employee_id = ?", param.EmployeeId)
	}

	// filter by finance_category_id
	if param.FinanceCategoryId != 0 {
		query = query.Where("finance_category_id = ?", param.FinanceCategoryId)
	}

	// filter by report_type
	if param.ReportType != "" {
		query = query.Where("report_type = ?", param.ReportType)
	}

	// filter by start_date and end_date
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", param.StartDate, param.EndDate)
	}

	if param.StartDate != "" && param.EndDate == "" {
		query = query.Where("created_at >= ?", param.StartDate)
	}

	// filter by from_amount and to_amount
	if param.FromAmount != 0 {
		query = query.Where("amount >= ?", param.FromAmount)
	}
	if param.ToAmount != 0 {
		query = query.Where("amount <= ?", param.ToAmount)
	}

	err = query.First(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)

}

// Update godoc
// @Summary Update a finance operation
// @Description Update a finance operation by ID
// @Tags finance operations
// @Security     BearerAuth
// @Produce json
// @Param id path int true "Finance operation ID"
// @Param  input body domain.FinanceOperationRequest true "Finance operation information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-operation/{id} [put]
func (h *FinanceOperationHandler) Update(c *gin.Context) {
	var (
		body domain.FinanceOperationRequest
		id   = c.Param("id")
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update finance operation
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.FinanceOperation{}).
		Where("id = ?", id).Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
