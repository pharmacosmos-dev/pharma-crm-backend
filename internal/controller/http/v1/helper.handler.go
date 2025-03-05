package v1

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
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
	INSERT INTO product_measurements (
			mxik_code, mxik_name_uz, unit_name, unit_code)
	VALUES (?, ?, ?, ?) ON CONFLICT (mxik_code) DO NOTHING;`

	// query1 := `
	// UPDATE product_measurements SET
	// 	mxik_name_ru = ?
	// WHERE mxik_code = ?;`

	// Process rows
	for i := len(rows) - 1; i >= 2; i-- {
		row := rows[i]
		if len(row) > 3 {
			// create measurements
			err = h.db.Debug().Exec(query, row[0], row[1], row[2], row[3]).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		}
	}

	handleResponse(c, OK, "Products MXIK CODE uploaded successfully")
}
