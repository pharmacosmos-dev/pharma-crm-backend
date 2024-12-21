package v1

import (
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
		brand = new(domain.BrandRequest)
		err   error
	)
	err = c.ShouldBind(brand)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	brand.Id = uuid.New().String()
	err = b.db.WithContext(c.Request.Context()).Model(&domain.Brand{}).Create(brand).Error
	if err != nil {
		b.log.Error("Error on creating brand: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, brand)
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
	res := new(domain.Brand)
	if err := b.db.First(res, "id = ?", c.Param("id")).Error; err != nil {
		b.log.Error("Error on getting brand: ", err.Error())
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
	var res []*domain.Brand
	if err := b.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		b.log.Error("Error on list brand: ", err.Error())
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
	)

	if err = c.ShouldBindJSON(&brand); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = b.db.WithContext(c.Request.Context()).
		Model(&domain.Brand{}).
		Where("id = ?", c.Param("id")).
		Updates(&brand).Error; err != nil {
		b.log.Error("Error on update brand: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, brand)
}

// Delete godoc
// @Summary Delete a brand
// @Description Delete a brand from the request body
// @Tags brands
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "brand ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /brand/{id} [delete]
func (b *BrandController) Delete(c *gin.Context) {
	if err := b.db.WithContext(c.Request.Context()).Delete(&domain.Brand{}, "id = ?", c.Param("id")).Error; err != nil {
		b.log.Error("Error on delete brand: ", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
