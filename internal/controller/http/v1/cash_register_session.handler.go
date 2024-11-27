package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

type CashRegisterSessionHandler struct {
	*Handler
}

func (h *Handler) NewCashRegisterSessionHandler(r *gin.RouterGroup) {
	cashRegisterSession := &CashRegisterSessionHandler{h}
	cashRegisterSession.CashRegisterSessionRoutes(r)
}

func (h *CashRegisterSessionHandler) CashRegisterSessionRoutes(r *gin.RouterGroup) {
	cashRegisterSession := r.Group("/cash_register_session")
	{
		cashRegisterSession.POST("", h.Create)
		cashRegisterSession.GET("/:id", h.Get)
		cashRegisterSession.GET("/list", h.List)
		cashRegisterSession.PUT("/:id", h.Update)
		cashRegisterSession.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a cash register session
// @Description Create a cash register session from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashRegisterSessionRequest true "Cash register session information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register_session [post]
func (h *CashRegisterSessionHandler) Create(c *gin.Context) {
	var (
		body domain.CashRegisterSessionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	cashRegisterSession := domain.CashRegisterSession{
		ID:             uuid.New().String(),
		CashRegisterID: body.CashRegisterID,
		EmployeeID:     body.EmployeeID,
		StoreID:        body.StoreID,
		Type:           body.Type,
		OpeningBalance: body.OpeningBalance,
		ClosingBalance: body.ClosingBalance,
		StartTime:      helper.TimePtr(time.Now()),
	}
	if err = h.db.WithContext(c.Request.Context()).
		Create(&cashRegisterSession).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, cashRegisterSession)
}

// Get godoc
// @Summary Get a cash register session
// @Description Get a cash register session from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register session ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register_session/{id} [get]
func (h *CashRegisterSessionHandler) Get(c *gin.Context) {
	var (
		body domain.CashRegisterSession
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
// @Summary Get a cash register session
// @Description Get a cash register session from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register_session/list [get]
func (h *CashRegisterSessionHandler) List(c *gin.Context) {
	var (
		body []domain.CashRegisterSession
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
		Preload("CashRegister").
		Preload("Employee").
		Find(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Update godoc
// @Summary Update a cash register session
// @Description Update a cash register session from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register session ID"
// @Param input body domain.CashRegisterSessionRequest true "Cash register session information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register_session/{id} [put]
func (h *CashRegisterSessionHandler) Update(c *gin.Context) {
	var (
		body domain.CashRegisterSessionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	cashRegisterSession := domain.CashRegisterSession{
		CashRegisterID: body.CashRegisterID,
		EmployeeID:     body.EmployeeID,
		StoreID:        body.StoreID,
		Type:           body.Type,
		OpeningBalance: body.OpeningBalance,
		ClosingBalance: body.ClosingBalance,
		EndTime:        helper.TimePtr(time.Now()),
	}
	if err = h.db.WithContext(c.Request.Context()).
		Model(&cashRegisterSession).
		Where("id = ?", c.Param("id")).
		Updates(&cashRegisterSession).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cashRegisterSession)
}

// Delete godoc
// @Summary Delete a cash register session
// @Description Delete a cash register session from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register session ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register_session/{id} [delete]
func (h *CashRegisterSessionHandler) Delete(c *gin.Context) {
	var (
		body domain.CashRegisterSession
		err  error
	)
	if err = h.db.Delete(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
