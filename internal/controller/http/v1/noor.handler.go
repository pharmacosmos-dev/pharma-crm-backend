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
	noor.GET("/product/list", h.List)
	noor.GET("/store-product/list", h.StoreProductList)
	noor.GET("/store/list", h.StoreList)
	noor.GET("/category/list", h.CategoryList)
	noor.POST("/sale", h.CreateSale)
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
func (h *NoorHandler) List(c *gin.Context) {
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
// @Router 	/noor/store-product/list 	[GET]
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
		res   []domain.Category
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

	err = h.db.Model(&domain.Category{}).Find(&res).Error
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
// @Router 	/noor/sale [post]
func (h *NoorHandler) CreateSale(c *gin.Context) {
	var (
		body domain.SaleOnline
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// create sale id
	saleId := uuid.New().String()

	// create online sale
	err = h.service.CreateOnlineSale(tx, saleId, body.TotalAmount)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Cannot create the online sale, something went wrong")
		tx.Rollback()
		return
	}

	// create online cart items
	for i := range body.Items {
		// create online cart item
		err = h.service.CreateOnlineCartItem(tx, &body.Items[i], saleId)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Cannot collect the items, something went wrong")
			tx.Rollback()
			return
		}
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Cannot commit the transaction")
		tx.Rollback()
		return
	}

	handleResponse(c, OK, "Success")
}
