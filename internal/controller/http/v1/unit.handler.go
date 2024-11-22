package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type UnitHandler struct {
	*Handler
}

func (h *Handler) NewUnitHandler(r *gin.RouterGroup) {
	unit := &UnitHandler{h}
	unit.UnitRoutes(r)
}

func (h *UnitHandler) UnitRoutes(r *gin.RouterGroup) {
	unit := r.Group("/unit")
	{
		unit.POST("", h.Create)
		unit.GET("", h.Get)
		unit.GET("/get-list", h.List)
		unit.PUT("", h.Update)
		unit.DELETE("", h.Delete)
	}
}

func (h *UnitHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Unit]
		res  domain.Unit
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	body.Data.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Model(&res).Create(&body.Data).Scan(&res).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrCreateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *UnitHandler) Get(c *gin.Context) {
	var res domain.Unit
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, http.StatusNotFound, MsgErrNotFount, nil)
			return
		}
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *UnitHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	res := []domain.Unit{}
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *UnitHandler) Update(c *gin.Context) {
	var (
		body RequestBody[domain.Unit]
		res  domain.Unit
	)
	if err := h.Db.WithContext(c.Request.Context()).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrUpdateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *UnitHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Unit{}, "id = ?", c.Query("id")).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrDeleteFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, nil)
}
