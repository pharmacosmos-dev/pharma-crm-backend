package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type FinanceCategoryHandler struct {
	*Handler
}

func (h *Handler) NewFinanceCategoryHandler(r *gin.RouterGroup) {
	category := &FinanceCategoryHandler{h}
	category.FinanceCategoryRoutes(r)
}

func (h *FinanceCategoryHandler) FinanceCategoryRoutes(r *gin.RouterGroup) {
	financeCategory := r.Group("/finance-category")
	{
		financeCategory.POST("", h.Create)
		financeCategory.GET("/:id", h.Get)
		financeCategory.GET("/list", h.List)
		financeCategory.PUT("/:id", h.Update)
		financeCategory.DELETE("", h.Delete)
	}
}

// Create godoc
// @Summary Create a new finance category
// @Description Create a new finance category from the request body
// @Tags finance categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param finance_category body domain.FinanceCategoryRequest true "Finance category information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-category [post]
func (h *FinanceCategoryHandler) Create(c *gin.Context) {
	var (
		body domain.FinanceCategoryRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Status = constants.GeneralStatusActive
	// create finance category
	err = h.db.WithContext(c.Request.Context()).
		Table("finance_categories").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a finance category by id
// @Description Get a finance category by id from the request body
// @Tags finance categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Finance category ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-category/{id} [get]
func (h *FinanceCategoryHandler) Get(c *gin.Context) {
	var (
		res domain.FinanceCategory
		id  = c.Param("id")
	)
	// get finance category by id
	err := h.db.First(&res, id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a list of finance categories
// @Description Get a list of finance categories from the request body
// @Tags finance categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param search query string false "Search"
// @Param account_group query string false "income || expense"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-category/list [get]
func (h *FinanceCategoryHandler) List(c *gin.Context) {
	var (
		search       = c.Query("search")
		accountGroup = c.Query("account_group")
		res          []domain.FinanceCategory
		totalCount   int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// build query
	query := h.db.Model(&domain.FinanceCategory{}).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Children", func(db *gorm.DB) *gorm.DB {
				return db.Preload("Children")
			})
		})
	// filter section
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	if accountGroup != "" {
		query = query.Where("account_group = ?", accountGroup)
	}
	query = query.Where("status = 'active' AND deleted_at IS NULL")
	// complete the query
	err = query.Count(&totalCount).Limit(limit).Offset(offset).Order("created_at DESC").Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// response
	data := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a finance category
// @Description Update a finance category from the request body
// @Tags finance categories
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Finance category ID"
// @Param finance_category body domain.FinanceCategoryRequest true "Finance category information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-category/{id} [put]
func (h *FinanceCategoryHandler) Update(c *gin.Context) {
	var (
		body domain.FinanceCategoryRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update finance category
	var updateCategory func(category domain.FinanceCategoryRequest) error
	updateCategory = func(category domain.FinanceCategoryRequest) error {

		// Save the current category
		res, err := h.service.CreateOrUpdateFinanceCategory(&category)
		if err != nil {
			return err
		}
		for _, children := range category.Children {
			if children.ParentId == nil {
				children.ParentId = &res.Id
			}
			err := updateCategory(children)
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

// Delete godoc
// @Summary Delete a finance category
// @Description Delete a finance category from the request body
// @Tags finance categories
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id body []int true "Finance category IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /finance-category [delete]
func (h *FinanceCategoryHandler) Delete(c *gin.Context) {
	var ids []int
	// bind request body
	if err := c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// delete finance category
	err := h.db.
		WithContext(c.Request.Context()).
		Where("id IN (?)", ids).
		Updates(map[string]any{"deleted_at": time.Now(), "status": "deleted"}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
