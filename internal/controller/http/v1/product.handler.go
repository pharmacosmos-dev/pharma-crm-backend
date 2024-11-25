package v1

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

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
		product.DELETE("/:id", h.Delete)
		product.POST("/upload", h.UploadProduct)
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
		Model(&domain.Product{}).
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
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
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

	// Prepare search field for ILIKE queries
	searchField := fmt.Sprintf("%%%s%%", c.Query("search"))

	categoryIDParam := c.Query("category_id")

	storeIDParam := c.Query("store_id")

	// Handle price range parameters
	supplyPriceFrom := c.Query("supply_price_from")
	if supplyPriceFrom == "" {
		supplyPriceFrom = "0" // or a suitable default value
	}
	supplyPriceTo := c.Query("supply_price_to")
	if supplyPriceTo == "" {
		supplyPriceTo = "999999999" // or a suitable default value
	}
	retailPriceFrom := c.Query("retail_price_from")
	if retailPriceFrom == "" {
		retailPriceFrom = "0" // or a suitable default value
	}
	retailPriceTo := c.Query("retail_price_to")
	if retailPriceTo == "" {
		retailPriceTo = "999999999" // or a suitable default value
	}
	// Build the query
	query := h.db.Model(&domain.Product{}).
		Preload("Category").
		Joins("LEFT JOIN categories c ON c.id = products.category_id").
		Where("products.name ILIKE ? OR products.barcode ILIKE ? OR c.name ILIKE ?", searchField, searchField, searchField).
		Where("is_active = ? ", true).
		Where("supply_price BETWEEN ? AND CASE WHEN ? = 0 THEN 999999999 ELSE ? END", supplyPriceFrom, supplyPriceTo, supplyPriceTo).
		Where("retail_price BETWEEN ? AND CASE WHEN ? = 0 THEN 999999999 ELSE ? END", retailPriceFrom, retailPriceTo, retailPriceTo).
		Where("(manufacturer = ? OR ? = '')", c.Query("producer"), c.Query("producer")).
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res)
	if categoryIDParam != "" {
		query.Where("category_id = ?", categoryIDParam)
	}
	if storeIDParam != "" {
		query.Where("store_id = ?", storeIDParam)
	}
	// Handle errors from the query
	if query.Error != nil {
		h.log.Error(query.Error)
		handleResponse(c, InternalError, query.Error.Error())
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
// @Param id path string true "product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Product{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// UploadProduct godoc
// @Summary Upload a product
// @Description Upload a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Product file"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/upload [post]
func (h *ProductHandler) UploadProduct(c *gin.Context) {
	file, handler, err := c.Request.FormFile("file")
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Read file contents into a byte slice
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		h.log.Error("Failed to read file: ", err)
		handleResponse(c, BadRequest, "Failed to read file")
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		h.log.Error("Failed to open file: ", err)
		handleResponse(c, InternalError, "Failed to open file")
		return
	}
	sheetNames := f.GetSheetMap()
	sheetName := f.GetSheetName(1)
	rows, _ := f.GetRows(sheetName)

	var products []domain.ProductUploadReq
	// Iterate through rows and create products
	for _, row := range rows { // Skip the header row
		if len(row) < 11 {
			h.log.Error("Row does not have enough columns")
			continue
		}

		product := domain.ProductUploadReq{
			Id:           uuid.New().String(),
			Name:         row[1],
			SupplyPrice:  parseFloat(row[2]),
			Vat:          parseInt(row[3]),
			RetailPrice:  parseFloat(row[4]),
			VatPrice:     parseFloat(row[5]),
			Quantity:     parseInt(row[6]),
			Sum:          parseFloat(row[7]),
			Manufacturer: row[8],
			ExpireDate:   row[9],
			Barcode:      row[10],
			Status:       "active",
		}
		products = append(products, product)
	}
	// Process rows
	fmt.Printf("Uploaded file: %s\n", handler.Filename)
	fmt.Printf("File size: %d bytes\n", handler.Size)
	fmt.Printf("Sheets in the file: %v\n", sheetNames)
	// Create all products in the database
	if err := h.db.WithContext(c.Request.Context()).Model(&domain.Product{}).Create(&products).Error; err != nil {
		h.log.Error("Failed to create products in database: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "Products uploaded successfully")
}

func (h *ProductHandler) GetProducerList(c *gin.Context) {
	var (
		res []*domain.ProductProducer
		err error
	)
	err = h.db.Select("manufacturer").Model(&domain.Product{}).Distinct("manufacturer").Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)

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
