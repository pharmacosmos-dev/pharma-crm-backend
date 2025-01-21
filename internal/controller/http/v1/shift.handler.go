package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type ShiftHandler struct {
	*Handler
}

func (h *Handler) NewShiftHandler(r *gin.RouterGroup) {
	shift := &ShiftHandler{h}
	shift.ShiftRoutes(r)
}

func (h *ShiftHandler) ShiftRoutes(r *gin.RouterGroup) {
	shift := r.Group("/shift")
	{
		shift.POST("", h.Create)
		shift.GET("/:id", h.Get)
		shift.GET("/list", h.List)
		shift.PUT("/:id", h.Update)
	}
}

// Create godoc
// @Summary Create a shift
// @Description Create a shift from the request body
// @Tags shifts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param shift body domain.ShiftRequest true "Shift information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shift [post]
func (h *ShiftHandler) Create(c *gin.Context) {
	var (
		body domain.ShiftRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("shifts").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.
		WithContext(c.Request.Context()).
		Raw(`
		UPDATE cashbox_operations 
		SET current_employee_id = ? 
		WHERE end_time IS NULL 
		AND cash_box_id = ? AND employee_id = ?`,
			body.ToEmployeeId, body.CashBoxId, body.FromEmployeeId).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	accessClaims := map[string]interface{}{
		"user_id": body.ToEmployeeId,
	}
	refreshClaims := map[string]interface{}{
		"user_id": body.ToEmployeeId,
	}
	accessToken, refreshToken, err := h.JwtHandler.GenerateTokens(accessClaims, refreshClaims)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, map[string]string{"access_token": accessToken, "refresh_token": refreshToken})
}

// Get godoc
// @Summary Get a shift
// @Description Get a shift from the request body
// @Tags shifts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "shift ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shift/{id} [get]
func (h *ShiftHandler) Get(c *gin.Context) {
	var (
		id  = c.Param("id")
		res domain.Shift
	)

	err := h.db.
		Preload("CashBox").
		Preload("FromEmployee").
		Preload("ToEmployee").
		First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a shift
// @Description Get a shift from the request body
// @Tags shifts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param employee_id query string false "Employee ID"
// @Param cash_box_id query string false "Cash Box ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shift/list [get]
func (h *ShiftHandler) List(c *gin.Context) {
	var (
		res        = []*domain.Shift{}
		cashBoxId  = c.Query("cash_box_id")
		employeeId = c.Query("employee_id")
		totalCount int64
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.Model(&domain.Shift{}).
		Preload("CashBox").
		Preload("FromEmployee").
		Preload("ToEmployee")
	if cashBoxId != "" {
		query = query.Where("cash_box_id = ?", cashBoxId)
	}

	if employeeId != "" {
		query = query.Where("from_employee_id = ?", employeeId)
	}

	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res, totalCount)
}

// Update godoc
// @Summary Update a shift
// @Description Update a shift from the request body
// @Tags shifts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "shift ID"
// @Param 	shift body domain.Shift true "Shift information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shift/{id} [put]
func (h *ShiftHandler) Update(c *gin.Context) {
	var (
		body domain.Shift
		err  error
		id   = c.Param("id")
	)

	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	now := time.Now()
	body.UpdatedAt = &now
	err = h.db.
		WithContext(c.Request.Context()).
		Table("shifts").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}
