package v1

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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
		product.PUT("/:id", h.Update)
		product.POST("/excel-upload", h.UploadProduct)
		product.GET("/producer", h.GetProducerList)
		product.GET("/similar/:id", h.SimilarProducts)
		product.GET("/store/:id", h.ListByStoreId)
		product.GET("/import/:id", h.GetProductImports)
		product.DELETE("/hard-delete", h.HardDelete)
		product.DELETE("/soft-delete", h.SoftDelete)
		product.GET("/store-product/:id", h.ListStoreProductProductId)
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
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.Id = uuid.New().String()
	body.Photos = utils.StringArray(body.Photos)
	body.Status = config.ACTIVE_PRODUCT
	body.MaterialCode = utils.GenerateMaterialCode()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("products").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if len(body.StoreProduct) > 0 {
		var imports = make([]domain.ImportRequest, len(body.StoreProduct))
		var importDetail = make([]domain.ImportDetailRequest, len(body.StoreProduct))
		for i := range body.StoreProduct {
			// store products table take required fields
			body.StoreProduct[i].ProductID = body.Id
			body.StoreProduct[i].UnitQuantity = body.StoreProduct[i].PackQuantity * body.UnitPerPack
			body.StoreProduct[i].UnitPerPack = body.UnitPerPack
			body.StoreProduct[i].SupplyPrice = body.SupplyPrice
			body.StoreProduct[i].RetailPrice = body.RetailPrice
			body.StoreProduct[i].Vat = body.Vat
			body.StoreProduct[i].ExpireDate = body.ExpireDate

			// imports table take required fields
			imports[i].Id = uuid.New().String()
			imports[i].StoreID = body.StoreProduct[i].StoreID
			imports[i].Status = config.COMPLETED_IMPORT
			imports[i].DocumentNumber = utils.GenerateDocumentNumber()
			imports[i].ImportDate = time.Now().Format("2006-01-02 15:04:05")

			// import detail take required fields
			importDetail[i].ImportID = imports[i].Id
			importDetail[i].ProductID = &body.Id
			importDetail[i].ReceivedCount = body.Quantity
			importDetail[i].ProductMaterialCode = body.MaterialCode
			importDetail[i].ReceivedAmount = float64(body.Quantity) * body.RetailPrice
		}
		err = h.db.
			WithContext(c.Request.Context()).
			Table("store_products").
			Create(&body.StoreProduct).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = h.db.
			WithContext(c.Request.Context()).
			Table("imports").
			Create(&imports).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = h.db.
			WithContext(c.Request.Context()).
			Table("import_details").
			Create(&importDetail).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if len(body.CategoryIds) > 0 {
		var categoryProduct = make([]domain.CategoryProduct, len(body.CategoryIds))
		for i := range body.CategoryIds {
			categoryProduct[i].ProductId = body.Id
			categoryProduct[i].CategoryId = body.CategoryIds[i]
			categoryProduct[i].IsOpen = true
		}
		err = h.db.
			WithContext(c.Request.Context()).
			Create(&categoryProduct).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product
	id := c.Param("id")
	err := h.db.
		Preload("UnitType").
		First(&res, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Product not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
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
// @Param producer query string false "Producer"
// @Param supply_price_from query int false "Supply From"
// @Param supply_price_to query int false "Supply To"
// @Param retail_price_from query int false "Retail Price From"
// @Param retail_price_to query int false "Retail Price To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/list [get]
func (h *ProductHandler) List(c *gin.Context) {
	var (
		res        []domain.Product
		totalCount int64
	)

	// Pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var (
		searchField     = c.Query("search")
		storeIDParam    = c.Query("store_id")
		supplyPriceFrom = c.Query("supply_price_from")
		supplyPriceTo   = c.Query("supply_price_to")
		retailPriceFrom = c.Query("retail_price_from")
		retailPriceTo   = c.Query("retail_price_to")
		producerName    = c.Query("producer")
		status          = c.Query("status")
	)

	// Build the query
	query := h.db.Model(&domain.Product{}).
		Select("products.*, DATE_PART('day', expire_date::timestamp - NOW()) AS expire_day").
		Preload("Categories")

	if storeIDParam != "" {
		query = query.
			Joins("JOIN store_products ON store_products.product_id = products.id").
			Where("store_products.store_id = ?", storeIDParam)
	}
	if status != "" {
		switch status {
		case "active":
			query = query.Where("products.status = ?", "active")
		case "inactive":
			query = query.Where("products.status = ?", "inactive")
		case "low-stock":
			query = query.Where("products.quantity <= ?", 10)
		case "zero-stock":
			query = query.Where("products.quantity = ?", 0)
		case "expired":
			query = query.Where("products.expire_date < ?", time.Now().Add(time.Hour*5))
		case "imminent":
			query = query.Where("products.expire_date BETWEEN ? AND ?", time.Now(), time.Now().AddDate(0, 0, 10))
		}
	} else {
		query = query.Where("products.status = ?", "active")
	}

	if searchField != "" {
		searchField = fmt.Sprintf("%%%s%%", searchField)
		query = query.Where("products.name ILIKE ? OR products.barcode ILIKE ?", searchField, searchField)
	}
	if supplyPriceFrom != "" {
		query = query.Where("products.supply_price >= ?", supplyPriceFrom)
	}
	if supplyPriceTo != "" {
		query = query.Where("products.supply_price <= ?", supplyPriceTo)
	}
	if retailPriceFrom != "" {
		query = query.Where("products.retail_price >= ?", retailPriceFrom)
	}
	if retailPriceTo != "" {
		query = query.Where("products.retail_price <= ?", retailPriceTo)
	}
	if producerName != "" {
		query = query.Where("products.manufacturer = ?", producerName)
	}

	err = query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("products.created_at DESC").
		Debug().
		Find(&res).Error
	// Handle errors from the query
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Prepare the response
	result := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, result)
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

	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Photos = utils.StringArray(body.Photos)
	err = h.db.
		WithContext(c.Request.Context()).
		Table("products").
		Where("id = ?", productID).
		Updates(body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(body.StoreProduct) > 0 {
		for _, storeProduct := range body.StoreProduct {
			storeProduct.ProductID = productID
			storeProduct.UnitQuantity = int(storeProduct.PackQuantity * body.UnitPerPack)
			err = h.db.WithContext(c.Request.Context()).
				Table("store_products").
				Where("product_id = ?", productID).
				Updates(&storeProduct).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
	}

	if len(body.CategoryIds) > 0 {
		var categoryProducts = make([]domain.CategoryProduct, len(body.CategoryIds))
		for i, categoryId := range body.CategoryIds {
			categoryProducts[i] = domain.CategoryProduct{
				ProductId:  productID,
				CategoryId: categoryId,
			}
		}
		err = h.db.WithContext(c.Request.Context()).
			Table("category_products").
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "product_id"}, {Name: "category_id"}}, // Define unique columns
				DoNothing: false,                                                        // If true, skips updates
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),             // Update specific columns
			}).Create(&categoryProducts).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
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
		res []*domain.ProductProducer
		err error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	search := c.Query("search")

	query := h.db.
		Model(&domain.Product{}).
		Select("DISTINCT manufacturer")
	if search != "" {
		query = query.Where("manufacturer ILIKE ?", search)
	}
	err = query.Limit(limit).
		Offset(offset).
		Find(&res).
		Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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

	// Process rows
	var products []domain.ProductUploadReq
	var product domain.ProductUploadReq
	for _, row := range rows[3:] {
		if len(row) < 11 {
			h.log.Warn("Row does not have enough columns, skipping row")
			continue
		}
		product = domain.ProductUploadReq{
			Id:           uuid.New().String(),
			Name:         row[1],
			SupplyPrice:  parseFloat(row[2]),
			Vat:          parsePercentage(row[3]),
			RetailPrice:  parseFloat(row[4]),
			VatPrice:     parseFloat(row[5]),
			Quantity:     parseInt(row[6]),
			Sum:          parseFloat(row[7]),
			Manufacturer: row[8],
			ExpireDate:   row[9],
			Barcode:      row[10],
			Status:       "active",
			IsActive:     true,
		}
		products = append(products, product)
	}

	// Insert into the database
	if err = h.db.WithContext(c.Request.Context()).Table("products").Create(&products).Error; err != nil {
		h.log.Error("Failed to create products in database: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
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
	res, err = h.storage.SimilarProducts(c.Request.Context(), id, offset, limit)
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
// @Param id path string true "Store ID"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store/{id} [get]
func (h *ProductHandler) ListByStoreId(c *gin.Context) {
	var (
		res     []*domain.StoreProductResponse
		err     error
		search  = c.Query("search")
		storeID = c.Param("id")
	)
	res, err = h.storage.ListStoreProduct(c.Request.Context(), storeID, search)
	if err != nil {
		handleResponse(c, InternalError, "Failed to fetch products")
		return
	}

	handleResponse(c, OK, res)
}

// GetProductImports godoc
// @Summary Get product imports
// @Description Get product imports
// @Tags products
// @Security     BearerAuth
// @Accept json
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
		storeID   = c.Query("store_id")
		productID = c.Param("id")
		res       []*domain.ImportDetail
	)
	var product domain.Product
	err := h.db.First(&product, "id = ?", productID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	query := h.db.
		Preload("Import", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Stores")
		}).
		Where("product_material_code = ?", product.MaterialCode)
	if storeID != "" {
		query = query.Joins("INNER JOIN imports ON imports.id = import_details.import_id").Where("imports.store_id = ?", storeID)
	}
	err = query.Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store-product/{id} [get]
func (h *ProductHandler) ListStoreProductProductId(c *gin.Context) {
	var (
		id         = c.Param("id")
		res        []domain.StoreProduct
		totalCount int64
	)
	if id == "" || id == "undefined" {
		handleResponse(c, BadRequest, "Product ID is required")
		return
	}

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		Model(&domain.StoreProduct{}).
		Preload("Store").
		Where("product_id = ?", id).
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("created_at desc").
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

// Helper function to safely parse float values
func parseFloat(value string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64) // Remove commas
	if err != nil {
		return 0
	}
	return f
}

// Helper function to safely parse integer values
func parseInt(value string) int {
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}

// Helper function to parse percentage values (e.g., "12%")
func parsePercentage(value string) float64 {
	// Remove the "%" symbol and trim spaces
	cleanValue := strings.TrimSuffix(strings.TrimSpace(value), "%")
	// Parse the remaining value as a float
	percentage, err := strconv.ParseFloat(cleanValue, 64)
	if err != nil {
		return 0 // Return 0 if parsing fails
	}
	return percentage
}

// Parse a string to int if value like 2324,34
func parseIntComma(value string) int {
	i, err := strconv.Atoi(strings.ReplaceAll(value, ",", ""))
	if err != nil {
		return 0
	}

	return i
}
