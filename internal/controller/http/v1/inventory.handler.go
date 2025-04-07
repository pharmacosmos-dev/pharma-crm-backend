package v1

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type InventoryHandler struct {
	*Handler
}

func (h *Handler) NewInventoryHandler(r *gin.RouterGroup) {
	inventoryHandler := &InventoryHandler{h}
	inventoryHandler.InventoryRoutes(r)
}

func (h *InventoryHandler) InventoryRoutes(r *gin.RouterGroup) {
	inventory := r.Group("/inventory")
	{
		inventory.POST("", h.Create)
		inventory.GET("/:id", h.Get)
		inventory.GET("/list", h.List)
		inventory.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		// inventory.GET("/export-excel", h.ExportImportExcel)
		// inventory.POST("/excel-upload", h.UploadExcelFile)
	}
	detail := r.Group("inventory-detail")
	{
		detail.GET("/list", h.InventoryDetailList)
	}

}

// Create godoc
// @Summary Create inventory
// @Description Create inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	inventory body domain.InventoryRequest true "Inventory"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory [POST]
func (h *InventoryHandler) Create(c *gin.Context) {
	var inventoryRequest domain.InventoryRequest
	// Bind the request body to the InventoryRequest struct
	if err := c.ShouldBindJSON(&inventoryRequest); err != nil {
		h.log.Warn("Error on binding request: %v", err.Error())
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}
	inventoryRequest.CreatedBy = userId.(string)
	// create inventory
	err := h.service.CreateInventory(&inventoryRequest)
	if err != nil {
		h.log.Warn("Error on creating inventory: %v", err.Error())
		handleResponse(c, InternalError, "Failed to create inventory")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get inventory
// @Description Get inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Inventory ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/{id} [GET]
func (h *InventoryHandler) Get(c *gin.Context) {
	// get inventory by id
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid inventory id")
		return
	}
	var inventory domain.Inventory
	err := h.db.Preload("Store").First(&inventory, "id = ?", id).Error
	if err != nil {
		h.log.Warn("Error on getting inventory: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get inventory")
		return
	}

	handleResponse(c, OK, inventory)
}

// Get List
// @Summary Get inventory list
// @Description Get inventory list
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Param   status 	query string false "STATUS (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/list [GET]
func (h *InventoryHandler) List(c *gin.Context) {
	var param domain.InventoryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.InventoryList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, data)
}

// Get List
// @Summary Get inventory list
// @Description Get inventory list
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   inventory_id query string true "Inventory ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/list [GET]
func (h *InventoryHandler) InventoryDetailList(c *gin.Context) {
	var param domain.InventoryDetailParam
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.InventoryDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}
	statsCount, err := h.service.InventoryDetailStatsCount(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail stats count")
		return
	}
	data := map[string]any{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     param.Limit,
			CurrentPage: (param.Offset / param.Limit) + 1,
			PageCount:   int((totalCount + int64(param.Limit) - 1) / int64(param.Limit)),
		},
		"stats_count": statsCount,
		"data":        res,
	}

	handleResponse(c, OK, data)
}

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Inventory ID"
// @Param 	body body domain.InventoryAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/{id}/add-product-by-barcode [PATCH]
func (h *InventoryHandler) AddProductByBarcode(c *gin.Context) {
	var request domain.InventoryAddProduct
	id := c.Param("id")
	// validate inventory id
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
		SELECT inventory_details.product_id FROM inventory_details JOIN products p ON p.id = inventory_details.product_id
		WHERE p.barcode = ? AND inventory_details.inventory_id = ?
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
		UPDATE inventory_details
		SET scanned_count = scanned_count + ?, updated_at = NOW()
		WHERE product_id = ? AND inventory_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	} else if request.Type == "MANUAL" {
		// add scanned count by product barcode
		err = h.db.Exec(`
		UPDATE inventory_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE product_id = ? AND inventory_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	}
	handleResponse(c, OK, "ADDED")
}
