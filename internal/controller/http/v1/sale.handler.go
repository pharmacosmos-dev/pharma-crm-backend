package v1

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type SaleHandler struct {
	*Handler
}

func (h *Handler) NewSaleHandler(r *gin.RouterGroup) {
	sale := &SaleHandler{h}
	sale.SaleRoutes(r)
}

func (h *SaleHandler) SaleRoutes(r *gin.RouterGroup) {
	sale := r.Group("/sale")
	{
		sale.POST("", h.Create)
		sale.GET("/:id", h.Get)
		sale.GET("/list", h.List)
		sale.PUT("/:id", h.Update)
		sale.DELETE("/:id", h.Delete)
		sale.POST("/final", h.FinalSale)
		sale.GET("/check", h.CheckSale)
	}
}

// Create godoc
// @Summary Create a sale
// @Description Create a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.SaleRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale [post]
func (h *SaleHandler) Create(c *gin.Context) {
	var (
		body domain.SaleRequest
		res  domain.Sale
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.SaleNumber = utils.GenerateCode()
	err = h.db.
		WithContext(c.Request.Context()).
		Raw(`
		INSERT INTO sales (id, employee_id, cash_box_operation_id, sale_number)
		VALUES (?, ?, ?, ?) RETURNING *`,
			body.ID, body.EmployeeID, body.CashBoxOperationId, body.SaleNumber).
		Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, res)
}

// Get godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [get]
func (h *SaleHandler) Get(c *gin.Context) {
	var (
		res domain.Sale
		id  = c.Param("id")
	)
	err := h.db.
		Preload("Employee").
		Preload("CashBox", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store")
		}).
		Preload("Customer").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).
		Preload("CartItems", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Product")
		}).First(&res, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, nil)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param employee_id query string false "Employee ID"
// @Param cash_box_id query string false "Cash Box ID"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list [get]
func (h *SaleHandler) List(c *gin.Context) {
	var totalAmount int64
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	res := []domain.Sale{}
	query := h.db.Model(&domain.Sale{}).
		Preload("Employee").
		Preload("CashBox", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store")
		}).
		Preload("Customer").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		})

	if employeeID := c.Query("employee_id"); employeeID != "" {
		query = query.Where("employee_id = ?", employeeID)
	}
	if cashBoxID := c.Query("cash_box_id"); cashBoxID != "" {
		query = query.Where("cash_box_id = ?", cashBoxID)
	}
	if startDate != "" && endDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	}

	err = query.Where("status = ?", "completed").
		Count(&totalAmount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	data := utils.ListResponse(res, totalAmount, limit, offset)
	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a sale
// @Description Update a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Param input body domain.SaleUpdateRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [put]
func (h *SaleHandler) Update(c *gin.Context) {
	var (
		body domain.SaleUpdateRequest
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("sales").
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
// @Summary Delete a sale
// @Description Delete a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [delete]
func (h *SaleHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Sale{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// FinalSale
// @Summary Final Sale
// @Description Final Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	var (
		body domain.FinalSale
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	err = h.db.Exec(`
	UPDATE sales SET status = 'completed', total_amount = ? WHERE id = ? RETURNING cash_box_operation_id`,
		body.TotalAmount, body.SaleID).Scan(&body.CashBoxOperationId).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.
		WithContext(c.Request.Context()).
		Table("cart_items").
		Where("id = ?", body.SaleID).
		Update("status", "sold").Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	var salePayment []domain.SalePaymentRequest
	for _, item := range body.PaymentTypes {
		salePayment = append(salePayment, domain.SalePaymentRequest{
			ID:                 uuid.New().String(),
			SaleID:             body.SaleID,
			CashBoxOperationID: body.CashBoxOperationId,
			PaymentTypeID:      item.PaymentTypeID,
			Amount:             item.Amount,
			PaidAt:             time.Now().Format("2006-01-02 15:04:05"),
			Status:             "paid",
			TransactionID:      uuid.New().String(),
		})
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sale_payments").
		Create(&salePayment).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	newSale := domain.SaleRequest{
		ID:                 uuid.New().String(),
		EmployeeID:         cast.ToString(userID),
		SaleNumber:         utils.GenerateCode(),
		CashBoxOperationId: body.CashBoxOperationId,
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("sales").
		Create(&newSale).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, newSale.ID)
}

// CheckSale
// @Summary Check Sale
// @Description Check cash box from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	store_id query string true "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/check [get]
func (h *SaleHandler) CheckSale(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	var res domain.Sale
	err := h.db.First(&res, "employee_id = ? AND status = ?", userID, "pending").Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, nil)
			return
		}
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
