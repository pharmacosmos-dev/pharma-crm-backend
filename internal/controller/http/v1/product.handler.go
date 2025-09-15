package v1

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	gen "math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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
		product.GET("/store/:id", h.ListByStoreId)
		product.GET("/import/:id", h.GetProductImports)
		product.DELETE("/hard-delete", h.HardDelete)
		product.DELETE("/soft-delete", h.SoftDelete)
		product.GET("/store-product/:id", h.ListStoreProductProductId)
		product.POST("/store/barcode", h.AddStoreProductByBarcode)
		product.POST("/generate-barcode", h.GenerateBarcode)
		product.GET("/total-status-count", h.TotalStatusCount)
		product.PUT("/update-barcode/:id", h.UpdateBarcode)
		product.POST("/attach-barcode", h.AttachBarcode)
		product.POST("/generate-marking", h.GenerateMarkingProducts)
		product.PATCH("/is-marking", h.UpdateIsMarking)
		product.GET("/:id/product-movement", h.ProductMovements)
		product.GET("/export-arzon", h.ArzonProductExport)
		product.GET("/list-arzon", h.ArzonProductList)
		product.GET("/list-by-import", h.ProductListByImport)
		product.GET("/export-by-import", h.ExportProductListByImport)
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
	var (
		body domain.ProductRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// begin transaction
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
	// generate id
	body.Id = uuid.New().String()
	body.Photos = utils.StringArray(body.Photos)
	body.Status = config.ACTIVE_PRODUCT
	body.MaterialCode = utils.GenerateMaterialCode()
	err = tx.
		WithContext(c.Request.Context()).
		Table("products").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// store products
	// if len(body.StoreProduct) > 0 {
	// 	var imports = make([]domain.ImportRequest, len(body.StoreProduct))
	// 	var importDetail = make([]domain.ImportDetailRequest, len(body.StoreProduct))
	// 	for i := range body.StoreProduct {
	// 		// store products table take required fields
	// 		body.StoreProduct[i].ProductID = body.Id
	// 		body.StoreProduct[i].UnitQuantity = body.StoreProduct[i].PackQuantity * body.UnitPerPack

	// 		// imports table take required fields
	// 		imports[i].Id = uuid.New().String()
	// 		imports[i].StoreID = body.StoreProduct[i].StoreID
	// 		imports[i].Status = config.COMPLETED_IMPORT
	// 		imports[i].DocumentNumber = utils.GenerateDocumentNumber()
	// 		imports[i].ImportDate = time.Now().Format("2006-01-02 15:04:05")

	// 		// import detail take required fields
	// 		importDetail[i].ImportID = imports[i].Id
	// 		importDetail[i].ProductID = &body.Id
	// 		importDetail[i].ReceivedCount = body.StoreProduct[i].PackQuantity
	// 		importDetail[i].AcceptedCount = body.StoreProduct[i].PackQuantity
	// 		importDetail[i].SupplyPrice = body.StoreProduct[i].SupplyPrice
	// 		importDetail[i].RetailPrice = body.StoreProduct[i].RetailPrice
	// 		importDetail[i].Vat = body.StoreProduct[i].Vat
	// 		importDetail[i].VatSum = body.StoreProduct[i].RetailPrice - body.StoreProduct[i].SupplyPrice
	// 		importDetail[i].ExpireDate = time.Now().Format("2006-01-02 15:04:05")
	// 	}
	// 	// store products
	// 	err = tx.
	// 		WithContext(c.Request.Context()).
	// 		Table("store_products").
	// 		Create(&body.StoreProduct).Error
	// 	if err != nil {
	// 		tx.Rollback()
	// 		h.log.Error(err)
	// 		handleResponse(c, InternalError, err.Error())
	// 		return
	// 	}
	// 	// create new import
	// 	err = tx.
	// 		WithContext(c.Request.Context()).
	// 		Table("imports").
	// 		Create(&imports).Error
	// 	if err != nil {
	// 		tx.Rollback()
	// 		h.log.Error(err)
	// 		handleResponse(c, InternalError, err.Error())
	// 		return
	// 	}
	// 	// create new import detail
	// 	err = tx.
	// 		WithContext(c.Request.Context()).
	// 		Table("import_details").
	// 		Create(&importDetail).Error
	// 	if err != nil {
	// 		tx.Rollback()
	// 		h.log.Error(err)
	// 		handleResponse(c, InternalError, err.Error())
	// 		return
	// 	}
	// }
	// check category length
	if len(body.CategoryIds) > 0 {
		var categoryProduct = make([]domain.CategoryProduct, len(body.CategoryIds))
		for i := range body.CategoryIds {
			categoryProduct[i].ProductId = body.Id
			categoryProduct[i].CategoryId = body.CategoryIds[i]
			categoryProduct[i].IsOpen = true
		}
		// create category products
		err = tx.
			WithContext(c.Request.Context()).
			Create(&categoryProduct).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, "CREATED")
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
		res     domain.Product
		id      = c.Param("id")
		storeId = c.Query("store_id")
	)

	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid product ID")
		return
	}

	// dynamic JOIN with subquery to get latest store_product
	rawJoin := `
		LEFT JOIN store_products sp ON sp.id = (
			SELECT id FROM store_products
			WHERE product_id = products.id
	`
	if storeId != "" {
		rawJoin += " AND store_id = ?"
	}
	rawJoin += `
			ORDER BY created_at DESC
			LIMIT 1
		)
	`

	pquery := h.db.
		Preload("UnitType").
		Preload("Shelf").
		Preload("Producer").
		Select(`
			products.*,
			ROUND(sp.retail_price / products.unit_per_pack, 2) AS retail_unit_price,
			ROUND((sp.retail_price),2) AS retail_price
		`).
		Joins(rawJoin, storeId)

	err := pquery.
		First(&res, "products.id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Product not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Get category tree
	category := []*domain.Category{}
	query := `
		WITH RECURSIVE category_tree AS (
			SELECT
				c.id,
				c.category_id,
				c.name::TEXT AS name_path,
				c.id AS root_category_id
			FROM categories c
			INNER JOIN category_products cp ON c.id = cp.category_id
			WHERE cp.product_id = ?

			UNION ALL

			SELECT
				parent.id,
				parent.category_id,
				(parent.name || ' > ' || child.name_path)::TEXT AS name_path,
				child.root_category_id
			FROM categories parent
			INNER JOIN category_tree child ON parent.id = child.category_id
		)
		SELECT DISTINCT ON (root_category_id) id, category_id, name_path AS name
		FROM category_tree
		ORDER BY root_category_id, LENGTH(name_path) DESC;
	`
	if err := h.db.Raw(query, id).Scan(&category).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	res.Categories = category

	// Get markings
	if err := h.db.Raw(`SELECT marking FROM product_markings WHERE product_id = ?`, id).Scan(&res.Markings).Error; err != nil {
		h.log.Warn("ERROR on getting product markings: %v", err)
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
	var (
		param domain.ProductQueryParam
		err   error
	)
	// bind
	err = c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	// Pagination parameters
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get products list
	products, totalCount, err := h.service.ListProduct(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare the response
	result := utils.ListResponse(products, totalCount, param.Limit, param.Offset)

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
	var (
		param domain.ProductQueryParam
		res   []domain.ProductData
	)
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Pagination parameters
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	// get products list
	res, err = h.service.ListProductExport(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Create excel file
	f := excelize.NewFile()

	if param.StoreID != "" {
		f, err = h.productListExportByStoreId(f, res)

		if err != nil {
			handleResponse(c, InternalError, "Failed to export product list")
			return
		}
	} else {
		f, err = h.productListExport(f, res)
		if err != nil {
			handleResponse(c, InternalError, "Failed to export product list")
			return
		}
	}
	saveExcelToUploads(c, f, *h.log, "Barcha_mahslulotlar")
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
	var (
		param domain.ProductQueryParam
	)
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	res, err := h.service.ListProductStats(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get product stats")
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
	// var storeProducts []map[string]any
	// if len(body.StoreProduct) > 0 {
	// 	for i := range body.StoreProduct {
	// 		// if body.StoreProduct[i].MeasurementValue != 0 {
	// 		status := config.COMPLETED_IMPORT
	// 		// operation := "+"
	// 		// if body.StoreProduct[i].MeasurementValue < 0 {
	// 		// 	status = config.WRITEOFF_IMPORT
	// 		// 	operation = "-"
	// 		// 	body.StoreProduct[i].MeasurementValue *= -1
	// 		// }
	// 		importReq := domain.ImportRequest{
	// 			Id:             uuid.New().String(),
	// 			StoreID:        body.StoreProduct[i].StoreID,
	// 			Status:         status,
	// 			ImportDate:     time.Now().Format(config.DATE_FORMAT),
	// 			DocumentNumber: utils.GenerateDocumentNumber(),
	// 		}
	// 		// create new import
	// 		err = tx.Table("imports").Create(&importReq).Error
	// 		if err != nil {
	// 			h.log.Error(err)
	// 			handleResponse(c, InternalError, err.Error())
	// 			tx.Rollback()
	// 			return
	// 		}
	// 		// create new import details
	// 		err = tx.Table("import_details").Create(&domain.ImportDetailRequest{
	// 			ImportID:      importReq.Id,
	// 			ProductID:     &productID,
	// 			ReceivedCount: body.StoreProduct[i].PackQuantity,
	// 			AcceptedCount: body.StoreProduct[i].PackQuantity,
	// 			RetailPrice:   body.StoreProduct[i].RetailPrice,
	// 			SupplyPrice:   body.StoreProduct[i].SupplyPrice,
	// 			Vat:           body.StoreProduct[i].Vat,
	// 			VatSum:        body.StoreProduct[i].RetailPrice - body.StoreProduct[i].SupplyPrice,
	// 			ExpireDate:    body.StoreProduct[i].ExpireDate.Format(config.DATE_FORMAT),
	// 		}).Error
	// 		if err != nil {
	// 			h.log.Error(err)
	// 			handleResponse(c, InternalError, err.Error())
	// 			tx.Rollback()
	// 			return
	// 		}
	// 		storeProducts = append(storeProducts, map[string]interface{}{
	// 			"store_id":       body.StoreProduct[i].StoreID,
	// 			"product_id":     productID,
	// 			"retail_price":   body.StoreProduct[i].RetailPrice,
	// 			"supply_price":   body.StoreProduct[i].SupplyPrice,
	// 			"vat":            body.StoreProduct[i].Vat,
	// 			"markup":         body.StoreProduct[i].Markup,
	// 			"pack_quantity":  body.StoreProduct[i].PackQuantity,
	// 			"unit_quantity":  body.StoreProduct[i].PackQuantity * body.UnitPerPack,
	// 			"small_quantity": body.StoreProduct[i].SmallQuantity,
	// 			"bonus_percent":  body.StoreProduct[i].BonusPercent,
	// 			"expire_date":    body.StoreProduct[i].ExpireDate,
	// 		})
	// 		// err = tx.Table("store_products").
	// 		// 	Where("product_id = ? AND store_id = ? ", productID, body.StoreProduct[i].StoreID).
	// 		// 	Updates(map[string]interface{}{
	// 		// 		"pack_quantity":  gorm.Expr("pack_quantity "+operation+" ?", body.StoreProduct[i].MeasurementValue),
	// 		// 		"unit_quantity":  gorm.Expr("(pack_quantity "+operation+" ?)*?", body.StoreProduct[i].MeasurementValue, body.UnitPerPack),
	// 		// 		"small_quantity": body.StoreProduct[i].SmallQuantity,
	// 		// 		"retail_price":   body.RetailPrice,
	// 		// 		"supply_price":   body.SupplyPrice,
	// 		// 		"vat":            body.Vat,
	// 		// 		"markup":         body.Markup,
	// 		// 	}).Error
	// 		// if err != nil {
	// 		// 	tx.Rollback()
	// 		// 	h.log.Error(err)
	// 		// 	handleResponse(c, InternalError, err.Error())
	// 		// 	return
	// 		// }
	// 		// }
	// 	}
	// 	err = tx.Table("store_products").Create(&storeProducts).Error
	// 	if err != nil {
	// 		h.log.Error(err)
	// 		handleResponse(c, InternalError, err.Error())
	// 		tx.Rollback()
	// 		return
	// 	}
	// }
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
		Find(&res).
		Error
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
func (h *ProductHandler) ListByStoreId(c *gin.Context) {
	var (
		res     []*domain.StoreProductResponse
		param   domain.StoreProductQueryParam
		storeId = c.Param("id")
		err     error
	)
	// bind query params
	if err = c.ShouldBindQuery(&param); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate store_id
	if err = uuid.Validate(storeId); err != nil {
		handleResponse(c, BadRequest, "Invalid store_id")
		return
	}
	param.StoreID = storeId
	// get limit offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get store products list
	res, err = h.service.ProductSearch(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to fetch products")
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
			newQuantity, cartItem.TotalPrice, cartItem.ID).Error
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
			storeProduct.RetailPrice, storeProduct.RetailPrice, config.PENDING,
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
	var (
		storeID    = c.Query("store_id")
		productID  = c.Param("id")
		res        []domain.ImportDetail
		totalCount int64
		employee   domain.Employee
	)
	// get user_id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, NotFound, "User ID not found")
		return
	}

	// get employee info
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// check user role
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			storeID = employee.StoreId
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
		Where("imd.product_id = ?", productID)
	if storeID != "" {
		query = query.
			Joins("INNER JOIN imports ON imports.id = imd.import_id").
			Where("imports.store_id = ?", storeID)
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
// @Param id path string true "Product ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store id"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store-product/{id} [get]
func (h *ProductHandler) ListStoreProductProductId(c *gin.Context) {
	var (
		id         = c.Param("id")
		res        []domain.StoreProduct
		totalCount int64
		companyID  string
		employee   domain.Employee
		storeID    string
	)
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid product id")
		return
	}
	storeID = c.Query("store_id")
	// get user_id from header context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, InternalError, "User ID not found in context")
		return
	}

	// get employee info
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check user role
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			storeID = employee.StoreId
		}
		companyID = employee.CompanyId
	}

	// get limit, offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// build query
	query := h.db.
		Model(&domain.StoreProduct{}).
		Preload("Store").
		Select("store_products.*, u.short_name,"+
			"CASE "+
			"WHEN store_products.supply_price = 0 THEN 100 "+
			"ELSE ROUND((store_products.retail_price / store_products.supply_price - 1) * 100, 3)"+
			"END AS markup ").
		Joins("JOIN products p ON p.id = store_products.product_id").
		Joins("LEFT JOIN unit_types u ON u.id = p.unit_type_id").
		Where("store_products.product_id = ?", id).
		Where("store_products.pack_quantity > 0 OR store_products.unit_quantity > 0")

	if storeID != "" {
		query = query.Where("store_products.store_id = ?", storeID)
	}
	if companyID != "" {
		query = query.Where("st.company_id = ?", companyID).Joins("LEFT JOIN stores st ON store_products.store_id = st.id ")
	}
	// complete query
	err = query.
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("store_products.created_at desc").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	for i := range res {
		res[i].Quantity = res[i].PackQuantity
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
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
func (h *ProductHandler) UpdateBarcode(c *gin.Context) {
	var (
		body domain.UpdateBarcodeRequest
		id   = c.Param("id")
	)
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid product id")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if body.Barcode != "" {
		// update barcode
		err = h.db.Model(&domain.Product{}).Where("id = ?", id).Update("barcode", body.Barcode).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		// update barcode store_p
		err = h.db.Model(&domain.StoreProduct{}).Where("product_id = ?", id).Update("barcode", body.Barcode).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	} else if body.Mxik != "" {
		// update mxik
		err = h.db.Model(&domain.Product{}).Where("id = ?", id).Update("mxik", body.Mxik).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		// store mxik
		err = h.db.Model(&domain.StoreProduct{}).Where("product_id = ?", id).Update("mxik", body.Mxik).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	} else if body.UnitCode != "" {
		err = h.db.Model(&domain.Product{}).Where("id = ?", id).Update("unit_code", body.UnitCode).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = h.db.Model(&domain.StoreProduct{}).Where("product_id = ?", id).Update("unit_code", body.UnitCode).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	} else if body.UnitLabel != "" {
		err = h.db.Model(&domain.Product{}).Where("id = ?", id).Update("unit_label", body.UnitLabel).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = h.db.Model(&domain.StoreProduct{}).Where("product_id = ?", id).Update("unit_label", body.UnitLabel).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	handleResponse(c, OK, "Barcode updated successfully")
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
	// get product id from the path param
	productId := c.Param("id")
	storeId := c.Query("store_id")
	var companyId string
	// validate product id
	if err := uuid.Validate(productId); err != nil {
		handleResponse(c, BadRequest, "Invalid product id")
		return
	}
	// validate store id
	if storeId != "" {
		if err := uuid.Validate(storeId); err != nil {
			handleResponse(c, BadRequest, "Invalid store id")
			return
		}
	}

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}
	// get pagination with default
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, "Received invalid pagination")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error("ERROR on getting employee info: ", err)
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			storeId = employee.StoreId
		}
		companyId = employee.CompanyId
	}
	// get product-movements data from the product service
	res, totalCount, err := h.service.GetProductMovements(productId, storeId, limit, offset, companyId)
	if err != nil {
		h.log.Info("Failed to get product-movement: %v", err)
		handleResponse(c, InternalError, "Can't get product-movements")
		return
	}
	// get pagintion data with _meta object
	data := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, data)
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
		handleResponse(c, BadRequest, "invalid.store_id")
		return
	}

	res, err := h.service.ProductListForArzon(storeId)
	if err != nil {
		handleResponse(c, InternalError, "failed.get.product_list")
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
		handleResponse(c, BadRequest, "invalid.store_id")
		return
	}

	res, err := h.service.ProductListForArzon(storeId)
	if err != nil {
		handleResponse(c, InternalError, "failed.get.product_list")
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
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
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
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (active || inactive || low-stock || zero-stock || expired || imminent)"
// @Param store_id query string true "Store ID"
// @Param category_id query string false "Category ID"
// @Param producer_id query string false "Producer ID"
// @Param no_barcode query bool false "No Barcode"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/list-by-import [GET]
func (h *ProductHandler) ProductListByImport(c *gin.Context) {
	var (
		param domain.ProductQueryParam
	)
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.GetProductListByImport(&param)
	if err != nil {
		handleResponse(c, InternalError, "failed.to.get.product_list")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)
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
func (h *ProductHandler) ExportProductListByImport(c *gin.Context) {
	var (
		param domain.ProductQueryParam
	)
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}
	res, _, err := h.service.GetProductListByImport(&param)
	if err != nil {
		handleResponse(c, InternalError, "failed.to.get.product_list")
		return
	}
	// Create excel file
	f := excelize.NewFile()

	sheetName := "Products"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименование", "Штрих-код", "Номер Импорт", "Производитель", "Кол-во", "IKPU", "Код.Уп", "Наз.Уп", "Маркировка", "Цена поставки", "Цена продажи"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Warn("Failed to create style: %v", err)
		handleResponse(c, InternalError, "failed.to.create.newstyle")
		return
	}

	// // Add product infos to excel column
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, product.Name)
		f.SetCellValue(sheetName, "C"+row, product.Barcode)
		f.SetCellValue(sheetName, "D"+row, product.ImportNumber)
		f.SetCellValue(sheetName, "E"+row, product.ProducerName)
		f.SetCellValue(sheetName, "F"+row, math.Round(product.Quantity+(product.UnitQuantity/float64(product.UnitPerPack))))
		f.SetCellValue(sheetName, "G"+row, product.Mxik)
		f.SetCellValue(sheetName, "H"+row, product.UnitCode)
		f.SetCellValue(sheetName, "I"+row, product.UnitLabel)
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
		body domain.UpdateBarcodeRequest
		id   = c.Param("id")
	)
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
	var param domain.ProductQueryParam

	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.GetMinMaxProducts(&param)
	if err != nil {
		h.log.Warn("ERROR on getting min_max product: %v", err)
		handleResponse(c, InternalError, "failed.to.get.min.max.products")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

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
// @Param status query string false "Status (active || inactive"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response "Product list"
// @Failure 400 {object} v1.Response "Invalid store_id"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /product/export-min-max [GET]
func (h *ProductHandler) ExportMinMaxProducts(c *gin.Context) {
	var param domain.ProductQueryParam

	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.GetMinMaxProducts(&param)
	if err != nil {
		h.log.Warn("ERROR on getting min_max product: %v", err)
		handleResponse(c, InternalError, "failed.to.get.min.max.products")
		return
	}

	// Create excel file
	f := excelize.NewFile()

	sheetName := "Products"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Филиал", "Код", "Наименование", "Квант", "Мин.зап", "Макс.зап", "Актив"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Warn("Failed to create style: %v", err)
		handleResponse(c, InternalError, "failed.to.create.newstyle")
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

// Helper function to safely parse float values
func parseFloat(value string) float64 {
	value = strings.TrimSpace(value)
	f, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64) // Remove commas
	if err != nil {
		return 0
	}
	return f
}

// Parse a string to int if value like 2324,34
func parseIntComma(value string) int {
	i, err := strconv.Atoi(strings.ReplaceAll(value, ",", ""))
	if err != nil {
		return 0
	}

	return i
}

// GenBarcode
func (h *ProductHandler) GenBarcode() string {
	var barcode string
	for {
		// Generate random 13-digit barcode
		barcode = generateRandomBarcode(13)

		// Check if barcode already exists in the database
		var count int64
		err := h.db.Model(&domain.Product{}).Where("barcode = ?", barcode).Count(&count).Error
		if err != nil {

			return ""
		}
		// If barcode is unique, return it
		if count == 0 {
			break
		}
	}
	return barcode
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
	headers := []string{"Аптека", "Код", "Наименования", "Штрих-код", "Кол-во", "Срок годности", "Серия", "Цена приход С НДС", "Цена продажа СНДС", "Cумма прихода С НДС", "Сумма продажа С НДС", "Производитель", "URL Фото"}

	err := setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		return nil, errors.New("failed to create style")
	}

	// give width to column
	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "C", 30)
	f.SetColWidth(sheetName, "D", "D", 10)
	f.SetColWidth(sheetName, "E", "E", 30)
	f.SetColWidth(sheetName, "F", "F", 15)
	f.SetColWidth(sheetName, "J", "J", 15)
	f.SetColWidth(sheetName, "H", "H", 20)

	// Ma'lumotlarni qo'shish
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.StoreName)
		f.SetCellValue(sheetName, "B"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "C"+row, product.Name)
		f.SetCellValue(sheetName, "D"+row, product.Barcode)
		f.SetCellValue(sheetName, "E"+row, math.Round((product.Quantity+product.UnitQuantity/float64(product.UnitPerPack))*10000)/10000)
		if product.ExpireDate != nil {
			f.SetCellValue(sheetName, "F"+row, product.ExpireDate)
		} else {
			f.SetCellValue(sheetName, "F"+row, "N/A")
		}
		f.SetCellValue(sheetName, "G"+row, product.SerialNumber)
		f.SetCellValue(sheetName, "H"+row, product.SupplyPrice)
		f.SetCellValue(sheetName, "I"+row, product.RetailPrice)
		f.SetCellValue(sheetName, "J"+row, math.Round((product.SupplyPrice*float64(product.Quantity)+(product.SupplyPrice/float64(product.UnitPerPack)*product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "K"+row, math.Round((product.RetailPrice*float64(product.Quantity)+(product.RetailPrice/float64(product.UnitPerPack)*product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "L"+row, product.Manufacturer)
		if len(product.Photos) > 0 && product.Photos[0] != "" {
			f.SetCellValue(sheetName, "M"+row, product.Photos[0])
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}
	}
	return f, nil
}

// product list export by store_id function
func (h *ProductHandler) productListExportByStoreId(f *excelize.File, res []domain.ProductData) (*excelize.File, error) {
	sheetName := "Products"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименование", "Штрих-код", "Производитель", "Кол-во", "Цена поставки", "Cумма поставки", "Цена продажи", "Cумма продажи", "Цена наценка", "Cумма наценка", "НДС", "Цена НДС", "Cумма НДС", "Категория", "MXIK", "Срок годности"}

	err := setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Warn("Failed to create style: %v", err)
		return nil, errors.New("failed to create style")
	}

	// give width to column
	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "B", 30)
	f.SetColWidth(sheetName, "C", "D", 20)
	f.SetColWidth(sheetName, "E", "N", 10)

	// Add product infos to excel column
	for i, product := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, product.Name)
		f.SetCellValue(sheetName, "C"+row, product.Barcode)
		f.SetCellValue(sheetName, "D"+row, product.Manufacturer)
		f.SetCellValue(sheetName, "E"+row, math.Round((product.Quantity+(product.UnitQuantity/float64(product.UnitPerPack)))*10000)/10000)
		f.SetCellValue(sheetName, "F"+row, product.SupplyPrice)
		f.SetCellValue(sheetName, "G"+row, math.Round((product.SupplyPrice*float64(product.Quantity)+(product.SupplyPrice/float64(product.UnitPerPack)*product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "H"+row, product.RetailPrice)
		f.SetCellValue(sheetName, "I"+row, math.Round((product.RetailPrice*float64(product.Quantity)+(product.RetailPrice/float64(product.UnitPerPack)*product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "J"+row, product.RetailPrice-product.SupplyPrice)
		f.SetCellValue(sheetName, "K"+row, math.Round(((product.RetailPrice-product.SupplyPrice)*float64(product.Quantity)+(product.RetailPrice-product.SupplyPrice)/float64(product.UnitPerPack)*product.UnitQuantity)*100)/100)
		f.SetCellValue(sheetName, "L"+row, product.Vat)
		f.SetCellValue(sheetName, "M"+row, product.VatPrice)
		f.SetCellValue(sheetName, "N"+row, math.Round((product.VatPrice*float64(product.Quantity)+(product.VatPrice/float64(product.UnitPerPack)*product.UnitQuantity))*100)/100)
		f.SetCellValue(sheetName, "O"+row, product.CategoryName)
		f.SetCellValue(sheetName, "P"+row, product.MXIK)
		f.SetCellValue(sheetName, "Q"+row, product.ExpireDate.Format("2006-01-02"))
	}
	return f, nil
}

func (h *ProductHandler) GenerateMarkingProducts(c *gin.Context) {
	var products []domain.Product

	err := h.db.Find(&products).Error
	if err != nil {
		h.log.Error(err)
		return
	}

	for _, product := range products {
		h.GenerateMarking(product.Id, "0dd07714-33fd-4716-8ab3-f3816a2e8f10")
	}
}

type TmpProduct struct {
	Marking        string `gorm:"marking" json:"marking"`
	ProductId      string `gorm:"product_id" json:"product_id"`
	ImportDetailId string `gorm:"import_detail_id" json:"import_detail_id"`
}

func (h *ProductHandler) GenerateMarking(productId string, importDetailId string) {
	var p = make([]TmpProduct, 0, 100)

	for i := 0; i < 100; i++ {
		marking := RandomString(31)
		p = append(p, TmpProduct{marking, productId, importDetailId})
	}

	// Save the products to the database
	err := h.db.Table("product_markings").Create(&p).Error
	if err != nil {
		h.log.Error(err)
	}
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
	var param domain.ProductQueryParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters: "+err.Error())
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}
	// defaults
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// service call
	data, total, err := h.service.ListExcludedProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, utils.ListResponse(data, total, param.Limit, param.Offset))
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
