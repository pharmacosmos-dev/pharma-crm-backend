package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type CartItemHandler struct {
	*Handler
}

func (h *Handler) NewCartItemHandler(r *gin.RouterGroup) {
	cartItem := &CartItemHandler{h}
	cartItem.CartItemRoutes(r)
}

func (h *CartItemHandler) CartItemRoutes(r *gin.RouterGroup) {
	cartItem := r.Group("/cart_item")
	{
		cartItem.POST("", h.Create)
		cartItem.GET("/:id", h.Get)
		cartItem.GET("/list", h.List)
		cartItem.PUT("/:id", h.Update)
		cartItem.DELETE("/:id", h.Delete)
		cartItem.POST("/multiple", h.MultipleDelete)
		cartItem.PUT("/sale/:sale_id", h.UpdateBySaleID)
	}
}

// Create godoc
// @Summary Create a cart item
// @Description Create a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.CartItemRequest true "Cart item information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item [post]
func (h *CartItemHandler) Create(c *gin.Context) {
	var (
		body domain.CartItemRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.First(&domain.CartItem{},
		"store_product_id = ? AND sale_id = ? AND is_drafted = false AND status = 'pending'",
		body.StoreProductID, body.SaleId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			body.TotalPrice = body.UnitPrice * float64(body.Quantity)
			body.ID = uuid.New().String()
			body.Status = config.PENDING_CART_ITEM
			err = h.db.
				WithContext(c.Request.Context()).
				Table("cart_items").
				Create(&body).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			handleResponse(c, CREATED, "CREATED")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.CartItem{}).
		Where("store_product_id = ? AND sale_id = ? AND is_drafted = false AND status = 'pending'", body.StoreProductID, body.SaleId).
		Updates(map[string]interface{}{
			"quantity":    gorm.Expr("quantity + ?", 1),
			"total_price": body.UnitPrice * float64(body.Quantity+1),
		}).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a cart item
// @Description Get a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cart item ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id} [get]
func (h *CartItemHandler) Get(c *gin.Context) {
	var (
		cartItem domain.CartItem
		err      error
		id       = c.Param("id")
	)
	err = h.db.
		WithContext(c.Request.Context()).
		Where("id = ?", id).
		First(&cartItem).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, cartItem)
}

// List godoc
// @Summary Get a cart item
// @Description Get a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param sale_id query string true "Sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/list [get]
func (h *CartItemHandler) List(c *gin.Context) {
	var (
		res    *domain.CartItemData
		saleID = c.Query("sale_id")
		err    error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res, err = h.storage.CartItemList(saleID, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// UpdateBySaleID godoc
// @Summary Update a cart item
// @Description Update a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param sale_id path string true "Sale ID"
// @Param input body domain.CartItemBySaleIDUpdateRequest true "Cart item information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/sale/{sale_id} [put]
func (h *CartItemHandler) UpdateBySaleID(c *gin.Context) {
	var body domain.CartItemBySaleIDUpdateRequest
	var saleId = c.Param("sale_id")
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Chegirma hisoblash va yangilash uchun SQL so'rovi
	err = h.db.WithContext(c.Request.Context()).
		Exec(`
        UPDATE cart_items
        SET
            discount_type = ?,
            discount_value = ?,
            discount_price = CASE
                WHEN discount_value = 0 THEN 0
                WHEN discount_type = 'percent' THEN unit_price - (unit_price * ? / 100)
                WHEN discount_type = 'cash' THEN unit_price - ?
                ELSE unit_price
            END,
            discount_amount = CASE
                WHEN discount_value = 0 THEN 0
                WHEN discount_type = 'percent' THEN (unit_price * ? / 100)
                WHEN discount_type = 'cash' THEN ?
                ELSE 0
            END,
            total_discount_price = discount_amount * quantity,
            updated_at = NOW()
        WHERE sale_id = ?;
    `, body.DiscountType, body.DiscountValue,
			body.DiscountValue, body.DiscountValue,
			body.DiscountValue, body.DiscountValue,
			saleId).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// UpdateProductItem godoc
// @Summary Update a cart item
// @Description Update a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cart item ID"
// @Param input body domain.CartItemUpdateProductUnit true "Cart item information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id} [put]
func (h *CartItemHandler) Update(c *gin.Context) {
	var (
		body domain.CartItemUpdateProductUnit
		err  error
		id   = c.Param("id")
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var cartItem domain.CartItem
	err = h.db.
		WithContext(c.Request.Context()).
		Where("id = ?", id).
		First(&cartItem).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	var data = map[string]interface{}{}
	if body.Quantity != nil {
		data["quantity"] = body.Quantity
		data["total_price"] = cartItem.UnitPrice * float64(*body.Quantity)
	}
	if body.UnitQuantity != nil {
		data["unit_quantity"] = body.UnitQuantity
	}

	err = h.db.
		WithContext(c.Request.Context()).
		Table("cart_items").
		Where("id = ?", id).
		Updates(data).Error
	if err != nil {
		handleResponse(c, InternalError, "Error on saving cart")
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete godoc
// @Summary Delete a cart item
// @Description Delete a cart item from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cart item ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id} [delete]
func (h *CartItemHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.CartItem{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// MultipleDelete godoc
// @Summary Delete all cart items
// @Description Delete all cart items from the request body
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.Ids true "cart item IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/multiple [post]
func (h *CartItemHandler) MultipleDelete(c *gin.Context) {
	var (
		body domain.Ids
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.Delete(&domain.CartItem{}, "id in (?)", body.Ids).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
