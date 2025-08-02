package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
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
		cartItems []domain.CartItem
		err       error
	)
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// start transaction
	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	// update sale status to drafted
	var saleInfo domain.Sale
	err = tx.WithContext(ctx).Raw(`UPDATE sales SET status = 'drafted' WHERE id = ? RETURNING *`, body.SaleID).Scan(&saleInfo).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Error updating sale status")
		return
	}
	// get cart_item list by sale_id
	err = tx.WithContext(ctx).
		Where("sale_id = ?", body.SaleID).
		Find(&cartItems).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check cart_items status
	if len(cartItems) < 1 {
		handleResponse(c, BadRequest, "No items in the cart")
		return
	}
	body.ID = uuid.New().String()
	cartDrafts := []domain.CartItemDraft{}
	for _, item := range cartItems {
		cartDrafts = append(cartDrafts, domain.CartItemDraft{
			ID:         uuid.New().String(),
			CartItemID: item.ID,
			DraftID:    body.ID,
		})
		// decrease pack_quantity and unit_quantity on store_products
		err = tx.WithContext(ctx).Exec(`UPDATE store_products SET pack_quantity = GREATEST(pack_quantity - ?, 0), unit_quantity = GREATEST(unit_quantity - ?, 0) WHERE id = ?`,
			item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			h.log.Warn("ERROR on update quantity in store_products: %v", err)
			handleResponse(c, InternalError, "Can't update quantity")
			return
		}
	}
	// create new draft
	body.Status = config.PENDING
	err = tx.WithContext(ctx).Table("drafts").Create(&body).Error
	if err != nil {
		h.log.Warn("ERROR on creating new draft: %v", err)
		handleResponse(c, InternalError, "Can't create new draft")
		return
	}
	// create cart_item_draft for saving draft history
	err = tx.WithContext(ctx).
		Table("cart_item_drafts").
		Create(&cartDrafts).Error
	if err != nil {
		h.log.Warn("ERROR on creating cart_item_drafts: %v", err)
		handleResponse(c, InternalError, "Can't create cart_item_draft")
		return
	}
	// update cart items status to drafted
	err = tx.WithContext(ctx).Exec(`UPDATE cart_items SET status = 'drafted' WHERE sale_id = ?`, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on updating cart_item status: %v", err)
		handleResponse(c, InternalError, "Can't draft to cart items")
		return
	}
	// create or get sale
	res, err := h.service.CreateOrGetSale(ctx, tx, &domain.SaleRequest{
		EmployeeID:         body.CreatedBy,
		CashBoxOperationId: saleInfo.CashBoxOperationId,
		StoreId:            saleInfo.StoreId,
		CashboxId:          saleInfo.CashboxId,
	})
	if err != nil {
		h.log.Warn("ERROR on creating new sale: %v", err)
		handleResponse(c, InternalError, "Can't create new sale after drafted")
		return
	}
	// commit transcation
	err = tx.Commit().Error
	if err != nil {
		h.log.Warn("ERROR on commiting transaction: %v", err)
		handleResponse(c, InternalError, "Can't commit transcation")
		return
	}
	handleResponse(c, CREATED, res)
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
	id := c.Param("id")
	draft, err := h.service.GetDraftByID(id)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Query associated cart items
	var cartItems []domain.CartItemResponse
	err = h.db.Model(&domain.CartItem{}).
		Select(`
		cart_items.*, p.name, p.barcode, sp.bonus_percent, 
		((sp.bonus_percent*sp.retail_price)/100) as bonus_amount, 
		u.unit_name, u.short_name`).
		Joins("JOIN cart_item_drafts ON cart_item_drafts.cart_item_id = cart_items.id").
		Joins("JOIN store_products sp ON sp.id = cart_items.store_product_id").
		Joins("JOIN products p ON p.id = sp.product_id").
		Joins("LEFT JOIN unit_types u ON u.id = p.unit_type_id").
		Where("cart_item_drafts.draft_id = ?", id).
		Find(&cartItems).Error
	if err != nil {
		h.log.Error(err)
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
// @Param customer_id query string false "Customer ID"
// @Param search query string false "Search"
// @Param draft_date query string false "Draft Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/list [get]
func (h *DraftHandler) List(c *gin.Context) {
	var (
		res        []*domain.Draft
		totalCount int64
		search     = c.Query("search")
		storeID    = c.Query("store_id")
		customerID = c.Query("customer_id")
		draftDate  = c.Query("draft_date")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Base query with joins and aggregate fields
	query := h.db.Model(&domain.Draft{}).
		Preload("Store").
		Preload("Customer").
		Preload("Employee").
		Select(`drafts.*, 
                SUM(cart_items.quantity) AS quantity, 
                COALESCE(SUM(cart_items.total_price), 0) AS total_price`).
		Joins("JOIN cart_item_drafts ON cart_item_drafts.draft_id = drafts.id").
		Joins("JOIN cart_items ON cart_items.id = cart_item_drafts.cart_item_id")

	// Filters
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.
			Joins("LEFT JOIN customers ON customers.id = drafts.customer_id").
			Joins("LEFT JOIN employees e ON e.id = drafts.created_by").
			Where("CAST(drafts.draft_number AS TEXT) LIKE ? OR customers.full_name ILIKE ? OR e.full_name ILIKE ? OR ? = ANY(customers.phone)",
				search, search, search, strings.Trim(search, "%"))
	}
	if storeID != "" {
		query = query.Where("drafts.store_id = ?", storeID)
	}
	if draftDate != "" {
		// Validate the date format
		if _, err := time.Parse("2006-01-02", draftDate); err != nil {
			handleResponse(c, BadRequest, "Invalid date format")
			return
		}
		query = query.Where("drafts.draft_time::date = ?", draftDate)
	}
	if customerID != "" {
		query = query.Where("drafts.customer_id = ?", customerID)
	}

	// Execute the query
	err = query.
		Where("drafts.status = ?", "pending").
		Group("drafts.id").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("drafts.created_at DESC").
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
		id   = c.Param("id")
	)
	// validate draft id
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid draft id")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update draft info
	err = h.db.
		WithContext(c.Request.Context()).
		Table("drafts").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Warn("ERROR on updating draft: %v", err)
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
// @Param 	id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [delete]
func (h *DraftHandler) Delete(c *gin.Context) {
	var (
		id  = c.Param("id")
		err error
	)

	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	draft, err := h.service.UpdateDraftField(tx, "status", "deleted", "id", id)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	cartItems, err := h.service.ListCartItemsBySaleID(draft.SaleID)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	for _, item := range cartItems {
		err = h.service.ChangeStoreProductStock(tx, item.StoreProductID, item.Quantity, item.UnitQuantity, true)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	err = tx.Commit().Error
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
// @Param 	id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/complete/{id} [post]
func (h *DraftHandler) CompleteDraft(c *gin.Context) {
	var (
		id  = c.Param("id")
		err error
	)
	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// update draft status
	draft, err := h.service.UpdateDraftField(tx, "status", "completed", "id", id)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// update sale status to pending
	sale, err := h.service.UpdateSaleField("status", "pending", "id", draft.SaleID)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// update cart item status to pending
	_, err = h.service.UpdateCartItemField("status", "pending", "sale_id", draft.SaleID)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get cart items which are depends on the draft sale
	cartItems, err := h.service.ListCartItemsBySaleID(sale.ID)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// return product to store_products
	for _, item := range cartItems {
		err = h.service.ChangeStoreProductStock(tx, item.StoreProductID, item.Quantity, item.UnitQuantity, true)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, draft.SaleID)
}
