package v1

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ProductBonusHandler struct {
	*Handler
}

func (h *Handler) NewProductBonusHandler(r *gin.RouterGroup) {
	productBonus := &ProductBonusHandler{h}
	productBonus.ProductBonusRoutes(r)
}

func (h *ProductBonusHandler) ProductBonusRoutes(r *gin.RouterGroup) {
	bonus := r.Group("/product-bonus")
	{
		bonus.POST("", h.Create)
		bonus.GET("/:id", h.Get)
		bonus.GET("/list", h.List)
		bonus.GET("/export", h.ExcelExport)
		bonus.PUT("/:id", h.Update)
		bonus.POST("/excel-import", h.ImportProductBonus)
		bonus.DELETE("", h.Delete)
		bonus.POST("/sold", h.SoldProductBonus)
		bonus.POST("/balance", h.BalanceProductBonus)
	}
}

// create product bonus
// @Summary Create product bonus
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.ProductBonusRequest true "Product Bonus"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus [post]
func (h *ProductBonusHandler) Create(c *gin.Context) {
	var (
		body domain.ProductBonusRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	companyId, ok := c.Get("company_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "Company ID not found")
		return
	}
	body.CompanyId = companyId.(string)
	// check product bonus with product id
	var count int64
	err = h.db.Table("product_bonuses").Where("product_id = ?", body.ProductId).Count(&count).Error
	if err != nil {
		h.log.Warn("ERROR on checking product bonus count: %v", err)
		handleResponse(c, InternalError, "Failed to check product bonus")
		return
	}
	// checking product bonus count
	if count > 0 {
		handleResponse(c, BadRequest, "Can't create duplicate bonus for one product")
		return
	}

	// create new product bonus
	body.Status = 1
	err = h.db.Table("product_bonuses").Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "CREATED")
}

// get product bonus
// @Summary Get product bonus
// @Description Get product bonus
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "Product Bonus ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/{id} [get]
func (h *ProductBonusHandler) Get(c *gin.Context) {
	var (
		id  = c.Param("id")
		res domain.ProductBonus
	)
	// validate id
	if id == "" {
		handleResponse(c, BadRequest, "invalid product bonus id")
		return
	}
	// get one product bonus
	err := h.db.Table("product_bonuses").First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// get product bonus list
// @Summary Get product bonus list
// @Description Get product bonus list
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param 	store_id query string false "Store ID"
// @Param   search  query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/list [get]
func (h *ProductBonusHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// get default limit offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreID = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get bonus product list
	res, totalCount, err := h.service.GetProductBonuses(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// get with pagination data
	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	// return response
	handleResponse(c, OK, data)
}

// get product bonus list
// @Summary Get product bonus list
// @Description Get product bonus list
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit    query int false "Limit"
// @Param 	offset   query int false "Offset"
// @Param 	store_id query string false "Store ID"
// @Param   search   query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/export [get]
func (h *ProductBonusHandler) ExcelExport(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.QueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// get default limit offset
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreID = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	// get bonus product list
	res, _, err := h.service.GetProductBonuses(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "NAME", "BONUS", "START_DATE", "END_DATE"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create excel style:", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// Set information to columns
	for i, value := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, value.Product.Value.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, value.Product.Value.Name)
		f.SetCellValue(sheetName, "C"+row, value.BonusAmount)
		f.SetCellValue(sheetName, "D"+row, value.StartDate)
		f.SetCellValue(sheetName, "E"+row, value.EndDate)
	}

	saveExcelToUploads(c, f, *h.log, "bonusli_mahsulotlar")

}

// update product bonus
// @Summary Update product bonus
// @Description Update product bonus
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param  id path string true "Product Bonus ID"
// @Param  body body domain.ProductBonusRequest true "Product Bonus"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/{id} [put]
func (h *ProductBonusHandler) Update(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.ProductBonusRequest
		err  error
	)
	// validate id
	if id == "" {
		handleResponse(c, BadRequest, "invalid.id")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Status = 1
	// update product bonus
	err = h.db.Table("product_bonuses").Where("id = ?", id).Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// import product bonus
// @Summary Import product bonus
// @Description Import product bonus
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing bonus data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/excel-import [post]
func (h *ProductBonusHandler) ImportProductBonus(c *gin.Context) {
	var file domain.File
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Check file extension
	ext := filepath.Ext(file.File.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		h.log.Error("Unsupported file format: ", ext)
		handleResponse(c, BadRequest, "Unsupported file format")
		return
	}

	// Save the uploaded file
	newFilename := uuid.New().String() + ext
	savePath := filepath.Join("uploads", newFilename)
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}
	//
	defer os.Remove(savePath)
	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Error("Failed to open .xlsx file: ", err.Error())
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()
	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// create query
	query := `
	INSERT INTO  product_bonuses(product_id, bonus_amount, start_date, end_date)
	SELECT id, ?, '2025-03-10', '2050-03-10'
	FROM products WHERE barcode = ? AND barcode is not null and barcode <> ''`
	count := 0
	// Process rows
	for _, row := range rows[1:] {
		if len(row) > 3 {
			err = tx.Exec(query, parseFloat(row[2]), row[3]).Error
			if err != nil {
				_ = tx.Rollback()
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			count++
		}
	}

	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
}

// delete product bonus
// @Summary Delete product bonus
// @Description Delete product bonus
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param  	ids body []int true "Product Bonus IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus [delete]
func (h *ProductBonusHandler) Delete(c *gin.Context) {
	var (
		ids []int
		err error
	)
	// bind request body
	if err = c.ShouldBindJSON(&ids); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// delete products
	err = h.db.Table("product_bonuses").Delete(&domain.ProductBonus{}, "id in (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// BalanceProductBonus godoc
// get employee bonus balance
// @Summary Get employee bonus balance
// @Description Get total bonus amount and total sale count for current employee
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/balance [post]
func (h *ProductBonusHandler) BalanceProductBonus(c *gin.Context) {

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	// get employee info
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	res := domain.EmployeeBonusBalance

	// query to aggregate employee bonus data
	err = h.db.
		Model(&domain.EmployeeBonus{}).
		Select(`
			COALESCE(SUM(bonus_amount), 0) AS total_bonus,
			COUNT(DISTINCT sale_id) AS total_sales,
			COALESCE(SUM(quantity), 0) AS total_products
		`).
		Where("employee_id = ?", employee.Id).
		Scan(&res).Error

	if err != nil {
		h.log.Error("failed to get employee bonus balance: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, res)
}

// get sold product bonus list
// @Summary Get sold product bonus list
// @Description Get sold product bonus list
// @Tags Product Bonus
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit 		query int false "Limit"
// @Param 	offset 		query int false "Offset"
// @Param 	store_id 	query string false "Store ID"
// @Param 	search 		query string false "Search"
// @Param 	employee_id 	query string false "Employee ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product-bonus/sold [post]
func (h *ProductBonusHandler) SoldProductBonus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.QueryParam

	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	params.EmployeeId = user.UserId

	// get sold product bonuses
	res, totalCount, err := h.service.SoldProductBonusList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}
