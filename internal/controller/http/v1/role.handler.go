package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
)

type RoleHandler struct {
	c *services.RoleService
	l logger.Interface
}

func NewRoleHandler(handler *gin.RouterGroup, c *services.RoleService, l logger.Interface) {
	r := &RoleHandler{c, l}
	handler.POST("", r.Create)
	handler.GET("", r.Get)
	handler.GET("/get-list", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("", r.Delete)
}

func (h *RoleHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Role]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := h.c.Create(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *RoleHandler) Get(c *gin.Context) {
	Id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := h.c.Get(ctx, Id)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *RoleHandler) List(c *gin.Context) {
	limit, err := getLimitParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	offset, err := getOffsetParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := h.c.GetList(ctx, &domain.Params{Limit: limit, Offset: offset})
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Role]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := h.c.Update(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	Id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := h.c.Delete(ctx, Id)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, Id)
}
