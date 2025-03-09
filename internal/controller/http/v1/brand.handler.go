package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type BrandController struct {
	*Handler
}

func (h *Handler) NewBrandController(r *gin.RouterGroup) {
	brand := &BrandController{h}
	brand.BrandRoutes(r)
}

func (b *BrandController) BrandRoutes(r *gin.RouterGroup) {
	apiGroup := r.Group("/brand")
	{
		apiGroup.POST("", b.Create)
		apiGroup.GET("/:id", b.Get)
		apiGroup.GET("/list", b.List)
		apiGroup.PUT("/:id", b.Update)
		apiGroup.DELETE("/:id", b.Delete)
	}

}

// Create godoc
// @Summary Create a brand
// @De a brand from the request body
// @Tags brands
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param brand body domain.BrandRequest true "Brand information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand [post]
func (b *BrandController) Create(c *gin.Context) {
	var (
		brand domain.BrandRequest
		err   error
	)
	if err = c.ShouldBindJSON(&brand); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	brand.Id = uuid.New().String()
	err = b.db.WithContext(c.Request.Context()).Model(&domain.Brand{}).Create(brand).Error
	if err != nil {
		b.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get a brand
// @Description Get a brand from the request body
// @Tags brands
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "brand ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand/{id} [get]
func (b *BrandController) Get(c *gin.Context) {
	var res domain.Brand
	var id = c.Param("id")
	err := b.db.First(&res, "id = ?", id).Error
	if err != nil {
		b.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a brand
// @Description Get a brand from the request body
// @Tags brands
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand/list [get]
func (b *BrandController) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	query := b.db.Model(&domain.Brand{})
	var search = c.Query("search")
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	var res []*domain.Brand
	err = query.Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		b.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a brand
// @Description Update a brand from the request body
// @Tags brands
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "brand ID"
// @Param brand body domain.BrandRequest true "Brand information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand/{id} [put]
func (b *BrandController) Update(c *gin.Context) {
	var (
		brand domain.BrandRequest
		err   error
		id    = c.Param("id")
	)
	// bind request body
	if err = c.ShouldBindJSON(&brand); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update brand
	err = b.db.
		WithContext(c.Request.Context()).
		Model(&domain.Brand{}).
		Where("id = ?", id).
		Updates(&brand).Error
	if err != nil {
		b.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, brand)
}

// Delete godoc
// @Summary Delete a brand
// @Description Delete a brand from the request body
// @Tags 	brands
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "brand ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand/{id} [delete]
func (b *BrandController) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := b.db.
		WithContext(c.Request.Context()).
		Delete(&domain.Brand{}, "id = ?", id).Error
	if err != nil {
		b.log.Error("Error on delete brand: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
