package v1

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/logger"
	"net/http"
	"time"
)

type ProductHandler struct {
	p storage.ProductRepo
	l *logger.Logger
}

func NewProductRoutes(hander *gin.RouterGroup, p storage.ProductRepo, l *logger.Logger) {
	r := ProductHandler{p, l}
	hander.POST("", r.Create)
	hander.GET("", r.Get)
	hander.GET("/get-list", r.List)
	hander.PUT("", r.Update)
	hander.DELETE("", r.Delete)
}

func (h *ProductHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Product]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
	defer cancel()
	res, err := h.p.Create(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *ProductHandler) Get(c *gin.Context) {
	Id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
	defer cancel()
	res, err := h.p.Get(ctx, Id)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *ProductHandler) List(c *gin.Context) {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
	defer cancel()
	res, err := h.p.GetList(ctx, &domain.Params{Limit: limit, Offset: offset})
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *ProductHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Product]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
	defer cancel()
	res, err := h.p.Update(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	Id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
	defer cancel()
	err := h.p.Delete(ctx, Id)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, Id)
}
