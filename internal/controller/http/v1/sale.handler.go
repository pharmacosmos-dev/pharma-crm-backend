package v1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type SaleHandler struct {
	*Handler
}

func (h *Handler) NewSaleHandler(r *gin.RouterGroup) {
	sale := &SaleHandler{h}
	sale.SaleRoutes(r)
}

func (h *SaleHandler) SaleRoutes(r *gin.RouterGroup) {
	sale := r.Group("/sale")
	{
		sale.POST("", h.Create)
		sale.POST("/return", h.CreateReturn)
		sale.GET("/:id", h.Get)
		sale.GET("/list", h.GetSales)
		sale.GET("/export-excel", h.ExportSalesExcel)
		sale.GET("/stats", h.GetSalesStats)
		sale.GET("/get-list", h.GetSaleList)
		sale.GET("/pending-list", h.PendingSaleList)
		sale.GET("/dmed/prescriptions", h.DMEDGetPrescriptions)
		sale.GET("/online-list", h.OnlineSaleList)
		sale.GET("/online-count", h.GetOnlineSaleCount)
		sale.PUT("/:id", h.Update)
		sale.POST("/final", h.FinalSale)
		sale.POST("/epos-result", h.EposResult)
		sale.POST("/discount-card", h.AddDiscountCard)
		sale.DELETE("/discount-card", h.RemoveCustomerDiscount)
		sale.POST("/online-accept", h.AcceptOnlineSale)
		sale.POST("online-cancel", h.CancelOnlineSale)
		sale.POST("/asil-belgi-barcode", h.AsilBelgiBarcode)
		sale.POST("/asil-belgi-barcode-confirm/:id", h.AsilBelgiBarcodeConfirm)
		sale.PUT("/pending/:id", h.PendingSale)
		sale.GET("/online-orders", h.FetchOnlineOrders)
	}
}

// region Create

// Create godoc
// @Summary 	Create a sale
// @Description Create a sale from the request body
// @Tags 	sales
// @Security BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body domain.SaleRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale [post]
func (h *SaleHandler) Create(c *gin.Context) {
	// get user id from header
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var body domain.SaleRequest
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.BadRequestError)
		return
	}

	if body.StoreId != user.StoreId {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	body.EmployeeId = user.UserId
	res, err := h.service.CreateSale(ctx, h.db, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// Create return sale godoc
// @Summary Create a return sale
// @Description Create a return sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.SaleReturnRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/return [post]
func (h *SaleHandler) CreateReturn(c *gin.Context) {
	// get user id in context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.SaleReturnRequest
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	body.EmployeeId = user.UserId
	body.SaleType = constants.SaleTypeReturn
	// create sale return
	sale, err := h.service.CreateReturnSale(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, sale)
}

// region Get

// Get godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale_id"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [get]
func (h *SaleHandler) Get(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get one sale
	res, err := h.service.GetSaleOne(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit 			query int false 	"limit"
// @Param offset 			query int false 	"offset"
// @Param vendor_id 		query string false 	"vendor_id"
// @Param store_id 			query string false 	"store_id"
// @Param cashbox_id 		query string false 	"cashbox_id"
// @Param payment_type_id 	query string false 	"payment_type_id"
// @Param search 			query string false 	"search"
// @Param start_date 		query string false 	"start_date"
// @Param end_date 			query string false 	"end_date"
// @Param total_amount_from query int false 	"total_amount_from"
// @Param total_amount_to 	query int false 	"total_amount_to"
// @Param cash 				query bool false 	"cash"
// @Param humo 				query bool false 	"humo"
// @Param uzcard 			query bool false 	"uzcard"
// @Param click 			query bool false 	"click"
// @Param payme 			query bool false 	"payme"
// @Param alif 				query bool false 	"alif"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list [get]
func (h *SaleHandler) GetSales(c *gin.Context) {
	// get user from the context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// bind query params
	var params domain.SaleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Errorf("bind query error: %v", err)
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset with checking default
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get sale list data
	res, totalCount, err := h.service.GetSales(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// added _meta section to response
	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// List godoc
// @Summary Get a sale list excel
// @Description Get a sale list excel
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit 			query int false 	"limit"
// @Param offset 			query int false 	"offset"
// @Param vendor_id 		query string false 	"vendor_id"
// @Param store_id 			query string false 	"store_id"
// @Param cashbox_id 		query string false 	"cashbox_id"
// @Param payment_type_id 	query string false 	"payment_type_id"
// @Param search 			query string false 	"search"
// @Param start_date 		query string false 	"start_date"
// @Param end_date 			query string false 	"end_date"
// @Param total_amount_from query int false 	"total_amount_from"
// @Param total_amount_to 	query int false 	"total_amount_to"
// @Param cash 				query bool false 	"cash"
// @Param humo 				query bool false 	"humo"
// @Param uzcard 			query bool false 	"uzcard"
// @Param click 			query bool false 	"click"
// @Param payme 			query bool false 	"payme"
// @Param alif 				query bool false 	"alif"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/export-excel [get]
func (h *SaleHandler) ExportSalesExcel(c *gin.Context) {
	var params domain.SaleQueryParams
	// get user_id from the context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	// bind query params
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get limit offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get sale list data
	res, _, err := h.service.GetSales(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Филиал", "Наличный", "Humo", "Uzcard", "Payme", "Click", "AlifBank", "Обшая сумма", "Дата продажа", "Время продажа", "Касса", "Продавец", "Клиент"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Errorf("could not create style: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	// Ma'lumotlarni qo'shish
	for i, sale := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, helper.SaleTypeToRussian(sale.SaleType, sale.SaleNumber))
		f.SetCellValue(sheetName, "B"+row, sale.StoreName)
		f.SetCellValue(sheetName, "C"+row, sale.Cash)
		f.SetCellValue(sheetName, "D"+row, sale.Humo)
		f.SetCellValue(sheetName, "E"+row, sale.Uzcard)
		f.SetCellValue(sheetName, "F"+row, sale.Payme)
		f.SetCellValue(sheetName, "G"+row, sale.Click)
		f.SetCellValue(sheetName, "H"+row, sale.Alif)
		f.SetCellValue(sheetName, "I"+row, sale.TotalAmount)
		f.SetCellValue(sheetName, "J"+row, sale.CompletedAt.Add(time.Hour*5).Format(time.DateOnly))
		f.SetCellValue(sheetName, "K"+row, sale.CompletedAt.Add(time.Hour*5).Format(time.TimeOnly))
		f.SetCellValue(sheetName, "L"+row, sale.CashBoxName)
		f.SetCellValue(sheetName, "M"+row, sale.FullName)
		if sale.CustomerName != nil {
			f.SetCellValue(sheetName, "N"+row, *sale.CustomerName)
		} else {
			f.SetCellValue(sheetName, "N"+row, "N/A")
		}

	}

	saveExcelToUploads(c, f, *h.log, "Barcha_sotuvlar")
}

// List godoc
// @Summary Get a sale stats
// @Description Get a sale stats from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param vendor_id 		query string false 	"vendor_id"
// @Param store_id 			query string false 	"store_id"
// @Param cashbox_id 		query string false 	"cashbox_id"
// @Param payment_type_id 	query string false 	"payment_type_id"
// @Param search 			query string false 	"search"
// @Param start_date 		query string false 	"start_date"
// @Param end_date 			query string false 	"end_date"
// @Param total_amount_from query int false 	"total_amount_from"
// @Param total_amount_to 	query int false 	"total_amount_to"
// @Param cash 				query bool false 	"cash"
// @Param humo 				query bool false 	"humo"
// @Param uzcard 			query bool false 	"uzcard"
// @Param click 			query bool false 	"click"
// @Param payme 			query bool false 	"payme"
// @Param alif 				query bool false 	"alif"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/stats [get]
func (h *SaleHandler) GetSalesStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// bind query param
	var params domain.SaleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// check user role
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, err := h.service.GetSalesStats(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param vendor_id query string false "Vendor ID"
// @Param store_id query string true "Store ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/get-list [get]
func (h *SaleHandler) GetSaleList(c *gin.Context) {

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var params domain.SaleQueryParams
	// bind query params
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get sale list data
	res, totalCount, err := h.service.GetSaleList(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// added _meta section to response
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Get online pending sale count
// @Summary 	Get online pending sale count
// @Description Get online pending sale count
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param	store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-count [GET]
func (h *SaleHandler) GetOnlineSaleCount(c *gin.Context) {
	// get user from the set header
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	storeId := c.Query("store_id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			storeId = user.StoreId
		}
	}

	// get online order count
	qb := h.db.WithContext(ctx).
		Model(&domain.Sale{}).
		Where("online_status IN(?)", constants.OnlinePendingStages).
		Where("type = ?", constants.SaleTypeOnline)

	if storeId != "" {
		qb = qb.Where("store_id = ?", storeId)
	}

	var count int64
	if err := qb.Count(&count).Error; err != nil {
		h.log.Errorf("could not get online sale count: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, gin.H{"count": count})
}

// Get online pending sale list
// @Summary Get online pending sale list
// @Description Get online pending sale list
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   limit query int false "Limit"
// @Param	offset query int false "Offset"
// @Param   store_id query string false "Store ID"
// @Param   search query string false "Search"
// @Param	start_date query string false "StartDate"
// @Param	end_date query string false "EndDate"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-list [GET]
func (h *SaleHandler) OnlineSaleList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var params domain.SaleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Errorf("bind query error: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, totalCount, err := h.service.GetOnlinePendingSales(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	// get response data with pagination _meta data
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// PendingSaleList godoc
// @Summary Get pending sales
// @Description Get all sales with status 'pending' filtered by store, date, search
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search (sale number or store name)"
// @Param start_date query string false "Created At Start Date (RFC3339)"
// @Param end_date query string false "Created At End Date (RFC3339)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/pending-list [get]
func (h *SaleHandler) PendingSaleList(c *gin.Context) {
	var params domain.SaleQueryParams

	// get user_id from context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// get pending sales
	res, totalCount, err := h.service.GetPendingSales(ctx, &params, user)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// DMEDGetPrescriptions godoc
// @Summary      Get prescriptions from DMED
// @Description  Fetch prescriptions for a patient from DMED API
// @Tags         sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        patient_id query string true "Patient ID"
// @Param        safe_code  query string true "Safe Code"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /sale/dmed/prescriptions [get]
func (h *SaleHandler) DMEDGetPrescriptions(c *gin.Context) {
	patientID := c.Query("patient_id")
	safeCode := c.Query("safe_code")

	if patientID == "" || safeCode == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	respBody, err := h.service.GetPrescriptionsFromDMED(patientID, safeCode)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, respBody)
}

// FetchOnlineOrders godoc
// @Summary      Fetch online orders
// @Description  Fetch online orders
// @Tags         sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param 		 limit query int false "Limit"
// @Param 		 offset query int false "Offset"
// @Param 		 store_id query string false "Store ID"
// @Param 		 search query string false "Search"
// @Param 		 start_date query string false "Start Date"
// @Param 		 end_date query string false "End Date"
// @Param        status query string false "Status"
// @Param 		 stage query int false "Stage"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /sale/online-orders [GET]
func (h *SaleHandler) FetchOnlineOrders(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var params domain.SaleQueryParams
	// bind query params
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get sale list data
	res, totalCount, err := h.service.GetOnlineOrders(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// added _meta section to response
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// region Update

// Update godoc
// @Summary Update a sale
// @Description Update a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Param input body domain.SaleUpdateRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [put]
func (h *SaleHandler) Update(c *gin.Context) {
	var (
		body domain.SaleUpdateRequest
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err = h.db.
		WithContext(ctx).
		Table("sales").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Errorf("could not update sale: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, body)
}

// FinalSale
// @Summary Final Sale
// @Description Final Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// bind request body
	var body domain.FinalSale
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	if body.StoreId != user.StoreId {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// lock parallel request
	mu := h.getOrderLock(body.SaleId)
	mu.Lock()
	defer mu.Unlock()

	res, err := h.service.FinalizeSale(ctx, &body)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			handleResponse(c, CONFLICT, notAddErr.Data)
			return
		}

		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// EposRequest godoc
// @Summary Epos Request
// @Description Epos Request from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.EposResponseRequest true "Epos Response info as json {}"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/epos-result [post]
func (h *SaleHandler) EposResult(c *gin.Context) {
	// get user id in context
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// bind request
	var body domain.EposResponseRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// Get order lock
	lock := h.getOrderLock(body.SaleId)
	lock.Lock()
	defer lock.Unlock()

	sale, err := h.service.EposResult(ctx, &body, user)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, sale)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.AddDiscountCard true "Add discount card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/discount-card [POST]
func (h *SaleHandler) AddDiscountCard(c *gin.Context) {
	var body domain.AddDiscountCard
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.AttachDiscountCardToSale(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.AddDiscountCard true "Add discount card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/discount-card [DELETE]
func (h *SaleHandler) RemoveCustomerDiscount(c *gin.Context) {
	var body domain.AddDiscountCard
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.DeleteDiscountCardFromSale(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
	}

	handleResponse(c, OK, "DELETED")

}

// Confirm online sale
// @Summary Confirm online sale
// @Description Confirm online sale
// @Tags 	sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   body body domain.ConfirmOnlineSaleRequest true "confirm online sale"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/sale/online-accept [POST]
func (h *SaleHandler) AcceptOnlineSale(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}
	var body domain.ConfirmOnlineSaleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	body.EmployeeId = user.UserId

	sale, err := h.service.AcceptOnlineSale(ctx, &body)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, sale.Id)
}

// godoc cancel online sale
// @Summary Cancel online sale
// @Description Cancel online sale
// @Tags 	sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   body body domain.ConfirmOnlineSaleRequest true "cancel online sale"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-cancel [POST]
func (h *SaleHandler) CancelOnlineSale(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var body domain.ConfirmOnlineSaleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	body.EmployeeId = user.UserId

	err := h.service.CancelOnlineSale(ctx, &body)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	handleResponse(c, OK, "CANCELED")
}

// AsilBelgiBarcode godoc
// @Summary      Check and save product barcode by markingCode
// @Description  Markirovkani yuborib, AslBelgi API orqali productName va gtin ni oladi. Foydalanuvchi yuborgan productName bilan solishtiriladi. Agar 90%+ mos tushsa avtomatik update qiladi (status=completed), bo‘lmasa pending yoziladi va eski barcode ham log bo‘ladi.
// @Tags         sales
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body domain.AsilBelgiRequest true "Markirovka va productName"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /sale/asil-belgi-barcode [post]
func (h *SaleHandler) AsilBelgiBarcode(c *gin.Context) {
	var body domain.AsilBelgiRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
	}
	body.UserID = userId.(string)

	// get employee info by set user_id
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "user.not.found")
			return
		}
		h.log.Warn("ERROR on getting employee info: %v", err)
		handleResponse(c, InternalError, "not.get.user")
		return
	}

	// markingCodeni 31 belgigacha qisqartirish
	markingCode := body.Markirovka
	if len(markingCode) > 31 {
		markingCode = markingCode[:31]
	}

	// markirovkadan 3 tashlab, keyingi 13 belgini olish
	var markingPart string
	if len(body.Markirovka) > 16 {
		markingPart = body.Markirovka[3:16] // 3-chi indexdan boshlab 13 ta belgini olish
	}

	// shu barcode bazada bormi?
	var exists bool
	err = h.db.Raw(`
	SELECT EXISTS(
		SELECT 1 FROM product_barcodes 
		WHERE product_id = ? AND barcode = ?
	)
	`, body.ProductID, markingPart).Scan(&exists).Error
	if err != nil {
		h.log.Warn("ERROR on searching product_barcodes: %v", err)
		handleResponse(c, InternalError, "failed.search.product_barcodes")
		return
	}

	// agar topilsa -> completed qaytariladi
	if exists {
		resp := domain.AsilBelgiBarcodeResponse{
			Id:          body.ProductID,
			Status:      constants.GeneralStatusCompleted,
			OldBarcode:  markingPart,
			NewBarcode:  markingPart,
			RequestName: body.ProductName,
			Similarity:  100,
		}
		handleResponse(c, OK, resp)
		return
	}

	// Asl Belgi API chaqiramiz
	cisInfo, err := h.service.FetchCisInfo(markingCode)
	if err != nil {
		handleResponse(c, InternalError, "failed.get.cis.info")
		return
	}

	// similarity
	similarity := helper.CalcSimilarity(body.ProductName, cisInfo.ProductName)
	barcode := strings.TrimLeft(cisInfo.Gtin, "0")

	// eski barcode olish
	var oldBarcode string
	err = h.db.Raw(`
		SELECT barcode FROM products WHERE id = ? LIMIT 1
	`, body.ProductID).Scan(&oldBarcode).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.log.Warn("ERROR on getting old barcode: %v", err)
		handleResponse(c, InternalError, "failed.get.old.barcode")
		return
	}

	// transaction
	tx := h.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	var similarityStr, id string
	if similarity >= 0.9 {
		// update products va store_products
		if err = tx.Exec(`UPDATE products SET barcode = ? WHERE id = ?`, barcode, body.ProductID).Error; err != nil {
			_ = tx.Rollback()
			h.log.Warn("ERROR on updating product barcode: %v", err)
			handleResponse(c, InternalError, "failed.update.product.barcode")
			return
		}
		if err = tx.Exec(`UPDATE store_products SET barcode = ? WHERE product_id = ?`, barcode, body.ProductID).Error; err != nil {
			_ = tx.Rollback()
			h.log.Warn("ERROR on updating store_product barcode: %v", err)
			handleResponse(c, InternalError, "failed.update.store_product.barcode")
			return
		}
		// product_barcodes log
		if err = tx.Raw(`
			INSERT INTO product_barcodes(product_id, old_barcode, barcode, created_by, status, store_id)
			VALUES(?, ?, ?, ?, ?, ?)
			RETURNING id
		`, body.ProductID, oldBarcode, barcode, body.UserID, constants.GeneralStatusCompleted, employee.StoreId).Scan(&id).Error; err != nil {
			_ = tx.Rollback()
			h.log.Warn("ERROR on inserting product_barcode: %v", err)
			handleResponse(c, InternalError, "failed.save.barcode.log")
			return
		}
		similarityStr = constants.GeneralStatusCompleted
	} else if similarity <= 0.6 {
		_ = tx.Rollback()
		handleResponse(c, BadRequest, "similarity.not.enough")
		return
	} else {
		// pending log
		if err = tx.Raw(`
			INSERT INTO product_barcodes(product_id, old_barcode, barcode, created_by, status, store_id)
			VALUES(?, ?, ?, ?, ?, ?)
			RETURNING id
		`, body.ProductID, oldBarcode, barcode, body.UserID, constants.GeneralStatusPending, employee.StoreId).Scan(&id).Error; err != nil {
			_ = tx.Rollback()
			h.log.Warn("ERROR on inserting pending product_barcode: %v", err)
			handleResponse(c, InternalError, "failed.save.pending.barcode.log")
			return
		}
		similarityStr = constants.GeneralStatusPending
	}

	if err = tx.Commit().Error; err != nil {
		h.log.Warn("ERROR on commiting transaction: %v", err)
		handleResponse(c, InternalError, "not.completed.transaction")
		return
	}

	// response struct
	resp := domain.AsilBelgiBarcodeResponse{
		Id:                   id,
		Status:               similarityStr,
		OldBarcode:           oldBarcode,
		NewBarcode:           barcode,
		AsilBelgiProductName: cisInfo.ProductName,
		RequestName:          body.ProductName,
		Similarity:           similarity * 100,
	}

	handleResponse(c, OK, resp)
}

// AsilBelgiBarcodeConfirm godoc
// @Summary      Confirm pending product barcode
// @Description  Pending statusdagi barcode’ni admin tasdiqlaydi va product/store_products ga yoziladi
// @Tags         sales
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "ProductBarcode ID"
// @Success      200 {object} domain.ConfirmBarcodeResponse
// @Failure      400 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /sale/asil-belgi-barcode-confirm/{id} [post]
func (h *SaleHandler) AsilBelgiBarcodeConfirm(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		handleResponse(c, BadRequest, "id.required")
		return
	}
	var barcodeLog domain.ProductBarcode
	// pending yozuvni olish
	err := h.db.First(&barcodeLog, "id = ? AND status = ?", id, constants.GeneralStatusPending).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		handleResponse(c, NotFound, "pending.barcode.not.found")
		return
	}
	if err != nil {
		h.log.Warn("ERROR on getting pending barcode: %v", err)
		handleResponse(c, InternalError, "failed.get.barcode")
		return
	}

	// transaction boshlash
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// products update
	err = tx.Exec(`UPDATE products SET barcode = ? WHERE id = ?`, barcodeLog.Barcode, barcodeLog.ProductID).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Warn("ERROR on updating product barcode: %v", err)
		handleResponse(c, InternalError, "failed.update.product.barcode")
		return
	}

	// store_products update
	err = tx.Exec(`UPDATE store_products SET barcode = ? WHERE product_id = ?`, barcodeLog.Barcode, barcodeLog.ProductID).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Warn("ERROR on updating store_product barcode: %v", err)
		handleResponse(c, InternalError, "failed.update.store_product.barcode")
		return
	}

	// log statusni update qilish
	err = tx.Exec(`UPDATE product_barcodes SET status = ? WHERE id = ?`, constants.GeneralStatusPending, id).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Warn("ERROR on updating product_barcode status: %v", err)
		handleResponse(c, InternalError, "failed.update.barcode.log")
		return
	}

	// commit
	if err = tx.Commit().Error; err != nil {
		h.log.Warn("ERROR on commiting transaction: %v", err)
		handleResponse(c, InternalError, "not.completed.transaction")
		return
	}

	resp := domain.ConfirmBarcodeResponse{
		Status:     "completed",
		ProductID:  barcodeLog.ProductID,
		OldBarcode: barcodeLog.OldBarcode,
		NewBarcode: barcodeLog.Barcode,
	}

	handleResponse(c, OK, resp)
}

// PendingSale godoc
// @Summary      Move sale to pending
// @Description  Update a sale record status to pending
// @Tags         sales
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Sale ID"
// @Success      200 {object} domain.PendingSaleResponse
// @Failure      400 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /sale/pending/{id} [put]
func (h *SaleHandler) PendingSale(c *gin.Context) {
	var (
		sale domain.Sale
		err  error
	)

	id := c.Param("id")
	if id == "" {
		handleServiceResponse(c, BadRequest, domain.BadRequestError)
		return
	}

	// get sale record
	err = h.db.Take(&sale, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		handleServiceResponse(c, NotFound, domain.NotFoundError)
		return
	}
	if sale.Status == constants.GeneralStatusPending {
		resp := domain.PendingSaleResponse{
			ID:     id,
			Status: constants.GeneralStatusPending,
		}
		handleResponse(c, OK, resp)
		return
	} else if sale.SaleType == constants.SaleTypeReturn {
		handleServiceResponse(c, BadRequest, domain.BadRequestError)
		return
	}
	if err != nil {
		h.log.Errorf("could not gett sale: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// update sale status to pending
	err = h.db.Exec(`UPDATE sales SET status = ? WHERE id = ?`, constants.GeneralStatusPending, id).Error
	if err != nil {
		h.log.Errorf("could not update sale status: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	resp := domain.PendingSaleResponse{
		ID:     id,
		Status: constants.GeneralStatusPending,
	}

	handleResponse(c, OK, resp)
}

// lock order for parallel request
func (h *SaleHandler) getOrderLock(orderId string) *sync.Mutex {
	lock, _ := h.ordersToMutexes.LoadOrStore(orderId, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
