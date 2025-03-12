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
			importDetail[i].SupplyPrice = body.StoreProduct[i].SupplyPrice
			importDetail[i].RetailPrice = body.StoreProduct[i].RetailPrice
			importDetail[i].Vat = body.StoreProduct[i].Vat
			importDetail[i].VatSum = body.StoreProduct[i].RetailPrice - body.StoreProduct[i].SupplyPrice
			importDetail[i].ExpireDate = time.Now().Format("2006-01-02 15:04:05")
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
		Select(`products.*, COALESCE(AVG(sp.retail_price), 0) AS retail_price`).
		Joins("LEFT JOIN store_products sp ON products.id = sp.product_id").
		Group(`products.id, products.id, products.brand_id, products.unit_type_id, 
		products.shelf_id, products.producer_id, products.material_code, 
		products.name, products.barcode, products.photos, products.unit_per_pack, 
		products.description, products.status, products.created_at, products.updated_at, products.deleted_at`).
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
// @Param no_barcode query bool false "No Barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/list [get]
func (h *ProductHandler) List(c *gin.Context) {
	var (
		param domain.ProductQueryParam
	)
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
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
// @Param supply_price_from query int false "Supply From"
// @Param supply_price_to query int false "Supply To"
// @Param retail_price_from query int false "Retail Price From"
// @Param retail_price_to query int false "Retail Price To"
// @Param no_barcode query bool false "No Barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/export-excel [get]
func (h *ProductHandler) ExportProductExcel(c *gin.Context) {
	var param domain.ProductQueryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Pagination parameters
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get products list
	products, _, err := h.service.ListProduct(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "Products"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Наименование", "Категория", "НДС", "Цена наценка", "Цена продажи", "Цена НДС", "Количество", "Цена", "Производитель", "Код продукта", "Штрих-код"}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "000000",
		},
	})
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}

	// Ma'lumotlarni qo'shish
	for i, product := range products {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, product.Name)
		f.SetCellValue(sheetName, "B"+row, product.CategoryName)
		f.SetCellValue(sheetName, "C"+row, product.Vat)
		f.SetCellValue(sheetName, "D"+row, product.MarkupPrice)
		f.SetCellValue(sheetName, "E"+row, product.RetailPrice)
		f.SetCellValue(sheetName, "F"+row, product.VatPrice)
		f.SetCellValue(sheetName, "G"+row, product.Quantity)
		f.SetCellValue(sheetName, "H"+row, product.Sum)
		f.SetCellValue(sheetName, "I"+row, product.Manufacturer)
		f.SetCellValue(sheetName, "J"+row, product.MaterialCode)
		f.SetCellValue(sheetName, "K"+row, product.Barcode)
	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=products.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
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
	err := h.db.Debug().Raw(query).Scan(&res).Error
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
	// match array
	body.Photos = utils.StringArray(body.Photos)
	err = tx.
		Model(&domain.Product{}).
		Where("id = ?", productID).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
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
	// 		err = tx.Table("import_details").Debug().Create(&domain.ImportDetailRequest{
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
	// 		// err = tx.Debug().Table("store_products").
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
	// 	err = tx.Debug().Table("store_products").Create(&storeProducts).Error
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
			tx.Rollback()
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
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

	// get limit offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get store products list
	res, err = h.service.ListStoreProduct(&param, storeId)
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
	err = h.db.Debug().First(&cartItem, "store_product_id = ? AND status = 'pending' AND sale_id = ?", storeProduct.Id, body.SaleID).Error
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
		storeID    = c.Query("store_id")
		productID  = c.Param("id")
		res        []domain.ImportDetail
		totalCount int64
		product    domain.Product
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
		storeID = employee.StoreId
	}

	// get product info
	err = h.db.First(&product, "id = ?", productID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
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
		Debug().
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/store-product/{id} [get]
func (h *ProductHandler) ListStoreProductProductId(c *gin.Context) {
	var (
		id         = c.Param("id")
		res        []domain.StoreProduct
		totalCount int64
		employee   domain.Employee
		storeID    string
	)
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid product id")
		return
	}
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
		storeID = employee.StoreId
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
		Select("store_products.*, u.short_name").
		Joins("JOIN products p ON p.id = store_products.product_id").
		Joins("LEFT JOIN unit_types u ON u.id = p.unit_type_id").
		Where("store_products.product_id = ?", id).
		Where("store_products.pack_quantity > 0 OR store_products.unit_quantity > 0 AND store_products.expire_date::date >= CURRENT_DATE")
	if storeID != "" {
		query = query.Where("store_products.store_id = ?", storeID)
	}
	// complete query
	err = query.
		Count(&totalCount).
		Limit(limit).Offset(offset).
		Order("store_products.created_at desc").
		Debug().
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

	// update barcode
	err = h.db.Model(&domain.Product{}).Where("id = ?", body.Id).Update("barcode", body.Barcode).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
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
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	err = tx.Table("producers").Create(&producers).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	err = tx.Table("products").Debug().Create(&products).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
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
	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 10 {
			err = tx.Exec("UPDATE products SET barcode = ? WHERE material_code = ?", row[10], row[9]).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				tx.Rollback()
				return
			}
		}
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
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

// Helper function to safely parse integer values
// func parseInt(value string) int {
// 	i, err := strconv.Atoi(value)
// 	if err != nil {
// 		return 0
// 	}
// 	return i
// }

// Helper function to parse percentage values (e.g., "12%")
// func parsePercentage(value string) float64 {
// 	// Remove the "%" symbol and trim spaces
// 	cleanValue := strings.TrimSuffix(strings.TrimSpace(value), "%")
// 	// Parse the remaining value as a float
// 	percentage, err := strconv.ParseFloat(cleanValue, 64)
// 	if err != nil {
// 		return 0 // Return 0 if parsing fails
// 	}
// 	return percentage
// }

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
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	digits := "0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = digits[random.Intn(len(digits))]
	}
	return string(result)
}
