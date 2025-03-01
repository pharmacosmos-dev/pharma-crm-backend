package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
		helper.POST("/export-package-code", h.ExportPackageCodeExcel)
	}
}

// Export package code excel godoc
// @Summary Export package code excel
// @Description Export package code excel
// @Tags helper
// @Security     BearerAuth
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /helper/export-package-code [POST]
func (h *HelperHandler) ExportPackageCodeExcel(c *gin.Context) {
	url := "http://integration.epos.uz:8347/uzpos"
	requestMap := map[string]string{
		"token":     "DXJFX32CN1296678504F2",
		"method":    "getICPCPackage",
		"classCode": "03004",
	}
	jsonData, err := json.Marshal(requestMap)
	if err != nil {
		h.log.Error("JSON Marshal error: %w", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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
	var data domain.PackageCodeResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "Packages"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Code", "nameLat"}

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

	// Ma'lumotlarni qo'shish
	for i, imp := range data.Packages {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Code)
		f.SetCellValue(sheetName, "B"+row, imp.NameLat)
	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=package-codes.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
}
