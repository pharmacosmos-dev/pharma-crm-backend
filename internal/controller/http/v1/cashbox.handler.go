package v1

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type CashBoxHandler struct {
	*Handler
}

func (h *Handler) NewCashBoxHandler(r *gin.RouterGroup) {
	cashBox := &CashBoxHandler{h}
	cashBox.CashBoxRoutes(r)
}

func (h *CashBoxHandler) CashBoxRoutes(r *gin.RouterGroup) {
	cashBox := r.Group("/cash_box")
	{
		cashBox.POST("", h.Create)
		cashBox.GET("/:id", h.Get)
		cashBox.GET("/list", h.List)
		cashBox.PUT("/:id", h.Update)
		cashBox.GET("/check", h.CheckCashBox)
		cashBox.DELETE("/hard-delete", h.HardDelete)
		cashBox.DELETE("/soft-delete", h.SoftDelete)
	}
}

// Create godoc
// @Summary Create a cash box
// @Description Create a cash box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CashBoxRequest true "Cash box information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box [post]
func (h *CashBoxHandler) Create(c *gin.Context) {
	var (
		body domain.CashBoxRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	body.ID = uuid.New().String()
	body.IsOpen = false
	// Save to database
	err = tx.
		WithContext(c.Request.Context()).
		Table("cash_boxes").
		Create(&body).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(body.PaymentTypes) > 0 {
		for i := range body.PaymentTypes {
			body.PaymentTypes[i].CashBoxId = body.ID
		}
		err = tx.WithContext(c.Request.Context()).
			Table("cashbox_payment_types").
			Create(&body.PaymentTypes).Error
		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a cash register
// @Description Get a cash register from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/{id} [get]
func (h *CashBoxHandler) Get(c *gin.Context) {
	var (
		body domain.CashBox
		err  error
		id   = c.Param("id")
	)
	err = h.db.
		Preload("Store").
		First(&body, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// List godoc
// @Summary Get a cash register
// @Description Get a cash register from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/list [get]
func (h *CashBoxHandler) List(c *gin.Context) {
	var (
		res        []domain.CashBox
		totalCount int64
		err        error
		storeID    = c.Query("store_id")
		search     = c.Query("search")
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	query := h.db.
		Model(&domain.CashBox{}).
		Preload("Store")

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	err = query.
		Where("deleted_at IS NULL").
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result, totalCount)
}

// Update godoc
// @Summary Update a cash box
// @Description Update a cash box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box ID"
// @Param input body domain.CashBoxRequest true "Cash box information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/{id} [put]
func (h *CashBoxHandler) Update(c *gin.Context) {
	var (
		body domain.CashBoxRequest
		err  error
		id   = c.Param("id")
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("cash_boxes").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Update cashbox_payment_types
	if len(body.PaymentTypes) > 0 {
		for _, pt := range body.PaymentTypes {
			cashboxPaymentType := domain.CashboxPaymentType{
				CashBoxId:     id,
				PaymentTypeId: pt.PaymentTypeId,
				IsActive:      pt.IsActive,
			}

			// Upsert logic (insert or update)
			err = h.db.WithContext(c.Request.Context()).
				Table("cashbox_payment_types").
				Where("cash_box_id = ? AND payment_type_id = ?", id, pt.PaymentTypeId).
				Assign(cashboxPaymentType).
				Debug(). // Assign updates if record exists
				FirstOrCreate(&cashboxPaymentType).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
	}

	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Hard Delete a cash box
// @Description Hard Delete a cash box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id body []string true "cash box IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/hard-delete [delete]
func (h *CashBoxHandler) HardDelete(c *gin.Context) {
	var (
		ids []string
		err error
	)
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Delete(&domain.CashBox{}, "id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Delete godoc
// @Summary Soft Delete a cash box
// @Description Soft Delete a cash box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id 	body []string true "cash box IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/soft-delete [delete]
func (h *CashBoxHandler) SoftDelete(c *gin.Context) {
	var (
		ids []string
		err error
	)
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("cash_boxes").
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_enable":  false,
			"deleted_at": time.Now()}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// CheckCashBox go docs
// @Summary Check Cash Box is open or not
// @Description Check Cash Box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/check [get]
func (h *CashBoxHandler) CheckCashBox(c *gin.Context) {
	// Get the user ID from the context
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	// Check if there is an open cashbox operation for this employee
	var cashBoxOperationID string
	err := h.db.Raw(`
		SELECT id 
		FROM cashbox_operations
		WHERE current_employee_id = ? 
		AND end_time IS NULL
	`, userID).Scan(&cashBoxOperationID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to check cash box operations")
		return
	}
	// Prepare the response object
	var checkCashBox domain.CashBoxCheckResponse
	checkCashBox.CashBoxOperationID = cashBoxOperationID

	// If a cashbox operation exists
	if cashBoxOperationID != "" {
		// Check for a pending sale linked to this cashbox operation
		var sale domain.Sale
		err := h.db.Where("status = ? AND cash_box_operation_id = ?", "pending", cashBoxOperationID).First(&sale).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No pending sale found; create a new one
				newSale := domain.SaleRequest{
					CashBoxOperationId: cashBoxOperationID,
					EmployeeID:         userID.(string),
					ID:                 uuid.New().String(),
				}
				if createErr := h.db.Table("sales").Create(&newSale).Error; createErr != nil {
					h.log.Error(createErr)
					handleResponse(c, InternalError, "Failed to create new sale")
					return
				}

				// Set the new sale ID in the response
				checkCashBox.SaleID = newSale.ID
				checkCashBox.IsOpen = true
				handleResponse(c, OK, checkCashBox)
				return
			}
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to check for pending sale")
			return
		}

		// If a pending sale exists
		checkCashBox.SaleID = sale.ID
		checkCashBox.IsOpen = true
		handleResponse(c, OK, checkCashBox)
		return
	}

	// No open cashbox operation found
	handleResponse(c, OK, checkCashBox)
}
