package v1

import (
	"net/http"

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
		store.GET("", h.Get)
		store.GET("/get-list", h.List)
		store.PUT("", h.Update)
		store.DELETE("", h.Delete)
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
	var body domain.StoreRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Table("stores").Create(&body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrCreateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

// Get godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [get]
func (h *StoreHandler) Get(c *gin.Context) {
	var res domain.Store
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, http.StatusNotFound, MsgErrNotFount, nil)
			return
		}
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/get-list [get]
func (h *StoreHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	res := []*domain.Store{}
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

// Update godoc
// @Summary Update a store
// @Description Update a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param store body domain.StoreUpdateRequest true "Store information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [put]
func (h *StoreHandler) Update(c *gin.Context) {
	var body domain.Store
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	if err := h.Db.WithContext(c.Request.Context()).Table("stores").Where("id = ?", body.Id).Updates(&body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrUpdateFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body)
}

// Delete godoc
// @Summary Delete a store
// @Description Delete a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [delete]
func (h *StoreHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Store{}, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrDeleteFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
