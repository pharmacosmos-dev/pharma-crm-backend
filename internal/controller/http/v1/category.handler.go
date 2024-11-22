package v1

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type CategoryController struct {
	*Handler
}

func (h *Handler) NewCategoryController(r *gin.RouterGroup) {
	category := &CategoryController{h}
	category.CategoryRoutes(r)
}

func (h *CategoryController) CategoryRoutes(r *gin.RouterGroup) {
	category := r.Group("/category")
	{
		category.POST("", h.Create)
		category.GET("", h.Get)
		category.PUT("", h.Update)
		category.GET("/get-list", h.List)
		category.DELETE("", h.Delete)
	}
}

// Create godoc
// @Summary Create a new category
// @Description Create a new category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param category body domain.CategoryRequest true "Category information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category [post]
func (h *CategoryController) Create(c *gin.Context) {
	var body domain.CategoryRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Create(&body).Scan(&body).Error; err != nil {
		h.Log.Error(err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

// Get godoc
// @Summary Get a category
// @Description Get a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "category ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category [get]
func (h *CategoryController) Get(c *gin.Context) {
	var res domain.Category
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrFetchFailed, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

// List godoc
// @Summary Get a category
// @Description Get a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/get-list [get]
func (h *CategoryController) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var res []domain.Category
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

// Update godoc
// @Summary Update a category
// @Description Update a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param category body domain.CategoryUpdateRequest true "Category information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category [put]
func (h *CategoryController) Update(c *gin.Context) {
	var body domain.CategoryUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	if err := h.Db.WithContext(c.Request.Context()).Table("categories").Where("id = ?", body.Id).
		Updates(&body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body)
}

// Delete godoc
// @Summary Delete a category
// @Description Delete a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "category ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category [delete]
func (h *CategoryController) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).
		Delete(&domain.Category{}, "id = ?", c.Query("id")).Error; err != nil {
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
