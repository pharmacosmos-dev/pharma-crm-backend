package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
)

type TokenGeneratorHandler struct {
	*Handler
}

func (h *Handler) NewTokenGeneratorHandler(r *gin.RouterGroup) {
	tokenGenerator := &TokenGeneratorHandler{h}
	tokenGenerator.TokenGeneratorRoutes(r)
}

func (h *TokenGeneratorHandler) TokenGeneratorRoutes(r *gin.RouterGroup) {
	r.POST("/generate-token", h.Generate1CToken)
}

// @Summary Generate 1C token
// @Description Generate 1C token
// @Tags 1C token
// @Accept json
// @Produce json
// @Param request body domain.Generate1CTokenRequest true "Generate 1C token"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /generate-token [post]
func (h *TokenGeneratorHandler) Generate1CToken(c *gin.Context) {
	var req domain.Generate1CTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Errorf("err: %v", err).Error())
		return
	}

	token, err := etc.Encrypt(req.Password, h.cfg.HashKey)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, token)
}
