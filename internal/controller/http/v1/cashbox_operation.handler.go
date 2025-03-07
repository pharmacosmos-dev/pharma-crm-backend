package v1

import (
	"errors"
	"fmt"
	"time"

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
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.WithContext(c.Request.Context()).
		Table("cash_boxes").
		Where("id = ?", body.CashBoxID).
		Update("is_open", true).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	now := time.Now()
	body.ID = uuid.New().String()
	body.StartTime = &now
	body.EmployeeID = userId.(string)
	body.IsOpen = true
	var id string
	err = tx.Raw(`
	INSERT INTO 
		cashbox_operations 
			(id, cash_box_id, employee_id, opened_amount, is_open, start_time, description, current_employee_id) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, body.ID, body.CashBoxID, body.EmployeeID,
		body.OpenedAmount, body.IsOpen, body.StartTime,
		body.Description, body.EmployeeID).Scan(&id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	var sale domain.Sale
	err = tx.WithContext(c.Request.Context()).
		Raw(`
		INSERT INTO sales (id, employee_id, cash_box_operation_id) 
		VALUES (?, ?, ?) RETURNING *`,
			uuid.New().String(), userId.(string), id).Scan(&sale).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
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

	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.IsOpen = false
	now := time.Now()
	body.EndTime = &now
	err = h.db.WithContext(c.Request.Context()).
		Table("cashbox_operations").
		Where("id = ?", cashBoxOperationID).
		Updates(&body).Debug().Error
	if err != nil {
		h.log.Error(err)
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
