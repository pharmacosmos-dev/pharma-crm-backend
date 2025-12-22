package v1

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
)

type OstatokHandler struct {
	*Handler
}

func (h *Handler) NewOstatokHandler(r *gin.RouterGroup) {
	ostatok := &OstatokHandler{h}
	ostatok.OstatokRoutes(r)
}

func (h *OstatokHandler) OstatokRoutes(r *gin.RouterGroup) {
	ostatok := r.Group("/ostatok")
	{
		ostatok.GET("", h.GetOstatok)
		ostatok.POST("/correct", h.UploadCorrectOstatok)
		ostatok.GET("/excel/:xlsx", h.ServeExcelFile)
	}
}

// GetOstatok godoc
// @Summary Get Wrong Ostatok Data
// @Description Get Wrong Ostatok Data
// @Tags 		 Ostatok
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param 		 store_id   query    string  true "Store ID"
// @Param        limit      query    int     false "Limit"
// @Param        offset     query    int     false "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /ostatok [GET]
func (h *OstatokHandler) GetOstatok(c *gin.Context) {
	storeId := c.Query("store_id")
	if storeId == "" {
		handleResponse(c, BadRequest, "store_id is required")
		return
	}

	query := `
	WITH import_data AS (
		SELECT
			p.id as product_id,
			sp.id AS store_product_id,
			COALESCE(imd.scanned_count * p.unit_per_pack, 0) AS scanned_count
		FROM import_details imd
		JOIN products p ON p.id = imd.product_id
		JOIN store_products sp ON sp.import_detail_id = imd.id
		JOIN imports im ON im.id = imd.import_id
		WHERE im.entry_type = 1 AND im.status = 'completed' AND im.store_id = ?
	),
	sold AS (
		SELECT
			sp.product_id,
			sp.id AS store_product_id,
			COALESCE(SUM(ci.unit_quantity), 0) AS sold_quantity
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN sales s ON s.id = ci.sale_id
		WHERE s.stage = 9 AND s.sale_type = 'SALE' AND s.store_id = ?
		GROUP BY sp.id
	),
	return_sales AS (
		SELECT
			sp.product_id,
			sp.id AS store_product_id,
			COALESCE(SUM(ci.unit_quantity), 0) AS sold_quantity
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN sales s ON s.id = ci.sale_id
		WHERE s.stage = 11 AND s.sale_type = 'RETURN' AND s.store_id = ?
		GROUP BY sp.id
	),
	tranfer_in AS (
		SELECT
			p.id as product_id,
			sp.id as store_product_id,
			td.received_count * p.unit_per_pack AS received_count,
			COALESCE((td.accepted_count * p.unit_per_pack), 0) AS scanned_count
		FROM transfer_details td
		JOIN products p ON p.id = td.product_id
		JOIN store_products sp ON td.id = sp.import_detail_id
		JOIN transfers t ON t.id = td.transfer_id
		WHERE t.entry_type = 1 AND t.status = 'completed' AND t.to_store_id = ?
	),
	tranfer_out AS (
		SELECT
			p.id as product_id,
			sp.id as store_product_id,
			COALESCE((td.accepted_count * p.unit_per_pack), 0) AS scanned_count
		FROM transfer_details td
		JOIN products p ON p.id = td.product_id
		JOIN store_products sp ON td.store_product_id = sp.id
		JOIN transfers t ON t.id = td.transfer_id
		WHERE t.entry_type = 1 AND t.status = 'completed' AND t.from_store_id = ?
	),
	vozvrat AS (
		SELECT
			p.id as product_id,
			sp.id as store_product_id,
			COALESCE(td.accepted_count * p.unit_per_pack, 0) AS scanned_count
		FROM transfer_details td
		JOIN products p ON p.id = td.product_id
		JOIN store_products sp ON sp.id = td.store_product_id
		JOIN transfers t ON t.id = td.transfer_id
		WHERE t.entry_type = 2 AND t.status = 'sent-to-1c' AND t.from_store_id = ?
	)
	SELECT
		sp.id                                 AS store_product_id,
		p.id                                  AS product_id,
		st.name                               AS store_name,
		p.name                                AS product_name,
		p.unit_per_pack                       AS unit_per_pack,
		COALESCE(im.scanned_count, 0)         AS import,
		COALESCE(sp.unit_quantity, 0)         AS ostatok,
		COALESCE(s.sold_quantity, 0)          AS sale,
		COALESCE(rs.sold_quantity, 0)         AS returned,
		COALESCE(tin.scanned_count, 0)        AS transfer_in,
		COALESCE(tout.scanned_count, 0)       AS transfer_out,
		COALESCE(v.scanned_count, 0)          AS vozvrat,
		COALESCE(im.scanned_count, 0) - COALESCE(s.sold_quantity, 0) + COALESCE(rs.sold_quantity, 0) + COALESCE(tin.scanned_count, 0) - COALESCE(tout.scanned_count, 0) - COALESCE(v.scanned_count, 0) AS correct
	FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			JOIN stores st ON sp.store_id = st.id
			LEFT JOIN import_data im ON im.store_product_id = sp.id
			LEFT JOIN sold s ON s.store_product_id = sp.id
			LEFT JOIN tranfer_in tin ON tin.store_product_id = sp.id
			LEFT JOIN tranfer_out tout ON tout.store_product_id = sp.id
			LEFT JOIN vozvrat v ON v.store_product_id = sp.id
			LEFT JOIN return_sales rs ON rs.store_product_id = sp.id
	WHERE sp.store_id = ?
	AND COALESCE(im.scanned_count, 0) - COALESCE(s.sold_quantity, 0) + COALESCE(rs.sold_quantity, 0) + COALESCE(tin.scanned_count, 0) - COALESCE(tout.scanned_count, 0) - COALESCE(v.scanned_count, 0) != sp.unit_quantity
	ORDER BY sp.created_at desc;
	`
	var results []struct {
		StoreProductID string  `gorm:"store_product_id" json:"store_product_id"`
		ProductID      string  `gorm:"product_id" json:"product_id"`
		StoreName      string  `gorm:"store_name" json:"store_name"`
		ProductName    string  `gorm:"product_name" json:"product_name"`
		UnitPerPack    float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
		Import         float64 `gorm:"import" json:"import"`
		Ostatok        float64 `gorm:"ostatok" json:"ostatok"`
		Sale           float64 `gorm:"sale" json:"sale"`
		Returned       float64 `gorm:"returned" json:"returned"`
		TransferIn     float64 `gorm:"transfer_in" json:"transfer_in"`
		TransferOut    float64 `gorm:"transfer_out" json:"transfer_out"`
		Vozvrat        float64 `gorm:"vozvrat" json:"vozvrat"`
		Correct        float64 `gorm:"correct" json:"correct"`
	}
	err := h.db.Raw(query, storeId, storeId, storeId, storeId, storeId, storeId, storeId).Scan(&results).Error
	if err != nil {
		h.log.Errorf("could not fetch ostatok by store_id(%s) err: %v", storeId, err)
		handleResponse(c, InternalError, "could not fetch ostatok data")
		return
	}

	// create excel
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// headers
	headers := []string{
		"Id", "ProductID", "Apteka",
		"ProductName", "№", "Ostatok",
		"ImportUnits", "SaleUnits",
		"ReturnedUnits", "TransferInUnits",
		"TransferOutUnits", "VozvratUnits", "CorrectUnits"}
	if err := setExcelHeaders(f, sheetName, headers); err != nil {
		h.log.Error("Excel style error:", err)
		handleResponse(c, InternalError, "Error on creating excel")
		return
	}
	var storeName string
	if len(results) > 0 {
		storeName = results[0].StoreName
	} else {
		storeName = "store"
	}
	// fill rows
	for i, item := range results {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, item.StoreProductID)
		f.SetCellValue(sheetName, "B"+row, item.ProductID)
		f.SetCellValue(sheetName, "C"+row, item.StoreName)
		f.SetCellValue(sheetName, "D"+row, item.ProductName)
		f.SetCellValue(sheetName, "E"+row, item.UnitPerPack)
		f.SetCellValue(sheetName, "F"+row, item.Ostatok)
		f.SetCellValue(sheetName, "G"+row, item.Import)
		f.SetCellValue(sheetName, "H"+row, item.Sale)
		f.SetCellValue(sheetName, "I"+row, item.Returned)
		f.SetCellValue(sheetName, "J"+row, item.TransferIn)
		f.SetCellValue(sheetName, "K"+row, item.TransferOut)
		f.SetCellValue(sheetName, "L"+row, item.Vozvrat)
		f.SetCellValue(sheetName, "M"+row, item.Correct)
	}

	fileName := "ostatok_" + strings.Replace(storeName, " ", "_", 10)
	// save
	saveExcelToUploads(c, f, *h.log, fileName)

}

// UploadCorrectOstatok godoc
// @Summary update correct ostatok
// @Description update correct ostatok
// @Tags 		 Ostatok
// @Security     BearerAuth
// @Accept 		json
// @Produce 	json
// @Param 	file formData file true "Excel file (.xlsx) containing ostatok corrections"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /ostatok/correct [POST]
func (h *OstatokHandler) UploadCorrectOstatok(c *gin.Context) {
	var file domain.File
	// bind request file
	if err := c.ShouldBind(&file); err != nil {
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
	err := c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}

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
		UPDATE store_products SET unit_quantity = ?, updated_at = NOW() WHERE id = ?;
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 11 {
			unitQuantity := cast.ToInt(row[12])
			err = h.db.Exec(query, unitQuantity, row[0]).Error
			if err != nil {
				h.log.Errorf("could not update store_product(%s) -> %v", row[0], err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			count++
		}
	}

	handleResponse(c, OK, "Successfully uploaded "+strconv.Itoa(count)+" records")
}

// ServeFile godoc
// @Summary Serve a file
// @Description Serve a file by its filename
// @Tags 	Ostatok
// @Produce octet/stream
// @Param xlsx path string true "File name"
// @Success 200 {file} file "File content"
// @Router /ostatok/excel/{xlsx} [get]
func (h *OstatokHandler) ServeExcelFile(c *gin.Context) {
	xlsx := c.Param("xlsx")
	if xlsx == "" {
		h.log.Warn("Filename not provided: %v", xlsx)
		handleResponse(c, BadRequest, "Filename not provided")
		return
	}

	filePath := filepath.Join("./uploads", xlsx)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			h.log.Warn("File not found: %v", err.Error())
			handleResponse(c, NotFound, "File not found")
			return
		}
		h.log.Error("Error opening file: %v", err)
		handleResponse(c, InternalError, "Could not open file")
		return
	}
	defer file.Close()

	// Set the headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", xlsx))
	c.Header("Content-Type", "application/octet-stream")

	// Stream the file to the client
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.log.Error("Error writing file to response: %v", err)
		handleResponse(c, InternalError, "Could not send file")
		return
	}

	// Remove the file after sending
	if err := os.Remove(filePath); err != nil {
		h.log.Error("Error deleting file after send: %v", err)
	}
}
