package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
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
		cartItem.PUT("/sale/:sale_id", h.UpdateCartItemDiscount)
		cartItem.PUT("/:id/markings", h.UpdateMarkings)
		cartItem.DELETE("/:id/markings", h.DeleteMarking)
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
	// get user id in context
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var body domain.CartItemRequest
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	// create cart item
	res, err := h.service.CreateCartItem(ctx, user, &body)
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
		res domain.CartItem
		id  = c.Param("id")
	)
	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeoutForReports)
	defer cancel()

	err := h.db.
		WithContext(ctx).
		Where("id = ?", id).
		First(&res).Error
	if err != nil {
		h.log.Errorf("could not get cart_item by id: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary 	Get a cart item
// @Description Get a cart item from the request body
// @Tags 		cart_items
// @Security    BearerAuth
// @Accept 		json
// @Produce 	json
// @Param 		limit query int false "Limit"
// @Param 		offset query int false "Offset"
// @Param 		sale_id query string true "saleId"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/list [get]
func (h *CartItemHandler) List(c *gin.Context) {
	var saleID = c.Query("sale_id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeoutForReports)
	defer cancel()

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	res, err := h.service.FetchCartItems(ctx, saleID, limit, offset)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
// @Param input body domain.CartItemDiscountRequest true "cartItemDiscount"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/sale/{sale_id} [put]
func (h *CartItemHandler) UpdateCartItemDiscount(c *gin.Context) {
	var (
		body   domain.CartItemDiscountRequest
		saleId = c.Param("sale_id")
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.UpdateCartItemDiscount(ctx, saleId, &body)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, body)
}

// Update cart item godoc
// @Summary Update a cart item
// @Description Update a cart item from the request body
// @Tags    cart_items
// @Security     BearerAuth
// @Accept  json
// @Produce json
// @Param   id path string true "cartItemId"
// @Param   input body domain.CartItemUpdateUnit true "Update unit"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id} [put]
func (h *CartItemHandler) Update(c *gin.Context) {
	var (
		body domain.CartItemUpdateUnit
		id   = c.Param("id")
	)

	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	body.Id = id

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateCartItemQuantity(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
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

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.DeleteCartItem(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
	var body domain.Ids

	// bind cart item ids
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if len(body.Ids) == 0 {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	err := h.service.DeleteCartItems(ctx, body.Ids)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}

// UpdateMarkings godoc
// @Summary Update cart item markings
// @Description Update the markings array for a specific cart item
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cart item ID"
// @Param body body domain.AppendMarkingRequest true "Markings payload"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 404 {object} v1.Response
// @Failure 409 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id}/markings [put]
func (h *CartItemHandler) UpdateMarkings(c *gin.Context) {
	var (
		id  = c.Param("id")
		req domain.AppendMarkingRequest
	)

	// bind request body
	if err := c.ShouldBindJSON(&req); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.UpdateCartItemMarkings(ctx, id, &req)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// DeleteMarking godoc
// @Summary Delete marking from cart item
// @Description Remove a single marking string from the markings array of a specific cart item
// @Tags cart_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "cart item ID"
// @Param body body domain.AppendMarkingRequest true "Marking payload"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 404 {object} v1.Response
// @Failure 409 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /cart_item/{id}/markings [delete]
func (h *CartItemHandler) DeleteMarking(c *gin.Context) {
	var (
		id  = c.Param("id")
		req domain.AppendMarkingRequest
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	// drop marking from cart_item markings list
	err := h.service.DeleteCartItemMarkings(context.TODO(), id, &req)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}
