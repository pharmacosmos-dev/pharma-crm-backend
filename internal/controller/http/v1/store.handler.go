package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
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
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	err = h.db.WithContext(c.Request.Context()).Model(&domain.Store{}).Create(&body).Error
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
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/list [get]
func (h *StoreHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Store{}
	err = h.db.Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
		body domain.StoreRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id = ?", c.Param("id")).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
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
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Store{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
