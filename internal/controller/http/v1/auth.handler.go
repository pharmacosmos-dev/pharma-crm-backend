package v1

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/etc"
	"gorm.io/gorm"
)

type AuthHandler struct {
	*Handler
}

func (h *Handler) NewAuthHandler(r *gin.RouterGroup) {
	auth := &AuthHandler{h}
	auth.AuthRoutes(r)
}

func (h *AuthHandler) AuthRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.Login)
	r.POST("/security/oauth/token", h.OAuthToken)
}

// @Summary      Loginx
// @Description  Login a user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body     domain.Login  true  "Login data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /login [post]
func (h *AuthHandler) Login(c *gin.Context) {

	var body domain.Login
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var employee domain.Employee
	err := h.db.WithContext(ctx).
		Preload("Store").
		Where("is_active = ?", true).
		First(&employee, "phone = ?", body.Phone).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Errorf("could not find employee by phone: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// decrypt saved password
	oldPassword, err := etc.Decrypt(employee.Password, h.cfg.HashKey)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check password
	if body.Password != oldPassword {
		handleResponse(c, CONFLICT, "Wrong password")
		return
	}

	userClaims := map[string]any{
		"user_id":    employee.Id,
		"company_id": employee.CompanyId,
		"store_id":   employee.StoreId,
		"role":       employee.RoleType,
	}
	// generate tokens
	accessToken, refreshToken, err := h.JwtHandler.GenerateTokens(userClaims)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect response
	data := domain.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		Employee:     employee,
	}
	// return response
	handleResponse(c, OK, data)
}

func (h *AuthHandler) Logout(c *gin.Context) {
}

func (h *AuthHandler) UpdateAccessToken(c *gin.Context) {
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
// @Router       /security/oauth/token [post]
func (h *AuthHandler) OAuthToken(c *gin.Context) {
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
