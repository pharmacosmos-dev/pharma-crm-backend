package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type CustomerHandler struct {
	*Handler
}

func (h *Handler) NewCustomerHandler(r *gin.RouterGroup) {
	customer := &CustomerHandler{h}
	customer.CustomerRoutes(r)
}

func (h *CustomerHandler) CustomerRoutes(r *gin.RouterGroup) {
	customer := r.Group("/customer")
	{
		customer.POST("", h.Create)
		customer.GET("", h.Get)
		customer.GET("/get-list", h.List)
		customer.PUT("", h.Update)
		customer.DELETE("", h.Delete)
	}
}

func (h *CustomerHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Customer]
	var res domain.Customer
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.Db.WithContext(ctx).Create(&body.Data).Scan(&res).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *CustomerHandler) Get(c *gin.Context) {
	var res domain.Customer
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, http.StatusNotFound, MsgErrNotFount, nil)
			return
		}
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *CustomerHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	res := []*domain.Customer{}
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *CustomerHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Customer]
	if err := h.Db.WithContext(c.Request.Context()).Model(&body.Data).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrUpdateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body.Data)
}

func (h *CustomerHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Customer{}, "id = ?", c.Query("id")).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrDeleteFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
