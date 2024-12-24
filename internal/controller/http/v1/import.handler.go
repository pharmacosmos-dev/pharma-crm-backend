package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
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
	body.PublicID = utils.GenerateRandomCode()
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("imports").Create(&body).Error
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import/list [get]
func (h *ImportHandler) List(c *gin.Context) {
	var (
		imports    []domain.Import
		totalCount int64
		search     = c.Query("search")
		err        error
	)

	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Fetch imports with detailed data
	query := h.db.Model(&domain.Import{}).
		Preload("Stores").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*, 
			SUM(import_details.received_amount) as received_amount, 
			SUM(import_details.accepted_amount) as accepted_amount, 
			SUM(import_details.received_count) as received_count, 
			SUM(import_details.accepted_count) as accepted_count
		`).
		Joins("LEFT JOIN import_details ON imports.id = import_details.import_id")
	if search != "" {
		query = query.Where("imports.public_id = ?", search)
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /import-detail/list [get]
func (h *ImportHandler) ListImportDetail(c *gin.Context) {
	var (
		importDetails []domain.ImportDetail
		totalCount    int64
		err           error
		importId      = c.Query("import_id")
		search        = c.Query("search")
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
		Joins("LEFT JOIN products ON import_details.product_id = products.id").
		Where("import_id = ?", importId)

	if search != "" {
		query = query.Where("products.barcode ILIKE ? OR products.name ILIKE ?", search, search)
	}
	err = query.
		Order("created_at DESC").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
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
	var (
		body  domain.AddScanRequest
		count int64
		err   error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.
		Table("import_details").
		Where("import_details.import_id = ?", body.ImportID).
		Where("import_details.product_id = (SELECT id FROM products WHERE barcode = ?)", body.Barcode).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Product not found")
			return
		}
		handleResponse(c, InternalError, err.Error())
		return
	}
	if count == 0 {
		handleResponse(c, NotFound, "Product not found")
		return
	}
	if body.Count < 1 {
		body.Count = 1
	}
	query := h.db.WithContext(c.Request.Context()).
		Table("import_details").
		Where("import_details.import_id = ?", body.ImportID).
		Where("import_details.product_id = (SELECT id FROM products WHERE barcode = ?)", body.Barcode).
		UpdateColumn("accepted_count", gorm.Expr("accepted_count + ?", body.Count))
	if query.Error != nil {
		handleResponse(c, InternalError, query.Error.Error())
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
	var (
		id = c.Param("id")
	)
	err := h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Where("id = ?", id).
		UpdateColumn("status", config.COMPLETED_IMPORT).Error
	if err != nil {
		h.log.Warn("Error on accepting import: %v", err.Error())
		handleResponse(c, InternalError, "Error on accepting import")
		return
	}
	// Update accepted_count and accepted_amount using a raw SQL query
	rawQuery := `
		UPDATE import_details
		SET 
			accepted_count = (SELECT received_count FROM import_details WHERE import_id = ?),
			accepted_amount = (SELECT received_amount FROM import_details WHERE import_id = ?)
		WHERE import_id = ?
	`
	err = h.db.WithContext(c.Request.Context()).Exec(rawQuery, id, id, id).Error
	if err != nil {
		h.log.Warn("Error on accepting import detail: %v", err.Error())
		handleResponse(c, InternalError, "Error on accepting import detail")
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
	var (
		id = c.Param("id")
	)
	err := h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Where("id = ?", id).
		UpdateColumn("status", config.CANCELED_IMPORT).Error
	if err != nil {
		h.log.Warn("Error on accepting import: %v", err.Error())
		handleResponse(c, InternalError, "Error on accepting import")
		return
	}
	// Update accepted_count and accepted_amount using a raw SQL query
	rawQuery := `
		UPDATE import_details
		SET 
			canceled_count = (SELECT received_count FROM import_details WHERE import_id = ?)
		WHERE import_id = ?
	`
	err = h.db.WithContext(c.Request.Context()).Exec(rawQuery, id, id).Error
	if err != nil {
		h.log.Warn("Error on accepting import detail: %v", err.Error())
		handleResponse(c, InternalError, "Error on accepting import detail")
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

	err := h.db.
		WithContext(c.Request.Context()).
		Table("imports").
		Where("id = ?", id).
		UpdateColumn("status", config.COMPLETED_IMPORT).Error
	if err != nil {
		h.log.Warn("Error on accepting import: %v", err.Error())
		handleResponse(c, InternalError, "Error on accepting import")
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
