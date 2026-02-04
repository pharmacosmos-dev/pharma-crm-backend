package v1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type RepricingHandler struct {
	*Handler
}

func (h *Handler) NewRepricingHandler(r *gin.RouterGroup) {
	returnHandler := &RepricingHandler{h}
	returnHandler.RepricingRoutes(r)
}

func (h *RepricingHandler) RepricingRoutes(r *gin.RouterGroup) {
	repricing := r.Group("/repricing")
	{
		repricing.POST("", h.Create)
		repricing.GET("/:id", h.Get)
		repricing.GET("/list", h.List)
		repricing.GET("/list-status", h.RepricingStatus)
		repricing.GET("/export-excel", h.ExportRepricingExcel)
		repricing.POST("/confirm/:id", h.Confirm)
		repricing.POST("/cancel/:id", h.Cancel)
		repricing.POST("/new-price/:id", h.AddRetailPrice)
	}
	detail := r.Group("repricing-detail")
	{
		detail.GET("/list/:id", h.ListDetail)
		detail.GET("/detail-status/:id", h.RepricingDetailStatus)
		detail.GET("/export-excel", h.ExportListDetail)
	}
}

// Create godoc
// @Summary Create Repricing
// @Description Create Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	repricing body domain.RepricingRequest true "Repricing"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing [POST]
func (h *RepricingHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var request domain.RepricingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	request.CreatedBy = &user.UserId
	// create repricing
	_, err := h.service.CreateRepricing(ctx, &request)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Repricing
// @Description Get Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/{id} [GET]
func (h *RepricingHandler) Get(c *gin.Context) {
	// get return by id
	id := c.Param("id")
	// validate id
	if id == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get repricing by id
	res, err := h.service.GetRepricingByID(ctx, id)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary List Repricing
// @Description List Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/list [get]
func (h *RepricingHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreID = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, totalCount, err := h.service.GetRepricingList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// RepricingStatus godoc
// @Summary Get Repricing summary stats
// @Description Get total count and retail price sums for repricing
// @Tags Repricing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date query string false "Start Date"
// @Param   end_date query string false "End Date"
// @Param   status query string false "Status"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/list-status [get]
func (h *RepricingHandler) RepricingStatus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreID = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, err := h.service.GetRepricingStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary List Repricing Export excel
// @Description List Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/export-excel [get]
func (h *RepricingHandler) ExportRepricingExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreID = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, _, err := h.service.GetRepricingList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Название", "Филиал", "Тип", "Кол-во", "Статус", "Создал", "Завершил", "Дата переоценки"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Id)
		f.SetCellValue(sheetName, "B"+row, imp.Name)
		if imp.Store != nil {
			f.SetCellValue(sheetName, "C"+row, imp.Store.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		f.SetCellValue(sheetName, "D"+row, imp.Type)
		f.SetCellValue(sheetName, "E"+row, imp.Count)
		f.SetCellValue(sheetName, "F"+row, imp.Status)
		if imp.CreatedBy != nil {
			f.SetCellValue(sheetName, "G"+row, imp.CreatedBy.FirstName)
		} else {
			f.SetCellValue(sheetName, "G"+row, "N/A")
		}
		if imp.CreatedBy != nil {
			f.SetCellValue(sheetName, "H"+row, imp.UpdatedBy.FirstName)
		} else {
			f.SetCellValue(sheetName, "H"+row, "N/A")
		}
		f.SetCellValue(sheetName, "I"+row, imp.CreatedAt.Format(time.DateTime))

	}
	saveExcelToUploads(c, f, *h.log, "product_repricing")
}

// confirm repricing
// @Summary Confirm Repricing
// @Description Confirm Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/confirm/{id} [POST]
func (h *RepricingHandler) Confirm(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var id = c.Param("id")
	repricingId, err := strconv.Atoi(id)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// confirm repricing service
	err = h.service.ConfirmRepricing(ctx, repricingId, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel Repricing
// @Summary Cancel Repricing
// @Description Cancel Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/cancel/{id} [POST]
func (h *RepricingHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate request uuid
	if id == "" {
		handleResponse(c, BadRequest, "invalid.repricing.id")
		return
	}
	// get user id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm repricing service
	err := h.service.CancelRepricing(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to cancel repricing")
		return
	}

	handleResponse(c, OK, "CANCELED")
}

// List godoc
// @Summary List Repricing
// @Description List Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   id path int false "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing-detail/list/{id} [get]
func (h *RepricingHandler) ListDetail(c *gin.Context) {
	var (
		param domain.QueryParam
		id    = c.Param("id")
	)
	// bind request query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	// convent to integer
	repricingID, err := strconv.Atoi(id)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.repricing.id")
		return
	}

	// default limit, offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.RepricingDetailList(repricingID, &param)
	if err != nil {
		h.log.Warn("ERROR on getting repricing list: %v", err)
		handleResponse(c, InternalError, "Failed to get repricing list")
		return
	}
	// _meta pagination data
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// RepricingDetailStatus godoc
// @Summary Get Repricing Detail summary stats
// @Description Get total count, sums and markup averages for repricing details
// @Tags Repricing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   id path int true "Repricing ID"
// @Param   search query string false "Search by product name or barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing-detail/detail-status/{id} [get]
func (h *RepricingHandler) RepricingDetailStatus(c *gin.Context) {
	var param domain.QueryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	repricingID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handleResponse(c, BadRequest, "Invalid repricing_id")
		return
	}

	res, err := h.service.RepricingDetailStatus(repricingID, &param)
	if err != nil {
		h.log.Error("ERROR on repricing detail status: %v", err)
		handleResponse(c, InternalError, "Failed to get repricing detail summary")
		return
	}

	handleResponse(c, OK, res)
}

// ExportListDetail godoc
// @Summary Export Repricing Details to Excel
// @Description Export repricing details to Excel
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   repricing_id query int false "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing-detail/export-excel [get]
func (h *RepricingHandler) ExportListDetail(c *gin.Context) {
	var params domain.QueryParam

	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// Get details list
	res, _, err := h.service.RepricingDetailList(params.RepricingID, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	fmt.Println("REPRICING LENGTH: ", len(res))
	f := excelize.NewFile()
	sheetName := "RepricingDetails"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{
		"ID", "Название", "Штрихкод",
		"Старая розничная цена", "Новая розничная цена",
		"Старая наценка", "Новая наценка",
		"Старый срок", "Новая срок",
		"Серийный номер",
	}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Ma'lumotlarni qo‘shish
	for i, item := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, i)
		f.SetCellValue(sheetName, "B"+row, item.Name)
		f.SetCellValue(sheetName, "C"+row, item.Barcode)
		f.SetCellValue(sheetName, "D"+row, item.OldRetailPrice)
		f.SetCellValue(sheetName, "E"+row, item.NewRetailPrice)
		f.SetCellValue(sheetName, "F"+row, fmt.Sprintf("%.2f %%", item.OldMarkup))
		f.SetCellValue(sheetName, "G"+row, fmt.Sprintf("%.2f %%", item.NewMarkup))

		// Null date handling
		if !item.OldExpireDate.IsZero() {
			f.SetCellValue(sheetName, "H"+row, item.OldExpireDate.Format("2006-01-02"))
		} else {
			f.SetCellValue(sheetName, "H"+row, "")
		}

		if item.NewExpireDate != nil && !item.NewExpireDate.IsZero() {
			f.SetCellValue(sheetName, "I"+row, item.NewExpireDate.Format("2006-01-02"))
		} else {
			f.SetCellValue(sheetName, "I"+row, "")
		}

		f.SetCellValue(sheetName, "J"+row, item.SerialNumber)
	}

	saveExcelToUploads(c, f, *h.log, fmt.Sprintf("Repricing_Detail_%d", params.RepricingID))
}

// AddRetailPrice godoc
// @Summary add new retail price
// @Description add new retail price
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   id path int true "Repricing ID"
// @Param   body body domain.UpdateNewPrice true "New price"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/new-price/{id} [POST]
func (h *RepricingHandler) AddRetailPrice(c *gin.Context) {
	var body domain.UpdateNewPrice
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get repricing id from path
	repricingId := c.Param("id")
	if repricingId == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	if body.Percent > 0 && repricingId != "" {
		query := `
				UPDATE price_revalution_details
				SET 
					new_retail_price = ROUND(old_supply_price * (1 + ?/100.00), -2), 
					updated_at = NOW()
				WHERE price_revalution_id = ?
			`
		err := tx.WithContext(ctx).Exec(query, body.Percent, repricingId).Error
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not update multiple product price: %v", err)
			handleResponse(c, BadRequest, domain.InternalServerError)
			return
		}
	} else if body.NewRetailPrice > 0 && body.Id != "" {
		query := `
			UPDATE price_revalution_details
			SET new_retail_price = ?, updated_at = NOW()
			WHERE id = ?
		`
		err := tx.WithContext(ctx).Exec(query, body.NewRetailPrice, body.Id).Error
		if err != nil {
			_ = tx.Rollback()
			handleServiceResponse(c, BadRequest, domain.InternalServerError)
			return
		}
	} else {
		_ = tx.Rollback()
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	if err := tx.Commit().Error; err != nil {
		h.log.Errorf("could not commit new repricing transaction: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, "UPDATED")
}
