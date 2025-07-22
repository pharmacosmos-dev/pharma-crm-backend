package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"net/http"
	"strings"
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
	var req domain.CreateDiscountCardRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// get user id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	card := &domain.DiscountCard{
		Barcode:    req.Barcode,
		CustomerID: req.CustomerID,
		Percent:    req.Percent,
		CreatedBy:  employee.Id,
	}

	if err = h.db.Create(&card).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Discount card with this barcode already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create discount card"})
		}
		return
	}

	handleResponse(c, CREATED, card)
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
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	id := c.Param("id")
	if id == "" {
		handleResponse(c, BadRequest, "ID is required")
		return
	}
	req.ID = id

	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "unauthorized")
		return
	}
	req.UpdatedBy = cast.ToString(userID)
	err := h.service.UpdateDiscountCard(&req)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
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
