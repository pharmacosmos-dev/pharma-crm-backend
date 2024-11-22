package v1

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/xuri/excelize/v2"
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
		product.GET("", h.Get)
		product.GET("/get-list", h.List)
		product.PUT("", h.Update)
		product.DELETE("", h.Delete)
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
	var body domain.ProductRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Table("products").Create(&body).Scan(&body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

// Get godoc
// @Summary Get a product
// @Description Get a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "product ID"
// @Success 200 {object} domain.Product
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product [get]
func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
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
// @Router /product/get-list [get]
func (h *ProductHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var (
		res        []domain.Product
		totalCount int64
		search     = c.Query("search")
	)

	// Perform a single query to get both total count and paginated results
	query := h.Db.Model(&domain.Product{}).Preload("Category").
		Joins("LEFT JOIN categories ON categories.id = products.category_id")
	if err := query.Where("products.name ILIKE ? OR products.barcode ILIKE ? OR categories.name ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}

	result := struct {
		Product []domain.Product `json:"data"`
		Meta    domain.Meta      `json:"_meta"`
	}{
		Product: res,
		Meta: domain.Meta{
			TotalCount:  int(totalCount),
			PerPage:     limit,
			CurrentPage: (offset / limit) + 1,
			PageCount:   int((totalCount + int64(limit) - 1) / int64(limit)),
		},
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, result)
}

// Get godoc
// @Summary Update a product
// @Description Update a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.ProductUpdateRequest true "Product information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product [put]
func (h *ProductHandler) Update(c *gin.Context) {
	var body domain.ProductUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	if err := h.Db.WithContext(c.Request.Context()).Table("products").Model(&body).Where("id = ?", body.Id).Updates(&body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body)
}

// Get godoc
// @Summary Delete a product
// @Description Delete a product from the request body
// @Tags products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id query string true "product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Product{}, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
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
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	// Read file contents into a byte slice
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		h.Log.Error("Failed to read file: ", err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, "Failed to read file")
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		h.Log.Error("Failed to open file: ", err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, "Failed to open file")
		return
	}
	sheetNames := f.GetSheetMap()
	sheetName := f.GetSheetName(1)
	rows, _ := f.GetRows(sheetName)

	var products []domain.ProductUploadReq
	// Iterate through rows and create products
	for _, row := range rows { // Skip the header row
		if len(row) < 11 {
			h.Log.Error("Row does not have enough columns")
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
	if err := h.Db.WithContext(c.Request.Context()).Model(&domain.Product{}).Create(&products).Error; err != nil {
		h.Log.Error("Failed to create products in database: ", err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}

	handleResponse(c, http.StatusOK, MsgSuccessCreate, MsgSuccessCreate)
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
