package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type PartnerApiHandler struct {
	*Handler
}

func (h *Handler) NewPartnerApiHandler(r *gin.RouterGroup) {
	partnerApi := &PartnerApiHandler{h}
	partnerApi.PartnerApiRoutes(r)
}

// Catalog APIs for external partners. They return the same data as the Noor APIs:
// only non-prescription products that have an Uzum online price.
func (h *PartnerApiHandler) PartnerApiRoutes(r *gin.RouterGroup) {
	partner := r.Group("/partner")
	partner.GET("/store/list", h.StoreList)
	partner.GET("/category/list", h.CategoryList)
	partner.GET("/product/list", h.ProductList)
	partner.GET("/store-product/list", h.StoreProductList)
}

// Store List godoc
// @Summary 	List stores
// @Description List active stores. Store id is used as shopId in /partner/store-product/list
// @Tags 		Partner API
// @Security    BearerAuth
// @Produce 	json
// @Success 	200 {object} []domain.NoorStore
// @Failure 	401 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/partner/store/list [get]
func (h *PartnerApiHandler) StoreList(c *gin.Context) {
	res, err := h.service.GetNoorStores()
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, emptyIfNil(res))
}

// Category List godoc
// @Summary 	List categories
// @Description List all categories, parents before children. parent_id is empty for root categories
// @Tags 		Partner API
// @Security    BearerAuth
// @Produce 	json
// @Success 	200 {object} []domain.NoorCategory
// @Failure 	401 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/partner/category/list [get]
func (h *PartnerApiHandler) CategoryList(c *gin.Context) {
	res, err := h.service.GetNoorCategories()
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, emptyIfNil(res))
}

// Product List godoc
// @Summary 	List products
// @Description List non-prescription products available for online orders
// @Tags 		Partner API
// @Security    BearerAuth
// @Produce 	json
// @Param   	limit 	query     int      false "Limit (default 10)"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	401 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/partner/product/list [get]
func (h *PartnerApiHandler) ProductList(c *gin.Context) {
	var params domain.NoorQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponseNoor(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, err := h.service.GetNoorProducts(&params)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, emptyIfNil(res))
}

// Store Product List godoc
// @Summary 	List store products
// @Description List stock quantity and price of products in stores
// @Tags 		Partner API
// @Security    BearerAuth
// @Produce 	json
// @Param		shopId  query   string   false "Store id from /partner/store/list, all stores if empty"
// @Param   	limit 	query     int      false "Limit (default 10)"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorStoreProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	401 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/partner/store-product/list [get]
func (h *PartnerApiHandler) StoreProductList(c *gin.Context) {
	var params domain.NoorQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponseNoor(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, err := h.service.GetNoorStoreProducts(&params)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, emptyIfNil(res))
}

// emptyIfNil makes an empty result encode as [] instead of null
func emptyIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
