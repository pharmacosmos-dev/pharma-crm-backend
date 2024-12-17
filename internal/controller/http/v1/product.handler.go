package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
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
		product.DELETE("/delete", h.MultipleDelete)
		product.POST("/excel-upload", h.UploadProduct)
		product.GET("/producer", h.GetProducerList)
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
// @Success 201 {object} domain.ProductRequest
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
	err = h.db.WithContext(c.Request.Context()).
		Table("products").
		Create(&body).
		Scan(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "product ID"
// @Success 200 {object} domain.Product
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
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
		Preload("Categories").
		Joins("LEFT JOIN category_products ON category_products.product_id = products.id").
		Joins("LEFT JOIN categories ON categories.id = category_products.category_id")
	if status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		case "low-stock":
			query = query.Where("quantity <= ?", 5)
		case "zero-stock":
			query = query.Where("quantity = ?", 0)
		case "expired":
			query = query.Where("expire_date < ?", time.Now())
		case "imminent":
			query = query.Where("expire_date BETWEEN ? AND ?", time.Now(), time.Now().AddDate(0, 0, 10))
		}
	}
	if storeIDParam != "" {
		query = query.Where("store_id = ?", storeIDParam)
	}
	if searchField != "" {
		searchField = fmt.Sprintf("%%%s%%", searchField)
		query = query.Where("name ILIKE ? OR barcode ILIKE ?", searchField, searchField)
	}
	if supplyPriceFrom != "" {
		query = query.Where("supply_price >= ?", supplyPriceFrom)
	}
	if supplyPriceTo != "" {
		query = query.Where("supply_price <= ?", supplyPriceTo)
	}
	if retailPriceFrom != "" {
		query = query.Where("retail_price >= ?", retailPriceFrom)
	}
	if retailPriceTo != "" {
		query = query.Where("retail_price <= ?", retailPriceTo)
	}
	if producerName != "" {
		query = query.Where("manufacturer = ?", producerName)
	}

	err = query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("quantity DESC").
		Find(&res).Error
	// Handle errors from the query
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
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
		body domain.ProductRequest
		err  error
	)

	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Photos = utils.StringArray(body.Photos)
	err = h.db.WithContext(c.Request.Context()).Model(&domain.Product{}).Where("id = ?", c.Param("id")).Updates(body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Get godoc
// @Summary Delete a product
// @Description Delete a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param body 	body []string true "product IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/delete [delete]
func (h *ProductHandler) MultipleDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
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

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
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
