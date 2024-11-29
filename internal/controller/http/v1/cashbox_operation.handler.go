package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
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
	}
	cashBoxHistory := r.Group("/cash_box_history")
	{
		cashBoxHistory.POST("", h.CreateCashHistory)
		cashBoxHistory.GET("/:cash_box_id", h.GetCashHistory)
	}
}

// Create godoc
// @Summary Create a cash box Operation
// @Description Create a cash box Operation from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashboxOperationRequest true "Cash box Operation information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_operation [post]
func (h *CashBoxOperationHandler) Create(c *gin.Context) {
	var (
		body domain.CashboxOperationRequest
		res  domain.CashboxOperation
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.StartTime = helper.TimePtr(time.Now())
	if err = h.db.WithContext(c.Request.Context()).
		Table("cashbox_operations").
		Create(&body).
		Scan(&res).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, res)
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
	)
	if err = h.db.First(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
// @Param limmit query int false "Limit"
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
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if err = h.db.WithContext(c.Request.Context()).
		Table("cashbox_operations").
		Where("id = ?", c.Param("id")).
		Updates(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
	)
	if err = h.db.Delete(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// CreateCashHistory godoc
// @Summary Create a cash history
// @Description Create a cash history from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashBoxHistoryRequest true "Cash history information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box_history [post]
func (h *CashBoxOperationHandler) CreateCashHistory(c *gin.Context) {
	var (
		body domain.CashBoxHistoryRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	if err = h.db.WithContext(c.Request.Context()).
		Table("cash_box_histories").
		Create(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// GetCashHistory godoc
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
// @Router /cash_box_history/{cash_box_id} [get]
func (h *CashBoxOperationHandler) GetCashHistory(c *gin.Context) {
	var (
		body domain.CashBoxHistory
		err  error
	)
	if err = h.db.Preload("CashBox").First(&body, "cash_box_id = ?", c.Param("cash_box_id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			body.CashBoxID = c.Param("cash_box_id")
			handleResponse(c, OK, body)
			return
		}
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}
