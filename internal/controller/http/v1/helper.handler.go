package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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

	url := h.cfg.SoliqIkpuBaseUrl + "/web-katalog"
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
		h.log.Error("Request yaratishda xatolik: %w", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// HTTP client yaratish va timeout o‘rnatish
	client := &http.Client{Timeout: 1 * time.Minute}

	// So‘rovni jo‘natish
	resp, err := client.Do(req)
	if err != nil {
		h.log.Error("So‘rov jo‘natishda xatolik: %w", err)
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
	fmt.Println("COUNT: ", count)
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
	UPDATE products SET mxik = ?, is_marking = ? WHERE material_code = ?
	`
	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 8 {
			if row[5] != "" && row[5] != "#N/A" && row[2] == "Да" {
				count++
				fmt.Println("ID: ", parseIntComma(row[1]), "Marking: ", row[2], "IKPU: ", row[4], "OLD IKPU: ", row[5])
				// // create measurements
				err = h.db.Exec(query, row[4], true, parseIntComma(row[1])).Error
				if err != nil {
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
			}
		}

	}
	fmt.Println("---=>>> ", count)
	handleResponse(c, OK, "Products MXIK CODE uploaded successfully: ")
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
			fmt.Println("ID: ", row[0], "Category: ", row[1])
			// // create measurements
			err = h.db.Exec(query, row[0], row[1]).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
	}
	fmt.Println("---->>> ", count)
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
		fmt.Println("FULLNAME: ", row[2], "PHONE: ", cast.ToString(row[3]), "DATE: ", cast.ToString(row[4]))
		fmt.Println("BARCODE: ", row[5], "PERCENT: ", row[8])
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
	fmt.Println("---->>> ", count)
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
	// query := `
	// 	UPDATE products SET unit_code = ?, unit_label = ? WHERE material_code = ?;
	// `

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if count <= 247 {
			fmt.Println("UCODE: ", row[4], "UNAME: ", row[7])
			// // create measurements
			// err = h.db.Debug().Exec(query, row[4], row[7], cast.ToInt(row[0])).Error
			// if err != nil {
			// 	h.log.Warn("ERROR on creating customers: %v", err)
			// }
			count++
		}
	}
	fmt.Println("---->>> ", count)
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
		Status:         config.NEW_IMPORT,
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
		fmt.Println("KOD:", row[0], "BARCODE: ", row[2], "MAR: ", row[5])

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
	fmt.Println("RESPONSE: ", res)

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
		err = h.db.Debug().Exec(`UPDATE products SET photos = ? WHERE mxik = ?`, utils.StringArray(res), mxikCode).Error
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
		storeMap   = make(map[int]any)
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
	for _, st := range stores {
		storeMap[st.StoreCode] = st.Id
	}

	err = h.db.Find(&products).Error
	if err != nil {
		h.log.Warn("ERROR on getting store list: %v", err)
		handleResponse(c, InternalError, "failed.get.store_list")
		return
	}

	for _, pr := range products {
		productMap[pr.MaterialCode] = pr.Id
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
		INSERT INTO store_product_thresholds(store_id, product_id, kvant, min_quantity, max_quantity)
		VALUES (?, ?, ?, ?, ?)
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		// fmt.Println("StoreID: ", cast.ToInt(row[0]), "ProductID: ", cast.ToString(row[2]))
		// fmt.Println("KVANT: ", row[4], "MIN: ", row[5], "Max: ", row[6])
		// // create measurements
		err = h.db.Debug().Exec(query, storeMap[cast.ToInt(row[0])], productMap[cast.ToInt(row[2])], cast.ToInt(row[4]), cast.ToInt(row[5]), cast.ToInt(row[6])).Error
		if err != nil {
			h.log.Warn("ERROR on creating customers: %v", err)
		}
		count++
	}
	fmt.Println("---->>> ", count)
	handleResponse(c, OK, "Successfully updated: "+cast.ToString(count))

}
