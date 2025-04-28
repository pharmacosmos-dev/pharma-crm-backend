package v1

import (
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
		returned.GET("/export-excel", h.ExportReturnExcel)
		returned.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		returned.POST("/send/:id", h.Send)
		returned.POST("/confirm/:id", h.Confirm)
		returned.POST("/cancel/:id", h.Cancel)
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
	var returnRequest domain.ReturnRequest
	// Bind the request body to the ReturnRequest struct
	if err := c.ShouldBindJSON(&returnRequest); err != nil {
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
	returnRequest.CreatedBy = userId.(string)     // get creator id from set header
	returnRequest.PublicId = utils.GenerateCode() // generate public id

	// create return
	err := h.service.CreateReturn(&returnRequest)
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
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	var res domain.Return
	err := h.db.Model(&domain.Transfer{}).Preload("Store").Raw(`SELECT * FROM transfers WHERE id = ?`, id).Scan(&res).Error
	if err != nil {
		h.log.Warn("Error on getting return: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get return")
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
	res, totalCount, err := h.service.ReturnList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Export return excel godoc
// @Summary Export return excel
// @Description Export return excel
// @Tags Return
// @Security     BearerAuth
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
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
	res, _, err := h.service.ReturnList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return list")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "Филиал", "Кол-во", "Полученная Сумма Поставки", "Принятая Сумма Поставки", "Полученная Сумма Продажи", "Принятая Сумма Продажи", "Статус", "Создание", "Завершение", "Создал", "Завершил"}

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
		if r.AcceptedBy != nil {
			f.SetCellValue(sheetName, "M"+row, r.AcceptedBy.FullName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}

	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=return.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
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
		request.Count, id).Error
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
	err := h.service.SendReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to send return")
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
	err := h.service.ConfirmReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "COMFIRMED")
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
	if err := c.ShouldBindQuery(&param); err != nil {
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
	headers := []string{"Код", "Наименование", "Штрих-код", "Срок годность", "Серия номер", "Кол-во", "Ед-изм", "Cканированные"}

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
		f.SetCellValue(sheetName, "H"+row, r.ScannedCount)

	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=return-detail.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}

}
