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
	"gorm.io/gorm/clause"
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
	}
}

// Create godoc
// @Summary Create a sale
// @Description Create a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.SaleRequest true "Sale information"
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
	user, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.EmployeeID = cast.ToString(user)
	err = h.db.
		WithContext(c.Request.Context()).
		Raw(`
		INSERT INTO sales (id, employee_id, cash_box_operation_id)
		VALUES (?, ?, ?) RETURNING *`,
			body.ID, body.EmployeeID, body.CashBoxOperationId).
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
		Preload("CartItems").First(&res, "id = ?", id).Error
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

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update sale status
	err = updateSaleStatus(tx, body.SaleID, body.TotalAmount)
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Update cart items
	err = updateCartItemStatus(tx, body.SaleID)
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	now := time.Now()
	// Insert sale payments
	var salePayments []domain.SalePaymentRequest
	for _, item := range body.PaymentTypes {
		salePayments = append(salePayments, domain.SalePaymentRequest{
			ID:                 uuid.New().String(),
			SaleID:             body.SaleID,
			CashBoxOperationID: body.CashBoxOperationId,
			PaymentTypeID:      item.PaymentTypeID,
			Amount:             item.Amount,
			Status:             "paid",
			PaidAt:             &now,
		})
	}
	err = tx.
		Table("sale_payments").
		Create(&salePayments).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	for _, salePayment := range salePayments {
		// `OnConflict` yordamida yaratish yoki yangilash
		err = tx.Table("sale_payment_summary").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "cash_box_operation_id"}, {Name: "payment_type_id"}}, // Unikal kalitlar
				DoUpdates: clause.Assignments(map[string]interface{}{
					"total_amount":         gorm.Expr("sale_payment_summary.total_amount + ?", salePayment.Amount),
					"total_expense_amount": 0,
					"total_net_amount":     0,
					"total_difference":     0,
					"updated_at":           time.Now(),
				}),
			}).Create(&domain.SalePaymentSummary{
			CashBoxOperationID: salePayment.CashBoxOperationID,
			PaymentTypeID:      salePayment.PaymentTypeID,
			TotalAmount:        salePayment.Amount,
			TotalExpenseAmount: 0,
			TotalNetAmount:     0,
			TotalDifference:    0,
			CreatedAt:          &now,
			UpdatedAt:          &now,
		}).Error

		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to update sale_payment_summary")
			return
		}
	}

	newSale := domain.SaleRequest{
		ID:                 uuid.New().String(),
		EmployeeID:         cast.ToString(userID),
		CashBoxOperationId: body.CashBoxOperationId,
	}
	err = tx.
		WithContext(c.Request.Context()).
		Table("sales").
		Create(&newSale).Error
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, newSale.ID)
}

func updateSaleStatus(tx *gorm.DB, saleID string, totalAmount float64) error {
	return tx.
		Table("sales").
		Where("id = ?", saleID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"total_amount": totalAmount,
		}).Error
}

func updateCartItemStatus(tx *gorm.DB, saleID string) error {
	return tx.
		Table("cart_items").
		Where("sale_id = ?", saleID).
		Updates(map[string]interface{}{"status": "sold"}).Debug().Error
}
