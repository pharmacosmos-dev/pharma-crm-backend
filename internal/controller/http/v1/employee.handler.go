package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/internal/storage"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/logger"
)

type EmployeeHandler struct {
	c storage.EmployeeRepo
	l logger.Interface
}

func NewEmployeeHandler(handler *gin.RouterGroup, c storage.EmployeeRepo, l logger.Interface) {
	r := &EmployeeHandler{c, l}
	handler.POST("/login", r.Login)
	handler.POST("/logout", r.Logout)
	handler.POST("", r.Create)
	handler.GET("", r.Get)
	handler.GET("/get-list", r.List)
	handler.PUT("", r.Update)
	handler.DELETE("", r.Delete)
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Employee]
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	hashedPassword, err := etc.HashPassword(body.Data.Password)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Data.Password = hashedPassword
	res, err := h.c.Create(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *EmployeeHandler) Get(c *gin.Context) {
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

func (h *EmployeeHandler) List(c *gin.Context) {
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

func (h *EmployeeHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Employee]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := h.c.Update(ctx, &body.Data)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	Id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := h.c.Delete(ctx, Id)
	if err != nil {
		h.l.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, nil)
}
