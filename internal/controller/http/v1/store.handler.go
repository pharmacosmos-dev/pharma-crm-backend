package v1

import (
	"fmt"
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

type StoreHandler struct {
	*Handler
}

func (h *Handler) NewStoreHandler(r *gin.RouterGroup) {
	store := &StoreHandler{h}
	store.StoreRoutes(r)
}

func (h *StoreHandler) StoreRoutes(r *gin.RouterGroup) {
	store := r.Group("/store")
	{
		store.POST("", h.Create)
		store.GET("/:id", h.Get)
		store.GET("/list", h.List)
		store.GET("/export-excel", h.ExportExcel)
		store.PUT("/:id", h.Update)
		store.DELETE("/:id", h.Delete)
		store.POST("/excel-upload", h.UploadExcel)

	}
}

// Create godoc
// @Summary Create a store
// @Description Create a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.StoreRequest true "Store information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [post]
func (h *StoreHandler) Create(c *gin.Context) {
	var (
		body domain.StoreRequest
		err  error
	)
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone number
	if body.Phone != nil && !utils.IsValidPhone(*body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number")
		return
	}

	// generate store id
	body.Id = uuid.New().String()
	// create new store info
	err = h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [get]
func (h *StoreHandler) Get(c *gin.Context) {
	var (
		res domain.Store
		err error
	)
	err = h.db.First(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/list [get]
func (h *StoreHandler) List(c *gin.Context) {
	var (
		res        []domain.StoreWithProducts
		totalCount int64
		CompanyID  string
		search     = c.Query("search")
		productID  = c.Query("product_id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		CompanyID = employee.CompanyId
	}

	query := h.db.
		Model(&domain.StoreWithProducts{}).Table("stores s")
	if productID != "" {
		query = query.Select(`
        s.*, 
        COALESCE(sp.pack_quantity, 0) AS pack_quantity, 
        sp.small_quantity,
        sp.expire_date, sp.vat, sp.markup, sp.retail_price, 
        sp.supply_price, sp.bonus_percent
    `).
			Joins(`
        LEFT JOIN (
            SELECT DISTINCT ON (sp.store_id) sp.*
            FROM store_products sp
            WHERE sp.product_id = ?
            ORDER BY sp.store_id, sp.created_at DESC
        ) sp ON s.id = sp.store_id
    `, productID)
	}

	if CompanyID != "" {
		query = query.Where("s.company_id = ?", CompanyID)
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("s.name ILIKE ?", search)
	}

	// Use conditional ordering at the end
	if productID != "" {
		query = query.Order("COALESCE(sp.pack_quantity, 0) DESC, s.store_code DESC")
	} else {
		query = query.Order("s.store_code DESC")
	}

	err = query.
		Where("s.is_active = ?", true).
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	var ids []string
	err = h.db.Table("stores").Select("id").Find(&ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, map[string]interface{}{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     limit,
			CurrentPage: (offset / limit) + 1,
			PageCount:   int((totalCount + int64(limit) - 1) / int64(limit)),
		},
		"data": res,
		"ids":  ids,
	})
}

// List godoc
// @Summary Get a store export in Excel format
// @Description Get a store export in Excel format
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/export-excel [get]
func (h *StoreHandler) ExportExcel(c *gin.Context) {
	var (
		res        []domain.StoreWithProducts
		totalCount int64
		companyId  string
		search     = c.Query("search")
		productID  = c.Query("product_id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// get employee info
	var employee domain.Employee
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		handleResponse(c, InternalError, "Can't get employee info")
		return
	}

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, h.cfg) {
		companyId = employee.CompanyId
	}

	query := h.db.
		Model(&domain.StoreWithProducts{}).Table("stores s")
	if productID != "" {
		query = query.Select(`
		s.*, 
		COALESCE(sp.pack_quantity, 0) AS pack_quantity, 
		sp.small_quantity,
		sp.expire_date, sp.vat, sp.markup, sp.retail_price, 
		sp.supply_price, sp.bonus_percent
	`).
			Joins(`
		LEFT JOIN (
			SELECT DISTINCT ON (sp.store_id) sp.*
			FROM store_products sp
			WHERE sp.product_id = ?
			ORDER BY sp.store_id, sp.created_at DESC
		) sp ON s.id = sp.store_id
	`, productID)
	}
	if companyId != "" {
		query = query.Where("s.company_id = ?", companyId)
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("s.name ILIKE ?", search)
	}

	err = query.
		Where("s.is_active = ?", true).
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("s.store_code DESC").
		Find(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "Режим работы", "Адрес", "Телефон"}

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
		f.SetCellValue(sheetName, "A"+row, r.StoreCode)
		f.SetCellValue(sheetName, "B"+row, r.Name)
		f.SetCellValue(sheetName, "C"+row, r.WorkHours)
		f.SetCellValue(sheetName, "D"+row, r.Address)
		f.SetCellValue(sheetName, "E"+row, r.Phone)
	}

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Filiallar_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
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

// Update godoc
// @Summary Update a store
// @Description Update a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Param store body domain.StoreRequest true "Store information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [put]
func (h *StoreHandler) Update(c *gin.Context) {
	var (
		body domain.StoreUpdateRequest
		id   = c.Param("id")
		err  error
	)
	// validate uuid
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}
	// get user_id from context
	updatedBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// validate phone number
	if body.Phone != nil && !utils.IsValidPhone(*body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.UpdatedBy = updatedBy.(string)
	// update store info
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a store
// @Description Delete a store from the request body
// @Tags 		 stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [delete]
func (h *StoreHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id = ?", id).
		Update("is_active", false).
		Update("deleted_at", time.Now()).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// UploadStore godoc
// @Summary Upload a stores
// @Description Upload a store file in .xlsx format. The file should include product details in specific columns.
// @Tags stores
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response "Products uploaded successfully"
// @Failure 400 {object} v1.Response "Invalid file format or processing error"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /store/excel-upload [post]
func (h *StoreHandler) UploadExcel(c *gin.Context) {
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
	var stores []map[string]interface{}
	for _, row := range rows[2:] {
		stores = append(stores, map[string]interface{}{
			"name":       row[0],
			"store_code": parseIntComma(row[1]),
		})
	}

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	err = tx.Table("stores").Create(&stores).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	handleResponse(c, OK, "UPLOADED")
}
