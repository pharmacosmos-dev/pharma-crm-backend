package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type PaymentTypeHandler struct {
	*Handler
}

func (h *Handler) NewPaymentTypeHandler(r *gin.RouterGroup) {
	paymentType := &PaymentTypeHandler{h}
	paymentType.PaymentTypeRoutes(r)
}

func (h *PaymentTypeHandler) PaymentTypeRoutes(r *gin.RouterGroup) {
	paymentType := r.Group("/payment-type")
	{
		paymentType.POST("", h.Create)
		paymentType.GET("/:id", h.Get)
		paymentType.GET("/list", h.List)
		paymentType.PUT("/:id", h.Update)
		paymentType.DELETE("/:id", h.Delete)
		paymentType.GET("/active-list", h.ListCashboxOperationID)
	}
	paymentService := r.Group("/payment-service")
	{
		paymentService.POST("", h.CreatePaymentService)
		paymentService.GET("/:id", h.GetPaymentService)
		paymentService.GET("/list", h.ListPaymentService)
		paymentService.PUT("/:id", h.UpdatePaymentService)
		paymentService.DELETE("/:id", h.DeletePaymentService)
	}
}

// Create godoc
// @Summary Create a payment type
// @Description Create a payment type from the request body
// @Tags 	payment_types
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body domain.PaymentTypeRequest true "payment type"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type [post]
func (h *PaymentTypeHandler) Create(c *gin.Context) {
	var (
		body domain.PaymentTypeRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("payment_types").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a payment type
// @Description Get a payment type from the request body
// @Tags payment_types
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "payment type ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type/{id} [get]
func (h *PaymentTypeHandler) Get(c *gin.Context) {
	var res domain.PaymentType
	var id = c.Param("id")
	err := h.db.First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a payment type
// @Description Get a payment type from the request body
// @Tags payment_types
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	cashbox_id query string false "Cash Box ID"
// @Param   type query string false "type"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type/list [get]
func (h *PaymentTypeHandler) List(c *gin.Context) {
	var (
		res         = []*domain.PaymentType{}
		cashBoxId   = c.Query("cashbox_id")
		paymentType = c.Query("type")
	)
	query := h.db.Model(&domain.PaymentType{})
	if cashBoxId != "" {
		query = h.db.
			Table("payment_types pt").
			Select("pt.*", "COALESCE(cpt.is_active, false) AS is_active").
			Joins("LEFT JOIN cashbox_payment_types cpt ON cpt.payment_type_id = pt.id").
			Where("cpt.cash_box_id = ?", cashBoxId)
	}
	if paymentType != "" {
		query = query.Where("type = ?", paymentType)
	}

	err := query.
		Where("is_active = TRUE").
		Order("order_number ASC").Find(&res).Error
	if err != nil {
		h.log.Warn("ERROR on getting payment type list: %v", err)
		handleResponse(c, InternalError, "Can't get payment type list")
		return
	}

	handleResponse(c, OK, res)
}

// ListCashboxID godoc
// @Summary Get a list of payment types by cash box ID
// @Description Get a list of payment types by cash box ID from the request body
// @Tags payment_types
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	cash_box_operation_id query string false "Cash Box Operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type/active-list [get]
func (h *PaymentTypeHandler) ListCashboxOperationID(c *gin.Context) {
	var (
		cashBoxOperationID = c.Query("cash_box_operation_id")
		res                = []*domain.PaymentType{}
	)
	err := h.db.Raw(`
	SELECT pt.*, cpt.is_active FROM payment_types pt
	JOIN cashbox_payment_types cpt ON cpt.payment_type_id = pt.id
	WHERE cpt.is_active = true
	AND cpt.cash_box_id = (SELECT cash_box_id FROM cashbox_operations WHERE id = ?)`,
		cashBoxOperationID).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a payment type
// @Description Update a payment type from the request body
// @Tags 	payment_types
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "payment type ID"
// @Param 	input body domain.PaymentTypeRequest true "payment type"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type/{id} [put]
func (h *PaymentTypeHandler) Update(c *gin.Context) {
	var (
		body domain.PaymentTypeRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.
		WithContext(c.Request.Context()).
		Table("payment_types").Updates(&body).
		Where("id = ?", c.Param("id")).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Delete a payment type
// @Description Delete a payment type from the request body
// @Tags payment_types
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "payment type ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-type/{id} [delete]
func (h *PaymentTypeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.PaymentType{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Create godoc
// @Summary Create a payment service
// @Description Create a payment service from the request body
// @Tags payment_services
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param payment_service body domain.PaymentServiceRequest true "payment service"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-service [post]
func (h *PaymentTypeHandler) CreatePaymentService(c *gin.Context) {
	var (
		body domain.PaymentServiceRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// create new payment service
	body.ID = uuid.New().String()
	body.IsActive = true
	err = h.db.
		WithContext(c.Request.Context()).
		Table("payment_services").
		Create(&body).Error
	if err != nil {
		h.log.Warn("ERROR on creating payment service")
		handleResponse(c, InternalError, "Can't create payment service")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a payment service
// @Description Get a payment service from the request body
// @Tags payment_services
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-service/{id} [get]
func (h *PaymentTypeHandler) GetPaymentService(c *gin.Context) {
	var (
		res domain.PaymentService
		id  = c.Param("id")
	)
	// get payment service by id
	err := h.db.
		Preload("Store").
		Preload("PaymentType").
		First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a payment service
// @Description Get a payment service from the request body
// @Tags 	payment_services
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   store_id query string false "Store ID"
// @Param	payment_type_id query string false "Payment Type ID"
// @Param   limit query int false "Limit"
// @Param   offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-service/list [get]
func (h *PaymentTypeHandler) ListPaymentService(c *gin.Context) {
	var (
		res        = []*domain.PaymentService{}
		param      domain.QueryParam
		totalCount int64
	)
	// bind request query params
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid request param")
		return
	}
	// get default pagination values
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// build payment service get list query
	query := h.db.
		Model(&domain.PaymentService{}).
		Preload("Store").
		Select(`
			id, store_id, 
			payment_type_id, 
			name, type, 
			is_active, 
			created_at, updated_at
		`)
	if param.StoreID != "" {
		query = query.Where("store_id = ?", param.StoreID)
	}
	if param.PaymentTypeID != "" {
		query = query.Where("payment_type_id = ?", param.PaymentTypeID)
	}
	// execute query
	err := query.
		Where("deleted_at IS NULL").
		Count(&totalCount).
		Order("created_at DESC").
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a payment service
// @Description Update a payment service from the request body
// @Tags payment_services
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "payment service ID"
// @Param payment_service body domain.PaymentServiceRequest true "payment service"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-service/{id} [put]
func (h *PaymentTypeHandler) UpdatePaymentService(c *gin.Context) {
	var (
		body domain.PaymentServiceRequest
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
		Table("payment_services").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Delete a payment service
// @Description Delete a payment service from the request body
// @Tags payment_services
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "payment service ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /payment-service/{id} [delete]
func (h *PaymentTypeHandler) DeletePaymentService(c *gin.Context) {
	id := c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Table("payment_services").
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
