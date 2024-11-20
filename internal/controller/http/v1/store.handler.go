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

type StoreHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewStoreHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *StoreHandler {
	return &StoreHandler{cfg, db, log}
}

func (h *StoreHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Store]
		res  domain.Store
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&res).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrCreateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *StoreHandler) Get(c *gin.Context) {
	var res domain.Store
	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, http.StatusNotFound, MsgErrNotFount, nil)
			return
		}
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *StoreHandler) List(c *gin.Context) {
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
	res := []*domain.Store{}
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *StoreHandler) Update(c *gin.Context) {
	var (
		body RequestBody[domain.Store]
		res  domain.Store
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrUpdateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *StoreHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).Delete(&domain.Store{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrDeleteFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
