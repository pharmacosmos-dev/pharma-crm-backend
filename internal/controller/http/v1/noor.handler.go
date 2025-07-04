package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type NoorHandler struct {
	*Handler
}

func (h *Handler) NewNoorHandler(r *gin.RouterGroup) {
	noor := &NoorHandler{h}
	noor.NoorRoutes(r)
}

func (h *NoorHandler) NoorRoutes(r *gin.RouterGroup) {
	noor := r.Group("/noor")
	noor.GET("/product/list", h.ProductList)
	noor.GET("/store-product/list", h.StoreProductList)
	noor.GET("/store/list", h.StoreList)
	noor.GET("/category/list", h.CategoryList)
	noor.POST("/order", h.CreateOrder)
}

// List Products
// @Summary List Products
// @Description List Products
// @Tags 	Noor API
// @Security     BasicAuth
// @Accept 	json
// @Produce json
// @Param   limit 	query     int      false "Limit"
// @Param   offset 	query     int      false "Offset"
// @Success 200 {object} []domain.NoorProduct
// @Failure 400 {object} v1.IntegrationErrorResponse
// @Failure 500 {object} v1.IntegrationErrorResponse
// @Router 	/noor/product/list 	[GET]
func (h *NoorHandler) ProductList(c *gin.Context) {
	var (
		res   []domain.NoorProduct
		param domain.NoorQueryParam
	)
	err := c.ShouldBindQuery(&param)
	if err != nil {
		h.log.Warn("ERROR on binding query param: %v", err)
		handleResponseNoor(c, http.StatusBadRequest, "invalid.query.param")
		return
	}
	// get default product
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, err = h.service.GetNoorProducts(&param)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 	Noor API
// @Security     BasicAuth
// @Accept 	json
// @Produce json
// @Param	updatedAt query   string   false "updatedAt"
// @Param   limit 	query     int      false "Limit"
// @Param   offset 	query     int      false "Offset"
// @Success 200 {object} []domain.NoorStoreProduct
// @Failure 400 {object} v1.IntegrationErrorResponse
// @Failure 500 {object} v1.IntegrationErrorResponse
// @Router 	/noor/store-product/list 	[GET]
func (h *NoorHandler) StoreProductList(c *gin.Context) {
	var param domain.NoorQueryParam

	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		h.log.Warn("ERROR on binding noor query param: %v", err)
		handleResponseNoor(c, http.StatusBadRequest, "invalied.query.param")
		return
	}

	// get store products info
	res, err := h.service.GetNoorStoreProducts(&param)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 	Noor API
// @Security     BasicAuth
// @Accept 	json
// @Produce json
// @Success 200 {object} []domain.NoorStore
// @Failure 400 {object} v1.IntegrationErrorResponse
// @Failure 500 {object} v1.IntegrationErrorResponse
// @Router 	/noor/store/list 	[GET]
func (h *NoorHandler) StoreList(c *gin.Context) {
	// get store products info
	res, err := h.service.GetNoorStores()
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Category List godoc
// @Summary Get a category list for filter
// @Description Get a category list for filter
// @Tags 		Noor API
// @Security    BasicAuth
// @Produce 	json
// @Param 		limit query int false "Limit"
// @Param 		offset query int false "Offset"
// @Success 200 {object} []domain.NoorCategory
// @Failure 400 {object} v1.IntegrationErrorResponse
// @Failure 500 {object} v1.IntegrationErrorResponse
// @Router 	/noor/category/list [get]
func (h *NoorHandler) CategoryList(c *gin.Context) {
	var (
		param domain.NoorQueryParam
		res   []domain.NoorCategory
	)
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		h.log.Warn("ERROR on binding noor query param: %v", err)
		handleResponseNoor(c, http.StatusBadRequest, "invalid.query.param")
		return
	}

	// get default product
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	query := `
	WITH RECURSIVE category_hierarchy AS (
		-- Start with root categories (those with no parent)
		SELECT
			id,
			name,
			category_id AS parent_id,
			photo,
			ARRAY[id] AS path
		FROM categories
		WHERE category_id IS NULL

		UNION ALL

		-- Recursively get children
		SELECT
			c.id,
			c.name,
			c.category_id AS parent_id,
			c.photo,
			ch.path || c.id
		FROM categories c
		INNER JOIN category_hierarchy ch ON c.category_id = ch.id
	)
	SELECT
		id,
		name,
		parent_id,
		photo
	FROM category_hierarchy
	ORDER BY path;
	`
	err = h.db.Raw(query).Scan(&res).Error
	if err != nil {
		h.log.Error("ERROR on getting noor categories: %v", err)
		handleResponseNoor(c, http.StatusInternalServerError, "internal.server.error")
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// CreateSale godoc
// @Summary Create a sale
// @Description Create a sale
// @Tags 	Noor API
// @Security     BasicAuth
// @Produce 	json
// @Param 	body body 	domain.OnlineOrderRequest true "Order Request body"
// @Success 200 {object} domain.OnlineOrderResponse
// @Failure 400 {object} v1.IntegrationErrorResponse
// @Failure 500 {object} v1.IntegrationErrorResponse
// @Router 	/noor/order [post]
func (h *NoorHandler) CreateOrder(c *gin.Context) {
	var body domain.OnlineOrderRequest

	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponseNoor(c, http.StatusBadRequest, "invalid.request.body")
		return
	}

	// // create sale id
	saleID := uuid.New().String()

	// checking product quantity and get collect cart_items
	cartItems, err := h.service.GetOrCheckOnlineCartItems(body.Products, saleID)
	if err != nil {
		handleResponseNoor(c, http.StatusBadRequest, err.Error())
		return
	}

	// create or get customer
	customer, err := h.service.GetOrCreateCustomerByPhone(&body.ClientInfo)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, "client.not.created.or.get")
		return
	}

	// create online sale
	res, err := h.service.CreateOnlineSale(saleID, body.ShopId, customer, cartItems)
	if err != nil {
		h.log.Error(err)
		handleResponseNoor(c, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponseNoor(c, http.StatusOK, domain.OnlineOrderResponse{Message: "success", OrderID: res.SaleNumber})
}
