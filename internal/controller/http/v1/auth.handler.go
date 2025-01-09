package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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

// @Summary      Login
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
	var (
		body domain.Login
		res  domain.Employee
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.WithContext(c.Request.Context()).
		Preload("Store").
		Where("is_active = ?", true).
		First(&res, "phone = ?", body.Phone).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	oldPassword, err := etc.Decrypt(res.Password, h.cfg.HeshKey)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if body.Password != oldPassword {
		handleResponse(c, CONFLICT, "Wrong password")
		return
	}

	m := map[string]interface{}{
		"user_id": res.Id,
	}
	refreshClaims := map[string]interface{}{
		"user_id": res.Id,
	}
	accessToken, refreshToken, err := h.JwtHandler.GenerateTokens(m, refreshClaims)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := domain.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		Employee:     res,
	}
	handleResponse(c, OK, data)
}

func (h *AuthHandler) Logout(c *gin.Context) {}

func (h *AuthHandler) UpdateAccessToken(c *gin.Context) {
}
