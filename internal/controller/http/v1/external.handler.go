package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
}

// List Products
// @Summary List Products
// @Description List Products
// @Tags 	External
// @Security     BasicAuth
// @Accept 	json
// @Produce json
// @Param   search 	query     string   false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/product/list 	[GET]
func (h *ExternalHandler) List(c *gin.Context) {
	var (
		res             []domain.ProductExternal
		search          = c.Query("search")
		searchCondition string
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		searchCondition = fmt.Sprintf("WHERE p.name ILIKE '%s' OR p.barcode ILIKE '%s'", search, search)
	}
	query := fmt.Sprintf(`
	SELECT 
		p.id, p.name, p.barcode, p.photos, p.description, 
		sum(sp.pack_quantity) as quantity, u.short_name as unit_name
	FROM products p 
	JOIN store_products sp ON p.id = sp.product_id 
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	%s
	GROUP BY p.id, u.short_name LIMIT ? OFFSET ?`, searchCondition)
	err = h.db.Raw(query, limit, offset).Scan(&res).Error
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
			ON s.id = sp.store_id WHERE sp.product_id = ?`, res[i].Id).
			Scan(&res[i].Stores).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, OK, res)
}
