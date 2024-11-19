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

type ProductHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewProductHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *ProductHandler {
	return &ProductHandler{cfg, db, log}

}

func (h *ProductHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Product]
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&domain.Product{}).Create(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product

	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
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

	res := []*domain.Product{}
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *ProductHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Product]
	var res domain.Product
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).Updates(&body).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
