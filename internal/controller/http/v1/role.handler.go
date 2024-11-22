package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type RoleHandler struct {
	*Handler
}

func (h *Handler) NewRoleHandler(r *gin.RouterGroup) {
	role := &RoleHandler{h}
	role.RoleRoutes(r)
}

func (h *RoleHandler) RoleRoutes(r *gin.RouterGroup) {
	role := r.Group("/role")
	{
		role.POST("", h.Create)
		role.GET("", h.Get)
		role.GET("/get-list", h.List)
		role.PUT("", h.Update)
		role.DELETE("", h.Delete)
	}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Role]
	var res domain.Role
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	body.Data.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Model(&domain.Role{}).Create(&body.Data).Scan(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

func (h *RoleHandler) Get(c *gin.Context) {
	var res domain.Role
	if err := h.Db.Model(&domain.Role{}).First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *RoleHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	res := []domain.Role{}
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Role]
	var res domain.Role
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	if err := h.Db.WithContext(c.Request.Context()).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Role{}, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
