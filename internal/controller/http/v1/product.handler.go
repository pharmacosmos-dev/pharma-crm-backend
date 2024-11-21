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
	var res domain.Product
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&domain.Product{}).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
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
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	var res []domain.Product
	var totalCount int64

	// Perform a single query to get both total count and paginated results
	query := h.db.Model(&domain.Product{})
	if err := query.Count(&totalCount).Preload("Category").Limit(limit).Offset(offset).Where("name ILIKE ?", "%"+c.Query("name")+"%").Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}

	result := struct {
		Product []domain.Product `json:"data"`
		Meta    domain.Meta      `json:"_meta"`
	}{
		Product: res,
		Meta: domain.Meta{
			TotalCount:  int(totalCount),
			PerPage:     limit,
			CurrentPage: (offset / limit) + 1,
			PageCount:   int((totalCount + int64(limit) - 1) / int64(limit)),
		},
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, result)
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

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
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
