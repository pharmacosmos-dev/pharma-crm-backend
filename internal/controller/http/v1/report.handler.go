package v1

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
		report.POST("/product-by-date", h.ProductByDate)
		report.POST("/product-by-date/export", h.ProductByDateExport)
		report.POST("/bonus", h.BonusReport)
		report.POST("/bonus-export", h.BonusReportExport)
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
// @Param   product_ids body []string false "Product ids"
// @Param   store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-by-date [POST]
func (h *ReportHandler) ProductByDate(c *gin.Context) {
	var (
		param      domain.ReportQueryParam
		productIds []string
	)
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}
	// bind product ids
	if err = c.ShouldBindJSON(&productIds); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid product ids received")
		return
	}
	param.ProductIds = productIds
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
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   product_ids body []string false "Product ids"
// @Param   store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /report/product-by-date/export [POST]
func (h *ReportHandler) ProductByDateExport(c *gin.Context) {
	var (
		param      domain.ReportQueryParam
		productIds []string
	)
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid query param received")
		return
	}
	// bind product ids
	if err := c.ShouldBindJSON(&productIds); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid product ids received")
		return
	}
	param.ProductIds = productIds

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

	// Yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=bioline-sale.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
		return
	}

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
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
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
	sheetName := "Бонусный отчет"
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

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=bonus-report.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
}
