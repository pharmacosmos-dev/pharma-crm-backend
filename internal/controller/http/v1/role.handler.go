package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type RoleHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewRoleHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *RoleHandler {
	return &RoleHandler{cfg, db, log}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Role]
	var res domain.Role
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&domain.Role{}).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

func (h *RoleHandler) Get(c *gin.Context) {
	var res domain.Role
	if err := h.db.Model(&domain.Role{}).First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
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

	res := []*domain.Role{}
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Role]
	var res domain.Role
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).Delete(&domain.Role{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
