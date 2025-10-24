package v1

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pdf "github.com/jung-kurt/gofpdf"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
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
		transfer.POST("/send/:id", h.Send)
		transfer.POST("/confirm/:id", h.Confirm)
		transfer.POST("/send1c/:id", h.Send1C)
		transfer.POST("/cancel/:id", h.Cancel)
		transfer.GET("/:id", h.Get)
		transfer.GET("/list", h.List)
		transfer.GET("/list-status", h.TransferStats)
		transfer.GET("/export-excel", h.ExportTransferExcel)
		transfer.GET("/logs/:transfer_id", h.GetTransferLogs)
		transfer.GET("/export-nakladnoy", h.ExportTransferNakladnoyPDF)
		transfer.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var request domain.TransferRequest
	// Bind the request body to the ReturnRequest struct
	if err := c.ShouldBindJSON(&request); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	// get creator id from set header
	request.CreatedBy = user.UserId

	// create return
	err := h.service.CreateTransfer(ctx, &request)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
	if id == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetTransferById(ctx, id)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get Transfer list
// @Description Get Transfer list
// @Tags 	 Transfer
// @Security BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit 		query int false "limit"
// @Param 	offset 		query int false "offset"
// @Param   store_id 	query string false "store_id"
// @Param   search 		query string false "search"
// @Param   status 		query string false "status (0->new|1->sent|2->completed)"
// @Param 	start_date  query string false "start_date"
// @Param	end_date 	query string false "end_date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/list [GET]
func (h *TransferHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReturnParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get default limit and offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get return list
	res, totalCount, err := h.service.TransferList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// TransferStatus godoc
// @Summary Get transfer total stats
// @Description Get total sum and count stats for transfers
// @Tags Transfer
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   store_id 	query string false "store_id"
// @Param   search 		query string false "search"
// @Param   status 		query string false "status (0->new|1->sent|2->completed)"
// @Param 	start_date  query string false "start_date"
// @Param	end_date 	query string false "end_date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/list-status [get]
func (h *TransferHandler) TransferStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReturnParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
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

	// get aggregated transfer stats
	res, err := h.service.TransferStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
// @Param   store_id 	query string false "store_id"
// @Param   search 		query string false "search"
// @Param   status 		query string false "status (0->new|1->sent|2->completed)"
// @Param 	start_date  query string false "start_date"
// @Param	end_date 	query string false "end_date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /transfer/export-excel [get]
func (h *TransferHandler) ExportTransferExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReturnParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get default limit and offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get return list
	res, _, err := h.service.TransferList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
		h.log.Errorf("could not create style: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	for i, r := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, r.PublicId)
		f.SetCellValue(sheetName, "B"+row, r.Name)
		if r.FromStore.Valid {
			f.SetCellValue(sheetName, "C"+row, r.FromStore.Value.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		if r.ToStore.Valid {
			f.SetCellValue(sheetName, "D"+row, r.ToStore.Value.Name)
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
		if r.CreatedBy.Valid {
			f.SetCellValue(sheetName, "K"+row, r.CreatedBy.Value.FullName)
		} else {
			f.SetCellValue(sheetName, "K"+row, "N/A")
		}
		if r.UpdatedBy.Valid {
			f.SetCellValue(sheetName, "L"+row, r.UpdatedBy.Value.FullName)
		} else {
			f.SetCellValue(sheetName, "L"+row, "N/A")
		}
		if r.AcceptedBy.Valid {
			f.SetCellValue(sheetName, "M"+row, r.AcceptedBy.Value.FullName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}

	}

	saveExcelToUploads(c, f, *h.log, "Permisheniya")
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	var request domain.ReturnAddProduct
	// bind request body
	if err := c.ShouldBindJSON(&request); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	request.TransferId = id

	err := h.service.UpdateReturnDetailQuantity(ctx, &request, user.UserId, constants.TransferTypeMove)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
// @Param request body domain.TransferBarcodeRequest true "Barcode request payload"
// @Success 200 {object} v1.Response "Update successful"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /transfer/update-by-barcode/{id} [put]
func (h *TransferHandler) UpdateByBarcode(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var (
		req domain.TransferBarcodeRequest
		id  = c.Param("id")
	)
	// bind request body
	if err := c.ShouldBindJSON(&req); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.TransferId = id
	err := h.service.UpdateTransferByBarcode(ctx, &req, user, constants.TransferTypeMove)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// send Transfer
// @Summary Send Return
// @Description Send Return
// @Tags 	 Transfer
// @Security BearerAuth
// @Accept 	 json
// @Produce  json
// @Param 	 id 	path string true "Return ID"
// @Success  200 {object} v1.Response
// @Failure  400 {object} v1.Response
// @Failure  500 {object} v1.Response
// @Router /transfer/send/{id} [POST]
func (h *TransferHandler) Send(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// confirm return service
	err := h.service.SendTransfer(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.EditStatusToCheckingReturn(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check is accepted_count is not null
	err := h.service.CheckAcceptedCount(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// confirm return service
	err = h.service.ConfirmTransfer(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
	transferId := c.Param("id")
	if transferId == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// send transfer info to 1C
	err := h.service.SendTransferTo1C(ctx, transferId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	transferId := c.Param("id")
	// validate return id
	if transferId == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// confirm return service
	err := h.service.CancelTransfer(ctx, transferId, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
	var params domain.ReturnDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, totalCount, err := h.service.TransferDetailList(&params)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return detail list")
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

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
	if transfer.Status != constants.GeneralStatusCompleted && transfer.Status != constants.GeneralStatusSent {
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
	fromStore := "Поставщик: " + transfer.FromStore.Value.Name
	toStore := "Получатель: " + transfer.ToStore.Value.Name
	fromStoreAddress := "Адрес: " + transfer.FromStore.Value.Name
	toStoreAddress := "Адрес: " + transfer.ToStore.Value.Name
	fromStorePhone := fmt.Sprintf("Тел: +%s,%s", transfer.FromStore.Value.Name, "filial@pharma")
	toStorePhone := fmt.Sprintf("Тел: +%s,%s", transfer.ToStore.Value.Name, "filial@pharma")

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
		expireDateStr := ""
		if p.ExpireDate != nil {
			expireDateStr = p.ExpireDate.Format("02.01.2006")
		}
		row := []string{
			strconv.Itoa(count),
			p.Name,
			p.SerialNumber,
			expireDateStr,
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

// GetTransferLogs godoc
// @Summary Get Transfer Logs
// @Description Get Transfer Logs
// @Tags Transfer
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   transfer_id path string true "transfer_id"
// @Success 200 {object} v1.Response "Transfer logs retrieved"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /transfer/logs/{transfer_id} [get]
func (s *TransferHandler) GetTransferLogs(c *gin.Context) {
	transferId := c.Param("transfer_id")
	if err := uuid.Validate(transferId); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	var params domain.ReturnDetailParam

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	params.TransferId = transferId

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	logs, totalCount, err := s.service.GetTransferLogs(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(logs, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
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
