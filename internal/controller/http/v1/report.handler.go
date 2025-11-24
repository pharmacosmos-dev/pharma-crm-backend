package v1

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type ReportHandler struct {
	*Handler
}

func (h *Handler) NewReportHandler(r *gin.RouterGroup) {
	report := &ReportHandler{h}
	report.ReportRoutes(r)
}

func (h *ReportHandler) ReportRoutes(r *gin.RouterGroup) {
	report := r.Group("/report")
	{
		report.POST("/product", h.GetProductsReport)
		report.POST("/product-status", h.GetProductsReportStats)
		report.POST("/product-export", h.GetProductsReportExcel)
		report.POST("/product-by-date", h.ProductReportByDate)
		report.POST("/product-by-date/export", h.ProductByDateExport)
		report.POST("/store-amount", h.StoreReportAmount)
		report.POST("/store-amount/export-excel", h.StoreReportAmountExport)
		report.POST("/store-stats", h.StoreReportStats)
		report.POST("/bonus", h.BonusReport)
		report.POST("/bonus-export", h.BonusReportExport)
		report.POST("/lfl", h.LflReport)
		report.POST("/top-products", h.ReportTopProducts)
		report.POST("/top-products/export-excel", h.TopProductsExportExcel)
		report.POST("/top-seller", h.ReportTopSeller)
		report.POST("/top-seller/export-excel", h.TopSellerExcel)
		report.POST("/top-stores", h.ReportTopStores)
		report.POST("/top-stores/export-excel", h.TopStoresExcel)
		report.POST("/bonus-products", h.ReportBonusProducts)
		report.POST("/bonus-products-stats", h.ReportBonusProductsStats)
		report.POST("/bonus-items", h.FetchBonusItemsByEmployee)
		report.POST("/bonus-items-export", h.FetchBonusItemsByEmployeeExport)
		report.POST("/bonus-products/export-excel", h.BonusProductsExportExcel)
		report.POST("/store-summary", h.ReportStoreSummary)
		report.POST("/store-summary-stats", h.ReportStoreSummaryStats)
		report.POST("/store-summary/export-excel", h.StoreSummaryExportExcel)
		report.POST("/store-products-given-day", h.StoreProductsGivenDay)
		report.POST("/store-products-given-day/export-excel", h.StoreProductsGivenDayExportExcel)
		report.POST("/discount-card", h.DiscountCardReport)
	}
}

// ListImportDetail godoc
// @Summary List import details
// @Description List import details
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   producer_id query string false "Producer ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-by-date [POST]
func (h *ReportHandler) ProductReportByDate(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.ReportQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}

	// Bind JSON body (store_ids)
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	res, err := h.service.ProductReportWithDate(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// ListImportDetail godoc
// @Summary List import details
// @Description List import details
// @Tags Report
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit 		query int 	false "Limit"
// @Param 	offset 		query int 	false "Offset"
// @Param   search 		query string false "Search"
// @Param   start_date 	query string false "start_date"
// @Param   end_date 	query string false "end_date"
// @Param   producer_id query string false "producer_id"
// @Param   store_ids 	body  []string false "store_ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-by-date/export [POST]
func (h *ReportHandler) ProductByDateExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.ReportQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}

	// Bind JSON body (store_ids)
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	res, err := h.service.ProductReportWithDate(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Excel file
	f := excelize.NewFile()
	sheetName := "Товары отчет"
	f.SetSheetName("Sheet1", sheetName)

	// StartDate va EndDate oralig'idagi sanalarni olish
	startDate, _ := time.Parse("2006-01-02", params.StartDate)
	endDate, _ := time.Parse("2006-01-02", params.EndDate)

	// Dinamik headerlar tayyorlash
	headers := []string{"Названия товаров"}
	var dates []string
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		headers = append(headers, day)
		dates = append(dates, day)
	}
	headers = append(headers, "Общий итог")

	// Headerlarga style berish
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "000000",
		},
	})
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Headerlarni yozish
	for i, hText := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheetName, cell, hText)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	// give width to column
	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "J", 15)
	// Ma'lumotlarni yozish
	for i, rowData := range res {
		rowIndex := i + 2 // Excel starts from 1
		// A: product_name
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), rowData["product_name"])

		// B...: date columns
		for j, dateStr := range dates {
			col, _ := excelize.ColumnNumberToName(j + 2) // B is column 2
			cell := fmt.Sprintf("%s%d", col, rowIndex)
			f.SetCellValue(sheetName, cell, rowData[dateStr])
		}

		// Last column: total_amount
		lastCol, _ := excelize.ColumnNumberToName(len(dates) + 2)
		cell := fmt.Sprintf("%s%d", lastCol, rowIndex)
		f.SetCellValue(sheetName, cell, rowData["total_amount"])
	}

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Tovar_kunlik_hisboti_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
	filePath := filepath.Join("uploads", fileName)

	// uploads/ papkasi mavjud bo‘lmasa, yaratish
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		err := os.Mkdir("uploads", os.ModePerm)
		if err != nil {
			h.log.Error("Failed to create uploads directory:", err)
			handleResponse(c, InternalError, "Failed to create uploads folder")
			return
		}
	}

	// Faylni diskka yozish
	if err := f.SaveAs(filePath); err != nil {
		h.log.Error("Failed to save Excel file:", err)
		handleResponse(c, InternalError, "Failed to save Excel file")
		return
	}

	// Foydalanuvchiga file path yoki URLni qaytarish
	handleResponse(c, OK, gin.H{
		"file_name": fileName,
	})

}

// Bonus report godoc
// @Summary Get employee bonus report
// @Description Get employee bonus report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search"
// @Param   store_ids body []string false "Store ids"
// @Param   order query string false "Order type: max_count, min_count, max_amount, min_amount" Enums(max_count, min_count, max_amount, min_amount)
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus [POST]
func (h *ReportHandler) BonusReport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReportQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get default limit and offset for pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get bonus reports
	res, totalCount, err := h.service.BonusReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	// get data with _meta pagination info
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Bonus report godoc
// @Summary Get employee bonus report
// @Description Get employee bonus report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search"
// @Param   store_ids body []string false "Store ids"
// @Param   order query string false "Order type: max_count, min_count, max_amount, min_amount" Enums(max_count, min_count, max_amount, min_amount)
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-export [POST]
func (h *ReportHandler) BonusReportExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get default limit and offset for pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get bonus reports
	res, _, err := h.service.BonusReport(ctx, &params)
	if err != nil {
		handleResponse(c, InternalError, "Can't get bonus report")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Ф.И.O", "Магазин", "Телефон", "Рол", "Сумма бонуса", "Кол-во продаж"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}
	// set column width
	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "A", "E", 20)
	f.SetColWidth(sheetName, "F", "G", 15)

	// Set information to columns
	for i, value := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, value.PublicId)
		f.SetCellValue(sheetName, "B"+row, value.FullName)
		f.SetCellValue(sheetName, "C"+row, value.StoreName)
		f.SetCellValue(sheetName, "D"+row, value.Phone)
		f.SetCellValue(sheetName, "E"+row, value.Role)
		f.SetCellValue(sheetName, "F"+row, value.Amount)
		f.SetCellValue(sheetName, "G"+row, value.Count)

	}

	saveExcelToUploads(c, f, *h.log, "Xodimlar_bonuslari")
}

// Bonus report godoc
// @Summary Get employee bonus report
// @Description Get employee bonus report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search"
// @Param   employee_id query string false "Employee Id"
// @Param   producer_id query string false "Producer ID"
// @Param   order query string false "Order by field: e.g. -product_name, +store_name, +expire_date, -retail_price_sum"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product [POST]
func (h *ReportHandler) GetProductsReport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.ContextTimeoutForReports)
	defer cancel()

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get default limit and offset for pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, totalCount, err := h.service.GetProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// @Summary Get product status report
// @Description Get total quantity and price stats for sales and returns
// @Tags Report
// @Security BearerAuth
// @Accept 	json
// @Produce json
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search"
// @Param   employee_id query string false "Employee Id"
// @Param   producer_id query string false "Producer ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response{data=domain.ProductStatusReport}
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-status [POST]
func (h *ReportHandler) GetProductsReportStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.ContextTimeoutForReports)
	defer cancel()

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	// parse store_ids from body
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, err := h.service.GetProductsReportStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// Bonus report godoc
// @Summary Get employee bonus report
// @Description Get employee bonus report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search"
// @Param   employee_id query string false "Employee Id"
// @Param   producer_id query string false "Producer ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-export [POST]
func (h *ReportHandler) GetProductsReportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.ContextTimeoutForReports)
	defer cancel()

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get default limit and offset for pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, _, err := h.service.GetProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Филиал", "Наименование", "Производитель", "Серия", "Срок Годности", "Кол-во", "Цена прихода", "Цена продажная", "Сумма прихода", "Сумма продажная", "Сумма наценки", "Сумма НДС", "Дата продажи", "Время продажи", "Пользователь", "ID ЧЕКА", "МК кол-во"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Set information to columns
	for i, value := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, value.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, value.StoreName)
		f.SetCellValue(sheetName, "C"+row, value.ProductName)
		f.SetCellValue(sheetName, "D"+row, value.ProducerName)
		f.SetCellValue(sheetName, "E"+row, value.SerialNumber)
		if value.ExpireDate != nil {
			f.SetCellValue(sheetName, "F"+row, value.ExpireDate.Format(time.DateOnly))
		} else {
			f.SetCellValue(sheetName, "F"+row, "N/A")
		}
		f.SetCellValue(sheetName, "G"+row, math.Round((float64(value.UnitQuantity)/float64(value.UnitPerPack))*100)/100)
		f.SetCellValue(sheetName, "H"+row, value.SupplyPrice)
		f.SetCellValue(sheetName, "I"+row, value.RetailPrice)
		f.SetCellValue(sheetName, "J"+row, value.SupplyPriceSum)
		f.SetCellValue(sheetName, "K"+row, value.RetailPriceSum)
		f.SetCellValue(sheetName, "L"+row, value.MarkupSum)
		f.SetCellValue(sheetName, "M"+row, value.VatSum)
		f.SetCellValue(sheetName, "N"+row, value.CompletedAt.Add(time.Hour*5).Format(time.DateOnly))
		f.SetCellValue(sheetName, "O"+row, value.CompletedAt.Add(time.Hour*5).Format(time.TimeOnly))
		f.SetCellValue(sheetName, "P"+row, value.FullName)
		f.SetCellValue(sheetName, "Q"+row, helper.SaleTypeToRussian(value.SaleType, value.SaleNumber))
		f.SetCellValue(sheetName, "R"+row, value.MarkingCount)
	}

	saveExcelToUploads(c, f, *h.log, "Sale_details")
}

// Bonus report godoc
// @Summary Get employee bonus report
// @Description Get employee bonus report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date Format(2025-03)"
// @Param   end_date query string false "End Date Format(2025-04)"
// @Param   search query string false "Search"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/lfl [POST]
func (h *ReportHandler) LflReport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		// bind store_ids
		_ = c.ShouldBindJSON(&params.StoreIds)

	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get default limit and offset for pagination
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, _, err := h.service.LflReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Bonus report godoc
// @Summary Get Store report Amount
// @Description Get Store report Amount
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date Format(2025-03)"
// @Param   end_date query string false "End Date Format(2025-04)"
// @Param   search query string false "Search"
// @Param   order query string false "Order: +store_code, -store_name, -sale_date, +uzcard etc."
// @Param   store_id query string false "Store ID"
// @Param   body body domain.DashboardBody false "ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-amount [POST]
func (h *ReportHandler) StoreReportAmount(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	params.StoreIds = body.StoreIds
	params.CompanyIds = body.CompanyIds

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyIds = []string{user.CompanyId}
	}
	// get store report with payment type amounts
	res, totalCount, err := h.service.GetStoreAmountReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Bonus report godoc
// @Summary Get Store report Amount
// @Description Get Store report Amount
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date Format(2025-03)"
// @Param   end_date query string false "End Date Format(2025-04)"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   body body domain.DashboardBody false "ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-amount/export-excel [POST]
func (h *ReportHandler) StoreReportAmountExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	params.StoreIds = body.StoreIds
	params.CompanyIds = body.CompanyIds

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeoutForReports)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyIds = []string{user.CompanyId}
	}
	// get store report with payment type amounts
	res, _, err := h.service.GetStoreAmountReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "АПТЕКА", "ДАТА", "НАЛИЧНЫЕ", "HUMO", "UZCARD", "CLICK", "PAYME", "ALIF", "ВОЗВРАТ", "ОБЩАЯ СУММА"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("Failed to create style: %v", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Set information to columns
	for i, value := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, value.StoreCode)
		f.SetCellValue(sheetName, "B"+row, value.StoreName)
		f.SetCellValue(sheetName, "C"+row, value.SaleDate)
		f.SetCellValue(sheetName, "D"+row, value.Cash)
		f.SetCellValue(sheetName, "E"+row, value.Humo)
		f.SetCellValue(sheetName, "F"+row, value.Uzcard)
		f.SetCellValue(sheetName, "G"+row, value.Click)
		f.SetCellValue(sheetName, "H"+row, value.Payme)
		f.SetCellValue(sheetName, "I"+row, value.Alif)
		f.SetCellValue(sheetName, "J"+row, value.ReturnAmount)
		f.SetCellValue(sheetName, "K"+row, value.TotalAmount)
	}

	saveExcelToUploads(c, f, *h.log, "apteka_reports")
}

// Bonus report godoc
// @Summary Get Store report stats
// @Description Get Store report stats
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date Format(2025-03)"
// @Param   end_date query string false "End Date Format(2025-04)"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   body body domain.DashboardBody false "ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-stats [POST]
func (h *ReportHandler) StoreReportStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	params.StoreIds = body.StoreIds
	params.CompanyIds = body.CompanyIds

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeoutForReports)
	defer cancel()

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyIds = []string{user.CompanyId}
	}

	// get store report with payment type amounts
	res, err := h.service.ReportByStoreStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// Top Products godoc
// @Summary Get top products
// @Description Get top products
// @Tags Report
// @Security     BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-products [POST]
func (h *ReportHandler) ReportTopProducts(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store ids
	if err = c.ShouldBindJSON(&params.StoreIds); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get report TopProducts data
	res, totalCount, err := h.service.GetTopProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)
	handleResponse(c, OK, result)
}

// Top Bonus Products godoc
// @Summary Get bonus products
// @Description Get bonus products
// @Tags Report
// @Security     BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-seller [POST]
func (h *ReportHandler) ReportTopSeller(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get top seller data
	res, totalCount, err := h.service.GetTopSellersReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// Top Sellers Export godoc
// @Summary Export Top Sellers to Excel
// @Description Export Top Sellers to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param store_id query string false "Store ID"
// @Param store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-seller/export-excel [POST]
func (h *ReportHandler) TopSellerExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get top seller data
	res, _, err := h.service.GetTopSellersReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create excel
	f := excelize.NewFile()
	sheet := "List1"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{"ID", "Ф.И.O", "Магазин", "Количество", "Общее количество", "Общая сумма"}
	err = setExcelHeaders(f, sheet, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.FullName)
		f.SetCellValue(sheet, "C"+row, val.StoreName)
		f.SetCellValue(sheet, "D"+row, val.Count)
		f.SetCellValue(sheet, "E"+row, val.TotalCount)
		f.SetCellValue(sheet, "F"+row, val.TotalAmount)
	}

	// save excel
	saveExcelToUploads(c, f, *h.log, "top_sellers")
}

// Top Stores godoc
// @Summary Get top stores
// @Description Get top stores
// @Tags Report
// @Security     BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   store_id 	query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-stores [POST]
func (h *ReportHandler) ReportTopStores(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get top stores data
	res, totalCount, err := h.service.GetTopStoresReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// Top Stores Export godoc
// @Summary Export Top Stores to Excel
// @Description Export Top Stores to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-stores/export-excel [POST]
func (h *ReportHandler) TopStoresExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get top stores data
	res, _, err := h.service.GetTopStoresReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create excel
	f := excelize.NewFile()
	sheet := "List1"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{"ID", "Название", "Количество", "Общая количество", "Общая сумма"}
	err = setExcelHeaders(f, sheet, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}
	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.Name)
		f.SetCellValue(sheet, "C"+row, val.Count)
		f.SetCellValue(sheet, "D"+row, val.TotalCount)
		f.SetCellValue(sheet, "E"+row, val.TotalAmount)
	}

	// save excel
	saveExcelToUploads(c, f, *h.log, "top_stores")
}

// Top Bonus Products godoc
// @Summary Get bonus products
// @Description Get bonus products
// @Tags Report
// @Security     BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param   limit 	query int false "Limit"
// @Param 	offset query int false 	"Offset"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   store_id 	query string false "Store ID"
// @Param   store_ids  	body  []string  false  "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-products [POST]
func (h *ReportHandler) ReportBonusProducts(c *gin.Context) {
	// // get user_id from the context
	// user := h.service.GetSignedUser(c)
	// if user.UserId == "" {
	// 	handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
	// 	return
	// }

	var params domain.ReportQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store ids
	if err := c.ShouldBindJSON(&params.StoreIds); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	// // check if employee is not admin or superadmin
	// if !utils.In(user.Role, constants.AllAdminRoles...) {
	// 	if user.StoreId != "" {
	// 		params.StoreId = user.StoreId
	// 	}
	// 	params.CompanyId = user.CompanyId
	// }

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get bonus products data
	res, totalCount, err := h.service.GetBonusProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// Top Products Export godoc
// @Summary Export Top Products to Excel
// @Description Export Top Products to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param store_id query string false "Store ID"
// @Param store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/top-products/export-excel [POST]
func (h *ReportHandler) TopProductsExportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// get top products data
	res, _, err := h.service.GetTopProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	f := excelize.NewFile()
	sheet := "List1"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{"ID", "Название", "Производитель", "Количество", "Общее количество", "Общая сумма"}
	err = setExcelHeaders(f, sheet, headers)
	if err != nil {
		h.log.Errorf("could not create excel style: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.Name)
		f.SetCellValue(sheet, "C"+row, val.ProducerName)
		f.SetCellValue(sheet, "D"+row, val.Count)
		f.SetCellValue(sheet, "E"+row, val.TotalCount)
		f.SetCellValue(sheet, "F"+row, val.TotalAmount)
	}

	saveExcelToUploads(c, f, *h.log, "Top_products")
}

// Bonus Products Export godoc
// @Summary Export Bonus Products to Excel
// @Description Export Bonus Products to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param store_id query string false "Store ID"
// @Param store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-products/export-excel [POST]
func (h *ReportHandler) BonusProductsExportExcel(c *gin.Context) {
	// get user_id from the context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store ids
	if err := c.ShouldBindJSON(&params.StoreIds); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get bonus products data
	res, _, err := h.service.GetBonusProductsReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create excel
	f := excelize.NewFile()
	sheet := "BonusProducts"
	f.SetSheetName("Sheet1", sheet)
	// set headers
	headers := []string{"ID", "Название", "Количество", "Общая количество", "Сумма бонуса"}
	err = setExcelHeaders(f, sheet, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.Name)
		f.SetCellValue(sheet, "C"+row, val.Count)
		f.SetCellValue(sheet, "D"+row, val.TotalCount)
		f.SetCellValue(sheet, "E"+row, val.BonusAmount)
	}

	saveExcelToUploads(c, f, *h.log, "Bonus_products")
}

// Bonus products stats godoc
// @Summary Get bonus products stats
// @Description Get bonus products stats (dashboard summary)
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   start_date  query string false "Start Date (RFC3339)"
// @Param   end_date    query string false "End Date (RFC3339)"
// @Param   store_id    query string false "Store ID"
// @Param   store_ids   body []string false "Store IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-products-stats [POST]
func (h *ReportHandler) ReportBonusProductsStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, err := h.service.GetBonusProductsReportStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Bonus products stats godoc
// @Summary Get bonus products stats
// @Description Get bonus products stats (dashboard summary)
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   limit  		query string false "limit"
// @Param   offset    	query string false "offset"
// @Param   employee_id query string false "employee_id"
// @Param   start_date  query string false "start_date"
// @Param   end_date    query string false "end_date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-items [POST]
func (h *ReportHandler) FetchBonusItemsByEmployee(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	if params.EmployeeId == "" {
		handleServiceResponse(c, nil, domain.UserIdIsRequiredError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetBonusProductsByEmployeeId(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Bonus products stats godoc
// @Summary Get bonus products stats
// @Description Get bonus products stats (dashboard summary)
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   limit  		query string false "limit"
// @Param   offset    	query string false "offset"
// @Param   employee_id query string false "employee_id"
// @Param   start_date  query string false "start_date"
// @Param   end_date    query string false "end_date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/bonus-items-export [POST]
func (h *ReportHandler) FetchBonusItemsByEmployeeExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	if params.EmployeeId == "" {
		handleServiceResponse(c, nil, domain.UserIdIsRequiredError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, _, err := h.service.GetBonusProductsByEmployeeId(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// create Excel
	f := excelize.NewFile()
	sheet := "List1"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{
		"Чек ID", "Товар ID", "Наименование", "Кол-во", "Сумма бонус", "Дата",
	}
	if err := setExcelHeaders(f, sheet, headers); err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// fill rows
	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, val.SaleNumber)
		f.SetCellValue(sheet, "B"+row, val.MaterialCode)
		f.SetCellValue(sheet, "C"+row, val.ProductName)
		f.SetCellValue(sheet, "D"+row, math.Round(float64(val.UQuantity)/float64(val.UnitPerPack)*100)/100)
		f.SetCellValue(sheet, "E"+row, val.BonusAmount)
		if val.CreatedAt != nil {
			f.SetCellValue(sheet, "F"+row, val.CreatedAt.Format(constants.TimeOnlyDateFormat))
		} else {
			f.SetCellValue(sheet, "F"+row, "N/A")
		}
		if val.CreatedAt != nil {
			f.SetCellValue(sheet, "G"+row, val.CreatedAt.Format(time.TimeOnly))
		} else {
			f.SetCellValue(sheet, "G"+row, "N/A")
		}
	}

	// save to /uploads
	saveExcelToUploads(c, f, *h.log, "bonus_items")

}

// ReportStoreSummary godoc
// @Summary Get daily store summary
// @Description Returns sales, stock and import summary of stores
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param   order       query string false "Order"
// @Param   search      query string false "Search"
// @Param   limit       query int false "Limit"
// @Param   offset      query int false "Offset"
// @Param   start_date  query string false "Start Date"
// @Param   end_date    query string false "End Date"
// @Param   store_id    query string false "Store ID"
// @Param   body   		body  domain.DashboardBody false "ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-summary [POST]
func (h *ReportHandler) ReportStoreSummary(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	params.StoreIds = body.StoreIds
	params.CompanyIds = body.CompanyIds

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyIds = []string{user.CompanyId}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// pagination fallback
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// call service layer
	data, total, err := h.service.GetStoreSummaryReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(data, total, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// ReportStoreSummaryStats godoc
// @Summary Get daily store summary stats
// @Description Returns sales, stock and import summary of stores
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param   order       query string false "Order"
// @Param   search      query string false "Search"
// @Param   limit       query int false "Limit"
// @Param   offset      query int false "Offset"
// @Param   start_date  query string false "Start Date"
// @Param   end_date    query string false "End Date"
// @Param   store_id    query string false "Store ID"
// @Param   body   		body  domain.DashboardBody false "ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-summary-stats [POST]
func (h *ReportHandler) ReportStoreSummaryStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var body domain.DashboardBody
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}

	params.StoreIds = body.StoreIds
	params.CompanyIds = body.CompanyIds

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreIds = []string{user.StoreId}
		}
		params.CompanyIds = []string{user.CompanyId}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	data, err := h.service.GetStoreSummaryReportStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, data)
}

// Store Summary Export godoc
// @Summary Export Store Summary to Excel
// @Description Export Store Summary to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param   order       query string   false "Order"
// @Param   search      query string   false "Search"
// @Param   limit       query int      false "Limit"
// @Param   offset      query int      false "Offset"
// @Param   start_date  query string   false "Start Date"
// @Param   end_date    query string   false "End Date"
// @Param   store_id    query string   false "Store ID"
// @Param   store_ids   body   []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-summary/export-excel [POST]
func (h *ReportHandler) StoreSummaryExportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// bind store ids optional
	_ = c.ShouldBindJSON(&params.StoreIds)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// pagination fallback
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// call service layer
	res, _, err := h.service.GetStoreSummaryReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create Excel
	f := excelize.NewFile()
	sheet := "Остаток Аптека"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{
		"№", "Аптека", "Общая сумма продажа", "Импорт ожидании", "Общая сумма баланса", "Итог",
	}
	if err := setExcelHeaders(f, sheet, headers); err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// fill rows
	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.Name)
		f.SetCellValue(sheet, "C"+row, val.SaleAmount)
		f.SetCellValue(sheet, "D"+row, val.ImportAmount)
		f.SetCellValue(sheet, "E"+row, val.StockAmount)
		f.SetCellValue(sheet, "F"+row, val.Total)
	}

	// save to /uploads
	saveExcelToUploads(c, f, *h.log, "Остаток_Аптека")
}

// StoreProductsGivenDay godoc
// @Summary Get Store Products Given Day
// @Description Get Store Products Given Day
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param   order       query string false "Order"
// @Param   search      query string false "Search"
// @Param   limit       query int false "Limit"
// @Param   offset      query int false "Offset"
// @Param   start_date  query string false "Start Date (YYYY-MM-DD)"
// @Param   store_id    query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-products-given-day [POST]
func (h *ReportHandler) StoreProductsGivenDay(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return

	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	data, total, err := h.service.GetStoreProductsGivenDay(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(data, total, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// StoreProductsGivenDayExportExcel godoc
// @Summary Export Store Products Given Day to Excel
// @Description Export Store Products Given Day to Excel
// @Tags Report
// @Security BearerAuth
// @Produce json
// @Param   order       query string false "Order"
// @Param   search      query string false "Search"
// @Param   limit       query int    false "Limit"
// @Param   offset      query int    false "Offset"
// @Param   start_date  query string false "Start Date (YYYY-MM-DD)"
// @Param   store_id    query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-products-given-day/export-excel [POST]
func (h *ReportHandler) StoreProductsGivenDayExportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return

	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	res, _, err := h.service.GetStoreProductsGivenDay(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// create Excel
	f := excelize.NewFile()
	sheet := "Остаток по дате"
	f.SetSheetName("Sheet1", sheet)

	// headers mapping
	headers := []string{
		"№",
		"Наименование товара",
		"Аптека",
		"Итог (упаковки)",
		"Итог (штуки)",
		"Текущий остаток (упаковки)",
		"Текущий остаток (штуки)",
		"Приход (упаковки)",
		"Приход (штуки)",
		"Продажа (упаковки)",
		"Продажа (штуки)",
		"Возврат (упаковки)",
		"Возврат (штуки)",
		"Перемещение (упаковки)",
		"Перемещение (штуки)",
		"Инвентаризация (упаковки)",
		"Инвентаризация (штуки)",
	}
	if err := setExcelHeaders(f, sheet, headers); err != nil {
		h.log.Error("Failed to set excel headers:", err)
		handleResponse(c, InternalError, "Error on creating excel")
		return
	}

	// fill rows
	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.Name)
		f.SetCellValue(sheet, "C"+row, val.StoreName)
		f.SetCellValue(sheet, "D"+row, val.UnitQuantity)
		f.SetCellValue(sheet, "E"+row, val.MaterialCode)
		f.SetCellValue(sheet, "F"+row, val.Barcode)
		f.SetCellValue(sheet, "G"+row, val.UnitPerPack)
		f.SetCellValue(sheet, "H"+row, val.MXIK)
		f.SetCellValue(sheet, "I"+row, val.UnitCode)
		f.SetCellValue(sheet, "J"+row, val.IsMarking)
		f.SetCellValue(sheet, "K"+row, val.Manufacturer)
		f.SetCellValue(sheet, "L"+row, val.UnitName)
		f.SetCellValue(sheet, "M"+row, val.UnitLabel)
		f.SetCellValue(sheet, "N"+row, val.CreatedAt)
		f.SetCellValue(sheet, "O"+row, val.UpdatedAt)
	}

	// save to /uploads
	saveExcelToUploads(c, f, *h.log, "Остаток по дате")
}

// Discount card report godoc
// @Summary Get discount card report
// @Description Get discount card report
// @Tags Report
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   search query string false "Search (customer full name)"
// @Param   store_ids body []string false "Store ids"
// @Param   order query string false "every field in response is sorted by this field")
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/discount-card [POST]
func (h *ReportHandler) DiscountCardReport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReportQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&params.StoreIds)
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get default limit and offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetDiscountCardReport(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}
