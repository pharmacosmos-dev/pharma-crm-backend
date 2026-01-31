package v1

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type ProducerHandler struct {
	*Handler
}

func (h *Handler) NewProducerHandler(r *gin.RouterGroup) {
	producer := &ProducerHandler{h}
	producer.ProducerRoutes(r)
}

func (h *ProducerHandler) ProducerRoutes(r *gin.RouterGroup) {
	producer := r.Group("/producer")
	{
		producer.POST("", h.Create)
		producer.GET("/list", h.List)
		producer.PUT("/:id", h.Update)
		producer.DELETE("/:id", h.Delete)
		producer.POST("/excel-upload", h.UploadProducer)
	}
	shelf := r.Group("/shelf")
	{
		shelf.POST("", h.CreateShelf)
		shelf.GET("/list", h.ListShelf)
		shelf.PUT("/:id", h.UpdateShelf)
		shelf.DELETE("/:id", h.DeleteShelf)
	}
}

// Create a new producer
// @Summary Create a new producer
// @Description Create a new producer
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.Producer true "Producer information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer 	[post]
func (h *ProducerHandler) Create(c *gin.Context) {
	var (
		body domain.Producer
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.Raw(`INSERT INTO producers (name) VALUES (?) RETURNING *`, body.Name).Scan(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// List all producers
// @Summary List all producers
// @Description List all producers
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param id query string false "producer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/list [get]
func (h *ProducerHandler) List(c *gin.Context) {
	var (
		res        []*domain.Producer
		err        error
		totalCount int64
		id         = c.Query("id")
		search     = c.Query("search")
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.Model(&domain.Producer{})

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}

	if id != "" {
		query = query.Where("id = ?", id)
	}

	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// Update a producer
// @Summary Update a producer
// @Description Update a producer from the request body
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "producer ID"
// @Param input body domain.Producer true "Producer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/{id} [put]
func (h *ProducerHandler) Update(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.Producer
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Model(&domain.Producer{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// Delete a producer
// @Summary Delete a producer
// @Description Delete a producer from the request body
// @Tags producers
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "producer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/{id} [delete]
func (h *ProducerHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Producer{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Create a new shelf
// @Summary Create a new shelf
// @Description Create a new shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.Shelf true "Shelf information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf 	[post]
func (h *ProducerHandler) CreateShelf(c *gin.Context) {
	var (
		body domain.Shelf
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Raw(`INSERT INTO shelves (name) VALUES (?) RETURNING *`, body.Name).Scan(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// List all shelves
// @Summary List all shelves
// @Description List all shelves
// @Tags shelves
// @Security     BearerAuth
// @Accept 	json
// @Produce 	json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param id query string false "shelf ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/list [get]
func (h *ProducerHandler) ListShelf(c *gin.Context) {
	var (
		res        []*domain.Shelf
		err        error
		search     = c.Query("search")
		id         = c.Query("id")
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.Model(&domain.Shelf{})
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	if id != "" {
		query = query.Where("id = ?", id)
	}
	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// Update a shelf
// @Summary Update a shelf
// @Description Update a shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "shelf ID"
// @Param input body domain.Shelf true "Shelf information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/{id} [put]
func (h *ProducerHandler) UpdateShelf(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.Shelf
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Model(&domain.Shelf{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete a shelf
// @Summary Delete a shelf
// @Description Delete a shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "shelf ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/{id} [delete]
func (h *ProducerHandler) DeleteShelf(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Shelf{}, "id = ?", id).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// UploadProduct godoc
// @Summary Upload a producer
// @Description Upload a producer file in .xlsx format. The file should include producer details in specific columns.
// @Tags 	producers
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing producer data"
// @Success 200 {object} v1.Response "Producers uploaded successfully"
// @Failure 400 {object} v1.Response "Invalid file format or processing error"
// @Failure 500 {object} v1.Response "Internal server error"
// @Router /producer/excel-upload [post]
func (h *ProducerHandler) UploadProducer(c *gin.Context) {
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

	// Process rows
	var producers []map[string]any
	for _, row := range rows[1:] {
		producers = append(producers, map[string]any{
			"name": row[0],
			"code": row[1],
		})
	}
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// create producers
	err = h.db.Table("producers").Create(&producers).Error
	if err != nil {
		_ = tx.Rollback()
		h.log.Error("Failed to create producers: ", err.Error())
		handleResponse(c, InternalError, "Failed to create producers")
		return
	}
	// complete transaction
	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "Products uploaded successfully")
}
