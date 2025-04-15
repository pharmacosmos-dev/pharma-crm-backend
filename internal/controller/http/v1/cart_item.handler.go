package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
// @Param 	input body domain.CartItemRequest true "Cart item information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item [post]
func (h *CartItemHandler) Create(c *gin.Context) {
	var (
		body domain.CartItemRequest
		err  error
	)
	// get user id in context
	vendorID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// get employee by user id
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", vendorID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// get store product
	storeProduct, err := h.service.GetStoreProductByIdOrBarcode(body.StoreProductID, body.Barcode, employee.StoreId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Product not found")
			return
		} else if err.Error() == "marking and barcode mismatch" {
			handleResponse(c, UnprocessableEntity, "Marking and barcode mismatch")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get cart item
	var cartItem domain.CartItem
	err = h.db.First(&cartItem,
		"store_product_id = ? AND sale_id = ?  AND status = 'pending'",
		storeProduct.Id, body.SaleId).Error
	if err == nil {
		cartItem.Quantity++
		if cartItem.Quantity > storeProduct.PackQuantity && cartItem.UnitQuantity == 0 {
			storeProduct.UnitQuantity -= storeProduct.PackQuantity * storeProduct.UnitPerPack
			handleQuantityConflict(c, storeProduct, cartItem.Quantity, cartItem.UnitQuantity)
			return
		}
		cartItem.TotalPrice += cartItem.UnitPrice
		err = h.db.Raw(`UPDATE cart_items SET quantity = ?, total_price = ? WHERE id = ? RETURNING *`,
			cartItem.Quantity, cartItem.TotalPrice, cartItem.ID).Scan(&cartItem).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, cartItem)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		handleResponse(c, InternalError, err.Error())
		return
	}
	if storeProduct.PackQuantity > 0 {
		body.Quantity = 1
		body.TotalPrice = storeProduct.RetailPrice
	} else if storeProduct.UnitQuantity > 0 && storeProduct.UnitPerPack > 0 {
		body.UnitQuantity = 1
		body.TotalPrice = storeProduct.RetailPrice / float64(storeProduct.UnitPerPack)
	} else {
		handleQuantityConflict(c, storeProduct, 1, 0)
		return
	}
	// Check discount if cart has already discount
	var discountPercent, discountPrice float64
	if body.DiscountType == "percent" && body.DiscountValue <= 100 {
		body.DiscountAmount = storeProduct.RetailPrice * body.DiscountValue / 100
		discountPercent = body.DiscountValue
	} else if body.DiscountType == "cash" {
		body.DiscountAmount = body.DiscountValue
		discountPercent = body.DiscountValue * 100 / storeProduct.RetailPrice
	} else {
		handleResponse(c, BadRequest, "Discount type or value is invalid")
		return
	}
	if body.DiscountAmount > 0 {
		discountPrice = storeProduct.RetailPrice - body.DiscountAmount
	}
	body.UnitPrice = storeProduct.RetailPrice
	body.EmployeeID = vendorID.(string)
	body.StoreProductID = storeProduct.Id
	res, err := h.service.CreateCartItem(&body, discountPercent, discountPrice)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
	res, err = h.service.CartItemList(saleID, limit, offset)
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
	var (
		body   domain.CartItemBySaleIDUpdateRequest
		saleId = c.Param("sale_id")
	)
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
	var (
		cartItems []domain.CartItem
		sum       float64
		count     int64
	)
	// get cart_items by sale_id
	err = h.db.Model(&domain.CartItem{}).Where("sale_id = ?", saleId).Count(&count).Find(&cartItems).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to fetch cart items")
		return
	}
	// check cart_items count with 0
	if count == 0 {
		handleResponse(c, BadRequest, "Cart items not added yet")
		return
	}
	// get sum of unit_prices
	err = h.db.Raw("SELECT SUM(unit_price) FROM cart_items WHERE sale_id = ?", saleId).Scan(&sum).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to get sum of unit prices")
		return
	}

	// check sum with discount value
	if body.DiscountType == "cash" && sum < body.DiscountValue {
		handleResponse(c, CONFLICT, "Discount value is greater than sum of unit prices")
		return
	}
	// check discount type with percent or cash
	var discountPercent float64
	for i := range cartItems {
		if body.DiscountValue == 0 {
			cartItems[i].DiscountAmount = 0
			discountPercent = 0
		} else if body.DiscountType == "percent" && body.DiscountValue <= 100 {
			cartItems[i].DiscountAmount = cartItems[i].UnitPrice * body.DiscountValue / 100
			discountPercent = body.DiscountValue
		} else if body.DiscountType == "cash" {
			// a = 1100 b = 1200  discount = 900
			// x = d / (a + b) = (900 / (1100 + 1200)) * 1100 = 430.47
			// y = d / (a + b) = (900 / (1100 + 1200)) * 1200 = 469.56
			// percent = (1 - (430.47/1100)) * 100
			discountPrice := (body.DiscountValue / sum) * cartItems[i].UnitPrice
			discountPercent = 1 - (discountPrice/cartItems[i].UnitPrice)*100
			cartItems[i].DiscountAmount = cartItems[i].UnitPrice - discountPrice
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
		body        domain.CartItemUpdateProductUnit
		err         error
		id          = c.Param("id")
		oldCartItem domain.CartItem
	)
	// validate cart item id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid cart item ID")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Eski cart item qiymatlarini olish
	err = h.db.WithContext(c.Request.Context()).
		Table("cart_items").
		Where("id = ?", id).
		First(&oldCartItem).Error
	if err != nil {
		handleResponse(c, InternalError, "Cart item not found")
		return
	}

	storeProduct, err := h.service.GetStoreProductByID(body.StoreProductID)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Unit quantityni pack quantityga o'zgartirish
	if body.UnitQuantity >= storeProduct.UnitPerPack && storeProduct.UnitPerPack > 0 {
		body.Quantity += body.UnitQuantity / storeProduct.UnitPerPack
		body.UnitQuantity = body.UnitQuantity % storeProduct.UnitPerPack
	}

	// Miqdor yetarliligini tekshirish
	if body.Quantity > 0 && body.UnitQuantity == 0 {
		if storeProduct.PackQuantity < body.Quantity {
			storeProduct.UnitQuantity -= storeProduct.PackQuantity * storeProduct.UnitPerPack
			handleQuantityConflict(c, storeProduct, body.Quantity, body.UnitQuantity)
			return
		}
	} else if body.UnitQuantity > 0 && body.Quantity == 0 {
		if storeProduct.UnitQuantity < body.UnitQuantity {
			storeProduct.UnitQuantity -= storeProduct.PackQuantity * storeProduct.UnitPerPack
			handleQuantityConflict(c, storeProduct, body.Quantity, body.UnitQuantity)
			return
		}
	} else if body.Quantity > 0 && body.UnitQuantity > 0 {
		if body.Quantity > storeProduct.PackQuantity || storeProduct.UnitQuantity-(body.Quantity*storeProduct.UnitPerPack) < body.UnitQuantity {
			storeProduct.UnitQuantity -= storeProduct.PackQuantity * storeProduct.UnitPerPack
			handleQuantityConflict(c, storeProduct, body.Quantity, body.UnitQuantity)
			return
		}
	} else {
		handleResponse(c, BadRequest, "Invalid quantity")
		return
	}

	// Eski va yangi qiymatlarni solishtirish
	quantityDiff := body.Quantity - oldCartItem.Quantity
	unitQuantityDiff := body.UnitQuantity - oldCartItem.UnitQuantity
	isIncrease := quantityDiff > 0 || unitQuantityDiff > 0

	var unitPrice float64
	if storeProduct.UnitPerPack > 0 {
		unitPrice = (storeProduct.RetailPrice / float64(storeProduct.UnitPerPack)) * float64(body.UnitQuantity)
	}

	// Cart item ni yangilash
	data := map[string]any{
		"store_product_id": body.StoreProductID,
		"quantity":         body.Quantity,
		"unit_quantity":    body.UnitQuantity,
		"total_price":      float64(body.Quantity)*storeProduct.RetailPrice + unitPrice,
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

	// Yangilangan response
	response := map[string]any{
		"id":                 id,
		"store_product_id":   body.StoreProductID,
		"increase":           isIncrease,
		"quantity":           body.Quantity,
		"unit_quantity":      body.UnitQuantity,
		"unit_per_pack":      storeProduct.UnitPerPack,
		"quantity_diff":      quantityDiff,
		"unit_quantity_diff": unitQuantityDiff,
	}

	handleResponse(c, OK, response)
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

func handleQuantityConflict(c *gin.Context, storeProduct *domain.StoreProduct, quantity, unitQuantity int) {
	handleResponse(c, CONFLICT, gin.H{
		"message":                "Not enough Product",
		"pack_quantity":          storeProduct.PackQuantity,
		"unit_quantity":          storeProduct.UnitQuantity,
		"received_pack_quantity": quantity,
		"received_unit_quantity": unitQuantity,
	})

}
