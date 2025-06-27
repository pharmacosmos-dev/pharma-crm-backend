package v1

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type WriteOffHandler struct {
	*Handler
}

func (h *Handler) NewWriteOffHandler(r *gin.RouterGroup) {
	writeOffHandler := &WriteOffHandler{h}
	writeOffHandler.WriteOffRoutes(r)
}

func (h *WriteOffHandler) WriteOffRoutes(r *gin.RouterGroup) {
	writeOff := r.Group("/write-off")
	{
		writeOff.POST("", h.Create)
		writeOff.GET("/:id", h.Get)
		writeOff.POST("/confirm/:id", h.Confirm)
		writeOff.POST("/cancel/:id", h.Cancel)
		writeOff.GET("/list", h.List)
		writeOff.GET("/list-status", h.WriteOffStatus)
		writeOff.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		writeOff.GET("/export-excel", h.ExportExcel)
	}
	detail := r.Group("write-off-detail")
	{
		detail.GET("/list", h.WriteOffDetailList)
		detail.GET("/export-excel", h.WriteOffDetailExportExcel)
	}

}

// Create godoc
// @Summary Create write-off
// @Description Create write-off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.WriteOffRequest true "WriteOff"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off [POST]
func (h *WriteOffHandler) Create(c *gin.Context) {
	var (
		body domain.WriteOffRequest
		err  error
	)
	// get user_id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found from the context")
		return
	}

	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Warn("ERROR on binding writeoff request: %v", err)
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// get created_by as a string
	body.CreatedBy = userId.(string)

	// create new write off
	err = h.service.CreateWriteOff(&body)
	if err != nil {
		h.log.Info("Failed to create new write off: %v", err)
		handleResponse(c, InternalError, "Can't create new write-off")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Write-Off
// @Description Get Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/{id} [GET]
func (h *WriteOffHandler) Get(c *gin.Context) {
	// get writeoff by id
	id := c.Param("id")
	// validate  uuid
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid write-off id")
		return
	}
	// get write-off info
	res, err := h.service.GetWriteOffById(id)
	if err != nil {
		h.log.Warn("Error on getting write-off: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get write-off")
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get WriteOff list
// @Description Get WriteOff list
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   status 	query string false "STATUS (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/list [GET]
func (h *WriteOffHandler) List(c *gin.Context) {
	var param domain.WriteOffParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.WriteOffList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, data)
}

// WriteOffStatus godoc
// @Summary Get Write-Off summary stats
// @Description Get total scanned count and retail price from write-offs
// @Tags Write-Off
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   store_id query string false "Store ID"
// @Param   search query string false "Search"
// @Param   status query string false "Status (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/list-status [get]
func (h *WriteOffHandler) WriteOffStatus(c *gin.Context) {
	var param domain.WriteOffParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	res, err := h.service.WriteOffStatus(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off summary")
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get Write Off export excel
// @Description Get Write Off export excel
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   status 	query string false "STATUS (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/export-excel [GET]
func (h *WriteOffHandler) ExportExcel(c *gin.Context) {
	var param domain.WriteOffParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.WriteOffList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off list")
		return
	}
	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "Магазин", "Кол-во", "Сумма по цене поставки", "Сумма по цене продажи", "Тип Списания", "Пользователь", "Статус", "Дата создание", "Дата завершения"}

	setExcelHeaders(f, sheetName, headers)

	// give width to column
	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "K", 15)

	// Ma'lumotlarni qo'shish
	for i, v := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, v.PublicId)
		f.SetCellValue(sheetName, "B"+row, v.Name)
		if v.Store != nil {
			f.SetCellValue(sheetName, "C"+row, v.Store.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		f.SetCellValue(sheetName, "D"+row, v.WriteoffCount)
		f.SetCellValue(sheetName, "E"+row, v.SupplyPriceSum)
		f.SetCellValue(sheetName, "F"+row, v.RetailPriceSum)
		f.SetCellValue(sheetName, "G"+row, v.Comment)
		if v.CreatedBy != nil {
			f.SetCellValue(sheetName, "H"+row, v.CreatedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "H"+row, "N/A")
		}
		f.SetCellValue(sheetName, "I"+row, helper.StatusToRussian(v.Status))
		f.SetCellValue(sheetName, "J"+row, v.CreatedAt.Format(time.DateTime))
		f.SetCellValue(sheetName, "K"+row, v.UpdatedAt.Format(time.DateTime))
	}

	saveExcelToUploads(c, f, *h.log, "Hisobdan_chiqarish")
}

// confirm Write-Off
// @Summary Confirm Write-Off
// @Description Confirm Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/confirm/{id} [POST]
func (h *WriteOffHandler) Confirm(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid write-off id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm write-off service
	err := h.service.ConfirmWriteOff(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm write-off")
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel Write-Off
// @Summary Cancel Write-Off
// @Description Cancel Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/cancel/{id} [POST]
func (h *WriteOffHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate write-off id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid write-off id")
		return
	}
	// get user_id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm write-off service
	err := h.service.CancelWriteOff(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm write-off")
		return
	}

	handleResponse(c, OK, "CANCELED")
}

// Get List
// @Summary Get Write-Off detail list
// @Description Get Write-Off detail list
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   writeoff_id query string true "WriteOff ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off-detail/list [GET]
func (h *WriteOffHandler) WriteOffDetailList(c *gin.Context) {
	var param domain.WriteOffDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get write-off detail list
	res, totalCount, err := h.service.WriteOffDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off detail list")
		return
	}
	// get write-off list _meta and pagination info
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Get List
// @Summary Get Write-Off detail list
// @Description Get Write-Off detail list
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   writeoff_id query string true "WriteOff ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off-detail/export-excel [GET]
func (h *WriteOffHandler) WriteOffDetailExportExcel(c *gin.Context) {
	var param domain.WriteOffDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get write-off detail list
	res, _, err := h.service.WriteOffDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off detail list")
		return
	}
	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Наименование", "Артикул", "Баркод", "Цена поставки", "Цена продажи", "Списание"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// give width to column
	f.SetColWidth(sheetName, "A", "F", 15)

	// Ma'lumotlarni qo'shish
	for i, v := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, v.Name)
		f.SetCellValue(sheetName, "B"+row, v.MaterialCode)
		f.SetCellValue(sheetName, "C"+row, v.Barcode)
		f.SetCellValue(sheetName, "D"+row, v.SupplyPriceVat)
		f.SetCellValue(sheetName, "E"+row, v.RetailPriceVat)
		f.SetCellValue(sheetName, "F"+row, v.ScannedCount)
	}

	saveExcelToUploads(c, f, *h.log, "Hisobdan_chiqarilgan_mahsulotlar")
}

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "WriteOff ID"
// @Param 	body body domain.InventoryAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/{id}/add-product-by-barcode [PATCH]
func (h *WriteOffHandler) AddProductByBarcode(c *gin.Context) {
	var (
		request    domain.WriteOffAddProduct
		writeOffID = c.Param("id")
	)

	// validate uuid
	if err := uuid.Validate(writeOffID); err != nil {
		handleResponse(c, BadRequest, "WriteOff id is invalid")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	// add scanned count by product barcode
	err = h.db.Exec(`
		UPDATE import_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE id = ? AND import_id = ?`,
		request.Count, request.Id, writeOffID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to add count")
		return
	}

	handleResponse(c, OK, "ADDED")
}
