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
	salePayment := r.Group("/sale-payment")
	{
		salePayment.POST("", h.Create)
		salePayment.GET("/:id", h.Get)
		salePayment.GET("/list", h.List)
		salePayment.PUT("/:id", h.Update)
		salePayment.DELETE("/:id", h.Delete)
		salePayment.GET("/list/close-cashbox/:cash_box_operation_id", h.ListByCashBoxId)
		salePayment.PUT("/amounts/:id", h.UpdateAmounts)
		salePayment.GET("/total-amount/:cash_box_id", h.GetTotalAmount)
	}
	transaction := r.Group("/transaction")
	{
		transaction.POST("", h.CreateTransaction)
		transaction.GET("/:id", h.GetTransaction)
		transaction.GET("/list", h.ListTransaction)
		transaction.PUT("/:id", h.UpdateTransaction)
		transaction.DELETE("/:id", h.DeleteTransaction)
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
// @Router /sale-payment [post]
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
	err = h.db.
		WithContext(c.Request.Context()).
		Table("sale_payments").
		Create(&body).Error
	if err != nil {
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
// @Router /sale-payment/{id} [get]
func (h *SalePaymentHandler) Get(c *gin.Context) {
	var res domain.SalePayment
	err := h.db.First(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
// @Router /sale-payment/list [get]
func (h *SalePaymentHandler) List(c *gin.Context) {
	res := []*domain.SalePayment{}
	err := h.db.Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// ListByCashBoxId godoc
// @Summary Get a sale payment
// @Description Get a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	cash_box_operation_id path string true "cash box operation ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale-payment/list/close-cashbox/{cash_box_operation_id} [get]
func (h *SalePaymentHandler) ListByCashBoxId(c *gin.Context) {
	var cashBoxOperationId = c.Param("cash_box_operation_id")
	res := []*domain.SalePaymentCloseCashBox{}
	err := h.db.Raw(`
	SELECT 
		sp.id,
		pt.name,
		sp.amount,
		sp.net_amount,
		sp.expense_amount,
		(sp.net_amount - sp.amount) as difference_amount 
	FROM 
		sale_payments sp
	RIGHT JOIN
		payment_types pt ON sp.payment_type_id = pt.id
	WHERE sp.cash_box_status = 'open' AND sp.cash_box_operation_id = ?
	ORDER BY pt.created_at
	`, cashBoxOperationId).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	var totalData domain.SalePaymentTotalAmount
	err = h.db.Raw(`
	SELECT 
		sum(sp.amount) as total_amount, 
		sum(sp.net_amount) as total_net_amount, 
		sum(sp.expense_amount) as total_expense_amount, 
		sum(sp.net_amount - sp.amount) as total_difference_amount
	FROM
		sale_payments sp
	RIGHT JOIN
		payment_types pt ON sp.payment_type_id = pt.id
	WHERE sp.cash_box_status = 'open' AND sp.cash_box_operation_id = ?
	`, cashBoxOperationId).Scan(&totalData).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := map[string]interface{}{
		"total_data": totalData,
		"data":       res,
	}
	handleResponse(c, OK, result)
}

// GetTotalAmount godoc
// @Summary Get a sale payment
// @Description Get a sale payment from the request body
// @Tags sale_payments
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	cash_box_operation_id path string true "cash box ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale-payment/total-amount/{cash_box_operation_id} [get]
func (h *SalePaymentHandler) GetTotalAmount(c *gin.Context) {
	var cashBoxID = c.Param("cash_box_operation_id")
	var totalData map[string]interface{}

	err := h.db.Raw(`
	SELECT
		SUM(CASE WHEN pt.type = 'cash' THEN sp.net_amount ELSE 0 END) AS cash_amount,
		SUM(CASE WHEN pt.type != 'cash' THEN sp.net_amount ELSE 0 END) AS cashless_amount
	FROM
		sale_payments sp
	JOIN
		payment_types pt ON sp.payment_type_id = pt.id
	WHERE 
		sp.cash_box_status = 'open' AND 
		sp.cash_box_operation_id = ?;
	`, cashBoxID).Scan(&totalData).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, totalData)
}

// Update godoc
// @Summary Update a sale payment
// @Description Update a sale payment from the request body
// @Tags 	sale_payments
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "sale payment ID"
// @Param 	sale_payment body domain.SalePaymentRequest true "sale payment"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale-payment/{id} [put]
func (h *SalePaymentHandler) Update(c *gin.Context) {
	var (
		body domain.SalePaymentRequest
		id   = c.Param("id")
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sale_payments").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// UpdateAmounts godoc
// @Summary Update a sale payment
// @Description Update a sale payment amounts from the request body
// @Tags 	sale_payments
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "sale payment ID"
// @Param 	sale_payment body domain.SalePaymentUpdateAmount true "sale payment"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale-payment/amounts/{id} [put]
func (h *SalePaymentHandler) UpdateAmounts(c *gin.Context) {
	var (
		body domain.SalePaymentUpdateAmount
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sale_payments").
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
// @Router /sale-payment/{id} [delete]
func (h *SalePaymentHandler) Delete(c *gin.Context) {
	var (
		id  = c.Param("id")
		err error
	)
	err = h.db.WithContext(c.Request.Context()).
		Delete(&domain.SalePayment{}, "id = ?", id).Error
	if err != nil {
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
	err = h.db.WithContext(c.Request.Context()).
		Table("transactions").
		Create(&body).Error
	if err != nil {
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
// @Router /transaction/{id} [get]
func (h *SalePaymentHandler) GetTransaction(c *gin.Context) {
	var res domain.Transaction
	var id = c.Param("id")
	err := h.db.First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
	err := h.db.Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
// @Router /transaction/{id} [put]
func (h *SalePaymentHandler) UpdateTransaction(c *gin.Context) {
	var (
		body domain.TransactionRequest
		err  error
		id   = c.Param("id")
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("transactions").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
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
// @Router /transaction/{id} [delete]
func (h *SalePaymentHandler) DeleteTransaction(c *gin.Context) {
	id := c.Param("id")
	err := h.db.WithContext(c.Request.Context()).
		Table("transactions").
		Delete(&domain.Transaction{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
