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
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get store product
	var storeProduct domain.StoreProduct
	err = h.db.First(&storeProduct, "id = ?", body.StoreProductID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get cart item
	var cartItem domain.CartItem
	err = h.db.First(&cartItem,
		"store_product_id = ? AND sale_id = ? AND is_drafted = false AND status = 'pending'",
		body.StoreProductID, body.SaleId).Error
	if err == nil {
		if storeProduct.PackQuantity < cartItem.Quantity+1 {
			handleResponse(c, CONFLICT, gin.H{
				"message":                "Not enough Product",
				"pack_quantity":          storeProduct.PackQuantity,
				"unit_quantity":          storeProduct.UnitQuantity,
				"received_pack_quantity": cartItem.Quantity,
				"received_unit_quantity": cartItem.UnitQuantity,
			})
			return
		}
		cartItem.Quantity++
		cartItem.TotalPrice = cartItem.UnitPrice * float64(cartItem.Quantity)
		err = h.db.Debug().Exec(`UPDATE cart_items SET quantity = ?, total_price = ? WHERE id = ?`,
			cartItem.Quantity, cartItem.TotalPrice, cartItem.ID).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "CREATED")
		return
	} else if errors.Is(err, gorm.ErrRecordNotFound) && storeProduct.PackQuantity > 0 {
		err = h.db.Exec(`
		INSERT INTO cart_items(
			id, store_product_id, 
			sale_id, employee_id, 
			quantity, unit_price, 
			total_price, status) 
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), body.StoreProductID, body.SaleId, userId.(string), 1,
			body.UnitPrice, body.UnitPrice, config.PENDING_CART_ITEM).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "CREATED")
		return
	}
	h.log.Info("ERROR on creating cart_item: %v", err.Error())

	handleResponse(c, CONFLICT, err.Error())
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
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var cartItems []domain.CartItem
	err = tx.Where("sale_id = ?", saleId).Find(&cartItems).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to fetch cart items")
		tx.Rollback()
		return
	}
	var discountPercent float64
	for i := range cartItems {
		// 1 pochka -> 10 000 so'm -> 1000 so'm discount
		// 1 dona -> 200 so'm - 20 so'm discount
		if body.DiscountType == "percent" && body.DiscountValue <= 100 {
			cartItems[i].DiscountAmount = cartItems[i].UnitPrice * body.DiscountValue / 100
			discountPercent = body.DiscountValue
		} else if body.DiscountType == "cash" {
			cartItems[i].DiscountAmount = body.DiscountValue
			discountPercent = body.DiscountValue * 100 / cartItems[i].UnitPrice
		} else {
			handleResponse(c, BadRequest, "Discount type or value is invalid")
			return
		}
		err = tx.Debug().Exec(`
		UPDATE cart_items 
		SET
			discount_amount = ?,
			discount_type = ?,
			discount_value = ?,
			discount_price = CASE
			WHEN ? = 0 THEN 0
			ELSE unit_price - ?
		END
		WHERE id = ?`,
			cartItems[i].DiscountAmount,
			body.DiscountType,
			discountPercent,
			body.DiscountValue,
			cartItems[i].DiscountAmount,
			cartItems[i].ID).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to update cart items")
			tx.Rollback()
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
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

	var storeProduct domain.StoreProduct
	err = h.db.Raw(`
	SELECT
		sp.*, p.unit_per_pack
	FROM store_products sp
	JOIN products p ON p.id = sp.product_id WHERE sp.id = ?`,
		body.StoreProductID).Scan(&storeProduct).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "product not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if body.Quantity > 0 && body.UnitQuantity == 0 {
		if storeProduct.PackQuantity < body.Quantity {
			handleQuantityConflict(c, storeProduct, body)
			return
		}
	} else if body.Quantity == 0 && body.UnitQuantity > 0 {
		if storeProduct.UnitQuantity < body.UnitQuantity {
			handleQuantityConflict(c, storeProduct, body)
			return
		}
	} else if body.Quantity > 0 && body.UnitQuantity > 0 {
		if storeProduct.PackQuantity < body.Quantity || storeProduct.UnitQuantity < body.UnitQuantity {
			handleQuantityConflict(c, storeProduct, body)
			return
		}
	} else {
		handleResponse(c, BadRequest, "Invalid quantity")
		return
	}

	// Cart item ni yangilash uchun ma'lumotlar tayyorlash
	data := map[string]interface{}{
		"store_product_id": body.StoreProductID,
		"quantity":         body.Quantity,
		"unit_quantity":    body.UnitQuantity,
		"total_price":      float64(body.Quantity)*storeProduct.RetailPrice + (storeProduct.RetailPrice/float64(storeProduct.UnitPerPack))*float64(body.UnitQuantity),
	}

	err = h.db.
		WithContext(c.Request.Context()).
		Table("cart_items").
		Where("id = ?", id).
		Updates(&data).Error
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
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// MultipleDelete godoc
// @Summary Delete all cart items
// @Description Delete all cart items from the request body
// @Tags 	cart_items
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.Ids true "cart item IDs"
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

func handleQuantityConflict(c *gin.Context, storeProduct domain.StoreProduct, body domain.CartItemUpdateProductUnit) {
	handleResponse(c, CONFLICT, gin.H{
		"message":                "Not enough Product",
		"pack_quantity":          storeProduct.PackQuantity,
		"unit_quantity":          storeProduct.UnitQuantity,
		"received_pack_quantity": body.Quantity,
		"received_unit_quantity": body.UnitQuantity,
	})

}
