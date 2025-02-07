package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type ExternalHandler struct {
	*Handler
}

func (h *Handler) NewExternalHandler(r *gin.RouterGroup) {
	external := &ExternalHandler{h}
	external.ExternalRoutes(r)
}

func (h *ExternalHandler) ExternalRoutes(r *gin.RouterGroup) {
	r.GET("/external/product/list", h.List)
	r.GET("/external/category/list", h.CategoryList)
}

// List Products
// @Summary List Products
// @Description List Products
// @Tags 	External API
// @Security     BasicAuth
// @Accept 	json
// @Produce json
// @Param   limit 	query     int      false "Limit"
// @Param   offset 	query     int      false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/product/list 	[GET]
func (h *ExternalHandler) List(c *gin.Context) {
	var (
		res []domain.ProductExternal
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		Table("products p").
		Preload("Categories").
		Select(`
		p.id, p.name, p.barcode, p.photos, p.description, 
		sum(sp.pack_quantity) as quantity, u.short_name as unit_name`).
		Joins("JOIN store_products sp ON p.id = sp.product_id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Group("p.id, u.short_name").
		Limit(limit).Offset(offset).
		Order("p.created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// var stores []domain.StoreExternal
	for i := range res {
		err = h.db.Raw(`
		SELECT 
			s.id, s.name, s.address, s.location, 
			sp.pack_quantity as quantity, sp.unit_quantity, 
			sp.retail_price, sp.expire_date
		FROM 
			stores s JOIN store_products sp 
			ON s.id = sp.store_id WHERE s.is_active = TRUE AND sp.product_id = ?`, res[i].Id).
			Scan(&res[i].Stores).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, OK, res)
}

// Category List godoc
// @Summary Get a category list for filter
// @Description Get a category list for filter
// @Tags 	External API
// @Security     BasicAuth
// @Produce 	json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/category/list [get]
func (h *ExternalHandler) CategoryList(c *gin.Context) {
	var res []domain.Category
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Preload SubCategories recursively
	query := h.db.Model(&domain.Category{}).
		Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
			return db.Preload("SubCategories", func(db *gorm.DB) *gorm.DB {
				return db.Preload("SubCategories")
			})
		})

	err = query.
		Limit(limit).
		Offset(offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
