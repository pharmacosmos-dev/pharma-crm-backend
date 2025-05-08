package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type CashBoxOperationHandler struct {
	*Handler
}

func (h *Handler) NewCashBoxOperationHandler(r *gin.RouterGroup) {
	cashBoxOperation := &CashBoxOperationHandler{h}
	cashBoxOperation.CashBoxOperationRoutes(r)
}

func (h *CashBoxOperationHandler) CashBoxOperationRoutes(r *gin.RouterGroup) {
	cashBoxOperation := r.Group("/cash_box_operation")
	{
		cashBoxOperation.POST("", h.Create)
		cashBoxOperation.GET("/:id", h.Get)
		cashBoxOperation.GET("/list", h.List)
		cashBoxOperation.PUT("/:id", h.Update)
		cashBoxOperation.DELETE("/:id", h.Delete)
		cashBoxOperation.PUT("/close/:cash_box_operation_id", h.CloseCashBox)
		cashBoxOperation.GET("/closed-info/:cash_box_id", h.CashBoxOperationClosedAmount)
		cashBoxOperation.GET("info/:id", h.CashBoxOperationInfo)
		cashBoxOperation.GET("/shift", h.OperationShiftList)
		cashBoxOperation.GET("/stats", h.OperationStats)
		cashBoxOperation.GET("/history", h.OperationHistory)
		cashBoxOperation.POST("/send-expense/:id", h.SendShiftExpense)
	}
}

// Create godoc
// @Summary Create a cash box Operation
// @Description Create a cash box Operation from the request body
// @Tags 	cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body domain.CashboxOperationRequest true "Cash box Operation information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation [post]
func (h *CashBoxOperationHandler) Create(c *gin.Context) {
	var (
		body domain.CashboxOperationRequest
		err  error
	)
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// check store id
	if body.StoreID == "" {
		handleResponse(c, BadRequest, "Store ID is required")
		return
	}
	// create cashbox operation and new sale
	sale, err := h.service.CreateCashboxOperation(&body, userId)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, sale)
}

// Get godoc
// @Summary Get a cash box Operation
// @Description Get a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box Operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/{id} [get]
func (h *CashBoxOperationHandler) Get(c *gin.Context) {
	var (
		body domain.CashboxOperation
		err  error
		id   = c.Param("id")
	)
	err = h.db.First(&body, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// List godoc
// @Summary Get a cash box Operation
// @Description Get a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/list [get]
func (h *CashBoxOperationHandler) List(c *gin.Context) {
	var (
		body []domain.CashboxOperation
		err  error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.
		Limit(limit).
		Offset(offset).
		Preload("CashBox").
		Preload("Employee").
		Find(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// CloseCashBox godoc
// @Summary Close a cash box Operation
// @Description Close a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	cash_box_operation_id path string true "cash box Operation ID"
// @Param 	input body domain.CloseCashboxOperation true "Cash box Operation close request body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/close/{cash_box_operation_id} [put]
func (h *CashBoxOperationHandler) CloseCashBox(c *gin.Context) {
	var (
		body               domain.CloseCashboxOperation
		err                error
		cashBoxOperationID = c.Param("cash_box_operation_id")
	)
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.service.CloseCashBoxOperation(cashBoxOperationID, &body, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "CLOSED")
}

// Update godoc
// @Summary Update a cash box Operation
// @Description Update a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box Operation ID"
// @Param input body domain.CashboxOperationRequest true "Cash box Operation information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/{id} [put]
func (h *CashBoxOperationHandler) Update(c *gin.Context) {
	var (
		body domain.CashboxOperationRequest
		err  error
		id   = c.Param("id")
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("cashbox_operations").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a cash box Operation
// @Description Delete a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box Operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/{id} [delete]
func (h *CashBoxOperationHandler) Delete(c *gin.Context) {
	var (
		body domain.CashboxOperation
		err  error
		id   = c.Param("id")
	)
	err = h.db.Delete(&body, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// CashBoxOperationClosedAmount godoc
// @Summary Get a cash history
// @Description Get a cash history from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param cash_box_id path string true "cash history ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/closed-info/{cash_box_id} [get]
func (h *CashBoxOperationHandler) CashBoxOperationClosedAmount(c *gin.Context) {
	var (
		cashBoxOperation domain.CashboxOperation
		cashBoxID        = c.Param("cash_box_id")
	)
	err := h.db.Raw(`
		SELECT * FROM cashbox_operations 
		WHERE cash_box_id = ? AND end_time IS NOT NULL 
		ORDER BY end_time DESC LIMIT 1`, cashBoxID).Scan(&cashBoxOperation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, cashBoxOperation)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cashBoxOperation)
}

// CashBoxOperationInfo godoc
// @Summary Get a cash operation
// @Description Get a cash operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "cash operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/info/{id} [get]
func (h *CashBoxOperationHandler) CashBoxOperationInfo(c *gin.Context) {
	var (
		cashBoxOperation domain.CashboxOperationInfo
		id               = c.Param("id")
	)
	err := h.db.
		Raw(`
		SELECT co.*, s.name as store_name, e.first_name as first_name 
		FROM cashbox_operations co 
		JOIN employees e ON co.employee_id = e.id
		JOIN cash_boxes cb ON co.cash_box_id = cb.id
		JOIN stores s ON cb.store_id = s.id
		WHERE co.id = ?
		`, id).Scan(&cashBoxOperation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, cashBoxOperation)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cashBoxOperation)
}

// OperationShiftList godoc
// @Summary Get a cash operation shift list
// @Description Get a cash operation shift list from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param store_id query string false "Store ID"
// @Param is_open query string false "Is open"
// @Param search query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/shift [get]
func (h *CashBoxOperationHandler) OperationShiftList(c *gin.Context) {
	var (
		storeID = c.Query("store_id")
		isOpen  = c.Query("is_open")
		search  = c.Query("search")
		shifts  []domain.CashboxOperationShift
	)
	// get limit offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get cash box operation shift list
	shifts, totalCount, err := h.service.GetOperationShiftList(storeID, isOpen, search, limit, offset)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get _meta data to add pagination items
	data := utils.ListResponse(shifts, totalCount, limit, offset)

	handleResponse(c, OK, data)

}

// OperationStats godoc
// @Summary Get a cash operation stats
// @Description Get a cash operation stats
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param store_id query string false "Store ID"
// @Param is_open query string false "Is open"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/stats [get]
func (h *CashBoxOperationHandler) OperationStats(c *gin.Context) {
	var (
		storeID = c.Query("store_id")
		isOpen  = c.Query("is_open")
		search  = c.Query("search")
	)
	// get cash box operation stats
	stats, err := h.service.GetOperationStats(storeID, isOpen, search)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, stats)
}

// OperationHistory godoc
// @Summary Get a cash operation history
// @Description Get a cash operation history
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param is_open query bool false "Is open"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/history [get]
func (h CashBoxOperationHandler) OperationHistory(c *gin.Context) {
	var (
		storeID = c.Query("store_id")
		isOpen  = c.Query("is_open")
		search  = c.Query("search")
	)
	// get limit offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get cash box operation history
	res, totalCount, err := h.service.OperationHistory(storeID, isOpen, search, limit, offset)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	data := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, data)
}

// Send shift expense to 1C godoc
// @Summary Send Shift Expenses
// @Description Send shift expense to 1C
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation/send-expense/{id} [POST]
func (h CashBoxOperationHandler) SendShiftExpense(c *gin.Context) {
	var id = c.Param("id")
	// validate cashbox_operation_id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid cashbox operation id")
		return
	}
	// send shift expense service
	err := h.service.SendExpenseTo1C(id)
	if err != nil {
		h.log.Warn("Failed to send shift expense: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Success")
}
