package v1

import (
	"context"
	"errors"

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
