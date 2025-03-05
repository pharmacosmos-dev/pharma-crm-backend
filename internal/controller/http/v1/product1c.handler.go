package v1

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
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
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// get store info
	var store domain.Store
	err = tx.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, "Store not found")
			return
		}
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect import data
	newImport := domain.ImportRequest{
		Id:             uuid.New().String(),
		StoreID:        store.Id,
		Status:         config.NEW_IMPORT,
		ImportDate:     body.Dok.DocumentDate,
		DocumentNumber: body.Dok.DocumentNumber,
	}
	// create new import
	err = tx.Table("imports").Create(&newImport).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			h.log.Warn("duplicate document_number: %v", err)
			handleResponse(c, OK, "Document with this number already exists")
			tx.Rollback()
			return
		}
		h.log.Error(fmt.Errorf("ERROR on creating dok: %v", err.Error()))
		handleResponse(c, InternalError, "Failed to creating new import")
		tx.Rollback()
		return
	}
	var productID string
	var importDetails []domain.ImportDetailRequest
	for i := range body.Товары {
		// get producer by code
		producer, err := h.service.GetProducerByCode(c.Request.Context(), body.Товары[i].Manufacturer)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Manufacturer not found or not created")
			tx.Rollback()
			return
		}
		// create product id
		productID = uuid.New().String()
		// create or update product
		err = tx.Raw(`
		INSERT INTO products (material_code, name, barcode, producer_id, mxik)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (material_code) DO UPDATE
		SET name = EXCLUDED.name,
			barcode = EXCLUDED.barcode,
			producer_id = EXCLUDED.producer_id,
			mxik = EXCLUDED.mxik
		RETURNING id;`,
			body.Товары[i].MaterialCode,
			body.Товары[i].Name, body.Товары[i].Barcode, producer.Id, body.Товары[i].Ikpu).Scan(&productID).Error
		if err != nil {
			h.log.Warn("ERROR on creating new product: %v", err.Error())
			handleResponse(c, BadRequest, "Error on checking product data")
			tx.Rollback()
			return
		}
		// check prices
		if body.Товары[i].SupplyPriceVat < 1 || body.Товары[i].RetailPriceVat < 1 {
			handleResponse(c, BadRequest, "SupplyPriceVat and RetailPriceVat must be greater than 0")
			tx.Rollback()
			return
		}
		// collect import details
		importDetails = append(importDetails, domain.ImportDetailRequest{
			ProductID:      &productID,
			ImportID:       newImport.Id,
			ReceivedCount:  body.Товары[i].Quantity,
			SupplyPrice:    body.Товары[i].SupplyPrice,
			SupplyPriceVat: body.Товары[i].SupplyPriceVat,
			RetailPrice:    body.Товары[i].RetailPrice,
			RetailPriceVat: body.Товары[i].RetailPriceVat,
			Vat:            cast.ToInt(body.Товары[i].Vat),
			VatSum:         body.Товары[i].VatSum,
			ExpireDate:     body.Товары[i].ExpireDate,
			SeriesNumber:   body.Товары[i].ProductSeriesNumber,
			SumVat:         body.Товары[i].SumVat,
			Marking:        utils.StringArray(body.Товары[i].Markirovka),
		})
	}
	// create import details
	err = tx.Table("import_details").Create(&importDetails).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "ERROR on creating import details")
		tx.Rollback()
		return
	}

	// check transaction completed
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to commit transaction")
		tx.Rollback()
		return
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
