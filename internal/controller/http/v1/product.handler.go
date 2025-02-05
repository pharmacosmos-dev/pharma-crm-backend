package v1

import (
	"errors"
	"fmt"
	"math/rand"
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
		product.POST("/store/barcode", h.GetStoreProductByBarcode)
		product.POST("/generate-barcode", h.GenerateBarcode)
		product.GET("/total-status-count", h.TotalStatusCount)
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
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	body.Id = uuid.New().String()
	body.Photos = utils.StringArray(body.Photos)
	body.Status = config.ACTIVE_PRODUCT
	body.MaterialCode = utils.GenerateMaterialCode()
	err = tx.
		WithContext(c.Request.Context()).
		Table("products").
		Create(&body).Error
	if err != nil {
		tx.Rollback()
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
			body.StoreProduct[i].Markup = body.Markup
			body.StoreProduct[i].ExpireDate = body.ExpireDate
			body.StoreProduct[i].BonusAmount = body.BonusAmount
			body.StoreProduct[i].BonusPercent = body.BonusPercent

			// imports table take required fields
			imports[i].Id = uuid.New().String()
			imports[i].StoreID = body.StoreProduct[i].StoreID
			imports[i].Status = config.COMPLETED_IMPORT
			imports[i].DocumentNumber = utils.GenerateDocumentNumber()
			imports[i].ImportDate = time.Now().Format("2006-01-02 15:04:05")

			// import detail take required fields
			importDetail[i].ImportID = imports[i].Id
			importDetail[i].ProductID = &body.Id
			importDetail[i].ReceivedCount = body.StoreProduct[i].PackQuantity
			importDetail[i].AcceptedCount = body.StoreProduct[i].PackQuantity
			importDetail[i].ReceivedAmount = float64(body.StoreProduct[i].PackQuantity) * body.RetailPrice
			importDetail[i].AcceptedAmount = float64(body.StoreProduct[i].PackQuantity) * body.RetailPrice
		}
		err = tx.
			WithContext(c.Request.Context()).
			Table("store_products").
			Create(&body.StoreProduct).Error
		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = tx.
			WithContext(c.Request.Context()).
			Table("imports").
			Create(&imports).Error
		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		err = tx.
			WithContext(c.Request.Context()).
			Table("import_details").
			Create(&importDetail).Error
		if err != nil {
			tx.Rollback()
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
		err = tx.
			WithContext(c.Request.Context()).
			Create(&categoryProduct).Error
		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product
	id := c.Param("id")
	err := h.db.
		Preload("UnitType").
		Preload("Shelf").
		Preload("Producer").
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
	category := []*domain.Category{}
	query := `
WITH RECURSIVE category_tree AS (
    -- Base case: Get categories directly linked to the product
    SELECT
        c.id,
        c.category_id,
        c.name::TEXT AS name_path,
        c.id AS root_category_id  -- Keep track of the original category
    FROM categories c
    INNER JOIN category_products cp ON c.id = cp.category_id
    WHERE cp.product_id = ?

    UNION ALL

    -- Recursive case: Build full category path
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

	// Execute the query
	if err := h.db.Raw(query, id).Scan(&category).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	res.Categories = category

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
		producerID      = c.Query("producer_id")
		status          = c.Query("status")
	)

	// Build the base query
	baseQuery := h.db.Model(&domain.Product{}).
		Table("products p").
		Joins("LEFT JOIN store_products sp ON sp.product_id = p.id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Joins("LEFT JOIN producers pr ON pr.id = p.producer_id")

	// Apply filters
	if storeIDParam != "" {
		baseQuery = baseQuery.Where("sp.store_id = ?", storeIDParam)
	}
	if status != "" {
		switch status {
		case "active":
			baseQuery = baseQuery.Where("p.status = ?", "active")
		case "inactive":
			baseQuery = baseQuery.Where("p.status = ?", "inactive")
		case "low-stock":
			baseQuery = baseQuery.Where("sp.small_quantity = sp.pack_quantity")
		case "zero-stock":
			baseQuery = baseQuery.Where("sp.pack_quantity = ? AND sp.unit_quantity = ?", 0, 0)
		case "expired":
			baseQuery = baseQuery.Where("sp.expire_date::date < ?", time.Now().Format("2006-01-02"))
		case "imminent":
			baseQuery = baseQuery.Where("sp.expire_date BETWEEN ? AND ?", time.Now(), time.Now().AddDate(0, 0, 10))
		}
	} else {
		baseQuery = baseQuery.Where("p.status = ?", "active")
	}

	if searchField != "" {
		searchField = fmt.Sprintf("%%%s%%", searchField)
		baseQuery = baseQuery.Where("p.name ILIKE ? OR p.barcode LIKE ?", searchField, searchField)
	}
	if supplyPriceFrom != "" {
		baseQuery = baseQuery.Where("sp.supply_price >= ?", supplyPriceFrom)
	}
	if supplyPriceTo != "" {
		baseQuery = baseQuery.Where("sp.supply_price <= ?", supplyPriceTo)
	}
	if retailPriceFrom != "" {
		baseQuery = baseQuery.Where("sp.retail_price >= ?", retailPriceFrom)
	}
	if retailPriceTo != "" {
		baseQuery = baseQuery.Where("sp.retail_price <= ?", retailPriceTo)
	}
	if producerID != "" {
		baseQuery = baseQuery.Where("p.producer_id = ?", producerID)
	}

	// Count total records using a subquery
	countQuery := baseQuery.Session(&gorm.Session{}).
		Select("COUNT(DISTINCT p.id)").
		Table("products p")

	err = countQuery.Debug().Count(&totalCount).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Execute main query with all fields
	err = baseQuery.
		Preload("Categories").
		Select(`
		p.id, p.name, p.barcode, p.status, p.description,
		p.photos, pr.name as manufacturer, p.material_code,
		AVG(sp.supply_price) AS supply_price,
		AVG(sp.vat) AS vat,
		AVG(p.markup) AS markup,
		AVG(sp.retail_price) AS retail_price,
		(AVG(sp.supply_price) * AVG(sp.vat) / 100) AS vat_price,
		(AVG(sp.supply_price) * AVG(p.markup) / 100) AS markup_price,
		SUM(sp.pack_quantity) AS quantity,
		(SUM(sp.pack_quantity) * AVG(sp.retail_price)) AS sum,
		AVG(sp.bonus_percent) AS bonus_percent,
		AVG(sp.bonus_amount) AS bonus_amount,
		u.short_name AS unit_name,
		p.created_at`).
		Group(`
			p.id, p.name, p.barcode, p.status, p.description, p.photos,
         	p.manufacturer, p.material_code, u.short_name, p.created_at, pr.name`).
		Order("p.created_at DESC").
		Limit(limit).
		Offset(offset).
		Debug().
		Find(&res).Error

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
// @Summary Get total count of products by status
// @Description Get total count of products by status
// @Tags products
// @Security     BearerAuth
// @Produce json
// @Param 	search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/total-status-count [get]
func (h *ProductHandler) TotalStatusCount(c *gin.Context) {
	var (
		res             domain.TotalStatusCount
		search          = c.Query("search")
		searchCondition string
	)
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		searchCondition = fmt.Sprintf("WHERE barcode LIKE '%s' OR name ILIKE '%s'", search, search)
	}
	query := fmt.Sprintf(`
	SELECT
		COUNT(*) AS total_count,
		COUNT(*) FILTER (WHERE status = 'active') AS active_count,
		COUNT(*) FILTER (WHERE status = 'inactive') AS inactive_count,
		(SELECT count(*) FROM store_products WHERE pack_quantity = 0 AND unit_quantity = 0) AS zero_stock_count,
		(SELECT count(*) FROM store_products WHERE small_quantity = pack_quantity) AS low_stock_count,
		(SELECT count(*) FROM store_products WHERE CURRENT_DATE <= expire_date AND expire_date <= CURRENT_DATE + INTERVAL '10 days') AS imminent_count,
		(SELECT count(*) FROM store_products WHERE expire_date::DATE < CURRENT_DATE) AS expired_count
	FROM products %s
	`, searchCondition)
	err := h.db.Raw(query).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	body.Photos = utils.StringArray(body.Photos)
	err = tx.
		WithContext(c.Request.Context()).
		Table("products").
		Where("id = ?", productID).
		Updates(&body).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if len(body.StoreProduct) > 0 {
		for i := range body.StoreProduct {
			if body.StoreProduct[i].MeasurementValue != 0 {
				status := config.COMPLETED_IMPORT
				operation := "+"
				if body.StoreProduct[i].MeasurementValue < 0 {
					status = config.WRITEOFF_IMPORT
					operation = "-"
					body.StoreProduct[i].MeasurementValue *= -1
				}
				importReq := domain.ImportRequest{
					Id:             uuid.New().String(),
					StoreID:        body.StoreProduct[i].StoreID,
					Status:         status,
					ImportDate:     time.Now().Format(config.DATE_FORMAT),
					DocumentNumber: utils.GenerateDocumentNumber(),
				}
				err = tx.Table("imports").Create(&importReq).Error
				if err != nil {
					tx.Rollback()
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
				err = tx.Table("import_details").Create(&domain.ImportDetailRequest{
					ImportID:       importReq.Id,
					ProductID:      &productID,
					ReceivedCount:  body.StoreProduct[i].MeasurementValue,
					AcceptedCount:  body.StoreProduct[i].MeasurementValue,
					ReceivedAmount: float64(body.StoreProduct[i].MeasurementValue) * body.RetailPrice,
					AcceptedAmount: float64(body.StoreProduct[i].MeasurementValue) * body.RetailPrice,
				}).Error
				if err != nil {
					tx.Rollback()
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}

				err = tx.Debug().Table("store_products").
					Where("product_id = ? AND store_id = ? ", productID, body.StoreProduct[i].StoreID).
					Updates(map[string]interface{}{
						"pack_quantity":  gorm.Expr("pack_quantity "+operation+" ?", body.StoreProduct[i].MeasurementValue),
						"unit_quantity":  gorm.Expr("(pack_quantity "+operation+" ?)*?", body.StoreProduct[i].MeasurementValue, body.UnitPerPack),
						"small_quantity": body.StoreProduct[i].SmallQuantity,
						"retail_price":   body.RetailPrice,
						"supply_price":   body.SupplyPrice,
						"vat":            body.Vat,
						"markup":         body.Markup,
						"unit_per_pack":  body.UnitPerPack,
					}).Error
				if err != nil {
					tx.Rollback()
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
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
				Columns:   []clause.Column{{Name: "product_id"}, {Name: "category_id"}},
				DoNothing: false,
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
			}).Create(&categoryProducts).Error
		if err != nil {
			tx.Rollback()
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
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
		search  = c.Query("search")
		storeID = c.Param("id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res, err = h.storage.ListStoreProduct(c.Request.Context(), storeID, search, limit, offset)
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
func (h *ProductHandler) GetStoreProductByBarcode(c *gin.Context) {
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
	err = h.db.First(&cartItem, "store_product_id = ? AND status = 'pending'", storeProduct.Id).Error
	if err == nil {
		// check quantity is enough in store_products table
		if storeProduct.PackQuantity < cartItem.Quantity+1 {
			handleResponse(c, CONFLICT, gin.H{
				"message":                "Not enough Product",
				"pack_quantity":          storeProduct.PackQuantity,
				"unit_quantity":          storeProduct.UnitQuantity,
				"received_pack_quantity": cartItem.Quantity + 1,
				"received_unit_quantity": cartItem.UnitQuantity,
			})
			return
		}
		// update cart_item
		newQuantity := cartItem.Quantity + 1
		cartItem.TotalPrice = storeProduct.RetailPrice * float64(newQuantity)
		err = h.db.Debug().Exec(`UPDATE cart_items SET quantity = ?, total_price = ? WHERE id = ?`,
			newQuantity, cartItem.TotalPrice, cartItem.ID).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "ADDED")
		return
	} else if errors.Is(err, gorm.ErrRecordNotFound) && storeProduct.PackQuantity > 0 {
		// create new cart_item
		err = h.db.Exec(`
		INSERT INTO cart_items(
			id, store_product_id, 
			employee_id, sale_id, 
			quantity, unit_price, 
			total_price, status
			) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), storeProduct.Id, userId.(string), body.SaleID, 1,
			storeProduct.RetailPrice, storeProduct.RetailPrice, "pending").Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		handleResponse(c, OK, "ADDED")
		return

	}

	handleResponse(c, BadRequest, "Not enough stock")
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
		Model(&domain.ImportDetail{}).
		Preload("Import", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Store")
		}).
		Where("product_id = ?", productID)
	if storeID != "" {
		query = query.
			Joins("INNER JOIN imports ON imports.id = import_details.import_id").
			Where("imports.store_id = ?", storeID)
	}
	var totalCount int64
	err = query.
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
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

// barcode generator
// generateRandomBarcode creates a random 13-digit numeric barcode
func generateRandomBarcode(length int) string {
	// Create a new random source and generator
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	digits := "0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = digits[random.Intn(len(digits))]
	}
	return string(result)
}
