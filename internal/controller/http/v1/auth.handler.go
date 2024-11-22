package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
)

func (h *EmployeeHandler) Login(c *gin.Context) {
	var body RequestBody[domain.Login]
	var res domain.Employee
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var count int64
	result := h.Db.Model(&domain.Employee{}).Where("phone = ?", body.Data.Phone).Count(&count)
	if result.Error != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, result.Error.Error())
		return
	}
	if count < 1 {
		handleResponse(c, http.StatusNotFound, MsgErrNotFount, MsgErrNotFount)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := h.Db.WithContext(ctx).Model(&domain.Employee{}).
		First(&res, "phone = ?", body.Data.Phone).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	if !etc.CheckPasswordHash(body.Data.Password, res.Password) {
		handleResponse(c, http.StatusConflict, MsgErrInvalidRequest, "Wrong password")
		return
	}
	m := map[string]interface{}{
		"user_id": res.Id,
		"role_id": res.RoleId,
	}

	accessToken, err := h.JwtHandler.GenerateJWT(m)
	if err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	data := domain.LoginResponse{
		Token:    accessToken,
		Employee: res,
		// Permissions: ,
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, data)
}

func (h *EmployeeHandler) Logout(c *gin.Context) {}
