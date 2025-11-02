package v1

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type ImportHandler struct {
	*Handler
}

func (h *Handler) NewImportHandler(r *gin.RouterGroup) {
	importHandler := &ImportHandler{h}
	importHandler.ImportRoutes(r)
}

func (h *ImportHandler) ImportRoutes(r *gin.RouterGroup) {
	imports := r.Group("/import")
	{
		imports.POST("", h.Create)
		imports.GET("/:id", h.Get)
		imports.GET("/list", h.List)
		imports.GET("/list-status", h.ListStatus)
		imports.GET("/export-excel", h.ExportImports)
	}
	importDetail := r.Group("/import-detail")
	{
		importDetail.POST("", h.CreateImportDetail)
		importDetail.GET("/list", h.ListImportDetail)
		importDetail.GET("/export-excel", h.ExportImporDetailExcel)
		importDetail.GET("/list/by-last-updated", h.ImportDetailListByLastUpdated)
		importDetail.PATCH("/add-scan", h.AddScann)
		importDetail.POST("/add-scan-by-id", h.AddAScanById)
		importDetail.PATCH("/accept-all/:id", h.AcceptImport)
		importDetail.PATCH("/cancel-all/:id", h.CancelImport)
		importDetail.PATCH("/accept-some/:id", h.AcceptSomeImport)
		importDetail.GET("/get-stock-status-counts/:id", h.GetStockStatusCounts)
		importDetail.PUT("/:id", h.UpdateImportDetail)
		// importDetail.GET("/product-marking/:id", h.ProductMarking)
	}
}

// Create godoc
// @Summary Create an import
// @Description Create an import from the request body
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.ImportRequest true "Import information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import [post]
func (h *ImportHandler) Create(c *gin.Context) {
	var (
		body domain.ImportRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// First godoc
// @Summary First imports
// @Description First imports
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/{id} [get]
func (h *ImportHandler) Get(c *gin.Context) {
	var (
		res domain.Import
		id  = c.Param("id")
	)
	err := h.db.First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Errorf("could not get import by Id: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary List imports
// @Description List imports
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Param   receive_amount_from 	query int false "Receive Amount From"
// @Param   receive_amount_to 	query int false "Receive Amount To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/list [get]
func (h *ImportHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ImportQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// Get import list data
	res, totalCount, err := h.service.GetImports(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// ListStatus godoc
// @Summary Get import status stats
// @Description Get aggregated import stats grouped by status
// @Tags imports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   status query string false "Status"
// @Param   receive_amount_from query int false "Receive Amount From"
// @Param   receive_amount_to query int false "Receive Amount To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/list-status [get]
func (h *ImportHandler) ListStatus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ImportQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetImportsStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, result)
}

// Export import excel godoc
// @Summary Export import excel
// @Description Export import excel
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Param   receive_amount_from 	query int false "Receive Amount From"
// @Param   receive_amount_to 	query int false "Receive Amount To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/export-excel [get]
func (h *ImportHandler) ExportImports(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ImportQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// Get import list data
	res, _, err := h.service.GetImports(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Импорный номер", "Номер документа", "Филиал", "Дата создания", "Дата закрытия", "Полученное количество", "Полученная сумма СНДС", "Принятое количество", "Принятая сумма СНДС", "Статус"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("could not create imports excel style: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.PublicId)
		f.SetCellValue(sheetName, "B"+row, imp.DocumentNumber)
		if imp.Store.Valid {
			f.SetCellValue(sheetName, "C"+row, imp.Store.Value.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}

		f.SetCellValue(sheetName, "D"+row, imp.ImportDate.Format(time.DateOnly))
		f.SetCellValue(sheetName, "E"+row, imp.UpdatedAt.Format(time.DateOnly))
		f.SetCellValue(sheetName, "F"+row, imp.ReceivedCount)
		f.SetCellValue(sheetName, "G"+row, imp.ReceivedAmountVat)
		f.SetCellValue(sheetName, "H"+row, imp.AcceptedCount)
		f.SetCellValue(sheetName, "I"+row, imp.AcceptedAmountVat)
		f.SetCellValue(sheetName, "J"+row, helper.StatusToRussian(imp.Status))

	}
	saveExcelToUploads(c, f, *h.log, "imports")
}

// Create godoc
// @Summary Create an import detail
// @Description Create an import detail from the request body
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.ImportDetailRequest true "Import detail information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail [post]
func (h *ImportHandler) CreateImportDetail(c *gin.Context) {
	var (
		body domain.ImportDetailRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Table("import_details").Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// ListImportDetail godoc
// @Summary List import details
// @Description List import details
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   import_id query string true "Import ID"
// @Param   received_amount_from query int false "Received Amount From"
// @Param   received_amount_to query int false "Received Amount To"
// @Param   no_barcode query bool false "Filter items with no barcode (true/false)"
// @Param   no_marking query bool false "Filter items with no marking (true/false)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/list [get]
func (h *ImportHandler) ListImportDetail(c *gin.Context) {
	var (
		importDetails []domain.ImportDetail
		totalCount    int64
		param         domain.ImportQueryParams
		err           error
	)

	// Bind query parameters
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Get pagination parameters
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// Get import detail list data
	importDetails, totalCount, err = h.service.ListImportDetail(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare response
	data := utils.ListResponse(importDetails, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, data)
}

// Export ImportDetail excel godoc
// @Summary Export ImportDetail excel
// @Description Export ImportDetail excel
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   import_id query string true "Import ID"
// @Param   received_amount_from query int false "Received Amount From"
// @Param   received_amount_to query int false "Received Amount To"
// @Param   no_barcode query bool false "Filter items with no barcode (true/false)"
// @Param   no_marking query bool false "Filter items with no marking (true/false)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/export-excel [get]
func (h *ImportHandler) ExportImporDetailExcel(c *gin.Context) {
	var (
		importDetails []domain.ImportDetail
		param         domain.ImportQueryParams
		err           error
	)

	// Bind query parameters
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Get pagination parameters
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// Get import detail list data
	importDetails, _, err = h.service.ListImportDetail(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Артикул", "Название", "Штрих-Код", "Цена Поставки", "Цена Поставки СНДС", "Цена Продажа", "Цена Продажа СНДС", "Статус", "Полученное количество", "Принятое количество", "Полученная сумма", "Принятая сумма", "Полученная сумма СНДС", "Принятая сумма СНДС", "Дата создания"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range importDetails {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Product.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, imp.Product.Name)
		f.SetCellValue(sheetName, "C"+row, imp.Product.Barcode)
		f.SetCellValue(sheetName, "D"+row, imp.SupplyPrice)
		f.SetCellValue(sheetName, "E"+row, imp.SupplyPriceVat)
		f.SetCellValue(sheetName, "F"+row, imp.RetailPrice)
		f.SetCellValue(sheetName, "G"+row, imp.RetailPriceVat)
		f.SetCellValue(sheetName, "H"+row, helper.StatusToRussian(imp.Import.Status))
		f.SetCellValue(sheetName, "I"+row, imp.ReceivedCount)
		f.SetCellValue(sheetName, "J"+row, imp.AcceptedCount)
		f.SetCellValue(sheetName, "K"+row, imp.ReceivedAmount)
		f.SetCellValue(sheetName, "L"+row, imp.AcceptedAmount)
		f.SetCellValue(sheetName, "M"+row, imp.ReceivedAmountVat)
		f.SetCellValue(sheetName, "N"+row, imp.AcceptedAmountVat)
		f.SetCellValue(sheetName, "O"+row, imp.CreatedAt.Format(time.DateTime))
	}

	saveExcelToUploads(c, f, *h.log, "import_details")
}

// ListImportDetail Scan list godoc
// @Summary List import details order by last update
// @Description List import details order by last update
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   import_id query string true "Import ID"
// @Param   received_amount_from query int false "Received Amount From"
// @Param   received_amount_to query int false "Received Amount To"
// @Param   type query string false "Type"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/list/by-last-updated [get]
func (h *ImportHandler) ImportDetailListByLastUpdated(c *gin.Context) {
	var (
		importDetails []domain.ImportDetail
		totalCount    int64
		err           error
	)

	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Get import detail list data
	importDetails, totalCount, err = h.service.ListImportDetailByLastUpdated(c, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare response
	data := map[string]interface{}{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     limit,
			CurrentPage: (offset / limit) + 1,
			PageCount:   int((totalCount + int64(limit) - 1) / int64(limit)),
		},
		"data":  importDetails,
		"stats": gin.H{},
	}

	handleResponse(c, OK, data)
}

// AddScann godoc
// @Summary Add scan to import detail
// @Description Add scan to import detail
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.AddScanRequest true "Add scan information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/add-scan [PATCH]
func (h *ImportHandler) AddScann(c *gin.Context) {
	var (
		body          domain.AddScanRequest
		surplus       = false
		importDetails []domain.ImportDetail
	)
	// Bind the JSON body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// validate barcode
	if body.Barcode == "" {
		handleResponse(c, BadRequest, "Barcode is required")
		return
	}

	err := h.db.Model(&domain.ImportDetail{}).
		Preload("Product").
		Preload("Import").
		Select(`
		import_details.*,
		(import_details.retail_price*received_count) as received_amount,
		(import_details.retail_price*accepted_count) as accepted_amount,
		sum_vat as received_amount_vat,
		(import_details.retail_price_vat*accepted_count) as accepted_amount_vat,
		COALESCE(unit_types.short_name, '') as unit_name`).
		Joins("JOIN products ON import_details.product_id = products.id").
		Joins("LEFT JOIN unit_types ON products.unit_type_id = unit_types.id").
		Where("import_id = ? AND products.barcode = ?", body.ImportID, body.Barcode).
		Order("products.name").
		Find(&importDetails).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if len(importDetails) > 1 {
		handleResponse(c, PartialContent, importDetails)
		return
	}

	// Check if the count is valid
	if body.Count < 1 {
		body.Count = 1
	}
	var importDetail domain.ImportDetail
	// Perform a single query to find and update the record
	result := h.db.Raw(`
	UPDATE import_details SET
		accepted_count = accepted_count + ?, updated_at = NOW()
	WHERE import_id = ? AND product_id IN (
		SELECT id
		FROM products
		WHERE barcode = ?
	)
	`, body.Count, body.ImportID, body.Barcode).Scan(&importDetail)

	if result.RowsAffected == 0 {
		handleResponse(c, NotFound, "Product not found")
		return
	}
	// Check if the record was updated
	if result.Error != nil {
		h.log.Error("Error on updating accepted_count: %v", result.Error)
		handleResponse(c, InternalError, result.Error.Error())
		return
	}

	if importDetail.AcceptedCount > importDetail.ReceivedCount {
		surplus = true
	}
	handleResponse(c, OK, map[string]interface{}{
		"surplus": surplus,
	})
}

// AddScannById godoc
// @Summary Add scan by import detail id
// @Description Add scan by import detail id
// @Tags import_details
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body domain.AddScanRequest true "Add scan information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/add-scan-by-id [POST]
func (h *ImportHandler) AddAScanById(c *gin.Context) {
	var (
		body         domain.AddScanRequest
		importDetail domain.ImportDetail
		err          error
		surplus      = false
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate id
	if err = uuid.Validate(body.ID); err != nil {
		handleResponse(c, BadRequest, "Invalid import detail id")
		return
	}
	// Check if the count is valid
	if body.Count < 1 {
		body.Count = 1
	}
	//
	err = h.db.Raw(`
		UPDATE import_details
		SET accepted_count = accepted_count + ?
		WHERE id = ? RETURNING *
	`, body.Count, body.ID).Scan(&importDetail).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check if there is a surplus
	if importDetail.AcceptedCount > importDetail.ReceivedCount {
		surplus = true
	}
	handleResponse(c, OK, map[string]interface{}{
		"surplus": surplus,
	})
}

// UpdateImportDetail
// @Summary Update an import detail
// @Description Update an import detail from the request body
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "import detail ID"
// @Param input body domain.ImportUpdateRequest true "Import detail information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/{id} [PUT]
func (h *ImportHandler) UpdateImportDetail(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.ImportUpdateRequest
		err  error
	)
	// validate uuid
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// update scanned_count
	err = h.db.
		WithContext(c.Request.Context()).
		Table("import_details").
		Where("id = ?", id).
		Update("scanned_count", body.ScannedCount).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// AcceptImport
// @Summary Accept import
// @Description Accept import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/accept-all/{id} [patch]
func (h *ImportHandler) AcceptImport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	importId := c.Param("id")
	if err := uuid.Validate(importId); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// lock parallel request
	mu := h.getImportLock(importId)
	mu.Lock()
	defer mu.Unlock()

	// update imports status to completed
	err := h.service.AcceptImport(ctx, importId, user.UserId, "all")
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// CancelImport
// @Summary Cancel import
// @Description Cancel import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/cancel-all/{id} [patch]
func (h *ImportHandler) CancelImport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var id = c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// lock parallel request
	mu := h.getImportLock(id)
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// update import status to cancel
	err := h.service.CancelImport(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// AcceptSomeImport
// @Summary Accept import
// @Description Accept import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/accept-some/{id} [patch]
func (h *ImportHandler) AcceptSomeImport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var id = c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid import id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// lock parallel request
	mu := h.getImportLock(id)
	mu.Lock()
	defer mu.Unlock()

	// update import status to completed
	err := h.service.AcceptImport(ctx, id, user.UserId, "some")
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// GetStockStatusCounts
// @Summary Get stock status counts
// @Description Get stock status counts
// @Tags import_details
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/get-stock-status-counts/{id} [get]
func (h *ImportHandler) GetStockStatusCounts(c *gin.Context) {
	var id = c.Param("id")
	var res domain.StockCountResponse
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid import id")
		return
	}
	// Use raw SQL to calculate the counts with surplus condition
	query := `
		SELECT
			COALESCE(SUM(accepted_count), 0) AS scanned_count,
			COALESCE(SUM(received_count - accepted_count), 0) AS shortage_count,
			COALESCE(SUM(received_count), 0) AS total_count,
			COALESCE(SUM(CASE WHEN accepted_count > received_count THEN accepted_count - received_count ELSE 0 END), 0) AS surplus_count
		FROM import_details
		WHERE import_id = ?
	`
	err := h.db.
		Raw(query, id).
		Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// lock order for parallel request
func (h *ImportHandler) getImportLock(importId string) *sync.Mutex {
	lock, _ := h.ordersToMutexes.LoadOrStore(importId, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
