package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	external := r.Group("/external")
	external.GET("/product/list", h.List)
	external.GET("/category/list", h.CategoryList)
	external.GET("/products/:product_id/stores", h.StoreListByProductId)
	external.POST("/sale", h.CreateSale)
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
// @Param   search 	query     string   false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/product/list 	[GET]
func (h *ExternalHandler) List(c *gin.Context) {
	var (
		res    []domain.ProductExternal
		search = c.Query("search")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res, err = h.service.GetExternalProducts(limit, offset, search)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// StoreListByProductId godoc
// @Summary Get a store list by product id
// @Description Get a store list by product id
// @Tags 	External API
// @Security     BasicAuth
// @Produce 	json
// @Param 	product_id path string true "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/products/{product_id}/stores [get]
func (h *ExternalHandler) StoreListByProductId(c *gin.Context) {
	var (
		res       []domain.StoreExternal
		productId = c.Param("product_id")
	)
	// validate product id
	if err := uuid.Validate(productId); err != nil {
		handleResponse(c, BadRequest, "Invalid product id")
		return
	}
	// get stores by product id
	res, err := h.service.GetExternalStoresByProductId(productId)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
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

// CreateSale godoc
// @Summary Create a sale
// @Description Create a sale
// @Tags 	External API
// @Security     BasicAuth
// @Produce 	json
// @Param 	body body domain.SaleOnline true "Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/external/sale [post]
func (h *ExternalHandler) CreateSale(c *gin.Context) {
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
