package v1

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportHandler struct {
	*Handler
}

func (h *Handler) NewImportHandler(r *gin.RouterGroup) {
	importHandler := &ImportHandler{h}
	importHandler.ImportRoutes(r)
}

func (h *ImportHandler) ImportRoutes(r *gin.RouterGroup) {
	imports := r.Group("/import")
	{
		imports.POST("", h.Create)
		imports.GET("/:id", h.Get)
		imports.GET("/list", h.List)
		imports.POST("/excel-upload", h.UploadExcelFile)
	}
	importDetail := r.Group("/import-detail")
	{
		importDetail.POST("", h.CreateImportDetail)
		importDetail.GET("/list", h.ListImportDetail)
		importDetail.PATCH("/add-scan", h.AddScann)
		importDetail.PATCH("/accept-all/:id", h.AcceptImport)
		importDetail.PATCH("/cancel-all/:id", h.CancelImport)
		importDetail.PATCH("/accept-some/:id", h.AcceptSomeImport)
		importDetail.GET("/get-stock-status-counts/:id", h.GetStockStatusCounts)
		importDetail.PUT("/:id", h.UpdateImportDetail)
	}
}

// Create godoc
// @Summary Create an import
// @Description Create an import from the request body
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.ImportRequest true "Import information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import [post]
func (h *ImportHandler) Create(c *gin.Context) {
	var (
		body domain.ImportRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// First godoc
// @Summary First imports
// @Description First imports
// @Tags imports
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/{id} [get]
func (h *ImportHandler) Get(c *gin.Context) {
	var (
		res domain.Import
		err error
		id  = c.Param("id")
	)
	err = h.db.First(&res, "id = ?", id).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary List imports
// @Description List imports
// @Tags imports
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
// @Param   receive_amount_from 	query int false "Receive Amount From"
// @Param   receive_amount_to 	query int false "Receive Amount To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/list [get]
func (h *ImportHandler) List(c *gin.Context) {
	var (
		imports          []domain.Import
		totalCount       int64
		search           = c.Query("search")
		storeID          = c.Query("store_id")
		startDate        = c.Query("start_date")
		endDate          = c.Query("end_date")
		status           = c.Query("status")
		receivePriceFrom = c.Query("receive_amount_from")
		receivePriceTo   = c.Query("receive_amount_to")
		err              error
	)

	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Fetch imports with detailed data
	query := h.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*, 
			SUM(import_details.retail_price) as received_amount, 
			SUM(import_details.accepted_retail_price) as accepted_amount, 
			SUM(import_details.received_count) as received_count, 
			SUM(import_details.accepted_count) as accepted_count
		`).Joins("LEFT JOIN import_details ON imports.id = import_details.import_id")

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		imports.document_number ILIKE ? OR 
		CAST(imports.public_id AS TEXT) LIKE ?`, search, search)
	}
	if storeID != "" {
		query = query.Where("imports.store_id = ?", storeID)
	}
	if startDate != "" {
		query = query.Where("imports.import_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("imports.import_date <= ?", endDate)
	}
	if status != "" {
		query = query.Where("imports.status = ?", status)
	}
	if receivePriceFrom != "" {
		query = query.Where("received_amount >= ?", receivePriceFrom)
	}
	if receivePriceTo != "" {
		query = query.Where("received_amount <= ?", receivePriceTo)
	}
	err = query.Group("imports.id").
		Order("imports.import_date DESC").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Find(&imports).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare response
	data := utils.ListResponse(imports, totalCount, limit, offset)

	handleResponse(c, OK, data)
}

// Create godoc
// @Summary Create an import detail
// @Description Create an import detail from the request body
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.ImportDetailRequest true "Import detail information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail [post]
func (h *ImportHandler) CreateImportDetail(c *gin.Context) {
	var (
		body domain.ImportDetailRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).Table("import_details").Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// ListImportDetail godoc
// @Summary List import details
// @Description List import details
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   import_id query string true "Import ID"
// @Param   received_amount_from query int false "Received Amount From"
// @Param   received_amount_to query int false "Received Amount To"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/list [get]
func (h *ImportHandler) ListImportDetail(c *gin.Context) {
	var (
		importDetails      []domain.ImportDetail
		totalCount         int64
		err                error
		importId           = c.Query("import_id")
		search             = c.Query("search")
		receivedAmountFrom = c.Query("received_amount_from")
		receivedAmountTo   = c.Query("received_amount_to")
	)

	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Fetch import details with detailed data
	query := h.db.Model(&domain.ImportDetail{}).
		Preload("Product").
		Preload("Import").
		Select("import_details.*, unit_types.short_name as unit_name").
		Joins("LEFT JOIN products ON import_details.product_id = products.id").
		Joins("LEFT JOIN unit_types ON products.unit_type_id = unit_types.id").
		Where("import_id = ?", importId)

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		products.barcode LIKE ? OR 
		products.name ILIKE ? OR
		CAST(products.material_code AS TEXT) LIKE ?`, search, search, search)
	}
	if receivedAmountFrom != "" {
		query = query.Where("import_details.received_amount >= ?", receivedAmountFrom)
	}
	if receivedAmountTo != "" {
		query = query.Where("import_details.received_amount <= ?", receivedAmountTo)
	}
	err = query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&importDetails).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Prepare response
	data := utils.ListResponse(importDetails, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// AddScann godoc
// @Summary Add scan to import detail
// @Description Add scan to import detail
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.AddScanRequest true "Add scan information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/add-scan [PATCH]
func (h *ImportHandler) AddScann(c *gin.Context) {
	var body domain.AddScanRequest
	var surplus = false
	// Bind the JSON body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Check if the count is valid
	if body.Count < 1 {
		body.Count = 1
	}
	var importDetail domain.ImportDetail
	// Perform a single query to find and update the record
	result := h.db.WithContext(c.Request.Context()).
		Table("import_details").
		Where(`
			import_id = ? AND
			product_id IN (
				SELECT id
				FROM products
				WHERE barcode = ?
			)`,
			body.ImportID, body.Barcode).
		Update("accepted_count", gorm.Expr("accepted_count + ?", body.Count)).
		Scan(&importDetail)

	if result.RowsAffected == 0 {
		handleResponse(c, NotFound, "Product not found")
		return
	}
	// Check if the record was updated
	if result.Error != nil {
		h.log.Error("Error updating accepted_count: %v", result.Error)
		handleResponse(c, InternalError, result.Error.Error())
		return
	}

	if importDetail.AcceptedCount > importDetail.ReceivedCount {
		surplus = true
	}
	handleResponse(c, OK, map[string]interface{}{
		"surplus": surplus,
	})
}

// UpdateImportDetail
// @Summary Update an import detail
// @Description Update an import detail from the request body
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "import detail ID"
// @Param input body domain.ImportUpdateRequest true "Import detail information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/{id} [PUT]
func (h *ImportHandler) UpdateImportDetail(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.ImportUpdateRequest
	)

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err := h.db.
		WithContext(c.Request.Context()).
		Table("import_details").
		Where("id = ?", id).
		Update("accepted_count", body.ScannedCount).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// AcceptImport
// @Summary Accept import
// @Description Accept import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/accept-all/{id} [patch]
func (h *ImportHandler) AcceptImport(c *gin.Context) {
	var id = c.Param("id")
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update imports status to completed
	importData, err := h.service.UpdateImportByField(tx, id, "status", config.COMPLETED_IMPORT)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		h.db.Rollback()
		return
	}
	// add products to store
	err = h.service.AddAllProductsToStore(tx, importData)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		h.db.Rollback()
		return
	}

	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		h.db.Rollback()
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// CancelImport
// @Summary Cancel import
// @Description Cancel import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/cancel-all/{id} [patch]
func (h *ImportHandler) CancelImport(c *gin.Context) {
	var id = c.Param("id")
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update import status to cancel
	importData, err := h.service.UpdateImportByField(tx, id, "status", config.CANCELED_IMPORT)
	if err != nil {
		handleResponse(c, InternalError, "Error on canceling import")
		tx.Rollback()
		return
	}

	// update import details to canceled_count
	err = h.service.UpdateImportDetailsToCancel(tx, importData.Id)
	if err != nil {
		handleResponse(c, InternalError, "Error on canceling import")
		tx.Rollback()
		return
	}
	// completed transaction
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Error on commit transaction")
		tx.Rollback()
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// AcceptSomeImport
// @Summary Accept import
// @Description Accept import
// @Tags import_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/accept-some/{id} [patch]
func (h *ImportHandler) AcceptSomeImport(c *gin.Context) {
	var id = c.Param("id")
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update import status to completed
	importData, err := h.service.UpdateImportByField(tx, id, "status", config.COMPLETED_IMPORT)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	// add products import_details to store_products
	err = h.service.AddImportedProductsToStore(tx, importData)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	// check transaction is commit
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// GetStockStatusCounts
// @Summary Get stock status counts
// @Description Get stock status counts
// @Tags import_details
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "IMPORT ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/get-stock-status-counts/{id} [get]
func (h *ImportHandler) GetStockStatusCounts(c *gin.Context) {
	var id = c.Param("id")
	var res domain.StockCountResponse

	// Use raw SQL to calculate the counts with surplus condition
	query := `
		SELECT
			COALESCE(SUM(accepted_count), 0) AS scanned_count,
			COALESCE(SUM(received_count - accepted_count), 0) AS shortage_count,
			COALESCE(COUNT(*), 0) AS total_count,
			COALESCE(SUM(CASE WHEN accepted_count > received_count THEN accepted_count - received_count ELSE 0 END), 0) AS surplus_count
		FROM import_details
		WHERE import_id = ?
	`
	err := h.db.
		Raw(query, id).
		Scan(&res).Error
	if err != nil {
		h.log.Error("Error getting stock status counts: %v", err)
		handleResponse(c, InternalError, "Failed to fetch stock status counts")
		return
	}

	handleResponse(c, OK, res)
}

// UploadExcelFile
// @Summary Upload excel file
// @Description Upload excel file
// @Tags imports
// @Security     BearerAuth
// @Accept 	multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing import data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/excel-upload [post]
func (h *ImportHandler) UploadExcelFile(c *gin.Context) {
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
	defer os.Remove(savePath)

	// Open the Excel file
	xlsx, err := excelize.OpenFile(savePath)
	if err != nil {
		h.log.Error("Failed to open .xlsx file: ", err.Error())
		handleResponse(c, BadRequest, "Failed to process file")
		return
	}
	defer xlsx.Close()

	sheetName := xlsx.GetSheetName(1)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		h.log.Error("Failed to get rows: ", err.Error())
		handleResponse(c, InternalError, "Failed to get rows")
		return
	}

	var products []map[string]interface{}
	var categories []map[string]interface{}
	var categoryProduct []map[string]interface{}

	existingCategories := make(map[string]string) // Key: Category Name, Value: Category ID

	// Load existing categories from DB
	var dbCategories []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	h.db.Table("categories").Select("id, name").Find(&dbCategories)
	for _, c := range dbCategories {
		existingCategories[c.Name] = c.ID
	}
	for _, row := range rows[1:] {
		if len(row) > 7 {
			productID := uuid.New().String()
			products = append(products, map[string]interface{}{
				"id":            productID,
				"name":          row[1],
				"barcode":       row[2],
				"vat":           12,
				"supply_price":  parseFloat(row[3]) - (parseFloat(row[3])*12)/100,
				"retail_price":  parseFloat(row[3]),
				"material_code": row[4],
			})

			// Category
			categoryID, exists := existingCategories[row[5]]
			if !exists {
				categoryID = uuid.New().String()
				existingCategories[row[5]] = categoryID
				categories = append(categories, map[string]interface{}{
					"id":   categoryID,
					"name": row[5],
				})
			}

			// Subcategory
			subCategoryID, exists := existingCategories[row[6]]
			if !exists {
				subCategoryID = uuid.New().String()
				existingCategories[row[6]] = subCategoryID
				categories = append(categories, map[string]interface{}{
					"id":          subCategoryID,
					"name":        row[6],
					"category_id": categoryID,
				})
			}

			// Child Category
			childCategoryID, exists := existingCategories[row[7]]
			if !exists {
				childCategoryID = uuid.New().String()
				existingCategories[row[7]] = childCategoryID
				categories = append(categories, map[string]interface{}{
					"id":          childCategoryID,
					"name":        row[7],
					"category_id": subCategoryID,
				})
			}

			categoryProduct = append(categoryProduct, map[string]interface{}{
				"category_id": childCategoryID,
				"product_id":  productID,
				"is_open":     true,
			})
		} else {
			continue
		}
	}

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Debug().Table("products").Create(&products).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}

	err = tx.Debug().Table("categories").Create(&categories).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}

	err = tx.Debug().Table("category_products").Create(&categoryProduct).Error
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

	handleResponse(c, OK, "CREATED")
}
