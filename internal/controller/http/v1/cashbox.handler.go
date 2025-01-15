package v1

import (
	"errors"
	"fmt"

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
		cashBox.DELETE("/:id", h.Delete)
		cashBox.GET("/check", h.CheckCashBox)
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
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.IsOpen = false
	// Save to database
	err = h.db.
		WithContext(c.Request.Context()).
		Table("cash_boxes").
		Create(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("failed to create cash box: %v", err))
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
	err = h.db.First(&body, "id = ?", id).Error
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/list [get]
func (h *CashBoxHandler) List(c *gin.Context) {
	var (
		body    []domain.CashBox
		err     error
		storeID = c.Query("store_id")
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
	err = query.
		Where("is_enable = ?", true).
		Limit(limit).Offset(offset).
		Find(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
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
	cashBox := domain.CashBox{
		ID:       body.ID,
		Name:     body.Name,
		StoreID:  body.StoreID,
		IsOpen:   body.IsOpen,
		IsEnable: body.IsEnable,
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.CashBox{}).
		Where("id = ?", id).
		Updates(&cashBox).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Delete a cash box
// @Description Delete a cash box from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cash box ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/{id} [delete]
func (h *CashBoxHandler) Delete(c *gin.Context) {
	var (
		body domain.CashBox
		err  error
		id   = c.Param("id")
	)
	err = h.db.Delete(&body, "id = ?", id).Error
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
// @Param 	store_id query string true "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/check [get]
func (h *CashBoxHandler) CheckCashBox(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	var storeID = c.Query("store_id")
	var checkCashBox domain.CashBoxCheckResponse
	err := h.db.Raw(`
	SELECT co.is_open, co.id AS cash_box_operation_id 
	FROM cashbox_operations co
	JOIN cash_boxes cb ON co.cash_box_id = cb.id
	WHERE cb.store_id = ? AND co.employee_id = ?
	`, storeID, userID).Scan(&checkCashBox).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	var sale domain.Sale
	if checkCashBox.IsOpen {
		err = h.db.First(&sale, "employee_id = ? AND status = ?", userID, "pending").Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				saleRequest := domain.SaleRequest{
					ID:                 uuid.New().String(),
					EmployeeID:         userID.(string),
					CashBoxOperationId: checkCashBox.CashBoxOperationID,
					SaleNumber:         utils.GenerateCode(),
				}
				err = h.db.Create(&saleRequest).Error
				if err != nil {
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
				checkCashBox.SaleID = saleRequest.ID
				handleResponse(c, OK, checkCashBox)
				return
			} else {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
		checkCashBox.SaleID = sale.ID
	}
	handleResponse(c, OK, checkCashBox)
}
