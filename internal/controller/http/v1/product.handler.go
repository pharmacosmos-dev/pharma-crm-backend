package v1

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	gen "math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductHandler struct {
	*Handler
}

func (h *Handler) NewProductHandler(r *gin.RouterGroup) {
	product := &ProductHandler{h}
	product.ProductRoutes(r)
}

func (h *ProductHandler) ProductRoutes(r *gin.RouterGroup) {
	product := r.Group("/product")
	{
		product.POST("", h.Create)
		product.GET("/:id", h.Get)
		product.GET("/list", h.List)
		product.GET("/product-list", h.ProductList)
		product.GET("/export-excel", h.ExportProductExcel)
		product.PUT("/:id", h.Update)
		product.POST("/excel-upload", h.UploadProduct)
		product.GET("/producer", h.GetProducerList)
		product.GET("/similar/:id", h.SimilarProducts)
		product.GET("/store/:id", h.GetProductsForSearch)
		product.GET("/import/:id", h.GetProductImports)
		product.DELETE("/hard-delete", h.HardDelete)
		product.DELETE("/soft-delete", h.SoftDelete)
		product.GET("/store-product/:id", h.ListStoreProductProductId)
		product.POST("/store/barcode", h.AddStoreProductByBarcode)
		product.POST("/generate-barcode", h.GenerateBarcode)
		product.GET("/total-status-count", h.TotalStatusCount)
		product.PUT("/update-barcode/:id", h.UpdateProductUnitValues)
		product.POST("/attach-barcode", h.AttachBarcode)
		product.PATCH("/is-marking", h.UpdateIsMarking)
		product.GET("/:id/product-movement", h.ProductMovements)
		product.GET("/:id/product-movement/export-excel", h.ExportProductMovementsExcel)
		product.GET("/export-arzon", h.ArzonProductExport)
		product.GET("/list-arzon", h.ArzonProductList)
		product.GET("/list-by-import", h.GetProductsByImport)
		product.GET("/export-by-import", h.GetProductsByImportExport)
		product.PUT("/update-mxik-import/:id", h.UpdateProductIkpuForStoreProduct)
		product.PATCH("/store-is-marking", h.UpdateStoreProductIsMarking)
		product.POST("/min-max", h.CreateMinMaxProduct)
		product.PUT("/min-max/:id", h.UpdateMinMaxProduct)
		product.GET("/min-max/:id", h.GetMinMaxProductById)
		product.GET("/list-min-max", h.GetMinMaxProducts)
		product.GET("/export-min-max", h.ExportMinMaxProducts)
		product.POST("/exclude", h.CreateExcludedProduct)
		product.DELETE("/exclude/:id", h.DeleteExcludedProduct)
		product.GET("/excluded-list", h.ListExcludedProducts)
		product.GET("/excluded-export", h.ExportExcludedProductsExcel)
		product.PUT("/update-packaging", h.UpdatePackaging)
		product.POST("/list-store-products", h.ListStoreProducts)
		product.POST("/photo-alert", h.CreateProductPhotoAlert)
		product.POST("/photo-alert/list", h.ListProductPhotoAlert)
		product.DELETE("/photo-alert/:id", h.DeleteProductPhotoAlert)
	}
}

// Create godoc
// @Summary Create a new product
// @Description Create a new product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param product body domain.ProductRequest true "Product information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var body domain.ProductRequest

	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateProduct(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "product ID"
// @Param store_id query string true "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	var (
		id      = c.Param("id")
		storeId = c.Query("store_id")
	)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetProductById(ctx, id, storeId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id query string false "Store ID"
// @Param category_id query string false "Category ID"
// @Param producer_id query string false "Producer ID"
// @Param no_barcode query bool false "No Barcode"
// @Param order query string false "Order by (+name || -name || +expire_date || -expire_date)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/list [get]
func (h *ProductHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	// bind request params
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// Pagination parameters
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// get products list
	products, totalCount, err := h.service.GetProducts(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Prepare the response
	result := utils.ListResponse(products, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id query string false "Store ID"
// @Param category_id query string false "Category ID"
// @Param producer_id query string false "Producer ID"
// @Param no_barcode query bool false "No Barcode"
// @Param order query string false "Order by (+name || -name || +expire_date || -expire_date)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/export-excel [get]
func (h *ProductHandler) ExportProductExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// Pagination parameters
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// Create excel file
	f := excelize.NewFile()

	if params.StoreId != "" {
		// get products list
		res, _, err := h.service.GetProducts(ctx, &params, user)
		if err != nil {
			handleServiceResponse(c, nil, err)
			return
		}

		f, err = h.productListExportByStoreId(f, res)
		if err != nil {
			handleServiceResponse(c, nil, domain.InternalServerError)
			return
		}
	} else {
		res, err := h.service.GetProductsByStores(ctx, &params, user)
		if err != nil {
			handleServiceResponse(c, nil, err)
			return
		}

		f, err = h.productListExport(f, res)
		if err != nil {
			handleServiceResponse(c, nil, domain.InternalServerError)
			return
		}
	}
	saveExcelToUploads(c, f, *h.log, "products")
}

// Get godoc
// @Summary Get total count of products by status
// @Description Get total count of products by status
// @Tags products
// @Security     BearerAuth
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id query string false "Store ID"
// @Param category_id query string false "Category ID"
// @Param producer_id query string false "Producer ID"
// @Param no_barcode query bool false "No Barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/total-status-count [get]
func (h *ProductHandler) TotalStatusCount(c *gin.Context) {
	// get user_id from the context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam

	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetProductStats(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/product-list [get]
func (h *ProductHandler) ProductList(c *gin.Context) {
	var products []struct {
		Id   string `gorm:"id" json:"id"`
		Name string `gorm:"name" json:"name"`
	}
	search := c.Query("search")
	// get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get products list
	query := h.db.Model(&domain.Product{})
	// add search fileter
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ? OR barcode LIKE ?", search, search)
	}
	var totalCount int64
	// complete query
	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&products).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Prepare the response
	data := utils.ListResponse(products, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// Get godoc
// @Summary Update a product
// @Description Update a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   id path string true "Product ID"
// @Param   input body domain.ProductUpdateRequest true "Product information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	var (
		body      domain.ProductUpdateRequest
		err       error
		productID = c.Param("id")
	)
	if productID == "" || productID == "undefined" {
		handleResponse(c, BadRequest, "Product ID is required")
		return
	}

	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	// match array
	body.Photos = utils.StringArray(body.Photos)
	err = tx.
		Model(&domain.Product{}).
		Where("id = ?", productID).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if len(body.CategoryIds) > 0 {
		var categoryProducts = make([]domain.CategoryProduct, len(body.CategoryIds))
		for i, categoryId := range body.CategoryIds {
			categoryProducts[i] = domain.CategoryProduct{
				ProductId:  productID,
				CategoryId: categoryId,
			}
		}
		err = tx.Table("category_products").
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "product_id"}, {Name: "category_id"}},
				DoNothing: false,
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
			}).Create(&categoryProducts).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	err = tx.Commit().Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/producer [get]
func (h *ProductHandler) GetProducerList(c *gin.Context) {
	var (
		res    []*domain.ProductProducer
		err    error
		search = c.Query("search")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.
		Model(&domain.Product{}).
		Select("DISTINCT manufacturer")
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("manufacturer ILIKE ?", search)
	}
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

// SimilarProducts godoc
// @Summary Get similar products
// @Description Get similar products based on category
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/similar/{id} [get]
func (h *ProductHandler) SimilarProducts(c *gin.Context) {
	var (
		id  = c.Param("id")
		res []domain.StoreProductResponse
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res, err = h.service.SimilarProducts(c.Request.Context(), id, offset, limit)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Step 3: Return the response
	handleResponse(c, OK, res)
}

// StoreProducts godoc
// @Summary Get products by store
// @Description Get products by store
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param id path string true "Store ID"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store/{id} [get]
func (h *ProductHandler) GetProductsForSearch(c *gin.Context) {
	var (
		params  domain.StoreProductQueryParam
		storeId = c.Param("id")
	)
	// bind query params
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.StoreId = storeId
	// get limit offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get store products list
	res, err := h.service.GetProductsForSearch(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// GetStoreProductByBarcode godoc
// @Summary Get store product by barcode
// @Description Get store product by barcode
// @Tags 	products
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.StoreProductBarcodeRequest true "Request body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store/barcode [POST]
func (h *ProductHandler) AddStoreProductByBarcode(c *gin.Context) {
	var (
		body domain.StoreProductBarcodeRequest
		err  error
	)
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var count int64
	err = h.db.Model(&domain.Sale{}).Where("id = ? AND status = 'completed'", body.SaleID).Count(&count).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if count > 0 {
		handleResponse(c, CONFLICT, "Sale already completed")
		return
	}

	// get store_product by barcode
	var storeProduct domain.StoreProduct
	err = h.db.First(&storeProduct, "id = ?", body.ID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get cart_items by store_product_id and status = 'pending'
	var cartItem domain.CartItem
	err = h.db.First(&cartItem, "store_product_id = ? AND status = 'pending' AND sale_id = ?", storeProduct.Id, body.SaleID).Error
	if err == nil {
		// check quantity is enough in store_products table
		if storeProduct.PackQuantity < cartItem.Quantity+1 {
			handleResponse(c, CONFLICT, gin.H{
				"message":                "Not enough Product",
				"pack_quantity":          storeProduct.PackQuantity,
				"unit_quantity":          storeProduct.UnitQuantity,
				"received_pack_quantity": 1,
				"received_unit_quantity": cartItem.UnitQuantity,
			})
			return
		}
		// update cart_item
		newQuantity := cartItem.Quantity + 1
		cartItem.TotalPrice = storeProduct.RetailPrice * float64(newQuantity)
		err = h.db.Exec(`UPDATE cart_items SET quantity = ?, total_price = ? WHERE id = ?`,
			newQuantity, cartItem.TotalPrice, cartItem.Id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "ADDED")
		return
	} else if errors.Is(err, gorm.ErrRecordNotFound) && storeProduct.PackQuantity > 0 {
		// calculation discount price and percent
		var discountPercent, discountPrice, discountAmount float64
		if body.DiscountType == "percent" && body.DiscountValue <= 100 {
			discountAmount = storeProduct.RetailPrice * body.DiscountValue / 100
			discountPercent = body.DiscountValue
		} else if body.DiscountType == "cash" {
			discountAmount = body.DiscountValue
			discountPercent = body.DiscountValue * 100 / storeProduct.RetailPrice
		} else {
			handleResponse(c, BadRequest, "Discount type or value is invalid")
			return
		}
		if discountAmount > 0 {
			discountPrice = storeProduct.RetailPrice - discountAmount
		}
		// create new cart_item
		err = h.db.Exec(`
		INSERT INTO cart_items(
			id, store_product_id, 
			employee_id, sale_id, 
			quantity, unit_price, 
			total_price, status, discount_type, 
			discount_value, discount_price, discount_amount
			) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), storeProduct.Id, userId.(string), body.SaleID, 1,
			storeProduct.RetailPrice, storeProduct.RetailPrice, constants.GeneralStatusPending,
			body.DiscountType, discountPercent, discountPrice, discountAmount).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "ADDED")
		return

	}

	handleResponse(c, CONFLICT, gin.H{
		"message":                "Not enough Product",
		"pack_quantity":          storeProduct.PackQuantity,
		"unit_quantity":          storeProduct.UnitQuantity,
		"received_pack_quantity": 1,
		"received_unit_quantity": 0,
	})
}

// GetProductImports godoc
// @Summary Get product imports
// @Description Get product imports
// @Tags products
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param id path string true "Product ID"
// @Param store_id query string false "Store ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/import/{id} [get]
func (h *ProductHandler) GetProductImports(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var (
		storeId    = c.Query("store_id")
		productId  = c.Param("id")
		res        []domain.ImportDetail
		totalCount int64
	)

	// check user role
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			storeId = user.StoreId
		}
	}

	// get limit offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// build query
	query := h.db.
		Model(&domain.ImportDetail{}).
		Table("import_details imd").
		Preload("Import", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store")
		}).
		Select(`imd.*, imd.retail_price * imd.received_count AS received_amount, imd.retail_price * imd.accepted_count AS accepted_amount`).
		Where("imd.product_id = ?", productId)
	if storeId != "" {
		query = query.
			Joins("INNER JOIN imports ON imports.id = imd.import_id").
			Where("imports.store_id = ?", storeId)
	}
	// complete query
	err = query.
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("imd.created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get _meta data for pagination information
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// ListStoreProductProductId godoc
// @Summary Get store products by product_id
// @Description Get store products by product_id
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id 			path  string 	true "ProductId"
// @Param 	limit 		query int 		false "Limit"
// @Param 	offset 		query int 		false "Offset"
// @Param 	store_id 	query string 	false "StoreId"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store-product/{id} [get]
func (h *ProductHandler) ListStoreProductProductId(c *gin.Context) {
	// get user_id from header context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, InternalError, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam

	params.ProductId = c.Param("id")
	params.StoreId = c.Query("store_id")

	// get limit, offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	params.Limit = limit
	params.Offset = offset

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetStoreProductsByProductId(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	data := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, data)
}

// UpdateBarcode godoc
// @Summary Update barcode
// @Description Update barcode
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body domain.UpdateBarcodeRequest true "Update barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/update-barcode/{id} [put]
func (h *ProductHandler) UpdateProductUnitValues(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	if !helper.IsAdmin(user) {
		handleServiceResponse(c, UNAUTHORIZED, domain.ForbiddinError)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var id = c.Param("id")
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.UpdateBarcodeRequest
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	body.Id = id
	err := h.service.UpdateProductUnitValues(ctx, &body, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// HardDelete godoc
// @Summary Hard delete a product
// @Description Hard delete a product
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   ids body []string true "Product IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/hard-delete [delete]
func (h *ProductHandler) HardDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Delete(&domain.CategoryProduct{}, "product_id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Delete(&domain.StoreProduct{}, "product_id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Delete(&domain.Product{}, "id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Get godoc
// @Summary Soft Delete a product
// @Description Soft Delete a product from the request body
// @Tags 	products
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body 	body []string true "product IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/soft-delete [delete]
func (h *ProductHandler) SoftDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("products").
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_active": false,
			"status":    "deleted"}).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// GenerateBarcode godoc
// @Summary Generate a product barcode
// @Description Generate a product barcode
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/generate-barcode [POST]
func (h *ProductHandler) GenerateBarcode(c *gin.Context) {
	const barcodeLength = 13
	var barcode string

	for {
		// Generate random 13-digit barcode
		barcode = generateRandomBarcode(barcodeLength)

		// Check if barcode already exists in the database
		var count int64
		err := h.db.Model(&domain.Product{}).Where("barcode = ?", barcode).Count(&count).Error
		if err != nil {
			handleResponse(c, InternalError, err.Error())
			return
		}
		// If barcode is unique, return it
		if count == 0 {
			break
		}
	}

	handleResponse(c, OK, gin.H{"barcode": barcode})
}

// UploadProduct godoc
// @Summary Upload a product
// @Description Upload a product file in .xlsx format. The file should include product details in specific columns.
// @Tags products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response "Products uploaded successfully"
// @Failure 400 {object} v1.Response "Invalid file format or processing error"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/excel-upload [post]
func (h *ProductHandler) UploadProduct(c *gin.Context) {
	var file domain.File
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Check file extension
	ext := filepath.Ext(file.File.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		h.log.Error("Unsupported file format: ", ext)
		handleResponse(c, BadRequest, "Unsupported file format")
		return
	}

	// Save the uploaded file
	newFilename := uuid.New().String() + ext
	savePath := filepath.Join("uploads", newFilename)
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}
	//
	defer os.Remove(savePath)
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Error("Failed to open .xlsx file: ", err.Error())
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	existingProducers := make(map[string]string)

	// Load existing producers from DB
	var dbProducers []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	h.db.Table("producers").Select("id, name").Find(&dbProducers)
	for _, c := range dbProducers {
		existingProducers[c.Name] = c.ID
	}

	// Process rows
	var products []map[string]any
	var producers []map[string]any
	for _, row := range rows[1:] {
		if len(row) < 8 {
			producerID, exists := existingProducers[row[5]]
			if !exists {
				producerID = uuid.New().String()
				existingProducers[row[5]] = producerID
				producers = append(producers, map[string]any{
					"id":   producerID,
					"name": row[5],
				})
			}
			products = append(products, map[string]any{
				"material_code": parseIntComma(row[1]),
				"barcode":       h.GenBarcode(),
				"producer_id":   producerID,
				"name":          row[2],
			})
			continue
		}
		producerID, exists := existingProducers[row[6]]
		if !exists {
			producerID = uuid.New().String()
			existingProducers[row[6]] = producerID
			producers = append(producers, map[string]any{
				"id":   producerID,
				"name": row[6],
			})
		}
		products = append(products, map[string]any{
			"material_code": parseIntComma(row[1]),
			"producer_id":   producerID,
			"name":          row[2],
			"barcode":       row[3],
		})
	}
	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	err = tx.Table("producers").Create(&producers).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = tx.Table("products").Create(&products).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
}

// UploadProduct godoc
// @Summary Upload a product
// @Description Upload a product file in .xlsx format. The file should include product details in specific columns.
// @Tags products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response "Products uploaded successfully"
// @Failure 400 {object} v1.Response "Invalid file format or processing error"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/attach-barcode [post]
func (h *ProductHandler) AttachBarcode(c *gin.Context) {
	var file domain.File
	// bind file
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Check file extension
	ext := filepath.Ext(file.File.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		h.log.Error("Unsupported file format: ", ext)
		handleResponse(c, BadRequest, "Unsupported file format")
		return
	}
	// Save the uploaded file
	newFilename := uuid.New().String() + ext
	savePath := filepath.Join("uploads", newFilename)
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}
	//
	defer os.Remove(savePath)
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Error("Failed to open .xlsx file: ", err.Error())
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(1)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	// start transaction
	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 2 {
			err = tx.Exec("UPDATE products SET barcode = ? WHERE material_code = ?", row[2], row[0]).Error
			if err != nil {
				h.log.Warn("ERROR on updating product barcode: %v", err)
				handleResponse(c, InternalError, "Can't update product barcode")
				return
			}
		}
	}
	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, "Can't commit transaction")
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
}

// Get product movements godoc
// @Summary Get product movements
// @Description Get product movements
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id}/product-movement [get]
func (h *ProductHandler) ProductMovements(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// get product id from the path param
	var productId = c.Param("id")
	// validate product id
	if err := uuid.Validate(productId); err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var params domain.ProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	// validate store id
	if params.StoreId != "" {
		if err := uuid.Validate(params.StoreId); err != nil {
			handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get pagination with default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	params.ProducerId = productId
	// get product-movements data from the product service
	res, totalCount, err := h.service.GetProductMovements(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// get pagintion data with _meta object
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// ExportProductMovementsExcel godoc
// @Summary Export product movements to Excel
// @Description Export product movements to Excel
// @Tags products
// @Security BearerAuth
// @Produce  application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path string true "Product ID"
// @Param store_id query string false "Store ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id}/product-movement/export-excel [get]
func (h *ProductHandler) ExportProductMovementsExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// get product id from the path param
	var productId = c.Param("id")
	// validate product id
	if err := uuid.Validate(productId); err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var params domain.ProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	// validate store id
	if params.StoreId != "" {
		if err := uuid.Validate(params.StoreId); err != nil {
			handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get pagination with default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	params.ProducerId = productId
	// get product-movements data from the product service
	res, _, err := h.service.GetProductMovements(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create excel
	f := excelize.NewFile()
	sheetName := "Movements"
	f.SetSheetName("Sheet1", sheetName)

	// headers
	headers := []string{"ID", "Номер", "Тип", "Колличество", "Сумма", "Название", "Аптека", "Дата создания"}
	if err := setExcelHeaders(f, sheetName, headers); err != nil {
		h.log.Error("Excel style error:", err)
		handleResponse(c, InternalError, "Error on creating excel")
		return
	}

	// fill rows
	for i, item := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, item.Id)
		f.SetCellValue(sheetName, "B"+row, item.PublicId)
		f.SetCellValue(sheetName, "C"+row, entryTypeToString(item.EntryType)) // helper func
		f.SetCellValue(sheetName, "D"+row, item.Count)
		f.SetCellValue(sheetName, "E"+row, item.Sum)
		f.SetCellValue(sheetName, "F"+row, item.Name)
		f.SetCellValue(sheetName, "G"+row, item.StoreName)
		if item.CreatedAt != nil {
			f.SetCellValue(sheetName, "H"+row, item.CreatedAt.Add(time.Hour*5).Format("2006-01-02 15:04:05"))
		}
	}

	// save
	saveExcelToUploads(c, f, *h.log, "product_movements")
}

// Update ismarking godoc
// @Summary Update product ismarking
// @Description Update product ismarking
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param 	body body domain.UpdateIsMarking true "Update product is marking"
// @Success 200 {object} v1.Response "Updated is marking"
// @Failure 400 {object} v1.Response "Invalid product id or is marking"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/is-marking [patch]
func (h *ProductHandler) UpdateIsMarking(c *gin.Context) {
	var (
		body domain.UpdateIsMarking
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, "Invalid received info, Please try again")
		return
	}
	// update is_marking service
	err = h.service.UpdateProductIsMarking(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Can't update is_marking status")
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Get product arzon apteka
// @Summary Get product arzon apteka
// @Description Get product arzon apteka
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param 	store_id query string true "Store ID"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/list-arzon [GET]
func (h *ProductHandler) ArzonProductList(c *gin.Context) {
	storeId := c.Query("store_id")

	if err := uuid.Validate(storeId); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.ProductListForArzon(ctx, storeId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Get product arzon apteka
// @Summary Get product arzon apteka
// @Description Get product arzon apteka
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param 	store_id query string true "Store ID"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/export-arzon [GET]
func (h *ProductHandler) ArzonProductExport(c *gin.Context) {
	storeId := c.Query("store_id")

	if err := uuid.Validate(storeId); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.ProductListForArzon(ctx, storeId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Наименование", "Производитель", "Цена"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("could not create excel style: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Name)
		f.SetCellValue(sheetName, "B"+row, imp.ProducerName)
		f.SetCellValue(sheetName, "C"+row, imp.RetailPrice)
	}

	saveExcelToUploads(c, f, *h.log, "product_list")
}

// Get product list by import
// @Summary Get product list by import
// @Description Get product list by import
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param limit 	query int false "limit"
// @Param offset 	query int false "offset"
// @Param search 	query string false "search"
// @Param status 	query string false "status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id  query string false "store_id"
// @Param producer_id query string false "producer_id"
// @Param no_barcode query bool false "no_barcode"
// @Success 200 {object} v1.Response "products"
// @Failure 400 {object} v1.Response "invalid request query"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/list-by-import [GET]
func (h *ProductHandler) GetProductsByImport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetProductsByImport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Get product list by import
// @Summary Get product list by import
// @Description Get product list by import
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id query string false "Store ID"
// @Param category_id query string false "Category ID"
// @Param producer_id query string false "Producer ID"
// @Param no_barcode query bool false "No Barcode"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/export-by-import [GET]
func (h *ProductHandler) GetProductsByImportExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, _, err := h.service.GetProductsByImport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// Create excel file
	f := excelize.NewFile()

	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименование", "Кол-во", "Штрихкод", "Производитель", "ИКПУ", "Код.Уп", "Наз.Уп", "Маркировка", "Цена поставки", "Цена продажи"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("could not create excel style: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	// // Add product infos to excel column
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, product.Name)
		f.SetCellValue(sheetName, "C"+row, math.Round(float64(product.UQuantity)/float64(product.UnitPerPack)))
		f.SetCellValue(sheetName, "D"+row, product.Barcode)
		f.SetCellValue(sheetName, "E"+row, product.ProducerName)
		f.SetCellValue(sheetName, "F"+row, product.Mxik)
		f.SetCellValue(sheetName, "G"+row, product.UnitCode)
		f.SetCellValue(sheetName, "H"+row, product.UnitLabel)
		f.SetCellValue(sheetName, "J"+row, product.IsMarking)
		f.SetCellValue(sheetName, "K"+row, product.SupplyPrice)
		f.SetCellValue(sheetName, "L"+row, product.RetailPrice)

	}

	saveExcelToUploads(c, f, *h.log, "products_by_import")
}

// Update Mxik by product import
// @Summary Update Mxik by product import
// @Description Update Mxik by product import
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "Product ID"
// @Param 	body body domain.UpdateBarcodeRequest true "Update barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/update-mxik-import/{id} [put]
func (h *ProductHandler) UpdateProductIkpuForStoreProduct(c *gin.Context) {
	var (
		body   domain.UpdateBarcodeRequest
		id     = c.Param("id")
		userID string
	)
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	userID = userId.(string)

	// validate id
	err := uuid.Validate(id)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.store_product_id")
		return
	}
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if body.Mxik != "" {
		// update mxik
		err = h.db.Exec("UPDATE store_products SET mxik = ? WHERE id = ?", body.Mxik, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if body.UnitCode != "" {
		err = h.db.Exec("UPDATE store_products SET unit_code = ? WHERE id = ?", body.UnitCode, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if body.UnitLabel != "" {
		err = h.db.Exec("UPDATE store_products SET unit_label = ? WHERE id = ?", body.UnitLabel, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	if body.Barcode != "" {
		err = h.db.Exec("UPDATE store_products SET barcode = ? WHERE id = ?", body.Barcode, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		// insert into product_barcodes
		err = h.db.Exec(`
		INSERT INTO product_barcodes (product_id, barcode, old_barcode, store_id, status, created_by)
		SELECT p.id, ?, sp.barcode, sp.store_id, 'completed', ? 
		FROM store_products sp
		JOIN products p ON p.id = sp.product_id
		WHERE sp.id = ?`,
			body.Barcode, userID, id,
		).Error

		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	if body.ExpireDate != "" {
		err = h.db.Exec("UPDATE store_products SET expire_date = ? WHERE id = ?", body.ExpireDate, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, OK, "UPDATED")
}

// Update store_product is_marking
// @Summary Update store_product is_marking
// @Description Update store_product is_marking
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param 	body body domain.UpdateIsMarking true "Update product is marking"
// @Success 200 {object} v1.Response "Updated is marking"
// @Failure 400 {object} v1.Response "Invalid product id or is marking"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/store-is-marking [patch]
func (h *ProductHandler) UpdateStoreProductIsMarking(c *gin.Context) {
	var (
		body domain.UpdateIsMarking
		err  error
	)
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid received info, Please try again")
		return
	}

	// update is_marking service
	if body.IsMarking != nil {
		err = h.db.Exec(`UPDATE store_products SET is_marking = ? WHERE id = ?`, body.IsMarking, body.ID).Error
		if err != nil {
			h.log.Warn("ERROR on updating store_product is_marking: %v", err)
			handleResponse(c, InternalError, "failed.update.store_product.is_marking")
			return
		}
	}

	if body.IsChecking != nil {
		err = h.db.Exec(`UPDATE store_products SET is_checking = ? WHERE id = ?`, body.IsChecking, body.ID).Error
		if err != nil {
			h.log.Warn("ERROR on updating store_product is_checking: %v", err)
			handleResponse(c, InternalError, "failed.update.store_product.is_checking")
			return
		}
	}

	handleResponse(c, OK, "UPDATED")
}

// Create min max product
// @Summary Create min max product
// @Description Create min max product
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	body body domain.MinMaxProductRequest true "Min Max request body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/min-max [POST]
func (h *ProductHandler) CreateMinMaxProduct(c *gin.Context) {
	var body domain.MinMaxProductRequest
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// create store_product_thresholds
	err = h.db.Table("store_product_thresholds").Create(&body).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			handleResponse(c, BadRequest, "min.max.product.already.exists.with.store")
			return
		}
		// log error
		h.log.Warn("ERROR on creating store_product_thresholds: %v", err)
		handleResponse(c, InternalError, "failed.to.create.store_product_thresholds")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Update min max product
// @Summary Update min max product
// @Description Update min max product
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Min Max Product ID"
// @Param 	body body domain.MinMaxProductUpdate true "Min Max update"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/min-max/{id} [PUT]
func (h *ProductHandler) UpdateMinMaxProduct(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.MinMaxProductUpdate
	)

	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// update store_product_thresholds
	err = h.db.Exec(`
	UPDATE 
		store_product_thresholds 
	SET
		product_id = ?,
		kvant = ?,
		min_quantity = ?,
		max_quantity = ?
	WHERE id = ?`,
		body.ProductID,
		body.Kvant,
		body.MinQuantity,
		body.MaxQuantity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			handleResponse(c, BadRequest, "min.max.product.already.exists.with.store")
			return
		}
		handleResponse(c, InternalError, "failed.to.update.min.max.product")
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// Get min max product
// @Summary Get min max product
// @Description Get min max product
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "id"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/min-max/{id} [GET]
func (h *ProductHandler) GetMinMaxProductById(c *gin.Context) {
	var (
		id  = c.Param("id")
		res domain.StoreProductThreshold
	)

	// update store_product_thresholds
	err := h.db.
		Model(&domain.StoreProductThreshold{}).
		Preload("Store").
		Preload("Product").
		First(&res, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "min.max.product.not.found")
			return
		}
		handleResponse(c, InternalError, "failed.to.get.min.max.product")
		return
	}

	handleResponse(c, OK, res)
}

// Get min, max products
// @Summary min, max products
// @Description min, max products
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/list-min-max [GET]
func (h *ProductHandler) GetMinMaxProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, totalCount, err := h.service.GetMinMaxProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Get min, max products
// @Summary Get min, max products
// @Description Get min, max products
// @Tags products
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive)"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/export-min-max [GET]
func (h *ProductHandler) ExportMinMaxProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, _, err := h.service.GetMinMaxProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()

	sheetName := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Филиал", "Код", "Наименование", "Квант", "Мин.зап", "Макс.зап", "Актив"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("could not excel create style: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// // Add product infos to excel column
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.StoreName)
		f.SetCellValue(sheetName, "B"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "C"+row, product.Name)
		f.SetCellValue(sheetName, "D"+row, product.Kvant)
		f.SetCellValue(sheetName, "E"+row, product.MinQuantity)
		f.SetCellValue(sheetName, "F"+row, product.MaxQuantity)
		f.SetCellValue(sheetName, "G"+row, product.IsActive)
	}
	saveExcelToUploads(c, f, *h.log, "min_max_products")
}

// CreateExcludedProduct godoc
// @Summary Exclude a product from a store or globally
// @Description Exclude a specific product from a store or globally if store_id is omitted
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param exclude body domain.ProductExcludeRequest true "Exclude Product Request"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/exclude [post]
func (h *ProductHandler) CreateExcludedProduct(c *gin.Context) {
	var body domain.ProductExcludeRequest
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	companyId, ok := c.Get("company_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	// Validation: at least one of producer_id or product_id is required
	if body.ProducerID == nil && len(body.ProductID) == 0 {
		handleResponse(c, BadRequest, "Either producer_id or product_id(s) must be provided")
		return
	}
	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	var productIDs []string

	// Case 1 or 2: producer_id bor
	if body.ProducerID != nil && *body.ProducerID != "" {
		if err := tx.Raw(`SELECT id FROM products WHERE producer_id = ?`, *body.ProducerID).Scan(&productIDs).Error; err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to fetch products for producer")
			return
		}

		if len(productIDs) == 0 {
			handleResponse(c, BadRequest, "No products found for given producer")
			return
		}
	} else {
		// Case 3: product_id list berilgan
		productIDs = body.ProductID
	}

	// Insert logic
	now := time.Now()
	for _, productID := range productIDs {
		if len(body.StoreID) == 0 {
			// Global exclude (store_id = NULL)
			excludeID := uuid.New().String()
			err = tx.Exec(`
				INSERT INTO excluded_products (id, store_id, product_id, company_id, created_by, created_at)
				VALUES (?, NULL, ?, ?, ?, ?)
				ON CONFLICT (store_id, product_id, company_id) DO NOTHING
			`, excludeID, productID, companyId, userId, now).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, "Failed to exclude product globally")
				return
			}
		} else {
			// Store-specific exclude
			for _, store := range body.StoreID {
				if store == nil || *store == "" {
					continue
				}
				excludeID := uuid.New().String()
				err = tx.Exec(`
					INSERT INTO excluded_products (id, store_id, product_id, company_id, created_by, created_at)
					VALUES (?, ?, ?, ?, ?, ?)
					ON CONFLICT (store_id, product_id, company_id) DO NOTHING
				`, excludeID, *store, productID, companyId, userId, now).Error
				if err != nil {
					h.log.Error(err)
					handleResponse(c, InternalError, "Failed to exclude product for store")
					return
				}
			}
		}
	}

	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, "Product(s) successfully excluded")
}

// ListExcludedProducts godoc
// @Summary List excluded products
// @Description Get a paginated list of excluded products by store
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/excluded-list [get]
func (h *ProductHandler) ListExcludedProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// defaults
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// service call
	data, total, err := h.service.ListExcludedProducts(&params)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(data, total, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// ExportExcludedProductsExcel godoc
// @Summary Export excluded products to Excel
// @Description Export excluded products list to Excel by store or globally
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce octet-stream
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {file} file
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/excluded-export [get]
func (h *ProductHandler) ExportExcludedProductsExcel(c *gin.Context) {
	var param domain.ProductQueryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters: "+err.Error())
		return
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get data
	data, _, err := h.service.ListExcludedProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get excluded products: "+err.Error())
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	sheet := "ExcludedProducts"
	index, err := f.NewSheet(sheet)
	if err != nil {
		handleResponse(c, InternalError, "Failed to create sheet: "+err.Error())
		return
	}
	f.SetActiveSheet(index)

	// Set header
	headers := []string{"ID", "Наименование", "Магазин", "Создал", "Дата создания"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheet, cell, h)
	}

	// Fill rows
	for i, item := range data {
		row := i + 2 // start from row 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.ProductName)
		if *item.StoreName == "Global" {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "Все магазины")
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), *item.StoreName)
		}
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.CreatedBy)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.CreatedAt.Format("2006-01-02 15:04"))
	}

	// Save to uploads directory
	saveExcelToUploads(c, f, *h.log, "Qora-ro'yhat-maxsulotlar")
}

// DeleteExcludedProduct godoc
// @Summary Delete an excluded product
// @Description Delete a product from the excluded_products list by its ID
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Excluded Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 404 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/exclude/{id} [delete]
func (h *ProductHandler) DeleteExcludedProduct(c *gin.Context) {
	var (
		id  = c.Param("id")
		err error
	)
	if id == "" {
		handleResponse(c, BadRequest, "ID is required")
		return
	}

	tx := h.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	query := `DELETE FROM excluded_products WHERE id = ?`

	result := tx.Exec(query, id)
	if result.Error != nil {
		h.log.Error("Failed to delete excluded product: ", result.Error)
		handleResponse(c, InternalError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		handleResponse(c, NotFound, "Excluded product not found")
		return
	}

	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Excluded product deleted successfully")
}

// UpdatePackaging godoc
// @Summary Update product packaging
// @Description Update product unit_per_pack and recalculate store_products.unit_quantity
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param data body domain.UpdatePackagingRequest true "Update Packaging"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 404 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/update-packaging [put]
func (h *ProductHandler) UpdatePackaging(c *gin.Context) {
	var req domain.UpdatePackagingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, "Invalid request: "+err.Error())
		return
	}

	// service call
	if err := h.service.UpdatePackaging(&req); err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Packaging updated successfully")
}

// ListStoreProducts godoc
// @Summary Get list of store products
// @Description Get store products with product info and optional search
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search by product name"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/list-store-products [post]
func (h *ProductHandler) ListStoreProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var (
		res []struct {
			domain.StoreProduct
			ProductName string `json:"product_name"`
			UnitPerPack int    `json:"unit_per_pack"`
		}
		totalCount int64
		companyID  string
		storeID    string
		search     = c.Query("search")
	)

	// get store_id from query
	storeID = c.Query("store_id")

	// apply employee restrictions if not admin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			storeID = user.StoreId
		}
		companyID = user.CompanyId
	}

	// pagination
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// build query
	query := h.db.
		Table("store_products sp").
		Select(`sp.*, p.name as product_name, p.unit_per_pack`).
		Joins("JOIN products p ON p.id = sp.product_id").
		Where("sp.pack_quantity > 0 OR sp.unit_quantity > 0")

	if storeID != "" {
		query = query.Where("sp.store_id = ?", storeID)
	}
	if companyID != "" {
		query = query.Joins("JOIN stores st ON st.id = sp.store_id").
			Where("st.company_id = ?", companyID)
	}
	if search != "" {
		query = query.Where("p.name ILIKE ?", "%"+search+"%")
	}

	// execute query
	err = query.Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("sp.created_at desc").
		Scan(&res).Error
	if err != nil {
		h.log.Errorf("could not get store_products list: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// CreateProductPhotoAlert godoc
// @Summary Create a product photo alert
// @Description Create a new product photo alert (ml/doza noto'g'ri, ishlab chiqaruvchi noto'g'ri, rasm xato)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.ProductPhotoAlertCreate true "Product Photo Alert Create Body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/photo-alert [post]
func (h *ProductHandler) CreateProductPhotoAlert(c *gin.Context) {
	var req domain.ProductPhotoAlertCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// user_id from context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	req.CreatedBy = userId.(string)

	err := h.service.CreateProductPhotoAlert(&req)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Product photo alert created successfully")
}

// ListProductPhotoAlert godoc
// @Summary List product photo alerts
// @Description Get a paginated list of product photo alerts
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param status query string false "Status (pending || completed)"
// @Param category query int false "Category (1=ml/doza noto'g'ri, 2=ishlab chiqaruvchi noto'g'ri, 3=butunlay rasm xato)"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/photo-alert/list [post]
func (h *ProductHandler) ListProductPhotoAlert(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	// pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// service call
	alerts, totalCount, err := h.service.ListProductPhotoAlert(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(alerts, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// DeleteProductPhotoAlert godoc
// @Summary Delete product photo alert
// @Description Delete a product photo alert by ID
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Photo Alert ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 404 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/photo-alert/{id} [delete]
func (h *ProductHandler) DeleteProductPhotoAlert(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, BadRequest, "id is required")
		return
	}

	err := h.service.DeleteProductPhotoAlert(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "photo alert not found")
			return
		}
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Product photo alert deleted successfully")
}

const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func RandomString(length int) string {
	bytes := make([]byte, length)
	for i := range bytes {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		bytes[i] = charset[n.Int64()]
	}
	return string(bytes)
}

// barcode generator
// generateRandomBarcode creates a random 13-digit numeric barcode
func generateRandomBarcode(length int) string {
	// Create a new random source and generator
	source := gen.NewSource(time.Now().UnixNano())
	random := gen.New(source)

	digits := "0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = digits[random.Intn(len(digits))]
	}
	return string(result)
}

// just product list export function
func (h *ProductHandler) productListExport(f *excelize.File, res []domain.ProductData) (*excelize.File, error) {
	sheetName := "List1"

	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Аптека", "Код", "Наименования", "Штрих-код", "Кол-во", "Срок годности", "Цена прихода", "Cумма прихода", "Цена продажа", "Сумма продажа"}

	err := setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		return nil, errors.New("failed to create style")
	}

	// Ma'lumotlarni qo'shish
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.StoreName)
		f.SetCellValue(sheetName, "B"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "C"+row, product.Name)
		f.SetCellValue(sheetName, "D"+row, product.Barcode)
		f.SetCellValue(sheetName, "E"+row, math.Round((float64(product.UnitQuantity)/float64(product.UnitPerPack))*100)/100)
		if product.ExpireDate != nil {
			f.SetCellValue(sheetName, "F"+row, product.ExpireDate.Format(time.DateOnly))
		} else {
			f.SetCellValue(sheetName, "F"+row, "N/A")
		}
		f.SetCellValue(sheetName, "G"+row, product.SupplyPrice)
		f.SetCellValue(sheetName, "H"+row, math.Round(((product.SupplyPrice/float64(product.UnitPerPack))*float64(product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "I"+row, product.RetailPrice)
		f.SetCellValue(sheetName, "J"+row, math.Round(((product.RetailPrice/float64(product.UnitPerPack))*float64(product.UnitQuantity))*100)/100)
	}
	return f, nil
}

// product list export by store_id function
func (h *ProductHandler) productListExportByStoreId(f *excelize.File, res []domain.ProductData) (*excelize.File, error) {
	sheetName := "Products"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименование", "Штрих-код", "Производитель", "Кол-во", "Цена поставки", "Cумма поставки", "Цена продажи", "Cумма продажи", "Цена наценка", "Cумма наценка", "НДС", "Цена НДС", "Срок годности"}

	err := setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Warn("Failed to create style: %v", err)
		return nil, errors.New("failed to create style")
	}

	// Add product infos to excel column
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, product.Name)
		f.SetCellValue(sheetName, "C"+row, product.Barcode)
		f.SetCellValue(sheetName, "D"+row, product.Manufacturer)
		f.SetCellValue(sheetName, "E"+row, math.Round((float64(product.UnitQuantity)/float64(product.UnitPerPack))*100)/100)
		f.SetCellValue(sheetName, "F"+row, product.SupplyPrice)
		f.SetCellValue(sheetName, "G"+row, math.Round(((product.SupplyPrice/float64(product.UnitPerPack))*float64(product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "H"+row, product.RetailPrice)
		f.SetCellValue(sheetName, "I"+row, math.Round(((product.RetailPrice/float64(product.UnitPerPack))*float64(product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "J"+row, product.RetailPrice-product.SupplyPrice)
		f.SetCellValue(sheetName, "K"+row, math.Round((((product.RetailPrice-product.SupplyPrice)/float64(product.UnitPerPack))*float64(product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "L"+row, 12)
		f.SetCellValue(sheetName, "M"+row, math.Round((product.RetailPrice*12)/112))
		if product.ExpireDate != nil {
			f.SetCellValue(sheetName, "N"+row, product.ExpireDate.Format("2006-01-02"))
		} else {
			f.SetCellValue(sheetName, "N"+row, "N/A")
		}
	}

	return f, nil
}
