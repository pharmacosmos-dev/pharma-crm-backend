package v1

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"net/http"
	"time"
)

func (h *EmployeeHandler) Login(c *gin.Context) {
	var body = new(domain.Login)
	if err := c.ShouldBind(body); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res, err := h.c.GetRoles(ctx, body)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *EmployeeHandler) Logout(c *gin.Context) {}
