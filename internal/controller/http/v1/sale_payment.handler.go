package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type SalePaymentHandler struct {
	*Handler
}

func (h *Handler) NewSalePaymentHandler(r *gin.RouterGroup) {
	salePayment := &SalePaymentHandler{h}
	salePayment.SalePaymentRoutes(r)
}

func (h *SalePaymentHandler) SalePaymentRoutes(r *gin.RouterGroup) {
	salePayment := r.Group("/sale_payment")
	{
		salePayment.POST("", h.Create)
		salePayment.GET("/:id", h.Get)
		salePayment.GET("/list", h.List)
		salePayment.PUT("/:id", h.Update)
		salePayment.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a sale payment
// @Description Create a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param sale_payment body domain.SalePaymentRequest true "sale payment"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_payment [post]
func (h *SalePaymentHandler) Create(c *gin.Context) {
	var (
		body domain.SalePaymentRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	if err = h.db.WithContext(c.Request.Context()).
		Table("sale_payments").Create(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a sale payment
// @Description Get a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_payment/:id [get]
func (h *SalePaymentHandler) Get(c *gin.Context) {
	var res domain.SalePayment
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale payment
// @Description Get a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_payment/list [get]
func (h *SalePaymentHandler) List(c *gin.Context) {
	res := []*domain.SalePayment{}
	if err := h.db.Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a sale payment
// @Description Update a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale payment ID"
// @Param sale_payment body domain.SalePaymentRequest true "sale payment"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_payment/:id [put]
func (h *SalePaymentHandler) Update(c *gin.Context) {
	var (
		body domain.SalePaymentRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.WithContext(c.Request.Context()).
		Table("sale_payments").Updates(&body).
		Where("id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a sale payment
// @Description Delete a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale payment ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_payment/:id [delete]
func (h *SalePaymentHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.SalePayment{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Create godoc
// @Summary Create a transaction
// @Description Create a transaction from the request body
// @Tags transactions
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param transaction body domain.TransactionRequest true "transaction"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transaction [post]
func (h *SalePaymentHandler) CreateTransaction(c *gin.Context) {
	var (
		body domain.TransactionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	if err = h.db.WithContext(c.Request.Context()).
		Table("transactions").Create(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a transaction
// @Description Get a transaction from the request body
// @Tags transactions
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transaction/:id [get]
func (h *SalePaymentHandler) GetTransaction(c *gin.Context) {
	var res domain.Transaction
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a transaction
// @Description Get a transaction from the request body
// @Tags transactions
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transaction/list [get]
func (h *SalePaymentHandler) ListTransaction(c *gin.Context) {
	res := []*domain.Transaction{}
	if err := h.db.Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a transaction
// @Description Update a transaction from the request body
// @Tags transactions
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "transaction ID"
// @Param transaction body domain.TransactionRequest true "transaction"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transaction/:id [put]
func (h *SalePaymentHandler) UpdateTransaction(c *gin.Context) {
	var (
		body domain.TransactionRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.WithContext(c.Request.Context()).
		Table("transactions").Updates(&body).
		Where("id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a transaction
// @Description Delete a transaction from the request body
// @Tags transactions
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "transaction ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transaction/:id [delete]
func (h *SalePaymentHandler) DeleteTransaction(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).
		Table("transactions").
		Delete(&domain.Transaction{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
