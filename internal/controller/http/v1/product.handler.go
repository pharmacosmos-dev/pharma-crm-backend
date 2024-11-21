package v1

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ProductHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewProductHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *ProductHandler {
	return &ProductHandler{cfg, db, log}

}

func (h *ProductHandler) Create(c *gin.Context) {
	var body RequestBody[domain.Product]
	var res domain.Product
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&domain.Product{}).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (h *ProductHandler) Get(c *gin.Context) {
	var res domain.Product

	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (h *ProductHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	search := c.Query("search")
	var res []domain.Product
	var totalCount int64

	// Perform a single query to get both total count and paginated results
	query := h.db.Model(&domain.Product{}).Preload("Category").
		Joins("LEFT JOIN categories ON categories.id = products.category_id")
	if err := query.Where("products.name ILIKE ? OR products.barcode ILIKE ? OR categories.name ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res).Error; err != nil {
		h.log.Error(err)
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

func (h *ProductHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Product]
	var res domain.Product
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := h.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}

func (h *ProductHandler) UploadProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Retrieve the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	fmt.Println("File extension: ", ext) // Log the file extension
	if ext != ".xlsx" && ext != ".xls" {
		h.log.Error("Invalid file type: ", ext)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, "Invalid file type")
		return
	}

	// Save the uploaded file locally (optional)
	filePath := "./uploads/" + file.Filename
	fmt.Println("Saving file to: ", filePath) // Log the file path
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		h.log.Error("Failed to save file: ", err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, "Failed to save file")
		return
	}
	defer os.Remove(filePath)

	// Open the Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		h.log.Error("Failed to open file with excelize: ", err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	defer f.Close()

	// Iterate over rows and columns to read data
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.log.Error("Failed to get rows from sheet: ", err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}

	var products []domain.ProductUploadReq
	// Example: Iterate through rows and create products
	for _, row := range rows {
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

	// Create all products in the database
	if err := h.db.WithContext(ctx).Model(&domain.Product{}).Create(&products).Error; err != nil {
		h.log.Error("Failed to create products in database: ", err)
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
