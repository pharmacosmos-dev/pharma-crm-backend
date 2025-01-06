package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type StoreHandler struct {
	*Handler
}

func (h *Handler) NewStoreHandler(r *gin.RouterGroup) {
	store := &StoreHandler{h}
	store.StoreRoutes(r)
}

func (h *StoreHandler) StoreRoutes(r *gin.RouterGroup) {
	store := r.Group("/store")
	{
		store.POST("", h.Create)
		store.GET("/:id", h.Get)
		store.GET("/list", h.List)
		store.PUT("/:id", h.Update)
		store.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a store
// @Description Create a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.StoreRequest true "Store information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [post]
func (h *StoreHandler) Create(c *gin.Context) {
	var (
		body domain.StoreRequest
		err  error
	)
	createdBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.CreatedBy = createdBy.(string)
	body.Id = uuid.New().String()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [get]
func (h *StoreHandler) Get(c *gin.Context) {
	var (
		res domain.Store
		err error
	)
	err = h.db.First(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/list [get]
func (h *StoreHandler) List(c *gin.Context) {
	var (
		res        []domain.Store
		totalCount int64
		search     = c.Query("search")
		productID  = c.Query("product_id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.
		Model(&domain.Store{})

	if productID != "" {
		query = query.
			Select("stores.*, sp.quantity as quantity, sp.small_quantity as small_quantity").
			Joins("JOIN store_products sp ON stores.id = sp.store_id").
			Where("sp.product_id = ?", productID).Order("sp.quantity DESC")
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}

	err = query.
		Where("is_active = ?", true).
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error

	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a store
// @Description Update a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Param store body domain.StoreRequest true "Store information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [put]
func (h *StoreHandler) Update(c *gin.Context) {
	var (
		body domain.StoreUpdateRequest
		id   = c.Param("id")
		err  error
	)
	updatedBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	body.UpdatedBy = updatedBy.(string)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a store
// @Description Delete a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [delete]
func (h *StoreHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	deletedBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	err := h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Where("id = ?", id).
		Update("is_active", false).
		Update("deleted_by", deletedBy.(string)).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
