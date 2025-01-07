package v1

import (
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

	var product domain.Product
	err = h.db.First(&product, "id = ?", body.ProductID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Error on getting product")
		return
	}
	body.TotalPrice = product.RetailPrice * float64(body.Quantity)
	body.TotalDiscountPrice = body.TotalPrice
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
		body       []domain.CartItem
		sumResult  domain.SumResult
		totalCount int64
		err        error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.Model(&domain.CartItem{}).
		Count(&totalCount).
		Preload("Product", func(db *gorm.DB) *gorm.DB {
			return db.Preload("ProductUnits")
		}).
		Where("sale_id = ? AND is_drafted = false AND status = 'pending'", c.Query("sale_id")).
		Limit(limit).
		Offset(offset).
		Order("created_at desc").
		Find(&body).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.Model(&domain.CartItem{}).
		Select("SUM(total_price) as total_price, SUM(discount_amount) as discount_amount").
		Where("sale_id = ?", c.Query("sale_id")).
		Scan(&sumResult).Error

	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, domain.CartItemResponse{
		TotalAmount:    sumResult.TotalPrice,
		DiscountAmount: sumResult.DiscountPrice,
		Count:          totalCount,
		Data:           body,
	})
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
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("cart_items").
		Where("sale_id = ?", saleId).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var product domain.Product
	err = h.db.First(&product, "id = ?", body.ProductID).Error
	if err != nil {
		handleResponse(c, InternalError, "Error on getting product")
		return
	}

	var totalAmount float64
	var cartItem domain.CartItemUpdateRequest
	for _, productUnit := range body.ProductUnits {
		if productUnit.UnitName == "piece" && productUnit.BoxGrainCount > 0 {
			cartItem.TotalPrice += float64(body.DrugCount) * (product.RetailPrice / float64(productUnit.BoxGrainCount))
		} else if productUnit.UnitName == "pack" {
			if product.Quantity > 1 {
				cartItem.TotalPrice += product.RetailPrice * float64(body.Quantity)
			}
		} else {
			handleResponse(c, BadRequest, "Invalid unit type")
			return
		}
	}

	cartItem.Quantity = body.Quantity
	cartItem.TotalPrice = totalAmount
	cartItem.DrugCount = body.DrugCount
	if body.DiscountType != nil && body.DiscountValue != nil {
		if *body.DiscountType == "percent" {
			cartItem.DiscountAmount = product.RetailPrice * (*body.DiscountValue) / 100
		} else if *body.DiscountType == "cash" {
			cartItem.DiscountAmount = *body.DiscountValue
		} else {
			cartItem.DiscountAmount = 0
		}
	}
	cartItem.TotalDiscountPrice = cartItem.TotalPrice - cartItem.DiscountAmount
	err = h.db.
		WithContext(c.Request.Context()).
		Table("cart_items").
		Where("id = ?", id).
		Updates(&cartItem).Error
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
	var (
		body domain.CartItem
		err  error
		id   = c.Param("id")
	)
	err = h.db.Delete(&body, "id = ?", id).Error
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
