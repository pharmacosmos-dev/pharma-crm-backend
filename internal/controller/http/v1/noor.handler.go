package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
// @Success 	200 {object} []domain.NoorProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/product/list 	[GET]
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
// @Tags 		Noor API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param		updatedAt query   string   false "updatedAt"
// @Param   	limit 	query     int      false "Limit"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorStoreProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/noor/store-product/list 	[GET]
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
		h.log.Error(err)
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	t, _ := json.Marshal(&body)
	h.log.Info("Noor CreateOrder request: %s", string(t))

	// create sale id
	saleId := uuid.New().String()

	// checking product quantity and get collect cart_items
	cartItems, err := h.service.GetOrCheckOnlineCartItems(ctx, body.Products, saleId)
	if err != nil {
		handleServiceResponse(c, http.StatusBadRequest, err)
		return
	}

	// create or get customer
	customer, err := h.service.GetOrCreateCustomerByPhone(ctx, &body.ClientInfo)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}

	// create online sale
	res, err := h.service.CreateOnlineSale(ctx, saleId, body.ShopId, customer, cartItems)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}
	fmt.Println("Noor create online sale: ", res.SaleNumber)

	go h.service.NotifyOnlineOrder(body.ShopId, res.SaleNumber)

	handleResponseNoor(c, http.StatusOK, domain.OnlineOrderResponse{Message: "success", OrderId: res.SaleNumber})
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
