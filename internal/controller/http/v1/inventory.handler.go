package v1

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
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
		inventory.POST("/confirm/:id", h.Confirm)
		inventory.POST("/cancel/:id", h.Cancel)
	}
	detail := r.Group("inventory-detail")
	{
		detail.GET("/list", h.InventoryDetailList)
		detail.GET("/export-excel", h.InventoryDetailExport)
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

	res, err := h.service.GetInventoryById(id)
	if err != nil {
		h.log.Warn("Error on getting inventory: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get inventory")
		return
	}

	handleResponse(c, OK, res)
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
	inventoryID := c.Param("id")
	// validate inventory id
	if err := uuid.Validate(inventoryID); err != nil {
		handleResponse(c, BadRequest, "Inventory id is invalid")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}

	// add scanned count by product barcode
	err = h.db.Exec(`
		UPDATE import_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE id = ? AND import_id = ?`,
		request.Count, request.Id, inventoryID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to add count")
		return
	}

	handleResponse(c, OK, "ADDED")
}

// confirm inventory
// @Summary Confirm Inventory
// @Description Confirm Inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Inventory ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/confirm/{id} [POST]
func (h *InventoryHandler) Confirm(c *gin.Context) {
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
	err := h.service.ConfirmInventory(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm inventory")
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel inventory
// @Summary Cancel Inventory
// @Description Cancel Inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Inventory ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/cancel/{id} [POST]
func (h *InventoryHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate inventory id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid inventory id")
		return
	}
	// get user_id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.CancelInventory(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm inventory")
		return
	}

	handleResponse(c, OK, "CANCELED")
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
	// get inventory details status count
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

// Get List
// @Summary Get inventory list
// @Description Get inventory list
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   inventory_id query string true "Inventory ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/export-excel [GET]
func (h *InventoryHandler) InventoryDetailExport(c *gin.Context) {
	var param domain.InventoryDetailParam
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.InventoryDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Код", "Наименования", "Штрих-Код", "Текущее кол-во", "Срок Годности", "Цена Поставки СНДС", "Цена Продажа СНДС"}

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
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Id)
		f.SetCellValue(sheetName, "B"+row, imp.MaterialCode)
		f.SetCellValue(sheetName, "C"+row, imp.Name)
		f.SetCellValue(sheetName, "D"+row, imp.Barcode)
		f.SetCellValue(sheetName, "E"+row, imp.ReceivedCount)
		if imp.ExpireDate != nil {
			f.SetCellValue(sheetName, "F"+row, imp.ExpireDate.Format(time.DateOnly))
		} else {
			f.SetCellValue(sheetName, "F"+row, "N/A")
		}
		f.SetCellValue(sheetName, "G"+row, imp.SupplyPriceVat)
		f.SetCellValue(sheetName, "H"+row, imp.RetailPriceVat)
	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=import-detail.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}

}
