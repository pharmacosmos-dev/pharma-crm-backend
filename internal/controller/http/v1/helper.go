package v1

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	pdf "github.com/jung-kurt/gofpdf"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
)

func setExcelHeaders(f *excelize.File, sheet string, headers []string) error {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
	})
	if err != nil {
		return err
	}
	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheet, col, h)
		f.SetCellStyle(sheet, col, col, headerStyle)
	}
	return nil
}

func saveExcelToUploads(c *gin.Context, f *excelize.File, log logger.Logger, prefix string) {
	fileName := prefix + "_" + time.Now().Format("2006-01-02_15-04-05") + ".xlsx"
	filePath := filepath.Join("uploads", fileName)

	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		if err := os.Mkdir("uploads", os.ModePerm); err != nil {
			log.Error("Failed to create uploads directory:", err)
			handleResponse(c, InternalError, "Failed to create uploads folder")
			return
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		log.Error("Failed to save Excel file:", err)
		handleResponse(c, InternalError, "Failed to save Excel file")
		return
	}

	handleResponse(c, OK, gin.H{"file_name": fileName})
}

func savePdfToUploads(c *gin.Context, f *pdf.Fpdf, log logger.Logger, prefix string) {
	fileName := prefix + "_" + time.Now().Format("2006-01-02_15-04-05") + ".pdf"
	filePath := filepath.Join("uploads", fileName)
	// Ensure uploads directory exists
	_, err := os.Stat("uploads")
	if os.IsNotExist(err) {
		if err := os.Mkdir("uploads", os.ModePerm); err != nil {
			log.Warn("Failed to create uploads directory: %v", err)
			handleResponse(c, InternalError, "Failed to create uploads folder")
			return
		}
	}
	// Save the PDF file
	if err = f.OutputFileAndClose(filePath); err != nil {
		log.Warn("Failed to save PDF file: %v", err)
		handleResponse(c, InternalError, "Failed to save PDF file")
		return
	}

	handleResponse(c, OK, gin.H{"file_name": fileName})
}
