package v1

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type DraftHandler struct {
	*Handler
}

func (h *Handler) NewDraftHandler(r *gin.RouterGroup) {
	draft := &DraftHandler{h}
	draft.DraftRoutes(r)
}

func (h *DraftHandler) DraftRoutes(r *gin.RouterGroup) {
	draft := r.Group("/draft")
	{
		draft.POST("", h.Create)
		draft.GET("/:id", h.Get)
		draft.GET("/list", h.List)
		draft.PUT("/:id", h.Update)
		draft.DELETE("/:id", h.Delete)
		draft.PUT("/complete/:id", h.CompleteDraft)
	}
}

// Create godoc
// @Summary Create a draft
// @Description Create a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   input body domain.DraftRequest true "Draft information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft [post]
func (h *DraftHandler) Create(c *gin.Context) {
	var (
		body      domain.DraftRequest
		res       domain.Draft
		cartItems []domain.CartItem
		err       error
	)

	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sales").Where("id = ?", body.SaleID).
		Update("status", "drafted").Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Error updating sale status")
		return
	}

	body.ID = uuid.New().String()
	body.DraftNumber = utils.GenerateCode()
	err = h.db.
		Table("drafts").
		Create(&body).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.
		Where("sale_id = ?", body.SaleID).
		Find(&cartItems).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if len(cartItems) > 0 {
		cartDrafts := []domain.CartItemDraft{}
		for _, item := range cartItems {
			cartDrafts = append(cartDrafts, domain.CartItemDraft{
				ID:         uuid.New().String(),
				CartItemID: item.ID,
				DraftID:    res.ID,
			})
		}
		err = h.db.Table("cart_item_drafts").
			Create(&cartDrafts).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	err = h.db.Model(&domain.CartItem{}).
		Where("sale_id = ?", body.SaleID).
		Updates(map[string]interface{}{
			"is_drafted": true,
			"status":     config.DRAFTED_CART_ITEM,
		}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	var saleInfo domain.Sale
	err = h.db.First(&saleInfo, "id = ?", body.SaleID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	saleID := uuid.New().String()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("sales").Create(&domain.SaleRequest{
		ID:                 saleID,
		EmployeeID:         body.CreatedBy,
		CashBoxOperationId: saleInfo.CashBoxOperationId,
	}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	saleInfo.ID = saleID
	handleResponse(c, CREATED, saleInfo)
}

// Get godoc
// @Summary Get a draft
// @Description Get a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [get]
func (h *DraftHandler) Get(c *gin.Context) {
	var draft domain.Draft
	id := c.Param("id")

	// Query the draft
	err := h.db.Preload("Customer").
		Preload("Store").
		Preload("Employee").
		First(&draft, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Query associated cart items
	var cartItems []*domain.CartItem
	err = h.db.Model(&domain.CartItem{}).
		Preload("Product").
		Select("cart_items.*").
		Joins("JOIN cart_item_drafts ON cart_item_drafts.cart_item_id = cart_items.id").
		Where("cart_item_drafts.draft_id = ?", id).
		Find(&cartItems).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Attach cart items to the draft
	draft.CartItems = cartItems

	// Respond with the draft and its associated cart items
	handleResponse(c, OK, draft)
}

// List godoc
// @Summary Get a draft
// @Description Get a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search"
// @Param draft_date query string false "Draft Date"
// @Param customer_id query string false "Customer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/list [get]
func (h *DraftHandler) List(c *gin.Context) {
	var (
		res        []*domain.Draft
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Base query with joins and aggregate fields
	query := h.db.Model(&domain.Draft{}).
		Select(`drafts.*, 
                SUM(cart_items.quantity) AS quantity, 
                COALESCE(SUM(cart_items.total_price), 0) AS total_price`).
		Joins("JOIN cart_item_drafts ON cart_item_drafts.draft_id = drafts.id").
		Joins("JOIN cart_items ON cart_items.id = cart_item_drafts.cart_item_id").
		Group("drafts.id").
		Preload("Store").Preload("Customer").Preload("Employee")

	// Filters
	if search := c.Query("search"); search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Joins("LEFT JOIN customers ON customers.id = drafts.customer_id").
			Where("customers.phone LIKE ? OR customers.first_name ILIKE ?", search, search)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if draftDate := c.Query("draft_date"); draftDate != "" {
		// Validate the date format
		if _, err := time.Parse("2006-01-02", draftDate); err != nil {
			handleResponse(c, BadRequest, "Invalid date format")
			return
		}
		query = query.Where("drafts.draft_time::date = ?", draftDate)
	}
	if customerID := c.Query("customer_id"); customerID != "" {
		query = query.Where("drafts.customer_id = ?", customerID)
	}

	// Execute the query
	err = query.Limit(limit).Offset(offset).
		Count(&totalCount).Order("drafts.created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare and send the response
	data := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a draft
// @Description Update a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Param   input body domain.DraftRequest true "Draft information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [put]
func (h *DraftHandler) Update(c *gin.Context) {
	var (
		body domain.DraftRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("drafts").Where("id = ?", c.Param("id")).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Delete a draft
// @Description Delete a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [delete]
func (h *DraftHandler) Delete(c *gin.Context) {
	var id = c.Param("id")

	err := h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.CartItemDraft{}, "draft_id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.Draft{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// CompleteDraft
// @Summary Complete a draft
// @Description Complete a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/complete/{id} [post]
func (h *DraftHandler) CompleteDraft(c *gin.Context) {
	var (
		id  = c.Param("id")
		res domain.Draft
		err error
	)
	err = h.db.WithContext(c.Request.Context()).
		First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sales").Where("id = ?", res.SaleID).
		Update("status", "pending").Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.WithContext(c.Request.Context()).
		Table("cart_items").
		Where("sale_id = ?", res.SaleID).
		Updates(map[string]interface{}{
			"is_drafted": false,
			"status":     "pending"}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res.SaleID)
}
