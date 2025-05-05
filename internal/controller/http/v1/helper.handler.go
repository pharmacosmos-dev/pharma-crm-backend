package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
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
		helper.POST("/upload-package-code", h.UploadPackageCodeExcel)
		helper.POST("/upload-unit-count", h.UploadProductUnitCount)
		helper.POST("/upload-mxik", h.CorrectMXIK)
		helper.POST("/epos", h.EposTransmitter)
		helper.POST("/upload-category", h.UploadCategory)
		helper.POST("/upload-customer", h.UploadCustomer)
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
// @Router /helper/upload-package-code [POST]
func (h *HelperHandler) UploadPackageCodeExcel(c *gin.Context) {
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
	UPDATE products SET unit_code = ?, unit_label = ? WHERE material_code = ? AND mxik = ? AND (unit_code is null OR unit_code = '')
	`

	// query1 := `
	// UPDATE product_measurements SET
	// 	mxik_name_ru = ?
	// WHERE mxik_code = ?;`
	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		fmt.Println("KOD: ", row[0], "IKPU: ", row[2], "UKOD: ", row[3], "UName: ", row[4])
		count++
		err := h.db.Exec(query, row[3], row[4], cast.ToInt(row[0]), row[2]).Error
		if err != nil {
			h.log.Warn("ERROR on updating products: %v", err)
		}
	}
	fmt.Println("COUNT: ", count)
	handleResponse(c, OK, "Products MXIK CODE uploaded successfully")
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
				err = h.db.Debug().Exec(query, row[4], true, parseIntComma(row[1])).Error
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

	client := &http.Client{}
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
			err = h.db.Debug().Exec(query, row[0], row[1]).Error
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
		err = h.db.Debug().Exec(query, customer.Id, customer.FullName, customer.Phone, customer.Birthday).Error
		if err != nil {
			h.log.Warn("ERROR on creating customers: %v", err)
		}
		err = h.db.Debug().Exec(queryd, customer.Id, row[5], row[8]).Error
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
	query := `
		update products SET unit_per_pack = ? WHERE material_code = ? AND unit_per_pack = 0
	`

	var count = 0

	// Process rows
	for _, row := range rows[:] {
		count++
		fmt.Println("==>> ", row[0], row[2])
		// // create measurements
		err = h.db.Debug().Exec(query, cast.ToInt(row[2]), cast.ToInt(row[0])).Error
		if err != nil {
			h.log.Warn("ERROR on creating customers: %v", err)
		}

	}
	fmt.Println("---->>> ", count)
	handleResponse(c, OK, "Products Customer uploaded successfully: ")
}
