package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var req domain.LoyaltyCardCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid request: %s", err.Error()))
		return
	}

	if !(req.VirtualLoyaltyCardNeeded || *req.LoyaltyCardBarcode != "") {
		handleResponse(c, BadRequest, "Either virtual_loyalty_card_needed must be true or loyalty_card_barcode must be provided")
		return
	}

	req.LoyaltyCardCreatedBy = user.UserId
	customer, err := h.service.CreateLoyaltyCard(&req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, CREATED, customer)
}
