package v1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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
		report.POST("/product-by-date", h.ProductReportByDate)
		report.POST("/product-by-date/export", h.ProductByDateExport)
		report.POST("/bonus", h.BonusReport)
		report.POST("/bonus-export", h.BonusReportExport)
		report.POST("/product", h.ProductReport)
		report.POST("/product-export", h.ProductReportExportExcel)
		report.POST("/lfl", h.LflReport)
		report.POST("/store-amount", h.StoreReportAmount)
		report.POST("/store-amount/export-excel", h.StoreReportAmountExport)
		report.POST("/store-stats", h.StoreReportStats)
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
	var (
		param domain.ReportQueryParam
	)
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}

	// Bind JSON body (store_ids)
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&param.StoreIds)
	}

	res, err := h.service.ProductReportWithDate(&param)
	if err != nil {
		h.log.Error("ERROR on getting product report: %v", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
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
// @Router /report/product-by-date/export [POST]
func (h *ReportHandler) ProductByDateExport(c *gin.Context) {
	var (
		param domain.ReportQueryParam
	)
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}
	// bind store ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&param.StoreIds)
	}

	// Reportni olish
	res, err := h.service.ProductReportWithDate(&param)
	if err != nil {
		h.log.Error("ERROR on getting product report: %v", err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Excel file
	f := excelize.NewFile()
	sheetName := "Товары отчет"
	f.SetSheetName("Sheet1", sheetName)

	// StartDate va EndDate oralig'idagi sanalarni olish
	startDate, _ := time.Parse("2006-01-02", param.StartDate)
	endDate, _ := time.Parse("2006-01-02", param.EndDate)

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
	var param domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&param.StoreIds)
	}
	// get default limit and offset for pagination
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get bonus reports
	res, totalCount, err := h.service.BonusReport(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get bonus report")
		return
	}
	// get data with _meta pagination info
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

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
	var param domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&param.StoreIds)
	}
	fmt.Println("param:", param.StoreIds)
	// get default limit and offset for pagination
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get bonus reports
	res, _, err := h.service.BonusReport(&param)
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

	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
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
	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Xodimlar_bonuslari_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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
// @Param   employee_id query string false "Employee Id"
// @Param   producer_id query string false "Producer ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product [POST]
func (h *ReportHandler) ProductReport(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), config.ContextTimeoutForReports)
	defer cancel()

	var param domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		// bind store_ids
		_ = c.ShouldBindJSON(&param.StoreIds)
	}

	// get default limit and offset for pagination
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.ProductReport(ctx, &param)
	if err != nil {
		h.log.Warn("Failed to get product report: %v", err)
		handleResponse(c, InternalError, "failed to get product report")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

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
// @Param   employee_id query string false "Employee Id"
// @Param   producer_id query string false "Producer ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-export [POST]
func (h *ReportHandler) ProductReportExportExcel(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), config.ContextTimeoutForReports)
	defer cancel()

	var param domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		// bind store_ids
		_ = c.ShouldBindJSON(&param.StoreIds)
	}
	// get default limit and offset for pagination
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.ProductReport(ctx, &param)
	if err != nil {
		h.log.Warn("Failed to get product report: %v", err)
		handleResponse(c, InternalError, "failed to get product report")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Филиал", "Наименование", "Производитель", "Серия", "Срок Годности", "Кол-во", "Цена прихода", "Цена продажная", "Сумма прихода", "Сумма продажная", "Сумма наценки", "Сумма НДС", "Дата продажи", "Время продажи", "Пользователь", "ID ЧЕКА", "МК кол-во"}

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

	//
	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
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
		f.SetCellValue(sheetName, "G"+row, value.Quantity)
		f.SetCellValue(sheetName, "H"+row, value.SupplyPrice)
		f.SetCellValue(sheetName, "I"+row, value.RetailPrice)
		f.SetCellValue(sheetName, "J"+row, value.SupplyPriceSum)
		f.SetCellValue(sheetName, "K"+row, value.RetailPriceSum)
		f.SetCellValue(sheetName, "L"+row, value.MarkupSum)
		f.SetCellValue(sheetName, "M"+row, value.VatSum)
		f.SetCellValue(sheetName, "N"+row, value.CompletedAt.Format(time.DateOnly))
		f.SetCellValue(sheetName, "O"+row, value.CompletedAt.Format(time.TimeOnly))
		f.SetCellValue(sheetName, "P"+row, value.FullName)
		f.SetCellValue(sheetName, "Q"+row, helper.SaleTypeToRussian(value.SaleType, value.SaleNumber))
		f.SetCellValue(sheetName, "R"+row, value.MarkingCount)
	}

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Sale_details_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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
// @Param   start_date query string false "Start Date Format(2025-03)"
// @Param   end_date query string false "End Date Format(2025-04)"
// @Param   search query string false "Search"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/lfl [POST]
func (h *ReportHandler) LflReport(c *gin.Context) {
	var param domain.ReportQueryParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	// bind store_ids
	if c.Request.Body != nil {
		// bind store_ids
		_ = c.ShouldBindJSON(&param.StoreIds)

	}
	// get default limit and offset for pagination
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.LflReport(&param)
	if err != nil {
		h.log.Warn("Failed to get product report: %v", err)
		handleResponse(c, InternalError, "failed to get product report")
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
// @Param   store_id query string false "Store ID"
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-amount [POST]
func (h *ReportHandler) StoreReportAmount(c *gin.Context) {
	var (
		param domain.ReportQueryParam
	)
	// bind request query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get store report with payment type amounts
	res, totalCount, err := h.service.StoreReportAmount(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get store report amounts")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

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
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-amount/export-excel [POST]
func (h *ReportHandler) StoreReportAmountExport(c *gin.Context) {
	var (
		param domain.ReportQueryParam
	)
	// bind request query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get store report with payment type amounts
	res, _, err := h.service.StoreReportAmount(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get store report amounts")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Филиал", "Дата", "Наличные", "HUMO", "UZCARD", "CLICK", "PAYME", "ALIF", "Возврат", "Общая сумма"}

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

	//
	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
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

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Filial_hisoboti_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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
// @Param   store_ids body []string false "Store ids"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/store-stats [POST]
func (h *ReportHandler) StoreReportStats(c *gin.Context) {
	var param domain.ReportQueryParam

	// bind request query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	// get store report with payment type amounts
	res, err := h.service.ReportByStoreStats(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get store report stats")
		return
	}

	handleResponse(c, OK, res)
}
