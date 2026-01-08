package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
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
	noor.POST("/send-ws-test-message", h.SendWSTestMessage)
}

// List Products
// @Summary List Products
// @Description List Products
// @Tags 		Noor API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param   	limit 	query     int      false "Limit"
// @Param   	offset 	query     int      false "Offset"
// @param		updatedAt query   string   false "updatedAt"
// @Success 	200 {object} []domain.NoorProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/product/list 	[GET]
func (h *NoorHandler) ProductList(c *gin.Context) {
	var params domain.NoorQueryParam

	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}
	// get default product
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, err := h.service.GetNoorProducts(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 		Noor API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param		updatedAt query   string   false "updatedAt"
// @Param		shopId  query   string   false "Shop ID"
// @Param   	limit 	query     int      false "Limit"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorStoreProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/store-product/list 	[GET]
func (h *NoorHandler) StoreProductList(c *gin.Context) {
	var params domain.NoorQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}

	// get store products info
	res, err := h.service.GetNoorStoreProducts(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 		Noor API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Success 	200 {object} []domain.NoorStore
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/store/list 	[GET]
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
// @Success 	200 {object} []domain.NoorCategory
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/category/list [get]
func (h *NoorHandler) CategoryList(c *gin.Context) {
	var (
		params domain.NoorQueryParam
		res    []domain.NoorCategory
	)
	// bind query param
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}

	// get default product
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	query := `
	WITH RECURSIVE category_hierarchy AS (
		-- Start with root categories (those with no parent)
		SELECT
			id,
			name AS name_ru,
			name_uz,
            name_en,
            name_kr,
			category_id AS parent_id,
			photo,
			ARRAY[id] AS path
		FROM categories
		WHERE category_id IS NULL

		UNION ALL

		-- Recursively get children
		SELECT
			c.id,
			c.name as name_ru,
			c.name_uz,
			c.name_en,
			c.name_kr,
			c.category_id AS parent_id,
			c.photo,
			ch.path || c.id
		FROM categories c
		INNER JOIN category_hierarchy ch ON c.category_id = ch.id
	)
	SELECT
		id,
		name_ru,
		name_uz,
		name_en,
		name_kr,
		parent_id,
		photo
	FROM category_hierarchy
	ORDER BY path;
	`
	err = h.db.Raw(query).Scan(&res).Error
	if err != nil {
		h.log.Errorf("could not get categories for noor: %v", err)
		handleResponseNoor(c, http.StatusInternalServerError, domain.InternalServerError)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// CreateSale godoc
// @Summary 	Create a sale
// @Description Create a sale
// @Tags 		Noor API
// @Security    BasicAuth
// @Produce 	json
// @Param 		body body 	domain.OnlineOrderRequest true "Order Request body"
// @Success 	200 {object} domain.OnlineOrderResponse
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/order [post]
func (h *NoorHandler) CreateOrder(c *gin.Context) {
	var body domain.OnlineOrderRequest
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind noor order create request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	orderNumber, err := h.service.CreateNoorSale(ctx, &body)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			handleResponseNoor(c, http.StatusConflict, notAddErr.Data)
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, domain.OnlineOrderResponse{Message: "success", OrderId: orderNumber})
}

func (h *NoorHandler) SendWSTestMessage(c *gin.Context) {
	storeID := c.Query("store_id")
	if storeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store_id is required"})
		return
	}

	go h.service.NotifyOnlineOrder(storeID, 12345)

	c.JSON(http.StatusOK, gin.H{"message": "Test message sent"})
}
