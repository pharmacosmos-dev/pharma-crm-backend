package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
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
// @Param	updatedAt query   string   false "UpdateAt"
// @Param   limit 	query     int      false "Limit"
// @Param   offset 	query     int      false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
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
// @Param 	body body domain.SaleOnline true "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/noor/order [post]
func (h *NoorHandler) CreateOrder(c *gin.Context) {
	var body domain.OnlineOrderRequest

	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer recoverTransaction(tx, h.log)

	// // create sale id
	// saleId := uuid.New().String()

	// // create online sale
	// err = h.service.CreateOnlineSale(tx, )
	// if err != nil {
	// 	h.log.Error(err)
	// 	handleResponse(c, InternalError, "Cannot create the online sale, something went wrong")
	// 	tx.Rollback()
	// 	return
	// }

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Cannot commit the transaction")
		tx.Rollback()
		return
	}

	handleResponse(c, OK, "Success")
}
