package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type DiscountCardHandler struct {
	*Handler
}

func (h *Handler) NewDiscountCardHandler(r *gin.RouterGroup) {
	discountCard := &DiscountCardHandler{h}
	discountCard.DiscountCardRoutes(r)
}

func (h *DiscountCardHandler) DiscountCardRoutes(r *gin.RouterGroup) {
	discountCard := r.Group("/discount-card")
	{
		discountCard.POST("", h.CreateDiscountCard)
		discountCard.PUT("/:id", h.UpdateDiscountCard)
		discountCard.DELETE("/:id", h.DeleteDiscountCard)
	}
}

// CreateDiscountCard godoc
// @Summary Create discount card
// @Description Create discount card from the request body
// @Tags discount cards
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param req body domain.CreateDiscountCardRequest true "discount card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /discount-card [post]
func (h *DiscountCardHandler) CreateDiscountCard(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var req domain.CreateDiscountCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateCustomerDiscountCard(ctx, &req)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// Update godoc
// @Summary Update a discount card
// @Tags discount cards
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Discount Card ID"
// @Param body body domain.UpdateDiscountCardRequest true "Update Discount Card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /discount-card/{id} [put]
func (h *DiscountCardHandler) UpdateDiscountCard(c *gin.Context) {
	var req domain.UpdateDiscountCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.Id = id

	err := h.service.UpdateDiscountCard(ctx, &req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "updated successfully")
}

// Delete godoc
// @Summary Delete a discount card
// @Tags discount cards
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Discount Card ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /discount-card/{id} [delete]
func (h *DiscountCardHandler) DeleteDiscountCard(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		handleResponse(c, BadRequest, "ID is required")
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "unauthorized")
		return
	}

	err := h.service.DeleteDiscountCard(id, userID.(string))
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "deleted successfully")
}
