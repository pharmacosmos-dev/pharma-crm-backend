package v1

import (
	"context"
	"github.com/pharma-crm-backend/config"
	"gorm.io/gorm"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/logger"
)

type EmployeeHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewEmployeeHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *EmployeeHandler {
	return &EmployeeHandler{cfg: cfg, db: db, log: log}
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Employee]
		res  domain.Employee
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	hashedPassword, err := etc.HashPassword(body.Data.Password)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Data.Password = hashedPassword
	if err := h.db.WithContext(ctx).Create(&body.Data).Model(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *EmployeeHandler) Get(c *gin.Context) {
	var res domain.Employee
	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
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
	var res []*domain.Employee
	if err := h.db.Limit(limit).Offset(offset).Find(res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Employee]
	var res domain.Employee
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	hashedPassword, err := etc.HashPassword(body.Data.Password)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Data.Password = hashedPassword

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).
		Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Delete(&domain.Employee{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
