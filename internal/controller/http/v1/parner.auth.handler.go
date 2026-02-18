package v1

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type PartnerAuthHandler struct {
	*Handler
}

func (h *Handler) NewPartnerAuthHandler(r *gin.RouterGroup) {
	auth := &PartnerAuthHandler{h}
	auth.PartnerAuthRoutes(r)
}

func (h *PartnerAuthHandler) PartnerAuthRoutes(r *gin.RouterGroup) {
	r.POST("/uzum/security/oauth/token", h.OAuthToken)
}

// @Summary      OAuth2 Client Credentials Token
// @Description  Obtain an OAuth2 access token using client credentials grant
// @Tags         auth
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        grant_type formData string true "Grant Type" default(client_credentials)
// @Param        client_id formData string true "Client ID"
// @Param        client_secret formData string true "Client Secret"
// @Param        scope formData string false "Requested scopes (space-separated)" default(read write)
// @Success      200  {object}  v1.Response{data=domain.OAuthResponse}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /uzum/security/oauth/token [post]
func (h *PartnerAuthHandler) OAuthToken(c *gin.Context) {
	var body domain.OAuthRequest
	if err := c.ShouldBind(&body); err != nil {
		h.log.Warnf("invalid OAuth request format: %v", err)
		handleResponse(c, BadRequest, "Invalid request format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.OAuthToken(ctx, &body)
	if err != nil {
		h.log.Errorf("OAuth token error: %v", err)
		// Determine appropriate error code based on error message
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid client credentials") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": errMsg,
			})
		} else if strings.Contains(errMsg, "unsupported grant type") || strings.Contains(errMsg, "invalid scope") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errMsg,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
		}
		return
	}

	// Return 200 OK as per OAuth2 spec (not 201 CREATED)
	c.JSON(http.StatusOK, result)
}
