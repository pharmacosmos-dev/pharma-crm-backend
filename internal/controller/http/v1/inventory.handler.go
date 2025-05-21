package v1

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
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
		detail.GET("/detailed-flow", h.InventoryDetailedFlow)
		detail.GET("/export-excel", h.InventoryDetailExport)
		detail.POST("/upload-excel", h.InventoryDetailUpload)
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

	// UPDATE fact_quantity and scanned_count
	if request.FactQuantity > 0 {
		err = h.db.Exec(`
		UPDATE import_details 
		SET 
			scanned_count = ?
		WHERE id = ? AND import_id = ?;`,
			request.FactQuantity, request.Id, inventoryID).Error
		if err != nil {
			h.log.Warn("ERROR on updating inventory_details: %v", err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	}
	// update fact_unit and scanned_count
	if request.FactUnit > 0 {
		err = h.db.Exec(`
		UPDATE import_details 
		SET 
    		scanned_count = scanned_count + (? / p.unit_per_pack)
		FROM products p
		WHERE import_details.product_id = p.id AND import_details.id = ? AND import_id = ?;`,
			request.FactUnit, request.Id, inventoryID).Error
		if err != nil {
			h.log.Warn("ERROR on updating inventory_details: %v", err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
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
	res, err := h.service.ConfirmInventory(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm inventory")
		return
	}

	// attech inventory products to store_products
	err = h.service.AttachInventoryToStoreProduct(res)
	if err != nil {
		handleResponse(c, InternalError, "Failed to attech products")
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
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
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

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

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
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
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
	headers := []string{"Код", "Наименования", "УП", "Кол-во", "Кол-во", "Сумма", "Кол-во", "Кол-во", "Сумма", "Кол-во", "Кол-во", "Сумма"}

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
		f.SetCellValue(sheetName, "A"+row, imp.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, imp.Name)
		f.SetCellValue(sheetName, "C"+row, imp.UnitPerPack)
		f.SetCellValue(sheetName, "D"+row, imp.CurrentQuantity)
		f.SetCellValue(sheetName, "E"+row, fmt.Sprintf("%d(%d/%d)", int(imp.CurrentQuantity), int(imp.CurrentUnit), int(imp.UnitPerPack)))
		f.SetCellValue(sheetName, "F"+row, imp.CurrentSum)
		f.SetCellValue(sheetName, "G"+row, imp.FactQuantity)
		f.SetCellValue(sheetName, "H"+row, fmt.Sprintf("%d(%d/%d)", int(imp.FactQuantity), int(imp.FactUnit), int(imp.UnitPerPack)))
		f.SetCellValue(sheetName, "I"+row, imp.FactSum)
		f.SetCellValue(sheetName, "J"+row, imp.DifferenceQuantity)
		f.SetCellValue(sheetName, "K"+row, fmt.Sprintf("%d(%d/%d)", int(imp.DifferenceQuantity), int(imp.DifferenceUnit), int(imp.UnitPerPack)))
		f.SetCellValue(sheetName, "L"+row, imp.DifferenceSum)
	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=import-detail.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}

}

// Get List
// @Summary Get inventory detail flow
// @Description Get inventory detail flow
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   inventory_id query string true "Inventory ID"
// @Param   product_id 	query string true "Product ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/detailed-flow [GET]
func (h *InventoryHandler) InventoryDetailedFlow(c *gin.Context) {
	var param domain.InventoryDetailParam
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.InventoryDetailedFlow(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Upload inventory detail godoc
// @Summary Upload inventory detail excel
// @Description Upload inventory detail excel
// @Tags Inventory
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param 	file formData file true "Excel file (.xlsx) containing product data"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/upload-excel [POST]
func (h *InventoryHandler) InventoryDetailUpload(c *gin.Context) {
	var (
		file domain.File
		err  error
	)
	// bind request file
	if err = c.ShouldBind(&file); err != nil {
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

	// defer os.Remove(savePath)
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

	// build query
	query := `
	UPDATE import_details SET scanned_count = ?, supply_price_vat = ?, retail_price_vat = ?, expire_date = ?, updated_at = NOW()
	WHERE id = ?
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		count++
		fmt.Println("---->>> ", row[0])
		err := h.db.Exec(query, cast.ToFloat64(row[3]), cast.ToFloat64(row[5]), cast.ToFloat64(row[6]), row[4], row[0]).Error
		if err != nil {
			h.log.Warn("ERROR on updating products: %v", err)
		}
	}

	handleResponse(c, OK, "Successfully upload: "+cast.ToString(count))
}
