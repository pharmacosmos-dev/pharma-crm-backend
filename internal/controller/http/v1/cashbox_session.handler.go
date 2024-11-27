package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

type CashBoxSessionHandler struct {
	*Handler
}

func (h *Handler) NewCashBoxSessionHandler(r *gin.RouterGroup) {
	cashBoxSession := &CashBoxSessionHandler{h}
	cashBoxSession.CashBoxSessionRoutes(r)
}

func (h *CashBoxSessionHandler) CashBoxSessionRoutes(r *gin.RouterGroup) {
	cashBoxSession := r.Group("/cash_box_session")
	{
		cashBoxSession.POST("", h.Create)
		cashBoxSession.GET("/:id", h.Get)
		cashBoxSession.GET("/list", h.List)
		cashBoxSession.PUT("/:id", h.Update)
		cashBoxSession.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a cash box session
// @Description Create a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashBoxSessionRequest true "Cash box session information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session [post]
func (h *CashBoxSessionHandler) Create(c *gin.Context) {
	var (
		body domain.CashBoxSessionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	cashBoxSession := domain.CashBoxSession{
		ID:             uuid.New().String(),
		CashBoxID:      body.CashBoxID,
		EmployeeID:     body.EmployeeID,
		StoreID:        body.StoreID,
		Type:           body.Type,
		OpeningBalance: body.OpeningBalance,
		ClosingBalance: body.ClosingBalance,
		StartTime:      helper.TimePtr(time.Now()),
	}
	if err = h.db.WithContext(c.Request.Context()).
		Create(&cashBoxSession).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, cashBoxSession)
}

// Get godoc
// @Summary Get a cash box session
// @Description Get a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box session ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session/{id} [get]
func (h *CashBoxSessionHandler) Get(c *gin.Context) {
	var (
		body domain.CashBoxSession
		err  error
	)
	if err = h.db.First(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// List godoc
// @Summary Get a cash box session
// @Description Get a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session/list [get]
func (h *CashBoxSessionHandler) List(c *gin.Context) {
	var (
		body []domain.CashBoxSession
		err  error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.
		Limit(limit).
		Offset(offset).
		Preload("CashBox").
		Preload("Employee").
		Find(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Update godoc
// @Summary Update a cash box session
// @Description Update a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box session ID"
// @Param input body domain.CashBoxSessionRequest true "Cash box session information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session/{id} [put]
func (h *CashBoxSessionHandler) Update(c *gin.Context) {
	var (
		body domain.CashBoxSessionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	cashBoxSession := domain.CashBoxSession{
		CashBoxID:      body.CashBoxID,
		EmployeeID:     body.EmployeeID,
		StoreID:        body.StoreID,
		Type:           body.Type,
		OpeningBalance: body.OpeningBalance,
		ClosingBalance: body.ClosingBalance,
		EndTime:        helper.TimePtr(time.Now()),
	}
	if err = h.db.WithContext(c.Request.Context()).
		Model(&cashBoxSession).
		Where("id = ?", c.Param("id")).
		Updates(&cashBoxSession).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cashBoxSession)
}

// Delete godoc
// @Summary Delete a cash box session
// @Description Delete a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box session ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session/{id} [delete]
func (h *CashBoxSessionHandler) Delete(c *gin.Context) {
	var (
		body domain.CashBoxSession
		err  error
	)
	if err = h.db.Delete(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// LastCashBoxSession godoc
// @Summary Get a cash box session
// @Description Get a cash box session from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param cash_box_id path string true "cash box session ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_session/last/{cash_box_id} [get]
func (h *CashBoxSessionHandler) LastCashBoxSession(c *gin.Context) {
	var (
		body domain.CashBoxSession
		err  error
	)
	if err = h.db.
		Preload("CashBox").
		Where("end_time IS NULL NOT NULL").
		Where("cash_box_id = ?", c.Param("cash_box_id")).
		Limit(1).
		Order("created_at DESC").
		Find(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
