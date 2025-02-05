package v1

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type CategoryHander struct {
	*Handler
}

func (h *Handler) NewCategoryHander(r *gin.RouterGroup) {
	category := &CategoryHander{h}
	category.CategoryRoutes(r)
}

func (h *CategoryHander) CategoryRoutes(r *gin.RouterGroup) {
	category := r.Group("/category")
	{
		category.POST("", h.Create)
		category.GET("/:id", h.Get)
		category.PUT("/:id", h.Update)
		category.GET("/list", h.List)
		category.DELETE("/:id", h.Delete)
		category.GET("/list/product/:id", h.ListCategoryByProduct)
		category.GET("/list/filter", h.ListCategory)
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
func (h *CategoryHander) Create(c *gin.Context) {
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
func (h *CategoryHander) Update(c *gin.Context) {
	var body domain.CategoryUpdateRequest

	// Bind the JSON payload to the body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid request payload")
		return
	}

	var updateCategory func(category domain.CategoryUpdateRequest) error
	updateCategory = func(category domain.CategoryUpdateRequest) error {
		if category.Id == "" {
			category.Id = uuid.NewString()
		}
		// Save the current category
		err = h.db.WithContext(c.Request.Context()).
			Table("categories").
			Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}).
			Create(&category).Error
		if err != nil {
			return err
		}
		for _, subCategory := range category.SubCategories {
			if subCategory.CategoryId == nil {
				subCategory.CategoryId = &category.Id
			}
			err := updateCategory(subCategory)
			if err != nil {
				return err
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
	handleResponse(c, OK, "UPDATED")
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
func (h *CategoryHander) Get(c *gin.Context) {
	var (
		res domain.Category
		id  = c.Param("id")
	)

	err := h.db.
		Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
			return db.Preload("SubCategories")
		}).First(&res, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Category not found")
			return
		}
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
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param parent_id query string false "Parent ID"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/list [get]
func (h *CategoryHander) List(c *gin.Context) {
	var (
		res        []domain.Category
		parentID   = c.Query("parent_id")
		search     = c.Query("search")
		productID  = c.Query("product_id")
		totalCount int64
	)
	// Build the base query
	query := h.db.Model(&domain.Category{})

	// Filter by `parent_id`
	if parentID != "" {
		query = query.Where("category_id = ?", parentID)
	} else {
		// Root categories (no parent)
		query = query.Where("category_id IS NULL")
	}
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Apply search filter if provided
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}

	// Preload SubCategories recursively
	query = query.
		Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
			return db.Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
				return db.Preload("SubCategories") // Har bir keyingi levelni preload qilish
			})
		})

	// Modified recursive CTE to correctly check child categories
	if productID != "" {
		query = query.Select(`
				categories.*,
				EXISTS (
					WITH RECURSIVE category_tree AS (
						SELECT id, category_id, 1 as level
						FROM categories
						WHERE id = categories.id
						UNION ALL
						SELECT c.id, c.category_id, ct.level + 1
						FROM categories c
						INNER JOIN category_tree ct ON c.category_id = ct.id
					)
					SELECT 1
					FROM category_tree ct
					INNER JOIN category_products cp ON cp.category_id = ct.id
					WHERE cp.product_id = ?
				) AS is_open
			`, productID)
	}

	// Execute the query
	err = query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result, totalCount)
}

// Delete godoc
// @Summary Delete a category
// @Description Delete a category from the request body
// @Tags categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id 	body []string true "category IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/{id} [delete]
func (h *CategoryHander) Delete(c *gin.Context) {
	var (
		ids []string
		err error
	)
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.Category{}, "id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// ListCategoryByProduct godoc
// @Summary Get a category
// @Description Get a category from the request body
// @Tags 	categories
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/category/list/product/{id} [get]
func (h *CategoryHander) ListCategoryByProduct(c *gin.Context) {
	var res []domain.Category
	var id = c.Param("id")

	err := h.db.
		Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
			return db.Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
				return db.Preload("SubCategories")
			})
		}).
		Where("categories.category_id IS NULL").
		Select("categories.*, COALESCE(cp.is_open, false) AS is_open").
		Joins("LEFT JOIN category_products cp ON cp.category_id = categories.id AND cp.product_id = ?", id).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// ListCategory godoc
// @Summary Get a category list for filter
// @Description Get a category list for filter
// @Tags 	categories
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	search query string false "Search"
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /category/list/filter [get]
func (h *CategoryHander) ListCategory(c *gin.Context) {
	var (
		search = c.Query("search")
		res    []domain.Category
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	query := h.db.Model(&domain.Category{})

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	err = query.Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
