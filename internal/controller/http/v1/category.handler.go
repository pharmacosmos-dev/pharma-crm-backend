package v1

import (
	"fmt"

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

	// Bind the JSON body to the request struct
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("Error binding JSON: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Recursive function to save categories
	var saveCategory func(category domain.CategoryRequest, parentID *string) error
	saveCategory = func(category domain.CategoryRequest, parentID *string) error {
		category.Id = uuid.New().String() // Generate a unique ID
		// Save the current category
		err := h.db.WithContext(c.Request.Context()).
			Table("categories").
			Create(&domain.CategoryRequest{
				Id:         category.Id,
				Name:       category.Name,
				CategoryId: parentID,
			}).Error
		if err != nil {
			return err
		}

		// Save subcategories recursively
		if len(category.SubCategory) > 0 {
			for _, subCategory := range category.SubCategory {
				err := saveCategory(*subCategory, &category.Id)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Start saving categories recursively
	err = saveCategory(body, nil)
	if err != nil {
		h.log.Error("Error saving categories: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Update godoc
// @Summary Update a category
// @Description Update a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "category ID"
// @Param category body domain.Category true "Category information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/{id} [put]
func (h *CategoryController) Update(c *gin.Context) {
	var (
		body domain.Category
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var updateCategory func(category domain.Category) error
	updateCategory = func(category domain.Category) error {
		// Save the current category
		err := h.db.WithContext(c.Request.Context()).
			Table("categories").
			Where("id = ?", category.Id).
			Updates(&category).Error
		if err != nil {
			return err
		}

		// Save subcategories recursively
		if len(category.SubCategories) > 0 {
			for _, subCategory := range category.SubCategories {
				err := updateCategory(subCategory)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	err = updateCategory(body)
	if err != nil {
		h.log.Error(err)
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
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/list [get]
func (h *CategoryController) List(c *gin.Context) {
	var (
		res      []domain.Category
		parentID *string
		search   = c.Query("search")
	)

	// Handle optional `parent_id` query parameter
	if p := c.Query("parent_id"); p != "" {
		parentID = &p
	}

	// Build the base query
	query := h.db.Model(&domain.Category{})

	// Filter by `parent_id`
	if parentID != nil {
		query = query.Where("category_id = ?", *parentID)
	} else {
		// Root categories (no parent)
		query = query.Where("category_id IS NULL")
	}

	// Apply search filter if provided
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}

	// Preload SubCategories recursively
	query = query.Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
		return db.Preload("SubCategories")
	})

	// Execute the query
	if err := query.Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
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
