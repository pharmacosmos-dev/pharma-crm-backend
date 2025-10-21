package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type LoyaltyCardHandler struct {
	*Handler
}

func (h *Handler) NewLoyaltyCardHandler(r *gin.RouterGroup) {
	loyaltyCardHandler := &LoyaltyCardHandler{h}
	loyaltyCardHandler.LoyaltyCardRoutes(r)
}

func (h *LoyaltyCardHandler) LoyaltyCardRoutes(r *gin.RouterGroup) {
	loyaltyCard := r.Group("/loyalty_card")
	{
		loyaltyCard.POST("", h.Create)
		// loyaltyCard.GET("/:id", h.Get)
		// loyaltyCard.GET("/list", h.List)
		// loyaltyCard.PUT("/:id", h.Update)
		// loyaltyCard.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create Loyalty Card
// @Description create Loyalty Card
// @Tags loyalty_card
// Security BearerAuth
// @Accept json
// @Produce json
// @Param loyalty_card body domain.LoyaltyCardCreateRequest true "Loyalty Card info"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card [post]
func (h *LoyaltyCardHandler) Create(c *gin.Context) {
	var req domain.LoyaltyCardCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid request: %s", err.Error()))
		return
	}

	if !(req.VirtualLoyaltyCardNeeded || *req.LoyaltyCardBarcode != "") {
		handleResponse(c, BadRequest, "Either virtual_loyalty_card_needed must be true or loyalty_card_barcode must be provided")
		return
	}

	userId, ok := c.Get("user_id")
	if !ok {
		// handleResponse(c, UNAUTHORIZED, "User ID not found")
		// return
	}

	userId = "6673c653-60cb-4ada-bcd6-b8c1d17ffecb"

	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("Error on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	req.LoyaltyCardCreatedBy = employee.Id
	customer, err := h.service.CreateLoyaltyCard(&req)
	if err != nil {
		h.log.Error("error on creating customer: ", err)
		handleResponse(c, InternalError, fmt.Sprintf("error on creating loyalty card: %s", err.Error()))
		return
	}

	handleResponse(c, CREATED, customer)
}
