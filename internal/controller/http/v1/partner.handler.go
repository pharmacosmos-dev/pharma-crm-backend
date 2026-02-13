package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type PartnerHandler struct {
	*Handler
}

func (h *Handler) NewPartnerHandler(r *gin.RouterGroup) {
	partnerHandler := &PartnerHandler{h}
	partnerHandler.PartnerRoutes(r)
}

func (h *PartnerHandler) PartnerRoutes(r *gin.RouterGroup) {
	r.POST("/partners", h.CreatePartner)
}

// @Summary      Create Partner
// @Description  Create a new partner
// @Tags         partner
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        partner body domain.OAuthClient true "Partner"
// @Success      201 {object} domain.OAuthClient
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /partners [post]
func (h *PartnerHandler) CreatePartner(c *gin.Context) {
	var body domain.OAuthClient
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.CreatePartnerOAuthClient(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	c.JSON(http.StatusCreated, nil)
}
