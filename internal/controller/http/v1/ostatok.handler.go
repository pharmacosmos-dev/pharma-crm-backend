package v1

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		// ostatok.POST("/correct", h.UploadCorrectOstatok)
		ostatok.GET("/excel/:xlsx", h.ServeExcelFile)
		ostatok.POST("/fixed-plus", h.FixedPlus)
		ostatok.POST("/fixed-minus", h.FixedMinus)
		ostatok.GET("/fixed-stores", h.FixedStores)
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
        p.id AS product_id,
        p.unit_per_pack,
        SUM(imd.scanned_count * p.unit_per_pack) AS import_count
    FROM import_details imd
    JOIN products p ON imd.product_id = p.id
    JOIN imports im ON imd.import_id = im.id
    WHERE  im.entry_type = 1 AND im.status = 'completed' AND im.store_id = ?
    GROUP BY p.id
    ),
    sale_data AS (
        SELECT
            p.id AS product_id,
            SUM(ci.unit_quantity) AS sale_count
        FROM cart_items ci
        JOIN sales s on ci.sale_id = s.id
        JOIN store_products sp on ci.store_product_id = sp.id
        JOIN products p ON sp.product_id = p.id
        WHERE s.stage = 9 AND s.store_id = ?
        GROUP BY p.id
    ),
    return_data AS (
        SELECT
            p.id AS product_id,
            SUM(ci.unit_quantity) AS sale_count
        FROM cart_items ci
        JOIN sales s on ci.sale_id = s.id
        JOIN store_products sp on ci.store_product_id = sp.id
        JOIN products p ON sp.product_id = p.id
        WHERE s.stage = 11 AND s.store_id = ?
        GROUP BY p.id
    ),
    vozvrat_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS vozvrat_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE tr.entry_type = 2 AND tr.from_store_id = ? AND tr.status = 'sent-to-1c'
        GROUP BY p.id
    ),
    transfer_in_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE  tr.entry_type = 1 AND tr.to_store_id = ? AND tr.status = 'completed'
        GROUP BY p.id
    ),
    transfer_out_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE  tr.entry_type = 1 AND tr.from_store_id = ? AND tr.status = 'completed'
        GROUP BY p.id
    ),
    ostatok AS (
        SELECT
            p.id AS product_id,
            SUM(sp.unit_quantity) AS ostatok
        FROM store_products sp
        JOIN products p ON sp.product_id = p.id
        WHERE sp.store_id = ?
        GROUP BY p.id
    )
        SELECT
            p.id AS product_id,
            p.name,
            p.unit_per_pack,
            COALESCE(o.ostatok, 0) AS ostatok,
            COALESCE(imd.import_count, 0) AS import,
            COALESCE(sd.sale_count, 0) AS sale,
            COALESCE(rd.sale_count, 0) AS return,
            COALESCE(vd.vozvrat_count, 0) AS vozvrat,
            COALESCE(tid.transfer_count, 0) AS transfer_in,
            COALESCE(tod.transfer_count, 0) AS transfer_out,
            COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) + COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) + COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) AS fixed_count
        FROM products p
        LEFT JOIN import_data imd ON imd.product_id = p.id
        LEFT JOIN sale_data sd ON sd.product_id = p.id
        LEFT JOIN return_data rd ON rd.product_id = p.id
        LEFT JOIN vozvrat_data vd ON vd.product_id = p.id
        LEFT JOIN transfer_in_data tid ON tid.product_id = p.id
        LEFT JOIN transfer_out_data tod ON tod.product_id = p.id
        LEFT JOIN ostatok o ON o.product_id = p.id
        WHERE
            COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
            COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
            COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) != COALESCE(o.ostatok, 0);
	`
	var results []struct {
		ProductId   string  `gorm:"product_id" json:"product_id"`
		Name        string  `gorm:"name" json:"name"`
		UnitPerPack float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
		Ostatok     float64 `gorm:"ostatok" json:"ostatok"`
		Import      float64 `gorm:"import" json:"import"`
		Sale        float64 `gorm:"sale" json:"sale"`
		Return      float64 `gorm:"return" json:"return"`
		Vozvrat     float64 `gorm:"vozvrat" json:"vozvrat"`
		TransferIn  float64 `gorm:"transfer_in" json:"transfer_in"`
		TransferOut float64 `gorm:"transfer_out" json:"transfer_out"`
		FixedCount  float64 `gorm:"fixed_count" json:"fixed_count"`
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
		"ProductID", "ProductName", "№", "Ostatok",
		"ImportUnits", "SaleUnits",
		"ReturnedUnits", "TransferInUnits",
		"TransferOutUnits", "VozvratUnits", "FixedCount"}
	if err := setExcelHeaders(f, sheetName, headers); err != nil {
		h.log.Error("Excel style error:", err)
		handleResponse(c, InternalError, "Error on creating excel")
		return
	}

	var storeName string
	err = h.db.Raw("SELECT name AS store_name FROM stores WHERE id = ?", storeId).Scan(&storeName).Error
	if err != nil {
		h.log.Errorf("could not fetch store name: %v", err)
		storeName = "store"
	}

	// fill rows
	for i, item := range results {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, item.ProductId)
		f.SetCellValue(sheetName, "B"+row, item.Name)
		f.SetCellValue(sheetName, "C"+row, item.UnitPerPack)
		f.SetCellValue(sheetName, "D"+row, item.Ostatok)
		f.SetCellValue(sheetName, "E"+row, item.Import)
		f.SetCellValue(sheetName, "F"+row, item.Sale)
		f.SetCellValue(sheetName, "G"+row, item.Return)
		f.SetCellValue(sheetName, "H"+row, item.Vozvrat)
		f.SetCellValue(sheetName, "I"+row, item.TransferIn)
		f.SetCellValue(sheetName, "J"+row, item.TransferOut)
		f.SetCellValue(sheetName, "K"+row, item.FixedCount)
	}

	fileName := strings.Replace(storeName, " ", "_", 10) + "_ostatok_" + time.Now().Format("2006-01-02")
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

// FixedPlus godoc
// @Summary fixed plus ostatok
// @Description fixed plus ostatok
// @Tags 		 Ostatok
// @Security     BearerAuth
// @Accept 		json
// @Produce 	json
// @Param 	store_id   query    string  true "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /ostatok/fixed-plus [POST]
func (h *OstatokHandler) FixedPlus(c *gin.Context) {
	storeId := c.Query("store_id")
	if storeId == "" {
		handleResponse(c, BadRequest, "store_id is required")
		return
	}

	query := `
	WITH import_data AS (
		SELECT
			p.id AS product_id,
			p.unit_per_pack,
			SUM(imd.scanned_count * p.unit_per_pack) AS import_count
		FROM import_details imd
		JOIN products p ON imd.product_id = p.id
		JOIN imports im ON imd.import_id = im.id
		WHERE  im.entry_type = 1 AND im.status = 'completed' AND im.store_id = ?
		GROUP BY p.id
		),
		sale_data AS (
			SELECT
				p.id AS product_id,
				SUM(ci.unit_quantity) AS sale_count
			FROM cart_items ci
			JOIN sales s on ci.sale_id = s.id
			JOIN store_products sp on ci.store_product_id = sp.id
			JOIN products p ON sp.product_id = p.id
			WHERE s.stage = 9 AND s.store_id = ?
			GROUP BY p.id
		),
		return_data AS (
			SELECT
				p.id AS product_id,
				SUM(ci.unit_quantity) AS sale_count
			FROM cart_items ci
			JOIN sales s on ci.sale_id = s.id
			JOIN store_products sp on ci.store_product_id = sp.id
			JOIN products p ON sp.product_id = p.id
			WHERE s.stage = 11 AND s.store_id = ?
			GROUP BY p.id
		),
		vozvrat_data AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS vozvrat_count
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE tr.entry_type = 2 AND tr.from_store_id = ? AND tr.status = 'sent-to-1c'
			GROUP BY p.id
		),
		transfer_in_data AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE  tr.entry_type = 1 AND tr.to_store_id = ? AND tr.status = 'completed'
			GROUP BY p.id
		),
		transfer_out_data AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE  tr.entry_type = 1 AND tr.from_store_id = ? AND tr.status = 'completed'
			GROUP BY p.id
		),
		ostatok AS (
			SELECT
				p.id AS product_id,
				SUM(sp.unit_quantity) AS ostatok
			FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			WHERE sp.store_id = ?
			GROUP BY p.id
		),
		fixed_ostatok AS (
			SELECT
				p.id AS product_id,
				p.name,
				p.unit_per_pack,
				COALESCE(o.ostatok, 0) AS ostatok,
				COALESCE(imd.import_count, 0) AS import,
				COALESCE(sd.sale_count, 0) AS sale,
				COALESCE(rd.sale_count, 0) AS return,
				COALESCE(vd.vozvrat_count, 0) AS vozvrat,
				COALESCE(tid.transfer_count, 0) AS transfer_in,
				COALESCE(tod.transfer_count, 0) AS transfer_out,
				COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) + COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) + COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) AS fixed_count,
				COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
				COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
				COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) - COALESCE(o.ostatok, 0) AS add_count
			FROM products p
			LEFT JOIN import_data imd ON imd.product_id = p.id
			LEFT JOIN sale_data sd ON sd.product_id = p.id
			LEFT JOIN return_data rd ON rd.product_id = p.id
			LEFT JOIN vozvrat_data vd ON vd.product_id = p.id
			LEFT JOIN transfer_in_data tid ON tid.product_id = p.id
			LEFT JOIN transfer_out_data tod ON tod.product_id = p.id
			LEFT JOIN ostatok o ON o.product_id = p.id
			WHERE
				COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
				COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
				COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) != COALESCE(o.ostatok, 0)
			AND COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
				COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
				COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) - COALESCE(o.ostatok, 0) > 0
				),
		latest_store_products AS (
		SELECT DISTINCT ON (sp.product_id)
			sp.id,
			sp.product_id,
			sp.unit_quantity,
			sp.store_id
		FROM store_products sp
		WHERE sp.store_id = ?
		ORDER BY sp.product_id, sp.created_at DESC
	)
	UPDATE store_products sp
	SET unit_quantity = sp.unit_quantity + fo.add_count
	FROM latest_store_products lsp
	JOIN fixed_ostatok fo ON lsp.product_id = fo.product_id;
	`
	go func() {
		err := h.db.Exec(query, storeId, storeId, storeId, storeId, storeId, storeId, storeId, storeId).Error
		if err != nil {
			h.log.Errorf("could not fixed plus ostatok for store_id(%s) err: %v", storeId, err)
			handleResponse(c, InternalError, "could not fixed plus ostatok")
			return
		}
		err = h.db.Exec("UPDATE stores SET fixed_stage = 1 WHERE id = ?", storeId).Error
		if err != nil {
			h.log.Errorf("could not update fixed_stage for store_id(%s) err: %v", storeId, err)
			handleResponse(c, InternalError, "could not fixed plus ostatok")
			return
		}
	}()

	handleResponse(c, OK, "Ostatok fixed plus successfully")
}

// FixedMinus godoc
// @Summary fixed minus ostatok
// @Description fixed minus ostatok
// @Tags 		 Ostatok
// @Security     BearerAuth
// @Accept 		json
// @Produce 	json
// @Param 	store_id   query    string  true "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /ostatok/fixed-minus [POST]
func (h *OstatokHandler) FixedMinus(c *gin.Context) {
	storeId := c.Query("store_id")
	if storeId == "" {
		handleResponse(c, BadRequest, "store_id is required")
		return
	}

	query := `
	WITH import_data AS (
    SELECT
        p.id AS product_id,
        p.unit_per_pack,
        SUM(imd.scanned_count * p.unit_per_pack) AS import_count
    FROM import_details imd
    JOIN products p ON imd.product_id = p.id
    JOIN imports im ON imd.import_id = im.id
    WHERE  im.entry_type = 1 AND im.status = 'completed' AND im.store_id = ?
    GROUP BY p.id
    ),
    sale_data AS (
        SELECT
            p.id AS product_id,
            SUM(ci.unit_quantity) AS sale_count
        FROM cart_items ci
        JOIN sales s on ci.sale_id = s.id
        JOIN store_products sp on ci.store_product_id = sp.id
        JOIN products p ON sp.product_id = p.id
        WHERE s.stage = 9 AND s.store_id = ?
        GROUP BY p.id
    ),
    return_data AS (
        SELECT
            p.id AS product_id,
            SUM(ci.unit_quantity) AS sale_count
        FROM cart_items ci
        JOIN sales s on ci.sale_id = s.id
        JOIN store_products sp on ci.store_product_id = sp.id
        JOIN products p ON sp.product_id = p.id
        WHERE s.stage = 11 AND s.store_id = ?
        GROUP BY p.id
    ),
    vozvrat_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS vozvrat_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE tr.entry_type = 2 AND tr.from_store_id = ? AND tr.status = 'sent-to-1c'
        GROUP BY p.id
    ),
    transfer_in_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE  tr.entry_type = 1 AND tr.to_store_id = ? AND tr.status = 'completed'
        GROUP BY p.id
    ),
    transfer_out_data AS (
        SELECT
            p.id AS product_id,
            SUM(td.accepted_count * p.unit_per_pack) AS transfer_count
        FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN products p ON td.product_id = p.id
        WHERE  tr.entry_type = 1 AND tr.from_store_id = ? AND tr.status = 'completed'
        GROUP BY p.id
    ),
    ostatok AS (
        SELECT
            p.id AS product_id,
            SUM(sp.unit_quantity) AS ostatok
        FROM store_products sp
        JOIN products p ON sp.product_id = p.id
        WHERE sp.store_id = ?
        GROUP BY p.id
    ),
    fixed_ostatok AS (
        SELECT
            p.id AS product_id,
            p.name,
            p.unit_per_pack,
            COALESCE(o.ostatok, 0) AS ostatok,
            COALESCE(imd.import_count, 0) AS import,
            COALESCE(sd.sale_count, 0) AS sale,
            COALESCE(rd.sale_count, 0) AS return,
            COALESCE(vd.vozvrat_count, 0) AS vozvrat,
            COALESCE(tid.transfer_count, 0) AS transfer_in,
            COALESCE(tod.transfer_count, 0) AS transfer_out,
            COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) + COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) + COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) AS fixed_count,
            COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
            COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
            COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) - COALESCE(o.ostatok, 0) AS minus_count
        FROM products p
        LEFT JOIN import_data imd ON imd.product_id = p.id
        LEFT JOIN sale_data sd ON sd.product_id = p.id
        LEFT JOIN return_data rd ON rd.product_id = p.id
        LEFT JOIN vozvrat_data vd ON vd.product_id = p.id
        LEFT JOIN transfer_in_data tid ON tid.product_id = p.id
        LEFT JOIN transfer_out_data tod ON tod.product_id = p.id
        LEFT JOIN ostatok o ON o.product_id = p.id
        WHERE
            COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
            COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
            COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) != COALESCE(o.ostatok, 0)
        AND COALESCE(imd.import_count, 0) - COALESCE(sd.sale_count, 0) +
            COALESCE(rd.sale_count, 0) - COALESCE(vd.vozvrat_count, 0) +
            COALESCE(tid.transfer_count, 0) - COALESCE(tod.transfer_count, 0) - COALESCE(o.ostatok, 0) < 0
            ),
    store_products_with_deduction AS (
    SELECT
        sp.id,
        sp.product_id,
        sp.unit_quantity,
        fo.minus_count,
        ABS(fo.minus_count) AS minus_to_apply,

        -- oxiridan boshlab yig‘ilayotgan summa
        SUM(sp.unit_quantity) OVER (
            PARTITION BY sp.product_id
            ORDER BY sp.created_at DESC
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS running_sum
    FROM store_products sp
    JOIN fixed_ostatok fo ON fo.product_id = sp.product_id
    WHERE sp.store_id = ?
	),
		deduction_calc AS (
		SELECT
			id,
			product_id,
			unit_quantity,
			minus_to_apply,
			running_sum,

			CASE
				WHEN running_sum <= minus_to_apply
					THEN unit_quantity
				WHEN running_sum - unit_quantity < minus_to_apply
					THEN minus_to_apply - (running_sum - unit_quantity)
				ELSE 0
			END AS deduct_quantity
		FROM store_products_with_deduction
	)

	UPDATE store_products sp
	SET unit_quantity = sp.unit_quantity - dc.deduct_quantity
	FROM deduction_calc dc
	WHERE sp.id = dc.id
	AND dc.deduct_quantity > 0;
	`
	go func() {
		err := h.db.Exec(query, storeId, storeId, storeId, storeId, storeId, storeId, storeId, storeId).Error
		if err != nil {
			h.log.Errorf("could not fixed plus ostatok for store_id(%s) err: %v", storeId, err)
			handleResponse(c, InternalError, "could not fixed plus ostatok")
			return
		}

		err = h.db.Exec("UPDATE stores SET fixed_stage = 2 WHERE id = ?", storeId).Error
		if err != nil {
			h.log.Errorf("could not update fixed_stage for store_id(%s) err: %v", storeId, err)
			handleResponse(c, InternalError, "could not fixed plus ostatok")
			return
		}
	}()

	handleResponse(c, OK, "Ostatok fixed minus successfully")
}

// FixedStores godoc
// @Summary 	fixed stores
// @Description fixed stores
// @Tags 		 Ostatok
// @Security     BearerAuth
// @Accept 		json
// @Produce 	json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /ostatok/fixed-stores [GET]
func (h *OstatokHandler) FixedStores(c *gin.Context) {
	var stores []struct {
		Id          string     `gorm:"id" json:"id"`
		Name        string     `gorm:"name" json:"name"`
		HasInventor bool       `gorm:"has_inventor" json:"has_inventor"`
		FixedStage  int        `gorm:"fixed_stage" json:"fixed_stage"`
		CreatedAt   *time.Time `gorm:"created_at" json:"created_at"`
	}

	query := `
	SELECT
		id,
		name,
		created_at,
		exists(SELECT 1 FROM imports where entry_type = 2 AND store_id = s.id AND status = 'completed') AS has_inventor
	FROM stores s WHERE is_active = true ORDER BY created_at;
	`

	err := h.db.Raw(query).Scan(&stores).Error
	if err != nil {
		h.log.Errorf("could not fetch fixed stores err: %v", err)
		handleResponse(c, InternalError, "could not fetch fixed stores")
		return
	}

	handleResponse(c, OK, stores)
}
