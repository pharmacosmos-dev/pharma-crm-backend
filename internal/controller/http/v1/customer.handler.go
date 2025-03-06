package v1

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
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
	var (
		body     domain.CustomerRequest
		customer domain.Customer
		err      error
	)
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone number
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	// get user id
	createdBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// generate id
	body.Id = uuid.New().String()
	body.CreatedBy = cast.ToString(createdBy)
	// insert customer
	err = h.db.
		WithContext(c.Request.Context()).Raw(`
		INSERT INTO customers 
			(id, store_id, first_name, last_name, full_name, phone, gender, birthday, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *`,
		body.Id, body.StoreId, body.FirstName, body.LastName, body.FirstName+" "+body.LastName,
		body.Phone, body.Gender, body.Birthday, body.CreatedBy).Scan(&customer).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, customer)
}

// Get godoc
// @Summary Get a customer
// @Description Get a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [get]
func (h *CustomerHandler) Get(c *gin.Context) {
	var (
		customer domain.Customer
		id       = c.Param("id")
	)
	// validate uuid
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}
	err := h.db.
		Preload("Tag").
		First(&customer, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, customer)
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
	var (
		search  = c.Query("search")
		storeID = c.Query("store_id")
	)
	// get limit and offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get customers data
	res, totalCount, err := h.service.ListCustomer(search, storeID, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	// add _meta data
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// Export customer excel godoc
// @Summary Export customer excel
// @Description Export customer excel
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/export-excel [get]
func (h *CustomerHandler) ExportCustomerExcel(c *gin.Context) {
	var (
		search  = c.Query("search")
		storeID = c.Query("store_id")
	)
	// get limit and offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get customers data
	res, _, err := h.service.ListCustomer(search, storeID, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "Clients"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "ФИО", "Номер Телефона", "Теги", "Сумма покупки", "Последний покупка", "Дата рождения", "Дата регистрации", "Зарегистрируйтесь в филиале", "Баланс", "Текущий долг"}

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

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=clients.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
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
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("customers").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, InternalError, err.Error())
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
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("customers").
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_active":  false,
			"deleted_at": time.Now()}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
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
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("customers").
		Where("id IN (?)", ids).
		Delete(&domain.Customer{}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, InternalError, err.Error())
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
	handleResponse(c, OK, "CREATED")
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
		res    []domain.Tag
		search = c.Query("search")
	)
	// get all tags
	query := h.db.Model(&domain.Tag{})

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	err := query.Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
