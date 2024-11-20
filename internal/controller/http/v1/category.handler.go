package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/pharma-crm-backend/config"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type CategoryHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewCategoryHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *CategoryHandler {
	return &CategoryHandler{cfg: cfg, db: db, log: log}
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Category]
		res  domain.Category
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := h.db.WithContext(ctx).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *CategoryHandler) Get(c *gin.Context) {
	var res domain.Category
	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *CategoryHandler) List(c *gin.Context) {
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
	var res []*domain.Category
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	var (
		body RequestBody[domain.Category]
		res  domain.Category
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).
		Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).
		Delete(&domain.Category{}, "id = ?", c.Query("id")).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
