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

type BrandHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewBrandHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *BrandHandler {
	return &BrandHandler{cfg, db, log}
}

func (h *BrandHandler) Create(c *gin.Context) {
	var (
		brand = new(domain.Brand)
		res   = new(domain.Brand)
	)

	if err := c.ShouldBind(brand); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	brand.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Create(brand).Error; err != nil {
		h.log.Error("Error on creating brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *BrandHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res := new(domain.Brand)
	if err := h.db.WithContext(ctx).First(res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error("Error on getting brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *BrandHandler) List(c *gin.Context) {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var res []*domain.Brand
	if err := h.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error("Error on list brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *BrandHandler) Update(c *gin.Context) {
	var brand RequestBody[domain.Brand]
	var res = new(domain.Brand)
	if err := c.ShouldBindJSON(&brand); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := h.db.WithContext(ctx).Model(res).Where("id = ?", brand.Data.Id).
		Updates(brand.Data).Error; err != nil {
		h.log.Error("Error on update brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *BrandHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := h.db.WithContext(ctx).Delete(&domain.Brand{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error("Error on delete brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
