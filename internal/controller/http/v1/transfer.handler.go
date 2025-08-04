package v1

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pdf "github.com/jung-kurt/gofpdf"
	"github.com/pharma-crm-backend/config"
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
	transfer := r.Group("/transfer")
	{
		transfer.POST("", h.Create)
		transfer.GET("/:id", h.Get)
		transfer.GET("/list", h.List)
		transfer.GET("/list-status", h.TransferStatus)
		transfer.GET("/export-excel", h.ExportTransferExcel)
		transfer.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		transfer.POST("/send/:id", h.Send)
		transfer.POST("/confirm/:id", h.Confirm)
		transfer.POST("/send1c/:id", h.Send1C)
		transfer.POST("/cancel/:id", h.Cancel)
		transfer.GET("/export-nakladnoy", h.ExportTransferNakladnoyPDF)
		transfer.PUT("/update-by-barcode/:id", h.UpdateByBarcode)
		transfer.PUT("/edit-status-to-checking/:id", h.EditStatusToChecking)
		transfer.DELETE("/:id", h.DeleteTransfer)

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
	err := c.ShouldBindJSON(&request)
	if err != nil {
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
	// get creator id from set header
	request.CreatedBy = userId.(string)

	// create return
	err = h.service.CreateTransfer(&request)
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
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	// get user info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
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

// TransferStatus godoc
// @Summary Get transfer total stats
// @Description Get total sum and count stats for transfers
// @Tags Transfer
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   store_id query string false "Store ID"
// @Param   search query string false "Search Keyword"
// @Param   status query string false "Transfer Status (0->new|1->sent|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/list-status [get]
func (h *TransferHandler) TransferStatus(c *gin.Context) {
	var param domain.ReturnParam

	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found")
		return
	}

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	// check if employee is not admin or superadmin
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		h.log.Warn("Failed to get user: %v", err)
		handleResponse(c, InternalError, "Failed to get user info")
		return
	}
	if !helper.IsAdmin(employee, h.cfg) && employee.StoreId != "" {
		param.StoreId = employee.StoreId
	}

	// get aggregated transfer stats
	res, err := h.service.TransferStatus(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
	headers := []string{"ID", "Наименование", "От Филиал", "До Филиал", "Кол-во", "Сумма Поставки", "Сумма Продажи", "Статус", "Создание", "Завершение", "Создал", "Отправитель", "Завершил"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
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
		if r.UpdatedBy != nil {
			f.SetCellValue(sheetName, "L"+row, r.UpdatedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "L"+row, "N/A")
		}
		if r.AcceptedBy != nil {
			f.SetCellValue(sheetName, "M"+row, r.AcceptedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}

	}

	saveExcelToUploads(c, f, *h.log, "Transferlar")
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
	err := uuid.Validate(id)
	if err != nil {
		handleResponse(c, BadRequest, "Return id is invalid")
		return
	}
	// bind request body
	err = c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	err = h.service.UpdateReturnDetailQuantity(id, &request)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "ADDED")
}

// UpdateByBarcode godoc
// @Summary Update return or transfer by barcode
// @Tags Transfer
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Transfer ID or Return ID"
// @Param request body domain.BarcodeRequest true "Barcode request payload"
// @Success 200 {object} v1.Response "Update successful"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /return/update-by-barcode/{id} [put]
func (h *TransferHandler) UpdateByBarcode(c *gin.Context) {
	var (
		req domain.BarcodeRequest
		id  = c.Param("id")
	)
	// bind request body
	err := c.ShouldBindJSON(&req)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// default count is 1
	if req.Count == 0 {
		req.Count = 1
	}

	// get default update field
	updatedField := "scanned_count"
	if req.Status == "checking" {
		updatedField = "accepted_count"
	}

	if req.Id != "" {
		err = h.db.Exec(fmt.Sprintf(`UPDATE transfer_details SET %s = COALESCE(%s, 0) + ? WHERE id = ? AND received_count >= COALESCE(%s,0) + ?;`, updatedField, updatedField, updatedField), req.Count, req.Id, req.Count).Error
		if err != nil {
			h.log.Error("could not update transfer_details(%s) scanned_count: %v", req.Id, err)
			handleResponse(c, InternalError, "internal.server.error")
			return
		}
	} else if req.Barcode != "" {
		var barcodeResponse []domain.TransferBarcodeResponse
		err = h.db.Raw(`SELECT t.id, p.name FROM transfer_details t JOIN products p ON p.id = t.product_id WHERE p.barcode = ? AND t.transfer_id = ?`, req.Barcode, id).Scan(&barcodeResponse).Error
		if err != nil {
			h.log.Error("could not get transfer_details by barcode(%s): %v", req.Barcode, err)
			handleResponse(c, InternalError, "internal.server.error")
			return
		}
		if len(barcodeResponse) > 1 {
			handleResponse(c, MultiStatus, barcodeResponse)
			return
		}
		err = h.db.Exec(fmt.Sprintf(`
		UPDATE transfer_details t 
		SET %s = COALESCE(%s, 0) + ? 
		FROM products p 
		WHERE 
			t.transfer_id = ? AND 
			p.id = t.product_id AND 
			p.barcode = ? AND 
			t.received_count >= COALESCE(t.%s,0) + ?;`, updatedField, updatedField, updatedField), req.Count, id, req.Barcode, req.Count).Error
		if err != nil {
			h.log.Error("could not update transfer_details by barcode(%s): %v", req.Barcode, err)
			handleResponse(c, InternalError, "internal.server.error")
			return
		}
		handleResponse(c, OK, "UPDATED")
		return
	} else {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	handleResponse(c, OK, "UPDATED")
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

// @Summary Edit status to checking
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   id path string true "Transfer ID"
// @Success 200 {object} v1.Response "Return PDF file"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /transfer/edit-status-to-checking/{id} [PUT]
func (h *TransferHandler) EditStatusToChecking(c *gin.Context) {

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user.not.authorized")
		return
	}

	id := c.Param("id")
	if id == "" {
		handleResponse(c, BadRequest, "invalid.id")
		return
	}

	err := h.service.EditStatusToCheckingReturn(id, userId.(string))
	if err != nil {
		log.Println("update by barcode error:", err)
		handleResponse(c, InternalError, "internal.server.error")
		return
	}

	handleResponse(c, OK, "updated successfully")
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

	// check is accepted_count is not null
	err := h.service.CheckAcceptedCount(id)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	// confirm return service
	err = h.service.ConfirmTransfer(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "COMFIRMED")
}

// send1c transfer
// @Summary Send Transfer
// @Description Send Transfer
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Transfer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/send1c/{id} [POST]
func (h *TransferHandler) Send1C(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid transfer id")
		return
	}
	// send transfer info to 1C
	err := h.service.SendTransferTo1C(id)
	if err != nil {
		h.log.Warn("ERROR on sending transfer to 1c: %v", err)
		handleResponse(c, InternalError, "failed_to_send_transfer")
		return
	}
	handleResponse(c, OK, "SENT")
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
// @Param   transfer_id query string true "Return ID"
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
	headers := []string{"Код", "Наименование", "Штрих-код", "Срок годность", "Серия номер", "Текущее Кол-во", "Ед-изм", "Текущее Cумма", "Отправленное кол-во", "Cканированные", "Cканированные Cумма"}

	setExcelHeaders(f, sheetName, headers)

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
		f.SetCellValue(sheetName, "I"+row, r.ExpectedCount)
		f.SetCellValue(sheetName, "J"+row, r.ScannedCount)
		f.SetCellValue(sheetName, "K"+row, r.ScannedSum)

	}

	saveExcelToUploads(c, f, *h.log, "Transfer_mahsulotlar")
}

// ExportNakladnoy godoc
// @Summary Export Nakladnoy
// @Description Export Nakladnoy
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   transfer_id query string true "Transfer ID"
// @Success 200 {object} v1.Response "Nakladnoy PDF file"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /transfer/export-nakladnoy [GET]
func (h *TransferHandler) ExportTransferNakladnoyPDF(c *gin.Context) {
	var transferId = c.Query("transfer_id")
	// validate transfer id
	err := uuid.Validate(transferId)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.transfer.id")
		return
	}
	var transfer domain.Transfer
	// get transfer by id
	err = h.db.
		Model(&domain.Transfer{}).
		Preload("FromStore").
		Preload("ToStore").
		First(&transfer, "id = ?", transferId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "transfer.not.found")
			return
		}
		handleResponse(c, InternalError, "failed.get.transfer")
		return
	}

	// check if transfer is not completed
	if transfer.Status != config.COMPLETED && transfer.Status != config.SENT {
		handleResponse(c, BadRequest, "transfer.not.completed")
		return
	}

	res, _, err := h.service.TransferDetailList(&domain.ReturnDetailParam{
		TransferId: transferId,
		Limit:      10000, // set a high limit to get all products
		Offset:     0})
	if err != nil {
		handleResponse(c, InternalError, "failed.get.transfer.products")
		return
	}

	// nakladnoy name
	nakladnoyName := fmt.Sprintf("НАКЛАДНАЯ № %s от %s г.", transfer.PublicId, time.Now().Format("02.01.2006"))
	fromStore := "Поставщик: " + transfer.FromStore.Name
	toStore := "Получатель: " + transfer.ToStore.Name
	fromStoreAddress := "Адрес: " + transfer.FromStore.Address
	toStoreAddress := "Адрес: " + transfer.ToStore.Address
	fromStorePhone := fmt.Sprintf("Тел: +%s,%s", transfer.FromStore.Phone, "filial@pharma")
	toStorePhone := fmt.Sprintf("Тел: +%s,%s", transfer.ToStore.Phone, "filial@pharma")

	pdf := pdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font("DejaVu", "", "./app/uploads/DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "./app/uploads/DejaVuSansBold.ttf")
	pdf.AddPage()

	pdf.SetFont("DejaVu", "B", 14)
	pdf.CellFormat(0, 10, nakladnoyName, "", 1, "C", false, 0, "")

	// Sender/Receiver section
	pdf.Ln(-1)
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(95, 8, fromStore, "1", 0, "L", false, 0, "")
	pdf.CellFormat(95, 8, toStore, "1", 1, "L", false, 0, "")

	// Addresses
	drawPairedMultiLineCells(pdf, fromStoreAddress, toStoreAddress, 95, 6)

	// Phone numbers
	drawPairedMultiLineCells(pdf, fromStorePhone, toStorePhone, 95, 6)

	// Bank details
	drawPairedMultiLineCells(pdf, "Р/счет: 20208000200621819001 в ИПОТЕКА БАНК", "Р/счет: 20208000200621819001 в ИПОТЕКА БАНК", 95, 6)
	// MFO and INN details
	drawPairedMultiLineCells(pdf, "МФО: 00408       ИНН: 303970073", "МФО: 00408       ИНН: 303970073", 95, 6)
	// OKONX details
	drawPairedMultiLineCells(pdf, "ОКОНХ: 46460", "ОКОНХ: 46460", 95, 6)
	pdf.Ln(5)

	// Optimized jadval sarlavhasi - ko'p qatorli
	headers1 := []string{"№", "Наименование товара", "Серия", "Срок", "Ед.", "Кол", "Базовая", "Приходная", "Наценка", "Отпускная", "Стоимость"}
	headers2 := []string{"", "", "", "", "изм.", "", "цена", "цена", "", "цена", "поставки"}
	widths := []float64{6, 45, 15, 20, 8, 9, 18, 18, 15, 18, 18}

	pdf.SetFont("DejaVu", "B", 8)

	// First row headers
	for i, h := range headers1 {
		if i == 5 || i == 6 || i == 7 || i == 9 || i == 10 { // "Базовая", "Приходная", "Стоимость" ustunlari uchun ko'p qatorli
			pdf.CellFormat(widths[i], 3.5, h, "LTR", 0, "C", false, 0, "")
		} else {
			pdf.CellFormat(widths[i], 7, h, "1", 0, "C", false, 0, "")
		}
	}
	pdf.Ln(-1)

	// Second row headers (faqat kerakli joylar uchun)
	for i, h := range headers2 {
		if i == 5 || i == 6 || i == 7 || i == 9 || i == 10 { // "цена", "цена", "поставки" uchun
			pdf.CellFormat(widths[i], 3.5, h, "LBR", 0, "C", false, 0, "")
		} else {
			pdf.CellFormat(widths[i], 3.5, "", "", 0, "C", false, 0, "")
		}
	}
	pdf.Ln(-1)

	pdf.SetFont("DejaVu", "", 6)
	var total float64
	var count = 1
	for _, p := range res {
		row := []string{
			strconv.Itoa(count),
			p.Name,
			p.SerialNumber,
			p.ExpireDate.Format("02.01.2006"),
			p.ShortName,
			strconv.FormatFloat(p.ExpectedCount, 'f', 2, 64),
			formatWithSpaceSeparator(p.SupplyPrice),
			formatWithSpaceSeparator(p.RetailPrice),
			formatWithSpaceSeparator(p.RetailPrice - p.SupplyPrice),
			formatWithSpaceSeparator(p.RetailPrice),
			formatWithSpaceSeparator(p.RetailPrice * p.ExpectedCount),
		}

		// Har bir ustun uchun maksimal qator sonini topish
		maxLines := 1
		var splitCells [][]string

		for i, val := range row {
			lines := splitText(pdf, val, widths[i], 6)
			splitCells = append(splitCells, lines)
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		// Each row will have the same number of lines
		// so we iterate through the maxLines
		for lineNum := 0; lineNum < maxLines; lineNum++ {
			for i := range row {
				align := "C"
				if i == 1 {
					align = "L"
				}

				var cellText string
				if lineNum < len(splitCells[i]) {
					cellText = splitCells[i][lineNum]
				}

				// drawing border based on line number
				// LTR - left top right, LBR - left bottom right, LR -
				border := "LR"
				if lineNum == 0 {
					border = "LTR"
				}
				if lineNum == maxLines-1 {
					border = "LBR"
				}
				if lineNum > 0 && lineNum < maxLines-1 {
					border = "LR"
				}
				// draw cell with text and border
				pdf.CellFormat(widths[i], 6, cellText, border, 0, align, false, 0, "")
			}
			pdf.Ln(-1)
		}
		// Update totals and count
		total += math.Round(p.RetailPrice * p.ExpectedCount)
		count++
	}

	totalWidth := 0.0
	for _, w := range widths {
		totalWidth += w
	}
	pdf.CellFormat(totalWidth, 7, "Итого: "+formatWithSpaceSeparator(total), "1", 1, "R", false, 0, "")

	pdf.SetFont("DejaVu", "B", 10)
	pdf.Ln(10)
	pdf.CellFormat(100, 7, "Руководитель предприятия: _______________", "", 0, "L", false, 0, "")
	pdf.CellFormat(100, 7, "Получил: _______________", "", 1, "L", false, 0, "")
	pdf.CellFormat(100, 7, "Гл. бухгалтер: _______________", "", 0, "L", false, 0, "")
	pdf.CellFormat(100, 7, "Товар отпустил: _______________", "", 1, "L", false, 0, "")

	savePdfToUploads(c, pdf, *h.log, "Nakladnoy_"+transfer.PublicId)
}

// DeleteTransfer godoc
// @Summary Delete Transfer
// @Description Delete Transfer
// @Tags 	Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   id path string true "Transfer ID"
// @Success 200 {object} v1.Response "Transfer deleted"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /transfer/{id} [delete]
func (h *TransferHandler) DeleteTransfer(c *gin.Context) {
	var transferId = c.Param("id")

	err := h.service.DeleteTransfer(transferId)
	if err != nil {
		handleResponse(c, InternalError, "transfer.delete.error")
		return
	}
	handleResponse(c, OK, "transfer.delete.success")
}
