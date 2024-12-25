package v1

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

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
		category.GET("/:id", h.Get)
		category.PUT("/:id", h.Update)
		category.GET("/list", h.List)
		category.DELETE("/:id", h.Delete)
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
	var (
		body domain.CategoryRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("err: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	err = h.db.WithContext(c.Request.Context()).
		Table("categories").
		Create(&body).Error
	if err != nil {
		h.log.Error("err: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Get godoc
// @Summary Get a category
// @Description Get a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "category ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/{id} [get]
func (h *CategoryController) Get(c *gin.Context) {
	var res domain.Category
	err := h.db.First(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a category
// @Description Get a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param parent_id query string false "Parent ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/list [get]
func (h *CategoryController) List(c *gin.Context) {
	// var res []domain.Category
	var parentID *string
	if p := c.Query("parent_id"); p != "" {
		parentID = &p
	}
	categories, err := fetchCategories(h.db, parentID)
	if err != nil {
		h.log.Error("err: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, categories)
}

// Update godoc
// @Summary Update a category
// @Description Update a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "category ID"
// @Param category body domain.CategoryRequest true "Category information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/{id} [put]
func (h *CategoryController) Update(c *gin.Context) {
	var (
		body domain.CategoryRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Category{}).
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
// @Summary Delete a category
// @Description Delete a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "category ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/{id} [delete]
func (h *CategoryController) Delete(c *gin.Context) {
	err := h.db.WithContext(c.Request.Context()).
		Delete(&domain.Category{}, "id = ?", c.Param("id")).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

func fetchCategories(db *gorm.DB, parentID *string) ([]domain.Category, error) {
	var categories []domain.Category
	query := db.Preload("SubCategories")
	if parentID != nil {
		query = query.Where("category_id = ?", parentID)
	}
	err := query.Find(&categories).Error
	if err != nil {
		return nil, err
	}
	// Recursively fetch subcategories for each category
	for i := range categories {
		categories[i].SubCategories, err = fetchCategories(db, &categories[i].Id)
		if err != nil {
			return nil, err
		}
	}

	return categories, nil
}
