package v1

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type WriteOffHandler struct {
	*Handler
}

func (h *Handler) NewWriteOffHandler(r *gin.RouterGroup) {
	writeOffHandler := &WriteOffHandler{h}
	writeOffHandler.WriteOffRoutes(r)
}

func (h *WriteOffHandler) WriteOffRoutes(r *gin.RouterGroup) {
	writeOff := r.Group("/write-off")
	{
		writeOff.POST("", h.Create)
		writeOff.GET("/:id", h.Get)
		writeOff.GET("/confirm/:id", h.Confirm)
		writeOff.POST("/cancel/:id", h.Cancel)
		writeOff.GET("/list", h.List)
		writeOff.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		// imports.GET("/export-excel", h.ExportImportExcel)
	}
	detail := r.Group("write-off-detail")
	{
		detail.GET("/list", h.WriteOffDetailList)
	}

}

// Create godoc
// @Summary Create write-off
// @Description Create write-off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.WriteOffRequest true "WriteOff"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off [POST]
func (h *WriteOffHandler) Create(c *gin.Context) {
	var (
		body domain.WriteOffRequest
		err  error
	)
	// get user_id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found from the context")
		return
	}

	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Warn("ERROR on binding writeoff request: %v", err)
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// get created_by as a string
	body.CreatedBy = userId.(string)

	// create new write off
	err = h.service.CreateWriteOff(&body)
	if err != nil {
		h.log.Info("Failed to create new write off: %v", err)
		handleResponse(c, InternalError, "Can't create new write-off")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Write-Off
// @Description Get Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/{id} [GET]
func (h *WriteOffHandler) Get(c *gin.Context) {
	// get inventory by id
	id := c.Param("id")
	// validate  uuid
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid write-off id")
		return
	}
	// get write-off info
	res, err := h.service.GetWriteOffById(id)
	if err != nil {
		h.log.Warn("Error on getting write-off: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get write-off")
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get inventory list
// @Description Get inventory list
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   status 	query string false "STATUS (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/list [GET]
func (h *WriteOffHandler) List(c *gin.Context) {
	var param domain.WriteOffParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.WriteOffList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, data)
}

// confirm Write-Off
// @Summary Confirm Write-Off
// @Description Confirm Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/confirm/{id} [POST]
func (h *WriteOffHandler) Confirm(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid inventory id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.ConfirmWriteOff(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm write-off")
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel Write-Off
// @Summary Cancel Write-Off
// @Description Cancel Write-Off
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Write-Off ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/cancel/{id} [POST]
func (h *WriteOffHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate inventory id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid write-off id")
		return
	}
	// get user_id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.CancelWriteOff(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm write-off")
		return
	}

	handleResponse(c, OK, "CANCELED")
}

// Get List
// @Summary Get Write-Off detail list
// @Description Get Write-Off detail list
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   writeoff_id query string true "WriteOff ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off-detail/list [GET]
func (h *WriteOffHandler) WriteOffDetailList(c *gin.Context) {
	var param domain.WriteOffDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get write-off detail list
	res, totalCount, err := h.service.WriteOffDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get write-off detail list")
		return
	}
	// get write-off list _meta and pagination info
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Write-Off
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "WriteOff ID"
// @Param 	body body domain.InventoryAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /write-off/{id}/add-product-by-barcode [PATCH]
func (h *WriteOffHandler) AddProductByBarcode(c *gin.Context) {
	var (
		request domain.InventoryAddProduct
		id      = c.Param("id")
	)

	// validate uuid
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Inventory id is invalid")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	var productId string
	if request.ProductId != "" {
		productId = request.ProductId
	} else if request.Barcode != "" {
		err = h.db.Raw(`
		SELECT import_details.product_id FROM import_details JOIN products p ON p.id = import_details.product_id
		WHERE p.barcode = ? AND import_details.import_id = ?
		`, request.Barcode, id).Scan(&productId).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handleResponse(c, BadRequest, "Product barcode not found")
				return
			}
			h.log.Warn("ERROR on getting product by barcode: %v", err)
			handleResponse(c, InternalError, "Failed to find product by barcode")
			return
		}
	} else {
		handleResponse(c, BadRequest, "product_id or barcode not sent")
		return
	}

	if request.Type == "SCAN" {
		if request.Count == 0 {
			request.Count = 1
		}
		// add scanned count by product barcode
		err = h.db.Exec(`
		UPDATE import_details
		SET scanned_count = scanned_count + ?, updated_at = NOW()
		WHERE product_id = ? AND import_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	} else if request.Type == "MANUAL" {
		// add scanned count by product barcode
		err = h.db.Exec(`
		UPDATE import_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE product_id = ? AND import_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	}
	handleResponse(c, OK, "ADDED")
}
