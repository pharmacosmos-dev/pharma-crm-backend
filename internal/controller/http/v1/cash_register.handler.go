package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type CashRegisterHandler struct {
	*Handler
}

func (h *Handler) NewCashRegisterHandler(r *gin.RouterGroup) {
	cashRegister := &CashRegisterHandler{h}
	cashRegister.CashRegisterRoutes(r)
}

func (h *CashRegisterHandler) CashRegisterRoutes(r *gin.RouterGroup) {
	cashRegister := r.Group("/cash_register")
	{
		cashRegister.POST("", h.Create)
		cashRegister.GET("/:id", h.Get)
		cashRegister.GET("/list", h.List)
		cashRegister.PUT("/:id", h.Update)
		cashRegister.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a cash register
// @Description Create a cash register from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashRegisterRequest true "Cash register information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register [post]
func (h *CashRegisterHandler) Create(c *gin.Context) {
	var (
		body domain.CashRegisterRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	// Map request to model
	cashRegister := domain.CashRegister{
		ID:      body.ID,
		Name:    body.Name,
		StoreID: body.StoreID,
	}

	// Save to database
	if err = h.db.WithContext(c.Request.Context()).Create(&cashRegister).Error; err != nil {
		h.log.Error(fmt.Errorf("failed to create cash register: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a cash register
// @Description Get a cash register from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register/{id} [get]
func (h *CashRegisterHandler) Get(c *gin.Context) {
	var (
		body domain.CashRegister
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
// @Summary Get a cash register
// @Description Get a cash register from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register/list [get]
func (h *CashRegisterHandler) List(c *gin.Context) {
	var (
		body []domain.CashRegister
		err  error
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.Limit(limit).Offset(offset).Preload("Store").Find(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Update godoc
// @Summary Update a cash register
// @Description Update a cash register from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register ID"
// @Param input body domain.CashRegisterRequest true "Cash register information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register/{id} [put]
func (h *CashRegisterHandler) Update(c *gin.Context) {
	var (
		body domain.CashRegisterRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	cashRegister := domain.CashRegister{
		ID:      body.ID,
		Name:    body.Name,
		StoreID: body.StoreID,
	}
	if err = h.db.WithContext(c.Request.Context()).
		Where("id = ?", c.Param("id")).
		Updates(&cashRegister).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cashRegister)
}

// Delete godoc
// @Summary Delete a cash register
// @Description Delete a cash register from the request body
// @Tags cash_registers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash register ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_register/{id} [delete]
func (h *CashRegisterHandler) Delete(c *gin.Context) {
	var (
		body domain.CashRegister
		err  error
	)
	if err = h.db.Delete(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
