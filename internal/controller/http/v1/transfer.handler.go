package v1

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type TransferHandler struct {
	*Handler
}

func (h *Handler) NewTransferHandler(r *gin.RouterGroup) {
	transferHandler := &TransferHandler{h}
	transferHandler.TransferRoutes(r)
}

func (h *TransferHandler) TransferRoutes(r *gin.RouterGroup) {
	returned := r.Group("/transfer")
	{
		returned.POST("", h.Create)
		returned.GET("/:id", h.Get)
		returned.GET("/list", h.List)
		returned.GET("/export-excel", h.ExportTransferExcel)
		returned.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		returned.POST("/send/:id", h.Send)
		returned.POST("/confirm/:id", h.Confirm)
		returned.POST("/cancel/:id", h.Cancel)
	}
	detail := r.Group("transfer-detail")
	{
		detail.GET("/list", h.TransferDetailList)
		detail.GET("/export-excel", h.ExportTransferDetailList)
	}

}

// Create godoc
// @Summary Create Return
// @Description Create Return
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	transfer body domain.TransferRequest true "Transfer"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer [POST]
func (h *TransferHandler) Create(c *gin.Context) {
	var request domain.TransferRequest
	// Bind the request body to the ReturnRequest struct
	if err := c.ShouldBindJSON(&request); err != nil {
		h.log.Warn("Error on binding request: %v", err.Error())
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}

	request.CreatedBy = userId.(string)     // get creator id from set header
	request.PublicId = utils.GenerateCode() // generate public id

	// create return
	err := h.service.CreateTransfer(&request)
	if err != nil {
		h.log.Warn("Error on creating return: %v", err.Error())
		handleResponse(c, InternalError, "Failed to create return")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Return
// @Description Get Return
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Transfer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/{id} [GET]
func (h *TransferHandler) Get(c *gin.Context) {
	// get return by id
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	res, err := h.service.GetTransferById(id)
	if err != nil {
		h.log.Warn("Error on getting transfer: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get transfer")
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get Transfer list
// @Description Get Transfer list
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   status 	query string false "STATUS (0->new|1->sent|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/list [GET]
func (h *TransferHandler) List(c *gin.Context) {
	var param domain.ReturnParam
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	// get user info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Warn("ERROR on getting user: %v", err)
		handleResponse(c, InternalError, "Failed to get user info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreId = employee.StoreId
		}
	}

	// get default limit and offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get return list
	res, totalCount, err := h.service.TransferList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Export Transfer excel godoc
// @Summary Export Transfer excel
// @Description Export Transfer excel
// @Tags Transfer
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   status 	query string false "STATUS (0->new|1->sent|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/export-excel [get]
func (h *TransferHandler) ExportTransferExcel(c *gin.Context) {
	var param domain.ReturnParam
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	// get user info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Warn("ERROR on getting user: %v", err)
		handleResponse(c, InternalError, "Failed to get user info")
		return
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		if employee.StoreId != "" {
			param.StoreId = employee.StoreId
		}
	}

	// get default limit and offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get return list
	res, _, err := h.service.TransferList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return list")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "От Филиал", "До Филиал", "Кол-во", "Сумма Поставки", "Сумма Продажи", "Статус", "Создание", "Завершение", "Создал", "Завершил"}

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
	for i, r := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, r.PublicId)
		f.SetCellValue(sheetName, "B"+row, r.Name)
		if r.FromStore != nil {
			f.SetCellValue(sheetName, "C"+row, r.FromStore.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		if r.ToStore != nil {
			f.SetCellValue(sheetName, "D"+row, r.ToStore.Name)
		} else {
			f.SetCellValue(sheetName, "D"+row, "N/A")
		}
		f.SetCellValue(sheetName, "E"+row, r.ReceivedCount)
		f.SetCellValue(sheetName, "F"+row, r.ReceivedRetailSum)
		f.SetCellValue(sheetName, "G"+row, r.AcceptedRetailSum)
		f.SetCellValue(sheetName, "H"+row, helper.StatusToRussian(r.Status))
		if r.CreatedAt != nil {
			f.SetCellValue(sheetName, "I"+row, r.CreatedAt.Format(time.DateTime))
		} else {
			f.SetCellValue(sheetName, "I"+row, "N/A")
		}
		if r.AcceptedAt != nil {
			f.SetCellValue(sheetName, "J"+row, r.AcceptedAt.Format(time.DateTime))
		} else {
			f.SetCellValue(sheetName, "J"+row, "N/A")
		}
		if r.CreatedBy != nil {
			f.SetCellValue(sheetName, "K"+row, r.CreatedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "K"+row, "N/A")
		}
		if r.AcceptedBy != nil {
			f.SetCellValue(sheetName, "L"+row, r.AcceptedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "L"+row, "N/A")
		}

	}
	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Transferlar_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Param 	body body domain.ReturnAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/{id}/add-product-by-barcode [PATCH]
func (h *TransferHandler) AddProductByBarcode(c *gin.Context) {
	var request domain.ReturnAddProduct
	id := c.Param("id")
	// validate return id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Return id is invalid")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	// add scanned count by transfer detail id
	err = h.db.Exec(`
		UPDATE transfer_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE id = ? AND transfer_id = ?`,
		request.Count, request.Id, id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to add count")
		return
	}

	handleResponse(c, OK, "ADDED")
}

// send Return
// @Summary Send Return
// @Description Send Return
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/send/{id} [POST]
func (h *TransferHandler) Send(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm return service
	err := h.service.SendTransfer(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to send return")
		return
	}

	handleResponse(c, OK, "SENT")
}

// confirm Return
// @Summary Confirm Return
// @Description Confirm Return
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/confirm/{id} [POST]
func (h *TransferHandler) Confirm(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm return service
	err := h.service.ConfirmTransfer(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "COMFIRMED")
}

// cancel return
// @Summary Cancel Return
// @Description Cancel Return
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/cancel/{id} [POST]
func (h *TransferHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate return id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	// get user_id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm return service
	err := h.service.CancelTransfer(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "CANCELED")
}

// Get List
// @Summary Get Transfer list
// @Description Get Transfer list
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   return_id query string true "Return ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer-detail/list [GET]
func (h *TransferHandler) TransferDetailList(c *gin.Context) {
	var param domain.ReturnDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.TransferDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return detail list")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Get List
// @Summary Get Transfer list
// @Description Get Transfer list
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   return_id query string true "Return ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer-detail/export-excel [GET]
func (h *TransferHandler) ExportTransferDetailList(c *gin.Context) {
	var param domain.ReturnDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get return detail list
	res, _, err := h.service.TransferDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return detail list")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименование", "Штрих-код", "Срок годность", "Серия номер", "Текущее Кол-во", "Ед-изм", "Текущее Cумма", "Cканированные", "Cканированные Cумма"}

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
	for i, r := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, r.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, r.Name)
		f.SetCellValue(sheetName, "C"+row, r.Barcode)
		f.SetCellValue(sheetName, "D"+row, r.ExpireDate)
		f.SetCellValue(sheetName, "E"+row, r.SerialNumber)
		f.SetCellValue(sheetName, "F"+row, r.ReceivedCount)
		f.SetCellValue(sheetName, "G"+row, r.ShortName)
		f.SetCellValue(sheetName, "H"+row, r.ReceivedSum)
		f.SetCellValue(sheetName, "I"+row, r.ScannedCount)
		f.SetCellValue(sheetName, "J"+row, r.ScannedSum)

	}
	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Transfer_mahsulotlar_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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
