package v1

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
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
		cashBox.GET("/open-list", h.OpenCashboxList)
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
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	if body.StoreID == "" {
		handleResponse(c, BadRequest, "store_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
		}
	}()
	body.ID = uuid.New().String()
	body.IsOpen = false
	// Save to database
	err := tx.
		WithContext(ctx).
		Table("cash_boxes").
		Create(&body).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Errorf("could not create cashbox: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	if len(body.PaymentTypes) > 0 {
		for i := range body.PaymentTypes {
			body.PaymentTypes[i].CashBoxId = body.ID
		}
		err = tx.WithContext(ctx).
			Table("cashbox_payment_types").
			Create(&body.PaymentTypes).Error
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not create cashbox_payment_type: %v", err)
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
	}
	if err = tx.Commit().Error; err != nil {
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var (
		res        []domain.CashboxOpenData
		totalCount int64
		err        error
		storeID    = c.Query("store_id")
		companyID  = c.Query("company_id")
		search     = c.Query("search")
		filter     = " WHERE c.is_active = true "
		args       = []any{}
	)

	// check employee role
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			storeID = user.StoreId
		}
		companyID = user.CompanyId
	}

	// get limit, offset with getting or default
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// build query
	query := `
	SELECT
		c.id,
		c.store_id,
		c.name,
		c.is_active,
		s.name AS store_name,
		COALESCE(
			(SELECT co.is_open
			FROM cashbox_operations co
			WHERE co.cash_box_id = c.id
			ORDER BY co.created_at DESC
			LIMIT 1),
			FALSE
		) AS is_open,
		COALESCE(
			(SELECT e.full_name
			FROM cashbox_operations co
			JOIN employees e ON e.id = co.current_employee_id
			WHERE co.cash_box_id = c.id
			AND co.is_open = TRUE
			ORDER BY co.created_at DESC
			LIMIT 1),
			''
		) AS full_name,
		COUNT(*) OVER() AS total_count
	FROM
		cash_boxes c
	JOIN stores s ON c.store_id = s.id
	`
	if storeID != "" {
		filter += " AND c.store_id = ? "
		args = append(args, storeID)
	}
	if companyID != "" {
		filter += " AND s.company_id = ? "
		args = append(args, companyID)
	}
	if search != "" {
		search = "%" + search + "%"
		filter += " AND c.name ILIKE ? "
		args = append(args, search)
	}
	query = query + filter + "ORDER BY c.created_at DESC " + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	// complete query
	err = h.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		h.log.Warn("ERROR on getting cashbox list: %v", err)
		handleResponse(c, InternalError, "Can't get cashbox list")
		return
	}
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}
	// build response with _meta
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
	// validate request id
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update cashbox info
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
			// Upsert logic (insert or update)
			err = h.db.WithContext(c.Request.Context()).
				Table("cashbox_payment_types").
				Where("cash_box_id = ? AND payment_type_id = ?", id, pt.PaymentTypeId).
				Update("is_active", pt.IsActive).Error
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
	// bind request body
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// hard delete
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
	// bind request body
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// soft delete
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
// @Accept 	json
// @Produce json
// @Param 	store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/check [get]
func (h *CashBoxHandler) CheckCashBox(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	storeId := c.Query("store_id")

	if storeId != user.StoreId {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// Check if there is an open cashbox operation for this employee
	var cashboxOperation domain.CashboxOperation
	err := h.db.WithContext(ctx).Raw(`
	SELECT 
		co.* 
	FROM 
		cashbox_operations co 
    JOIN 
		cash_boxes cb ON co.cash_box_id = cb.id 
    WHERE 
		co.end_time IS NULL AND 
		co.current_employee_id = ? AND 
		cb.store_id = ?;`, user.UserId, storeId).Scan(&cashboxOperation).Error

	if errors.Is(err, gorm.ErrRecordNotFound) || cashboxOperation.ID == "" {
		handleResponse(c, NotFound, "You have no open cashbox operation")
		return
	} else if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to check cash box operations")
		return
	}
	// Prepare the response object
	var checkCashBox domain.CashBoxCheckResponse
	checkCashBox.CashBoxOperationID = cashboxOperation.ID

	// If a cashbox operation exists
	if cashboxOperation.ID != "" {
		// Check for a pending sale linked to this cashbox operation
		var sale *domain.Sale
		sale, err = h.service.CreateSale(ctx, h.db, &domain.SaleRequest{
			CashBoxOperationId: cashboxOperation.ID,
			EmployeeId:         user.UserId,
			StoreId:            user.StoreId,
			CashboxId:          cashboxOperation.CashBoxID,
		})
		if err != nil {
			h.log.Warn("ERROR on creating new sale: %v", err)
			handleResponse(c, InternalError, "Can't create new sale on cheching cashbox")
			return
		}
		// If a pending sale exists
		checkCashBox.SaleID = sale.Id
		checkCashBox.IsOpen = true
		handleResponse(c, OK, checkCashBox)
		return
	}

	// No open cashbox operation found
	handleResponse(c, OK, checkCashBox)
}

// OpenCashboxList godoc
// @Summary Get open cashbox list
// @Description Get open cashbox list from the request body
// @Tags cash_boxes
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cash_box/open-list [get]
func (h *CashBoxHandler) OpenCashboxList(c *gin.Context) {
	var (
		res []domain.CashBox
	)
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Error("User ID not found")
		handleResponse(c, InternalError, "User ID not found")
		return
	}
	err := h.db.Raw(`
	SELECT cb.* 
	FROM cashbox_operations co 
	JOIN cash_boxes cb ON co.cash_box_id = cb.id 
	WHERE co.current_employee_id = ? AND co.end_time IS NULL`, userId).
		Scan(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
