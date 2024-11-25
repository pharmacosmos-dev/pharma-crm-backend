package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
)

func (h *EmployeeHandler) Login(c *gin.Context) {
	var (
		body domain.Login
		res  domain.Employee
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var count int64
	err = h.db.Model(&domain.Employee{}).Where("phone = ?", body.Phone).Count(&count).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if count < 1 {
		handleResponse(c, NotFound, "User not found")
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Employee{}).
		First(&res, "phone = ?", body.Phone).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if !etc.CheckPasswordHash(body.Password, res.Password) {
		handleResponse(c, CONFLICT, "Wrong password")
		return
	}
	m := map[string]interface{}{
		"user_id": res.Id,
		"role_id": res.RoleId,
	}

	accessToken, err := h.JwtHandler.GenerateJWT(m)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := domain.LoginResponse{
		Token:    accessToken,
		Employee: res,
		// Permissions: ,
	}
	handleResponse(c, OK, data)
}

func (h *EmployeeHandler) Logout(c *gin.Context) {}
