package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
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
	err = h.db.WithContext(c.Request.Context()).Table("imports").Create(&body).Error
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

	// Count total records (without Joins or heavy Selects)
	countQuery := h.db.Model(&domain.Import{})
	if search != "" {
		countQuery = countQuery.Where("public_id = ?", search)
	}
	err = countQuery.Count(&totalCount).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
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
		Joins("LEFT JOIN import_details ON imports.id = import_details.import_id").
		Group("imports.id").
		Order("import_date DESC").
		Limit(limit).
		Offset(offset)

	if search != "" {
		query = query.Where("imports.public_id = ?", search)
	}

	err = query.Find(&imports).Error
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
