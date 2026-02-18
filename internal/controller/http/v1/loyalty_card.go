package v1

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
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
		loyaltyCard.GET("/dashboard", h.GetDashboard)
		loyaltyCard.GET("/top", h.GetTopCustomers)
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
// @Security BearerAuth
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

// GetDashboard godoc
// @Summary Get Loyalty Card Dashboard Statistics
// @Description Returns loyalty card statistics including total cashback, card counts, and distribution by level
// @Tags loyalty_card
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param from_date query string false "Start date for new cards filter" example:"2024-01-01"
// @Param to_date query string false "End date for new cards filter" example:"2024-12-31"
// @Param is_loyalty query bool false "Filter customers by loyalty card status (true=has card, null=no filter)"
// @Param limit query int false "Number of customers to return" default:10
// @Param offset query int false "Offset for pagination" default:0
// @Success 200 {object} v1.Response{data=domain.LoyaltyCardDashboard}
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card/dashboard [get]
func (h *LoyaltyCardHandler) GetDashboard(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var req domain.LoyaltyCardDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid query parameters: %s", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.Limit, req.Offset = defaultLimitOffset(req.Limit, req.Offset)

	dashboard, err := h.service.GetLoyaltyCardDashboard(ctx, &req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, dashboard)
}

// GetTopCustomers godoc
// @Summary Get Top Loyalty Card Customers
// @Description Returns top customers by cashback earned, with optional date filtering
// @Tags loyalty_card
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Number of customers to return" default:10
// @Param offset query int false "Offset for pagination" default:0
// @Param from_date query string false "Start date for sales filter" example:"2024-01-01"
// @Param to_date query string false "End date for sales filter" example:"2024-12-31"
// @Success 200 {object} v1.Response{data=[]domain.LoyaltyCardTopCustomer}
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card/top [get]
func (h *LoyaltyCardHandler) GetTopCustomers(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var req domain.LoyaltyCardTopRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid query parameters: %s", err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.Limit, req.Offset = defaultLimitOffset(req.Limit, req.Offset)

	customers, count, err := h.service.GetLoyaltyCardTopCustomers(ctx, &req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(customers, count, req.Limit, req.Offset)

	handleResponse(c, OK, result)
}

