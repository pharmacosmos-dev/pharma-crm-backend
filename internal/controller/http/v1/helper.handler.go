package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/unicode/norm"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type HelperHandler struct {
	*Handler
}

func (h *Handler) NewHelperHandler(r *gin.RouterGroup) {
	helper := &HelperHandler{h}
	helper.HelperRoutes(r)
}

func (h *HelperHandler) HelperRoutes(r *gin.RouterGroup) {
	helper := r.Group("/helper")
	{
		helper.POST("/upload-tax-products", h.UploadTaxProducts)
		helper.POST("/upload-unit-count", h.UploadProductUnitCount)
		helper.POST("/upload-mxik", h.CorrectMXIK)
		helper.POST("/epos", h.EposTransmitter)
		helper.POST("/upload-category", h.UploadCategory)
		helper.POST("/upload-customer", h.UploadCustomer)
		helper.POST("/upload-import", h.UploadImport)
		helper.GET("picture", h.GetProductPictureFromTasnif)
		helper.POST("/product-min-max", h.UploadProductMinMax)
		helper.POST("/product-kvant", h.UploadProductKvant)
		helper.POST("/set-product-photo", h.SetProductPhoto)
		helper.POST("/fix-product-quantity", h.FixProductQuantity)
		helper.POST("/update-product-info", h.UpdateProductInfo)
		helper.POST("/update-wrong-items", h.UpdateSaleItems)
		helper.POST("/add-categories", h.AddCategories)
		helper.POST("/attach-category-to-products", h.AttachCategoryToProducts)
		helper.POST("/upload-categories-json", h.UploadCategoryJson)
		helper.DELETE("/delete-not-photos", h.DeleteNotFoundPhotos)
	}
}

// GetIKPUDatafromSoliq godoc
// @Summary Get IKPU data from Soliq
// @Description Get IKPU data from Soliq
// @Tags helper
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param 		 lang      	query    string  true "Lang: (uz_latn || uz_cyrl || ru)"
// @Param  		 groupCode  query    string  true  "Group code"
// @Param 		 classCode  query    string  true  "Class code"
// @Param        limit      query    int     true "Limit"
// @Param        offset     query    int     true "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /helper/get-ikpu-data-from-soliq [POST]
func (h *HelperHandler) GetIKPUDatafromSoliq(c *gin.Context) {
	var (
		lang      = c.Query("lang")
		groupCode = c.Query("groupCode")
		classCode = c.Query("classCode")
		limit     = c.Query("limit")
		offset    = c.Query("offset")
	)

	url := h.cfg.TasnifApiUrl + "/web-katalog"
	if lang != "" {
		url += "?lang=" + lang
	}
	if offset != "" {
		url += "&pageNo=" + offset
	}
	if limit != "" {
		url += "&pageSize=" + limit
	}
	if groupCode != "" {
		url += "&groupCode=" + groupCode
	}
	if classCode != "" {
		url += "&classCode=" + classCode
	}
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		h.log.Errorf("Request yaratishda xatolik: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// HTTP client yaratish va timeout o‘rnatish
	client := &http.Client{Timeout: 1 * time.Minute}

	// So‘rovni jo‘natish
	resp, err := client.Do(req)
	if err != nil {
		h.log.Errorf("So‘rov jo‘natishda xatolik: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	defer resp.Body.Close()
	var data domain.SoliqResponse

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	tx := h.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	query := `
	INSERT INTO product_measurements(
		mxik_code, class_code, mxik_name, mxik_name_uz, mxik_name_ru, unit_name, unit_code
	) VALUES(?, ?, ?, ?, ?, ?, ?) ON CONFLICT (mxik_code) DO NOTHING;`
	for _, item := range data.Data {
		err = tx.Exec(query, item.MxikCode, classCode, item.Name, item.Name, item.Units).Error
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

	handleResponse(c, OK, "CREATED")
}

// UploadPackageCodeExcel godoc
// @Summary Upload package code excel
// @Description Upload package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-tax-products [POST]
func (h *HelperHandler) UploadTaxProducts(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Error("Failed to open .xlsx file: ", err.Error())
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(2)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	// build query
	query := `
	INSERT INTO tax_products(name_uz, mxik, unit_code, unit_name)
	VALUES (?, ?, ?, ?)
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		count++
		err := h.db.Exec(query, row[1], row[0], row[3], row[2]).Error
		if err != nil {
			h.log.Warn("ERROR on updating products: %v", err)
		}

	}

	handleResponse(c, OK, "Tax Products uploaded successfully")
}

// UploadProduct godoc
// @Summary Upload package code excel
// @Description Upload package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-mxik [POST]
func (h *HelperHandler) CorrectMXIK(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	// build query
	query := `
	UPDATE products SET mxik = ?, unit_code = ?, updated_at = now() WHERE material_code = ?
	`
	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 3 {
			err := h.db.Debug().Exec(query, strings.TrimSpace(row[2]), strings.TrimSpace(row[3]), strings.TrimSpace(row[0])).Error
			if err != nil {
				h.log.Error("could not updated product(%s) mxik(%s): %v", strings.TrimSpace(row[2]), err)
			} else {
				count++
			}
		}

	}
	handleResponse(c, OK, "UPDATED: "+strconv.Itoa(count))
}

// Epos transmitter godoc
// @Summary transmit request to epos
// @Description transmit request to epos
// @Tags helper
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	body body interface{} true "Epos request body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/epos [POST]
func (h *HelperHandler) EposTransmitter(c *gin.Context) {
	var body any
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // Set a timeout for the request
	}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(body)
	if err != nil {
		handleResponse(c, BadRequest, "Can't encode request data")
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", "http://integration.epos.uz:8347/uzpos", &buf)
	if err != nil {
		h.log.Warn("ERROR on creating new request: %v", err)
		handleResponse(c, InternalError, "Can't create new request")
		return
	}
	// add headers
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		h.log.Warn("ERROR on doing request: %v", err)
		handleResponse(c, InternalError, "Can't do request")
		return
	}
	defer resp.Body.Close()
	var res any
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		h.log.Warn("ERROR on decoding epos response %v", err)
		handleResponse(c, InternalError, "Can't decode response data")
		return
	}
	c.JSON(http.StatusOK, res)
}

// UploadProduct godoc
// @Summary Upload package code excel
// @Description Upload package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-category [POST]
func (h *HelperHandler) UploadCategory(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	// build query
	query := `
	INSERT INTO temp_excel_data(code, category_name) VALUES(?, ?)
	`
	var count = 0

	// Process rows
	for _, row := range rows[1:] {
		if len(row) == 2 {
			count++
			// // create measurements
			err = h.db.Exec(query, row[0], row[1]).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
	}
	handleResponse(c, OK, "Products MXIK CODE uploaded successfully: ")
}

// UploadProduct godoc
// @Summary Upload package code excel
// @Description Upload package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-customer [POST]
func (h *HelperHandler) UploadCustomer(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	// build query
	query := `
	INSERT INTO customers(id, full_name, phone, birthday) VALUES(?, ?, ?, ?)
	`
	queryd := `
	INSERT INTO discount_cards(customer_id, barcode, percent) VALUES(?, ?, ?)
	`
	var count = 0

	// Process rows
	for _, row := range rows[3:] {

		count++
		customer := domain.CustomerRequest{
			Id:       uuid.New().String(),
			FullName: row[2],
			Phone:    cast.ToString(row[3]),
			Birthday: &row[4],
		}
		// // create measurements
		err = h.db.Exec(query, customer.Id, customer.FullName, customer.Phone, customer.Birthday).Error
		if err != nil {
			h.log.Warn("ERROR on creating customers: %v", err)
		}
		err = h.db.Exec(queryd, customer.Id, row[5], row[8]).Error
		if err != nil {
			h.log.Warn("ERROR on creatig discount_card: %v", err)
		}

	}
	handleResponse(c, OK, "Products Customer uploaded successfully: ")
}

// UploadProduct godoc
// @Summary Upload package code excel
// @Description Upload package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-unit-count [POST]
func (h *HelperHandler) UploadProductUnitCount(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	// build query
	query := `
		UPDATE products SET unit_code = ?, unit_label = ? WHERE material_code = ?;
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if count <= 247 {
			// // create measurements
			err = h.db.Exec(query, row[4], row[7], cast.ToInt(row[0])).Error
			if err != nil {
				h.log.Warn("ERROR on creating customers: %v", err)
			}
			count++
		}
	}
	handleResponse(c, OK, "Successfully updated: "+cast.ToString(count))
}

// Upload import godoc
// @Summary Upload import excel
// @Description Upload import excel
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-import [POST]
func (h *HelperHandler) UploadImport(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	var store domain.Store
	err = h.db.First(&store, "store_code = ?", 4963).Error
	if err != nil {
		h.log.Warn("ERROR on getting store: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect import data
	newImport := domain.ImportRequest{
		Id:             uuid.New().String(),
		StoreID:        store.Id,
		Status:         constants.GeneralStatusNew,
		ImportDate:     "2025-05-08 17:10:00",
		DocumentNumber: "NP-2025050817",
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
	for _, row := range rows[1:] {

		// // create product id
		productID := uuid.New().String()
		// create or update product
		err = tx.Raw(`
			INSERT INTO products (material_code, name, barcode, is_marking)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (material_code) DO UPDATE
			SET
				producer_id = EXCLUDED.producer_id,
				mxik = EXCLUDED.mxik,
				is_marking = EXCLUDED.is_marking
			RETURNING id`,
			cast.ToInt(row[0]), row[1], row[2], cast.ToBool(row[5])).Scan(&productID).Error
		if err != nil {
			h.log.Warn("ERROR on creating new product: %v", err.Error())
			handleResponse(c, BadRequest, "Error on checking product data")
			tx.Rollback()
			return
		}
		// create import_detail
		var id string
		err = tx.Raw(`
			INSERT INTO import_details(
				product_id, import_id,
				received_count, scanned_count, supply_price, supply_price_vat,
				retail_price, retail_price_vat,
				vat, vat_sum, expire_date, series_number,
				sum_vat, marking) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
			productID, newImport.Id, 100, 100, 7030.37, 7900.00, 8928.57, 10000.00,
			12, 107143, "2025-12-31", "test", 1000000, utils.StringArray([]string{})).Scan(&id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "ERROR on creating import details")
			tx.Rollback()
			return
		}
	}
	// check transaction completed
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to commit transaction")
		tx.Rollback()
		return
	}
	handleResponse(c, OK, "Products Customer uploaded successfully: ")
}

// Get Product Picture From Tasnif godoc
// @Summary Get product picture from Tasnif
// @Description Get product picture from Tasnif
// @Tags helper
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	mxik_code query string true "MXIK code of the product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/picture [GET]
func (h *HelperHandler) GetProductPictureFromTasnif(c *gin.Context) {
	mxikCode := c.Query("mxik_code")

	client := &http.Client{
		Timeout: 30 * time.Second, // Set a timeout for the request
	}
	buf := bytes.Buffer{}

	url := "https://tasnif.soliq.uz/api/cls-api/integration-mxik/references/get/mxik/picture-names?lang=uz_latn&mxik_code="
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url+mxikCode, &buf)
	if err != nil {
		h.log.Warn("ERROR on creating new request: %v", err)
		handleResponse(c, InternalError, "Can't create new request")
		return
	}
	// add headers
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		h.log.Warn("ERROR on doing request: %v", err)
		handleResponse(c, InternalError, "Can't do request")
		return
	}
	defer resp.Body.Close()
	var res utils.StringArray
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		h.log.Warn("ERROR on decoding epos response %v", err)
		handleResponse(c, InternalError, "Can't decode response data")
		return
	}

	for _, v := range res {
		fullImageURL := "https://tasnif.soliq.uz/api/cls-api/integration-mxik/references/get/file/" + v
		imageURL := fullImageURL

		// GET so'rovi orqali rasmni olish
		imageResp, err := http.Get(imageURL)
		if err != nil {
			h.log.Warn("Failed to download image: %v", err)
			handleResponse(c, InternalError, "Failed to download image")
			return
		}
		defer imageResp.Body.Close()

		// Faylni saqlash pathi
		savePath := filepath.Join("./app/uploads", v)
		outFile, err := os.Create(savePath)
		if err != nil {
			h.log.Warn("Failed to create file: %v", err)
			handleResponse(c, InternalError, "Failed to create file")
			return
		}
		defer outFile.Close()

		// Rasmni faylga yozish
		_, err = io.Copy(outFile, imageResp.Body)
		if err != nil {
			h.log.Warn("Failed to save file: %v", err)
			handleResponse(c, InternalError, "Failed to write image to file")
			return
		}
	}

	if len(res) > 0 {
		err = h.db.Exec(`UPDATE products SET photos = ? WHERE mxik = ?`, utils.StringArray(res), mxikCode).Error
		if err != nil {
			h.log.Error("ERROR on saving product picture: %v", err)
			handleResponse(c, InternalError, "Failed to save product picture")
			return
		}
	}

	c.JSON(http.StatusOK, res)
}

// UploadProductMinMax godoc
// @Summary Upload package min, max count
// @Description Upload package min, max count
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/product-min-max [POST]
func (h *HelperHandler) UploadProductMinMax(c *gin.Context) {
	var (
		stores     []domain.Store
		products   []domain.Product
		storeMap   = make(map[string]any)
		productMap = make(map[string]any)
		file       domain.File
		err        error
	)

	// Bind uploaded file
	if err = c.ShouldBind(&file); err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Load stores
	err = h.db.Find(&stores).Error
	if err != nil {
		h.log.Warn("ERROR on getting store list: %v", err)
		handleResponse(c, InternalError, "failed.get.store_list")
		return
	}
	for _, st := range stores {
		storeMap[st.Name] = st.Id
	}

	// Load products
	err = h.db.Find(&products).Error
	if err != nil {
		h.log.Warn("ERROR on getting product list: %v", err)
		handleResponse(c, InternalError, "failed.get.product_list")
		return
	}

	for _, pr := range products {
		productMap[normalizeName(pr.Name)] = pr.Id
	}

	// Validate file extension
	ext := filepath.Ext(file.File.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		h.log.Error("Unsupported file format: ", ext)
		handleResponse(c, BadRequest, "Unsupported file format")
		return
	}

	// Save uploaded file
	newFilename := uuid.New().String() + ext
	savePath := filepath.Join("uploads", newFilename)
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}

	defer os.Remove(savePath)

	// Open Excel
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

	// SQL update
	query := `
		UPDATE store_product_thresholds
		SET min_quantity = ?, max_quantity = ?
		WHERE store_id = ? AND product_id = ?
	`
	// Skipped rows to return as JSON
	var skippedRows []map[string]any

	updated := 0
	for idx, row := range rows[1:] {
		rowNumber := idx + 2 // Excel row number (1-based + header)
		if len(row) < 6 {
			h.log.Warn("Row %d skipped, not enough columns", rowNumber)
			skippedRows = append(skippedRows, map[string]any{
				"row":    rowNumber,
				"reason": "not enough columns",
				"data":   row,
			})
			continue
		}

		storeName := strings.TrimSpace(row[0])
		productName := strings.TrimSpace(row[2])
		minQty := cast.ToFloat64(row[4])
		maxQty := cast.ToFloat64(row[5])
		productName = normalizeName(productName)
		storeID, storeOk := storeMap[storeName]
		productID, productOk := productMap[productName]
		if !storeOk || !productOk {
			h.log.Warn("Row %d skipped, store or product not found: %s / %s", rowNumber, storeName, productName)
			reason := "not found: "
			if !storeOk {
				reason += "store "
			}
			if !productOk {
				if !storeOk {
					reason += "and "
				}
				reason += "product"
			}
			skippedRows = append(skippedRows, map[string]any{
				"row":     rowNumber,
				"reason":  reason,
				"store":   storeName,
				"product": productName,
				"data":    row,
			})
			continue
		}

		err = h.db.Exec(query, minQty, maxQty, storeID, productID).Error
		if err != nil {
			h.log.Warn("Failed to update row %d: %v", rowNumber, err)
			skippedRows = append(skippedRows, map[string]any{
				"row":     rowNumber,
				"reason":  err.Error(),
				"store":   storeName,
				"product": productName,
				"data":    row,
			})
			continue
		}
		updated++
	}
	// Put skipped rows to file
	if len(skippedRows) > 0 {
		skippedFile := filepath.Join("uploads", fmt.Sprintf("skipped_rows_%s.json", time.Now().Format("20060102_150405")))
		skippedJSON, err := json.MarshalIndent(skippedRows, "", "\t")
		if err != nil {
			h.log.Error("Failed to marshal skipped rows: ", err)
			return
		}
		if err = os.WriteFile(skippedFile, skippedJSON, 0644); err != nil {
			h.log.Error("Failed to write skipped rows to file: ", err)
			return
		}
		h.log.Info("Skipped rows saved to file: %v", skippedFile)
	}

	// Return full result
	handleResponse(c, OK, gin.H{
		"updated": updated,
		"skipped": skippedRows,
	})
}
func normalizeName(s string) string {
	// Tashqi bo‘shliqlarni olib tashlash + ketma-ket bo‘shliqlarni 1 ta bo‘shliqqa qisqartirish
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// UploadProductMinMax godoc
// @Summary Upload package min, max count
// @Description Upload package min, max count
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/product-kvant [POST]
func (h *HelperHandler) UploadProductKvant(c *gin.Context) {
	var (
		stores     []domain.Store
		products   []domain.Product
		productMap = make(map[int]any)
	)
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.Find(&stores).Error
	if err != nil {
		h.log.Warn("ERROR on getting store list: %v", err)
		handleResponse(c, InternalError, "failed.get.store_list")
		return
	}

	err = h.db.Find(&products).Error
	if err != nil {
		h.log.Warn("ERROR on getting store list: %v", err)
		handleResponse(c, InternalError, "failed.get.store_list")
		return
	}

	for _, p := range products {
		productMap[p.MaterialCode] = p.Id
	}

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

	// defer os.Remove(savePath)
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

	// build query
	query := `
		INSERT INTO store_product_thresholds(store_id, product_id, kvant)
		VALUES (?, ?, ?)
		ON CONFLICT (store_id, product_id) DO UPDATE
		SET
			kvant = ?
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		for _, st := range stores {
			// create measurements
			err = h.db.Exec(query, st.Id, productMap[cast.ToInt(row[0])], cast.ToInt(row[2]), cast.ToInt(row[2])).Error
			if err != nil {
				h.log.Warn("ERROR on creating customers: %v", err)
			}
			count++
		}
	}
	handleResponse(c, OK, "Successfully updated: "+cast.ToString(count))

}

// SetProductPhotos godoc
// @Summary Set product photos
// @Description Set product photos
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/set-product-photo [POST]
func (h *HelperHandler) SetProductPhoto(c *gin.Context) {

	imageDir := "./app/uploads/images"
	targetDir := "./app/uploads"

	files, err := os.ReadDir(imageDir)
	if err != nil {
		panic(err)
	}

	var products []struct {
		Id   string `gorm:"id"`
		Name string `gorm:"name"`
	}

	err = h.db.Raw(`SELECT id, name FROM products`).Scan(&products).Error
	if err != nil {
		h.log.Error("could not get product list: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// Map for productID -> photos[]
	productPhotos := map[string]utils.StringArray{}
	var (
		matchCount   = 0
		noMatchCount = 0
	)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		normImageName := norm.NFC.String(normalizeFileName(f.Name()))
		normImageNameLower := strings.ToLower(normImageName)

		bestMatchID := ""
		bestScore := 999 // smaller is better (for Levenshtein)

		for _, p := range products {
			normProductName := norm.NFC.String(strings.ToLower(p.Name))

			// Option 1: if image name is contained in product name
			if strings.Contains(normProductName, normImageNameLower) {
				bestMatchID = p.Id
				bestScore = 0
				break
			}

			// Option 2: fuzzy match
			distance := levenshtein.ComputeDistance(normImageNameLower, normProductName)
			if distance < bestScore && distance <= 15 { // set a threshold
				bestMatchID = p.Id
				bestScore = distance
			}
		}

		if bestMatchID == "" {
			noMatchCount++
			fmt.Printf("❌ No match for image: %s\n", normImageName)
			continue
		}

		// Generate new filename and copy
		newFileName := uuid.New().String() + filepath.Ext(f.Name())
		srcPath := filepath.Join(imageDir, f.Name())
		dstPath := filepath.Join(targetDir, newFileName)

		if err := copyFile(srcPath, dstPath); err != nil {
			fmt.Printf("❌ Copy error: %v\n", err)
			continue
		}

		productPhotos[bestMatchID] = append(productPhotos[bestMatchID], newFileName)
		fmt.Printf("✅ Matched: %s → %s (product_id: %s)\n", f.Name(), newFileName, bestMatchID)
		matchCount++

	}

	// Save photos to DB
	for productID, photos := range productPhotos {
		err = h.db.Exec(`
			UPDATE products
			SET photos = ?, updated_at = now()
			WHERE id = ?
		`, utils.StringArray(photos), productID).Error
		if err != nil {
			fmt.Printf("❌ Failed to update product %s: %v\n", productID, err)
		} else {
			fmt.Printf("✅ Updated product %s with %d photos\n", productID, len(photos))
		}
	}

	handleResponse(c, OK, gin.H{
		"match":    matchCount,
		"no_match": noMatchCount,
	})
}

// FixProductQuantity godoc
// @Summary Fix product quantity
// @Description Fix product quantity
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/fix-product-quantity [POST]
func (h *HelperHandler) FixProductQuantity(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

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

	// defer os.Remove(savePath)
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

	// build query
	query := `
		UPDATE store_products SET pack_quantity = ?, unit_quantity = ?, updated_at = now() WHERE id = ?;
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		quantity := cast.ToInt(row[10])
		if quantity > 0 {
			unitPerPack := cast.ToInt(row[3])
			packQuantity := utils.NearestRound(float64(quantity) / float64(unitPerPack))
			err = h.db.Debug().Exec(query, packQuantity, quantity, row[0]).Error
			if err != nil {
				h.log.Error("could not update store_products(%s) -> %v", row[0], err)
			}
			count++
		}
	}

	handleResponse(c, OK, "Successfully updated: "+cast.ToString(count))

}

// UpdateProductInfo godoc
// @Summary update product infos
// @Description update product infos
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/update-product-info [POST]
func (h *HelperHandler) UpdateProductInfo(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

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

	// defer os.Remove(savePath)
	go h.processExcel(c, savePath)

	handleResponse(c, OK, "Successfully uploaded")
}

// UpdateSaleItems godoc
// @Summary update product infos
// @Description update product infos
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/update-wrong-items [POST]
func (h *HelperHandler) UpdateSaleItems(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

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

	// defer os.Remove(savePath)
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Errorf("Failed to open .xlsx file: %v", err)
		handleServiceResponse(c, BadRequest, domain.InternalServerError)
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Errorf("Failed to get rows: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// build query
	query := `
		UPDATE cart_items SET unit_quantity = ?, total_price = ?, updated_at = NOW() WHERE id = ?;
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 12 {
			quantity := cast.ToInt(row[4])
			price := cast.ToFloat64(row[5])
			err = h.db.Debug().Exec(query, quantity, float64(quantity)*price, row[0]).Error
			if err != nil {
				h.log.Errorf("could not update cart_item(%s) -> %v", row[0], err)
			}
			count++
		}
	}
	handleResponse(c, OK, fmt.Sprintf("%d - items update succesfully", count))
}

// AddCategories godoc
// @Summary add categories
// @Description add categories
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/add-categories [POST]
func (h *HelperHandler) AddCategories(c *gin.Context) {
	// read categories from json file
	filePath := "./app/uploads/categories.json"
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		h.log.Errorf("Failed to read categories file: %v", err)
		handleResponse(c, InternalError, "Failed to read categories file")
		return
	}

	var tmpCategories []struct {
		UId       string  `json:"uid"`
		ParentUId *string `json:"parentUID"`
		Name      string  `json:"name"`
		// NameUz    string  `json:"name_uz"`
		// NameEn    string  `json:"name_en"`
		// NameKr    string  `json:"name_kr"`
		Photo string `json:"photo"`
	}

	type Categories struct {
		Id         string  `json:"uid" gorm:"id"`
		CategoryId *string `json:"category_id" gorm:"category_id"`
		Name       string  `json:"name_ru" gorm:"name"`
		// NameUz     string  `json:"name_uz" gorm:"name_uz"`
		// NameKr     string  `json:"name_kr" gorm:"name_kr"`
		// NameEn     string  `json:"name_en" gorm:"name_en"`
		Photo string `json:"photo" gorm:"photo"`
	}

	err = json.Unmarshal(fileData, &tmpCategories)
	if err != nil {
		h.log.Errorf("Failed to unmarshal categories: %v", err)
		handleResponse(c, InternalError, "Failed to parse categories data")
		return
	}

	err = h.db.Exec("DELETE FROM categories WHERE name is not null;").Error
	if err != nil {
		h.log.Errorf("could not delete categories: %v", err)
		handleResponse(c, InternalError, "Failed to delete categories")
		return
	}

	var parentCategories []Categories
	var childCategories []Categories
	for _, cat := range tmpCategories {
		photoUrl, err := DownloadAndSaveImage(cat.Photo, "./app/uploads/")
		if err != nil {
			h.log.Errorf("download image from web: %v", err)
		}
		if cat.ParentUId == nil {
			parentCategories = append(parentCategories, Categories{
				Id:   cat.UId,
				Name: cat.Name,
				// NameUz: cat.NameUz,
				// NameKr: cat.NameKr,
				// NameEn: cat.NameEn,
				Photo: photoUrl,
			})
		}

		if cat.ParentUId != nil {
			childCategories = append(childCategories, Categories{
				Id:         cat.UId,
				CategoryId: cat.ParentUId,
				Name:       cat.Name,
				Photo:      photoUrl,
			})
		}

	}

	err = h.db.Table("categories").Create(&parentCategories).Error
	if err != nil {
		h.log.Errorf("Failed to insert parent categories: %v", err)
		handleResponse(c, InternalError, err)
		return
	}

	if len(childCategories) > 0 {
		err = h.db.Table("categories").Create(&childCategories).Error
		if err != nil {
			h.log.Errorf("Failed to insert child categories: %v", err)
			handleResponse(c, InternalError, err)
			return
		}
	}

	handleResponse(c, OK, fmt.Sprintf("Successfully added %d categories", len(parentCategories)+len(childCategories)))
}

// AttachCategoryToProducts godoc
// @Summary attach category to products
// @Description attach category to products
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/attach-category-to-products [POST]
func (h *HelperHandler) AttachCategoryToProducts(c *gin.Context) {
	// read categories from json file
	filePath := "./app/uploads/category_products.json"
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		h.log.Errorf("Failed to read categories file: %v", err)
		handleResponse(c, InternalError, "Failed to read categories file")
		return
	}

	var products []struct {
		Id          string `json:"id"`
		Name        string `json:"name"`
		CategoryUid string `json:"category_uid"`
	}

	err = json.Unmarshal(fileData, &products)
	if err != nil {
		h.log.Errorf("Failed to unmarshal categories: %v", err)
		handleResponse(c, InternalError, "Failed to parse categories data")
		return
	}

	// Update products with category IDs
	for _, prod := range products {
		err = h.db.Exec("UPDATE products SET category_id = ? WHERE id = ?", prod.CategoryUid, prod.Id).Error
		if err != nil {
			h.log.Errorf("Failed to update product %s: %v", prod.Id, err)
		}
	}

	handleResponse(c, OK, fmt.Sprintf("Successfully added %d products", len(products)))
}

// UploadCategoryJson godoc
// @Summary Upload Category Json
// @Description Upload Category Json
// @Tags helper
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Product file"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/upload-categories-json [POST]
func (h *HelperHandler) UploadCategoryJson(c *gin.Context) {
	var file domain.File

	// Bind the file data
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err)
		handleResponse(c, BadRequest, "Failed to bind file")
		return
	}

	// Check the file size (maximum 5 MB)
	maxFileSize := int64(5 * 1024 * 1024) // 5 MB
	if file.File.Size > maxFileSize {
		h.log.Error("File size exceeds the maximum limit of 5 MB")
		handleResponse(c, BadRequest, "File size exceeds the maximum limit of 5 MB")
		return
	}

	// Check file type
	ext := filepath.Ext(file.File.Filename)
	if ext != ".json" {
		h.log.Error("Invalid file type")
		handleResponse(c, UnprocessableEntity, "Invalid file type. Only .jpg, .jpeg, and .png files are allowed.")
		return
	}

	// Define the save path (adjust the directory as needed)
	savePath := filepath.Join("./app/uploads", file.File.Filename)

	// Remove the file after sending
	if err := os.Remove(savePath); err != nil {
		h.log.Error("Error deleting file after send: %v", err)
	}

	// Save the file
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Warn("Failed to save file: %v", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}

	// Return the file URL in the response
	c.JSON(http.StatusOK, gin.H{
		"file_url": file.File.Filename,
	})
}

// DeleteNotFoundPhotos godoc
// @Summary Upload Category Json
// @Description Upload Category Json
// @Tags helper
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/delete-not-photos [DELETE]
func (h *HelperHandler) DeleteNotFoundPhotos(c *gin.Context) {
	var products []struct {
		Id           string            `gorm:"id"`
		MaterialCode string            `gorm:"material_code"`
		Photos       utils.StringArray `gorm:"type:text[]"`
	}

	err := h.db.Raw("SELECT id, material_code, photos FROM products WHERE photos IS NOT NULL").Scan(&products).Error
	if err != nil {
		h.log.Errorf("could not get products: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	baseURl := h.cfg.FileBaseURL
	deletedCount := 0

	for _, p := range products {
		for _, photo := range p.Photos {

			resp, err := http.Get(baseURl + photo)
			if err != nil || resp.StatusCode != http.StatusOK {
				if err := h.db.Exec("UPDATE products SET photos = NULL WHERE id = ? AND photos IS NOT NULL", p.Id); err != nil {
					h.log.Errorf("could not update product %s photos to null: %v", p.Id, err)
				}
				// Fayl mavjud emas, uni o'chirish
				photoPath := filepath.Join("./app/uploads", photo)
				err = os.Remove(photoPath)
				if err != nil {
					h.log.Warn("could not delete photo %s: %v", photoPath, err)
				} else {
					deletedCount++
				}
			}

			photoPath := filepath.Join("./app/uploads", photo)
			if _, err := os.Stat(photoPath); os.IsNotExist(err) {
				if err := h.db.Exec("UPDATE products SET photos = NULL WHERE id = ? AND photos IS NOT NULL", p.Id); err != nil {
					h.log.Warn("could not update product %s photos to null: %v", p.Id, err)
				}
				// Fayl mavjud emas, uni o'chirish
				err := os.Remove(photoPath)
				if err != nil {
					h.log.Warn("could not delete photo %s: %v", photoPath, err)
				} else {
					deletedCount++
				}
			}
		}
	}

	handleResponse(c, OK, fmt.Sprintf("Successfully deleted %d photos", deletedCount))
}

func (h *HelperHandler) processExcel(c *gin.Context, savePath string) {
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Errorf("Failed to open .xlsx file: %v", err)
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(1)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Errorf("Failed to get rows: ", err)
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	// build query
	query := `
	UPDATE products
	SET 
		photos = ?, 
		updated_at = now()
	WHERE material_code = ?;
	`

	var (
		totalRowsProcessed = 0
		successImagesCount = 0
	)

	for _, row := range rows[1:] {
		if len(row) > 4 {
			// --- image handle ---
			var photos utils.StringArray
			if row[4] != "" {
				localPath, downErr := DownloadAndSaveImage(row[4], "uploads")
				if downErr != nil {
					h.log.Errorf("image download error for product %s: %v", row[0], downErr)
					continue // agar rasm yuklanmasa, shu rowni o‘tkazib yubor
				}

				if localPath != "" {
					photos = append(photos, localPath)
					// UPDATE faqat to'g'ri yuklangan rasm bo'lsa
					err = h.db.Debug().Exec(query,
						utils.StringArray(photos),
						cast.ToInt(row[0]),
					).Error
					if err != nil {
						h.log.Errorf("could not update product(%s) -> %v", row[0], err)
					} else {
						successImagesCount++
						totalRowsProcessed++
					}
				}
			}
		}
	}

	h.log.Info("Excel processing finished. Total rows processed: %d, Successful images downloaded: %d",
		totalRowsProcessed, successImagesCount)
}

func DownloadAndSaveImage(urlStr string, uploadDir string) (string, error) {
	if urlStr == "" {
		return "", nil
	}

	// Check if URL is valid
	parsedURL, err := url.Parse(urlStr)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("invalid URL: %s", urlStr)
	}

	ext := filepath.Ext(parsedURL.Path)
	if ext == "" || len(ext) > 5 {
		ext = ".png"
	}

	newImgName := uuid.New().String() + ext
	localPath := filepath.Join(uploadDir, newImgName)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create upload dir: %w", err)
		}
	}

	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return newImgName, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func normalizeFileName(fileName string) string {
	// Remove extension first
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Remove trailing _digit
	parts := strings.Split(name, "_")
	if len(parts) > 1 && isNumeric(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1] // remove last part
	}

	// Join with space
	return strings.Join(parts, " ")
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
