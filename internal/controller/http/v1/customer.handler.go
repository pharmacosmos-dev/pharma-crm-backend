package v1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type CustomerHandler struct {
	*Handler
}

func (h *Handler) NewCustomerHandler(r *gin.RouterGroup) {
	customer := &CustomerHandler{h}
	customer.CustomerRoutes(r)
}

func (h *CustomerHandler) CustomerRoutes(r *gin.RouterGroup) {
	customer := r.Group("/customer")
	{
		customer.POST("", h.Create)
		customer.GET("/:id", h.Get)
		customer.GET("/list", h.List)
		customer.GET("/export-excel", h.ExportCustomerExcel)
		customer.GET("/list-discount-cards", h.ListDiscountCards)
		customer.GET("/export-excel-discount-cards", h.ExportDiscountCardExcel)
		customer.PUT("/:id", h.Update)
		customer.DELETE("/soft-delete", h.SoftDelete)
		customer.DELETE("/hard-delete", h.HardDelete)
	}
	tag := r.Group("/tag")
	{
		tag.POST("", h.CreateTag)
		tag.GET("/list", h.TagList)
	}
}

// Create godoc
// @Summary Create a customer
// @Description Create a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param customer body domain.CustomerRequest true "Customer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer [post]
func (h *CustomerHandler) Create(c *gin.Context) {
	var body domain.CustomerRequest
	// get user from the context
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone number
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	body.CreatedBy = user.UserId
	res, err := h.service.CreateCustomer(ctx, &body)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// Get godoc
// @Summary 	Get a customer
// @Description Get a customer from the request body
// @Tags 	customers
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "customer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [get]
func (h *CustomerHandler) Get(c *gin.Context) {
	var id = c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetCustomerById(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a customer
// @Description Get a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/list [get]
func (h *CustomerHandler) List(c *gin.Context) {
	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get customers data
	res, totalCount, err := h.service.GetCustomers(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	// add _meta data
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// Export customer excel godoc
// @Summary Export customer excel
// @Description Export customer excel
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/export-excel [get]
func (h *CustomerHandler) ExportCustomerExcel(c *gin.Context) {

	var params domain.QueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)
	// get customers data
	res, _, err := h.service.GetCustomers(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "Klients"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "ФИО", "Номер Телефона", "Теги", "Сумма покупки", "Последний покупка", "Дата рождения", "Дата регистрации", "Зарегистрируйтесь в филиале", "Баланс", "Текущий долг"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, client := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, client.PublicId)
		f.SetCellValue(sheetName, "B"+row, client.FullName)
		f.SetCellValue(sheetName, "C"+row, client.Phone)
		// check if store is not null
		if client.Tag != nil {
			f.SetCellValue(sheetName, "D"+row, client.Tag.Name)
		} else {
			f.SetCellValue(sheetName, "D"+row, "N/A")
		}
		f.SetCellValue(sheetName, "E"+row, client.SaleAmount)
		// check last sale date with null or not
		if client.SaleDate != nil {
			f.SetCellValue(sheetName, "F"+row, client.SaleDate)
		} else {
			f.SetCellValue(sheetName, "F"+row, "N/A")
		}
		f.SetCellValue(sheetName, "G"+row, client.Birthday)
		f.SetCellValue(sheetName, "H"+row, client.CreatedAt)
		// check if store is not null
		if client.Store != nil {
			f.SetCellValue(sheetName, "I"+row, client.Store.Name)
		} else {
			f.SetCellValue(sheetName, "I"+row, "N/A")
		}
		f.SetCellValue(sheetName, "J"+row, client.Balance)
		f.SetCellValue(sheetName, "K"+row, client.DebtAmount)

	}
	saveExcelToUploads(c, f, *h.log, "Mijozlar")
}

// List godoc
// @Summary Get discount cards
// @Description Get list of discount cards joined with customer info
// @Tags customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search by customer name or phone or barcode"
// @Param store_id query string false "Store ID filter"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/list-discount-cards [get]
func (h *CustomerHandler) ListDiscountCards(c *gin.Context) {
	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, total, err := h.service.ListDiscountCards(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(res, total, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// Export discount card excel godoc
// @Summary Export discount cards with customer info to Excel
// @Description Export discount cards with customer data to Excel
// @Tags customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/export-excel-discount-cards [get]
func (h *CustomerHandler) ExportDiscountCardExcel(c *gin.Context) {
	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, _, err := h.service.ListDiscountCards(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create Excel
	f := excelize.NewFile()
	sheetName := "DiscountCards"
	f.SetSheetName("Sheet1", sheetName)

	// Header titles
	headers := []string{
		"ID", "ФИО", "Телефон", "Дата рождения", "Пол", "Баланс",
		"Филиал", "Тег", "Процент", "Штрих-код",
	}

	// Set header row
	if err := setExcelHeaders(f, sheetName, headers); err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Fill rows
	for i, dc := range res {
		row := strconv.Itoa(i + 2)

		f.SetCellValue(sheetName, "A"+row, dc.Id)
		f.SetCellValue(sheetName, "B"+row, dc.FullName)
		f.SetCellValue(sheetName, "C"+row, dc.Phone)
		f.SetCellValue(sheetName, "D"+row, dc.Birthday.Format("2006-01-02"))
		f.SetCellValue(sheetName, "E"+row, dc.Gender)
		f.SetCellValue(sheetName, "F"+row, dc.Balance)
		if dc.StoreName != "" {
			f.SetCellValue(sheetName, "G"+row, dc.StoreName)
		} else {
			f.SetCellValue(sheetName, "G"+row, "N/A")
		}
		if dc.TagName != "" {
			f.SetCellValue(sheetName, "H"+row, dc.TagName)
		} else {
			f.SetCellValue(sheetName, "H"+row, "N/A")
		}
		f.SetCellValue(sheetName, "I"+row, dc.Percent)
		f.SetCellValue(sheetName, "J"+row, dc.Barcode)
	}

	saveExcelToUploads(c, f, *h.log, "DiscountCards")
}

// Update godoc
// @Summary Update a customer
// @Description Update a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Param customer body domain.CustomerRequest true "Customer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [put]
func (h *CustomerHandler) Update(c *gin.Context) {
	var (
		body domain.CustomerRequest
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, domain.InvalidPhoneError)
		return
	}

	err = h.db.WithContext(ctx).
		Table("customers").
		Where("id = ?", id).
		Updates(&body).Error

	if err != nil {
		h.log.Errorf("could not update customer %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, body)
}

// SoftDelete godoc
// @Summary Soft delete a customer
// @Description Soft delete a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param ids body []string true "customer IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/soft-delete [delete]
func (h *CustomerHandler) SoftDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("customers").
		Where("id IN (?)", ids).
		Updates(map[string]any{
			"is_active":  false,
			"deleted_at": time.Now()}).Error
	if err != nil {
		h.log.Errorf("could not soft_delete customer: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, "DELETED")
}

// HardDelete godoc
// @Summary Hard delete a customer
// @Description Hard delete a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param ids body []string true "customer IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/hard-delete [delete]
func (h *CustomerHandler) HardDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err = h.db.
		WithContext(ctx).
		Table("customers").
		Where("id IN (?)", ids).
		Delete(&domain.Customer{}).Error
	if err != nil {
		h.log.Errorf("could not hard_delete customer: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, "DELETED")
}

// CreateTag godoc
// @Summary Create a new tag
// @Description Create a new tag from the request body
// @Tags tags
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param tag body domain.Tag true "Tag information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /tag [post]
func (h *CustomerHandler) CreateTag(c *gin.Context) {
	var (
		body domain.Tag
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	// create new tag
	err = h.db.
		WithContext(c.Request.Context()).
		Table("tags").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// TagList godoc
// @Summary Get all tags
// @Description Get all tags from the request body
// @Tags tags
// @Security     BearerAuth
// @Produce json
// @Param 	search query string false "Search Key"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /tag/list [get]
func (h *CustomerHandler) TagList(c *gin.Context) {
	var (
		res        []domain.Tag
		search     = c.Query("search")
		totalCount int64
	)
	// get limit and offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// build query for getting tag list
	query := h.db.Model(&domain.Tag{})

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	// complete query
	err = query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get _meta data
	data := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, data)
}
