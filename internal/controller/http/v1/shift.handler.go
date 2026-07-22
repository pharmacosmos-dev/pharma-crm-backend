package v1

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/etc"
	"gorm.io/gorm"
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
		body         domain.ShiftRequest
		fromEmployee domain.Employee
		toEmployee   domain.Employee
		operation    domain.CashboxOperation
	)
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error("ERROR on binding body: ", err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get current employee
	err := h.db.WithContext(ctx).Take(&fromEmployee, "id = ?", body.FromEmployeeId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleServiceResponse(c, NotFound, domain.NotFoundError)
			return
		}
		h.log.Errorf("could not get employee: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// check current employee store_id
	if fromEmployee.StoreId == "" {
		handleResponse(c, CONFLICT, "Current employee not attached to store")
		return
	}

	// from_employee smenani topshirishdan oldin face id orqali check-out qilgan bo'lishi shart
	fromLastEvent, err := h.service.GetTodayLastAttendanceEventType(ctx, body.FromEmployeeId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	if fromLastEvent != domain.AttendanceEventCheckOut {
		handleServiceResponse(c, nil, domain.ShiftFromEmployeeNotCheckedOutError)
		return
	}

	// get open operation info
	err = h.db.WithContext(ctx).Raw(`
		SELECT
			co.*
		FROM cashbox_operations co
			JOIN cash_boxes cb ON co.cash_box_id = cb.id
		WHERE cb.store_id = ?
		AND co.current_employee_id = ?
		AND co.is_open = TRUE AND co.end_time IS NULL;`,
		fromEmployee.StoreId, body.FromEmployeeId).Scan(&operation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Open cashbox not found")
			return
		}
		h.log.Errorf("could not get operation info: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// check cashbox operation with empty
	if operation.ID == "" {
		handleResponse(c, NotFound, "Open cashbox not found")
		return
	}

	// get to employee info
	err = h.db.WithContext(ctx).Take(&toEmployee, "id = ?", body.ToEmployeeId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleServiceResponse(c, NotFound, domain.NotFoundError)
			return
		}
		h.log.Errorf("could not get employee: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// to_employee smenani qabul qilishdan oldin face id orqali check-in qilgan bo'lishi shart
	toLastEvent, err := h.service.GetTodayLastAttendanceEventType(ctx, body.ToEmployeeId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	if toLastEvent != domain.AttendanceEventCheckIn {
		handleServiceResponse(c, nil, domain.ShiftToEmployeeNotCheckedInError)
		return
	}

	// get decrypted password
	passoword, err := etc.Decrypt(toEmployee.Password, h.cfg.HashKey)
	if err != nil {
		h.log.Errorf("ERROR decryption password: %v", err)
		handleResponse(c, InternalError, "Failed to parse password")
		return
	}
	// check old and received password
	if body.Password != passoword {
		handleResponse(c, BadRequest, "Wrong password")
		return
	}
	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	body.CashBoxId = operation.CashBoxID
	// Create shift
	err = tx.WithContext(ctx).
		Table("shifts").
		Create(&body).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Errorf("could not create new shift: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	var id string
	err = tx.Raw(`
		SELECT id
		FROM cashbox_operations 
		WHERE is_open = TRUE AND current_employee_id = ? 
		LIMIT 1
	`, body.ToEmployeeId).Scan(&id).Error
	if err != nil {
		_ = tx.Rollback()
		handleServiceResponse(c, InternalError, domain.InternalServerError)
	}
	if id != "" {
		_ = tx.Rollback()
		handleServiceResponse(c, nil, domain.AlreadyHaveOpenCashboxOperationError)
		return
	}
	// update cashbox_operations current_employee_id
	err = tx.Exec(`
		UPDATE cashbox_operations
		SET current_employee_id = ?
		WHERE end_time IS NULL
		AND current_employee_id = ?`,
		body.ToEmployeeId, body.FromEmployeeId).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Errorf("ERROR on updating cashbox_operations current_employee_id: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// add user_id to claims
	userClaims := map[string]any{
		"user_id":    body.ToEmployeeId,
		"company_id": toEmployee.CompanyId,
		"store_id":   toEmployee.StoreId,
		"role":       toEmployee.RoleType,
	}

	// generating access and refresh tokens
	accessToken, refreshToken, err := h.JwtHandler.GenerateTokens(userClaims)
	if err != nil {
		_ = tx.Rollback()
		h.log.Errorf("ERROR on generating token: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// commiting transaction
	if err = tx.Commit().Error; err != nil {
		h.log.Errorf("ERROR on committing transaction: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
