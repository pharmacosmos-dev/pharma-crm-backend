package v1

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
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

type ReturnHandler struct {
	*Handler
}

func (h *Handler) NewReturnHandler(r *gin.RouterGroup) {
	returnHandler := &ReturnHandler{h}
	returnHandler.ReturnRoutes(r)
}

func (h *ReturnHandler) ReturnRoutes(r *gin.RouterGroup) {
	returned := r.Group("/return")
	{
		returned.POST("", h.Create)
		returned.GET("/:id", h.Get)
		returned.GET("/list", h.List)
		returned.GET("/list-status", h.ReturnStatus)
		returned.GET("/export-excel", h.ExportReturnExcel)
		returned.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		returned.POST("/send/:id", h.Send)
		returned.POST("/send1c/:id", h.ResendReturnToOnec)
		returned.POST("/confirm/:id", h.Confirm)
		returned.POST("/cancel/:id", h.Cancel)
		returned.GET("/export-nakladnoy", h.ExportReturnNakladnoyPDF)
		returned.PUT("/update-by-barcode/:id", h.UpdateByBarcode)
		returned.PUT("/edit-status-to-checking/:id", h.EditStatusToChecking)
		returned.PATCH("/:id/comment", h.UpdateComment)
	}
	detail := r.Group("return-detail")
	{
		detail.GET("/list", h.ReturnDetailList)
		detail.GET("/export-excel", h.ExportReturnDetailList)
	}
}

// Create godoc
// @Summary Create Return
// @Description Create Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	return body domain.ReturnRequest true "Return"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return [POST]
func (h *ReturnHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var returnRequest domain.ReturnRequest
	if err := c.ShouldBindJSON(&returnRequest); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	returnRequest.CreatedBy = user.UserId
	// create return
	err := h.service.CreateReturn(ctx, &returnRequest)
	if err != nil {
		handleResponse(c, InternalError, err)
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Return
// @Description Get Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/{id} [GET]
func (h *ReturnHandler) Get(c *gin.Context) {
	// get return by id
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetReturnById(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Get List
// @Summary Get Return list
// @Description Get Return list
// @Tags Return
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
// @Router /return/list [GET]
func (h *ReturnHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReturnParam

	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
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
	res, totalCount, err := h.service.ReturnList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// ReturnStatus godoc
// @Summary Get Return summary stats
// @Description Get total count and retail sums for returns
// @Tags Return
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   store_id query string false "Store ID"
// @Param   search query string false "Search"
// @Param   status query string false "Status (0->new|1->sent|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/list-status [get]
func (h *ReturnHandler) ReturnStatus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReturnParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get return summary
	res, err := h.service.GetReturnStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Export return excel godoc
// @Summary Export return excel
// @Description Export return excel
// @Tags Return
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
// @Router /return/export-excel [get]
func (h *ReturnHandler) ExportReturnExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.ReturnParam

	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
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
	res, _, err := h.service.ReturnList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "Филиал", "Кол-во", "Полученная Сумма Поставки", "Принятая Сумма Поставки", "Полученная Сумма Продажи", "Принятая Сумма Продажи", "Статус", "Создание", "Завершение", "Создал", "Отправитель", "Завершил"}

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
		if r.Store != nil {
			f.SetCellValue(sheetName, "C"+row, r.Store.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}

		f.SetCellValue(sheetName, "D"+row, r.ReturnCount)
		f.SetCellValue(sheetName, "E"+row, r.ReceivedSupplySum)
		f.SetCellValue(sheetName, "F"+row, r.AcceptedSupplySum)
		f.SetCellValue(sheetName, "G"+row, r.ReceivedSupplySum)
		f.SetCellValue(sheetName, "H"+row, r.AcceptedRetailSum)
		f.SetCellValue(sheetName, "I"+row, helper.StatusToRussian(r.Status))
		if r.CreatedAt != nil {
			f.SetCellValue(sheetName, "J"+row, r.CreatedAt.Format(time.DateTime))
		} else {
			f.SetCellValue(sheetName, "J"+row, "N/A")
		}
		if r.AcceptedAt != nil {
			f.SetCellValue(sheetName, "K"+row, r.AcceptedAt.Format(time.DateTime))
		} else {
			f.SetCellValue(sheetName, "K"+row, "N/A")
		}
		if r.CreatedBy != nil {
			f.SetCellValue(sheetName, "L"+row, r.CreatedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "L"+row, "N/A")
		}
		if r.UpdatedBy != nil {
			f.SetCellValue(sheetName, "M"+row, r.UpdatedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}
		if r.AcceptedBy != nil {
			f.SetCellValue(sheetName, "N"+row, r.AcceptedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "N"+row, "N/A")
		}

	}
	saveExcelToUploads(c, f, *h.log, "Vozvratlar")
}

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Param 	body body domain.ReturnAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/{id}/add-product-by-barcode [PATCH]
func (h *ReturnHandler) AddProductByBarcode(c *gin.Context) {
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
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	request.TransferId = id

	err := h.service.UpdateReturnDetailQuantity(ctx, &request, user.UserId, constants.TransferTypeReturn)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "ADDED")
}

// send Return
// @Summary Send Return
// @Description Send Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/send/{id} [POST]
func (h *ReturnHandler) Send(c *gin.Context) {
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

	// lock parallel request
	mu := h.getReturnLock(id)
	mu.Lock()
	defer mu.Unlock()

	// confirm return service
	err := h.service.SendReturn(ctx, id, user.UserId)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			handleResponse(c, CONFLICT, notAddErr.Data)
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "SENT")
}

// Resend Return
// @Summary ReSend Return to 1C
// @Description ReSend Return to 1C
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/send1c/{id} [POST]
func (h *ReturnHandler) ResendReturnToOnec(c *gin.Context) {
	returnId := c.Param("id")
	if returnId == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// Resend return to onec
	err := h.service.ReSendReturnToOnec(ctx, returnId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "SENT")
}

// confirm Return
// @Summary Confirm Return
// @Description Confirm Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/confirm/{id} [POST]
func (h *ReturnHandler) Confirm(c *gin.Context) {
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

	mu := h.getReturnLock(id)
	mu.Lock()
	defer mu.Unlock()

	// confirm return service
	err := h.service.ConfirmReturn(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel return
// @Summary Cancel Return
// @Description Cancel Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/cancel/{id} [POST]
func (h *ReturnHandler) Cancel(c *gin.Context) {
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
	err := h.service.CancelReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "CANCELED")
}



// UpdateComment godoc
// @Summary Update Return Comment
// @Description Update comment for a return by ID
// @Tags Return
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Return ID"
// @Param comment body domain.ReturnCommentRequest true "Comment"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/{id}/comment [PATCH]
func (h *ReturnHandler) UpdateComment(c *gin.Context) {
	var id = c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}

	var req domain.ReturnCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}

	if err := h.service.UpdateReturnComment(id, req.Comment, userId.(string)); err != nil {
		handleResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// Get List
// @Summary Get return list
// @Description Get return list
// @Tags Return
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
// @Router /return-detail/list [GET]
func (h *ReturnHandler) ReturnDetailList(c *gin.Context) {
	var param domain.ReturnDetailParam
	// bind query param
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.ReturnDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return detail list")
		return
	}
	// get return details status count
	statsCount, err := h.service.ReturnDetailStatsCount(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return detail stats count")
		return
	}
	data := map[string]any{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     param.Limit,
			CurrentPage: (param.Offset / param.Limit) + 1,
			PageCount:   int((totalCount + int64(param.Limit) - 1) / int64(param.Limit)),
		},
		"stats_count": statsCount,
		"data":        res,
	}

	handleResponse(c, OK, data)
}

// Get List
// @Summary Get return list
// @Description Get return list
// @Tags Return
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
// @Router /return-detail/export-excel [GET]
func (h *ReturnHandler) ExportReturnDetailList(c *gin.Context) {
	var param domain.ReturnDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get return detail list
	res, _, err := h.service.ReturnDetailList(&param)
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

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
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
		f.SetCellValue(sheetName, "I"+row, r.ExpectedCount)
		f.SetCellValue(sheetName, "J"+row, r.ScannedCount)
		f.SetCellValue(sheetName, "K"+row, r.ScannedSum)

	}
	saveExcelToUploads(c, f, *h.log, "Vozvrat_mahsulotlar")
}

// ExportNakladnoy godoc
// @Summary Export Nakladnoy
// @Description Export Nakladnoy
// @Tags 	Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   return_id query string true "Return ID"
// @Param   type 	query string false "TYPE of document"
// @Success 200 {object} v1.Response "Nakladnoy PDF file"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /return/export-nakladnoy [GET]
func (h *ReturnHandler) ExportReturnNakladnoyPDF(c *gin.Context) {
	var returnId = c.Query("return_id")
	var typeDoc = c.Query("type")
	// validate return id
	err := uuid.Validate(returnId)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.return.id")
		return
	}
	var returnData domain.Return
	// get return by id
	err = h.db.
		Model(&domain.Transfer{}).
		Preload("Store").
		Select("transfers.*, 0 as return_count").
		First(&returnData, "id = ?", returnId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "return.not.found")
			return
		}
		handleResponse(c, InternalError, "failed.get.return")
		return
	}

	// check if return is not completed
	if returnData.Status != constants.GeneralStatusCompleted && returnData.Status != constants.GeneralStatusSent && returnData.Status != constants.GeneralStatusSentOnec {
		handleResponse(c, BadRequest, "return.not.completed")
		return
	}

	res, _, err := h.service.ReturnDetailList(&domain.ReturnDetailParam{
		ReturnId: returnId,
		Limit:    10000, // set a high limit to get all products
		Offset:   0})
	if err != nil {
		handleResponse(c, InternalError, "failed.get.return.products")
		return
	}

	// nakladnoy name
	nakladnoyName := fmt.Sprintf("НАКЛАДНАЯ № %s от %s г.", returnData.PublicId, time.Now().Format("02.01.2006"))
	fromStore := "Поставщик: " + returnData.Store.Name
	fromStoreAddress := "Адрес: " + returnData.Store.Address
	toStoreAddress := "Адрес: г.Ташкент, Учтепинский район, ул.Богобод, Д.269"
	fromStorePhone := fmt.Sprintf("Тел: +%s,%s", returnData.Store.Phone, "filial@pharma")
	toStorePhone := "Тел: +998772770333,sklad@pharmacos"
	if typeDoc == "return" {
		fromStore = "Поставщик: MChJ \"PharmaCosmos\""
		toStoreAddress = "Адрес: " + returnData.Store.Address
		fromStoreAddress = "Адрес: г.Ташкент, Учтепинский район, ул.Богобод, Д.269"
		toStorePhone = fmt.Sprintf("Тел: +%s,%s", returnData.Store.Phone, "filial@pharma")
		fromStorePhone = "Тел: +998772770333,sklad@pharmacos"
	}

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
	if typeDoc == "return" {
		pdf.CellFormat(95, 8, "Получатель: "+returnData.Store.Name, "1", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(95, 8, "Получатель: MChJ \"PharmaCosmos\"", "1", 1, "L", false, 0, "")
	}
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
		var quantityStr string
		var totalPrice float64
		if typeDoc == "return" {
			quantityStr = strconv.FormatFloat(p.ExpectedCount-p.AcceptedCount, 'f', 2, 64)
			if p.ExpectedCount-p.AcceptedCount <= 0 {
				continue
			}
			totalPrice = p.RetailPrice * (p.ExpectedCount - p.AcceptedCount)
		} else {
			quantityStr = strconv.FormatFloat(p.ExpectedCount, 'f', 2, 64)
			totalPrice = p.RetailPrice * p.ExpectedCount
		}
		row := []string{
			strconv.Itoa(count),
			p.Name,
			p.SerialNumber,
			p.ExpireDate.Format("02.01.2006"),
			p.ShortName,
			quantityStr,
			formatWithSpaceSeparator(p.SupplyPrice),
			formatWithSpaceSeparator(p.RetailPrice),
			formatWithSpaceSeparator(p.RetailPrice - p.SupplyPrice),
			formatWithSpaceSeparator(p.RetailPrice),
			formatWithSpaceSeparator(totalPrice),
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
		total += math.Round(totalPrice)
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

	savePdfToUploads(c, pdf, *h.log, "Return_Nakladnoy_"+returnData.PublicId)
}

// UpdateByBarcode godoc
// @Summary Update return or transfer by barcode
// @Tags Return
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Transfer ID or Return ID"
// @Param request body domain.TransferBarcodeRequest true "Barcode request payload"
// @Success 200 {object} v1.Response "Update successful"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /return/update-by-barcode/{id} [put]
func (h *ReturnHandler) UpdateByBarcode(c *gin.Context) {
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

	err := h.service.UpdateReturnByBarcode(ctx, &req, user)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			handleResponse(c, MultiStatus, notAddErr.Data)
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// EditStatusToChecking godoc
// @Summary Edit status to checking
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   id path string true "Return ID"
// @Success 200 {object} v1.Response "Return"
// @Failure 400 {object} v1.Response "Invalid request parameters"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /return/edit-status-to-checking/{id} [PUT]
func (h *ReturnHandler) EditStatusToChecking(c *gin.Context) {
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
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// lock order for parallel request
func (h *ReturnHandler) getReturnLock(importId string) *sync.Mutex {
	lock, _ := h.ordersToMutexes.LoadOrStore(importId, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
