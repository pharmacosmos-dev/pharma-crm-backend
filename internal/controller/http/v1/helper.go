package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	pdf "github.com/jung-kurt/gofpdf"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// recoverTransaction handles panics and rolls back the transaction if necessary.
func recoverTransaction(tx *gorm.DB, log logger.Interface) {
	if r := recover(); r != nil {
		log.Error("panic recovered:", r)
		tx.Rollback()
	}
}

// RollbackIfError checks if the error pointer is not nil and if it contains an error.
func RollbackIfError(tx *gorm.DB, err *error) {
	if err != nil && *err != nil {
		tx.Rollback()
	}
}

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
			log.Errorf("could not create uploads directory: %v", err)
			handleServiceResponse(c, nil, domain.InternalServerError)
			return
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		log.Errorf("could not save Excel file: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
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

// formatWithSpaceSeparator formats a float64 number with spaces as thousands separators and two decimal places.
// For example, 1234567.89 becomes "1 234 567.89
func formatWithSpaceSeparator(n float64) string {
	// Format the float with 2 decimal places
	s := fmt.Sprintf("%.2f", n)

	// Split into integer and fractional parts
	parts := strings.Split(s, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Reverse the integer part for grouping
	reversed := reverseString(intPart)

	var grouped []string
	for i := 0; i < len(reversed); i += 3 {
		end := i + 3
		if end > len(reversed) {
			end = len(reversed)
		}
		grouped = append(grouped, reversed[i:end])
	}

	// Join groups with space and reverse back
	intWithSpaces := reverseString(strings.Join(grouped, " "))

	return intWithSpaces + "." + decPart
}

// reverseString reverses the characters in a string.
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Helper function for drawing paired multi-line cells
func drawPairedMultiLineCells(pdf *pdf.Fpdf, leftText, rightText string, width, height float64) {
	leftLines := splitText(pdf, leftText, width, 10)
	rightLines := splitText(pdf, rightText, width, 10)

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for i := 0; i < maxLines; i++ {
		// Left cell
		var leftCellText string
		if i < len(leftLines) {
			leftCellText = leftLines[i]
		}

		// Right cell
		var rightCellText string
		if i < len(rightLines) {
			rightCellText = rightLines[i]
		}

		// Border logic
		leftBorder := "LR"
		rightBorder := "LR"
		if i == 0 {
			leftBorder = "LTR"
			rightBorder = "LTR"
		}
		if i == maxLines-1 {
			leftBorder = "LBR"
			rightBorder = "LBR"
		}

		pdf.CellFormat(width, height, leftCellText, leftBorder, 0, "L", false, 0, "")
		pdf.CellFormat(width, height, rightCellText, rightBorder, 1, "L", false, 0, "")
	}
}

// splitText splits a string into multiple lines that fit within the specified width.
func splitText(pdf *pdf.Fpdf, text string, maxWidth float64, fontSize float64) []string {
	pdf.SetFont("DejaVu", "", fontSize)
	words := strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if pdf.GetStringWidth(testLine) <= maxWidth-2 { // -2 for padding
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		lines = append(lines, text)
	}
	return lines
}
