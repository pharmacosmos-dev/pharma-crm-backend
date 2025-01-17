package v1

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Product1cHandler struct {
	*Handler
}

func (h *Handler) NewProduct1cHandler(r *gin.RouterGroup) {
	product1c := &Product1cHandler{h}
	product1c.Product1cRoutes(r)
}

func (h *Product1cHandler) Product1cRoutes(r *gin.RouterGroup) {
	r.POST("/product1c", h.Create)
	r.POST("/store1c", h.CreateStore)
	r.POST("/store1c/excel-upload", h.UploadStores)
}

// Create 	godoc
// @Summary Create a product
// @Description Create a product from the request body
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	product body domain.CreateProduct1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c [post]
func (h *Product1cHandler) Create(c *gin.Context) {
	var (
		body domain.CreateProduct1C
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	var store domain.Store
	err = h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, "Store not found")
			return
		}
		handleResponse(c, InternalError, err.Error())
		return
	}
	importID := uuid.New().String()
	newImport := domain.ImportRequest{
		Id:             importID,
		StoreID:        store.Id,
		StoreCode:      body.Apteka.StoreCode,
		Status:         config.NEW_IMPORT,
		ImportDate:     time.Now().Format("2006-01-02 15:04:05"),
		DocumentNumber: body.Dok.DocumentNumber,
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Create(&newImport).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			h.log.Warn("duplicate document_number: %v", err)
			handleResponse(c, OK, "Document with this number already exists")
			return
		}
		// Log and handle other errors
		h.log.Error(fmt.Errorf("ERROR on creating dok: %v", err.Error()))
		handleResponse(c, InternalError, "Failed to creating new import")
		return
	}

	err = h.db.WithContext(c.Request.Context()).
		Table("products").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "material_code"}}, // Specify the column(s) to check for conflict
			DoNothing: true,                                     // Ignore if conflict occurs
		}).
		Create(&body.Товары).Error
	if err != nil {
		h.log.Warn("ERROR on creating new product: %v", err.Error())
		handleResponse(c, InternalError, "New import created but new product not created")
		return
	}
	var importDetails []domain.ImportDetailRequest
	for _, product := range body.Товары {
		importDetails = append(importDetails, domain.ImportDetailRequest{
			ImportID:            importID,
			ProductMaterialCode: product.MaterialCode,
			ReceivedCount:       product.Quantity,
			ReceivedAmount:      float64(product.Quantity) * product.RetailPrice,
		})
	}
	if len(importDetails) > 0 {
		err = h.db.
			WithContext(c.Request.Context()).
			Table("import_details").
			Create(&importDetails).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "ERROR on creating import details")
			return
		}
	}

	handleResponse(c, OK, "CREATED")
}

// Create godoc
// @Summary Create a store
// @Description Create a store from the request body
// @Tags 1C Api
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param store body []domain.StoreRequest1C true "store"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store1c [post]
func (h *Product1cHandler) CreateStore(c *gin.Context) {
	var (
		body []domain.StoreRequest1C
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Create stores from 1C
	err = h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Create(&body).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			h.log.Warn("duplicate document_number: %v", err)
			handleResponse(c, OK, "Store with this code already exists")
			return
		}
		// Log and handle other errors
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "CREATED")
}

// UploadStores godoc
// @Summary Upload a store
// @Description Upload a store file in .xlsx format. The file should include store details in specific columns.
// @Tags 1C Api
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file (.xlsx) containing store data"
// @Success 200 {object} v1.Response "Stores uploaded successfully"
// @Failure 400 {object} v1.Response "Invalid file format or processing error"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /store1c/excel-upload [post]
func (h *Product1cHandler) UploadStores(c *gin.Context) {
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
	var stores []domain.StoreRequest1C
	for _, row := range rows[2:] {
		store := domain.StoreRequest1C{
			Id:        uuid.New().String(),
			Name:      row[0],
			StoreCode: parseIntComma(row[1]),
		}
		if store.StoreCode == 4048 {
			continue
		}
		stores = append(stores, store)
	}

	if len(stores) > 0 {
		err = h.db.
			WithContext(c.Request.Context()).
			Table("stores").
			Create(&stores).Error
		if err != nil {
			h.log.Error(fmt.Errorf("err: %v", err))
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, OK, "Stores uploaded successfully")
}
