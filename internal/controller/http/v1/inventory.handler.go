package v1

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
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
		inventory.GET("/list-status", h.GetInventoryStats)
		inventory.GET("/export-excel", h.InventoryExportExcel)
		inventory.PATCH("/:id/add-product-by-barcode", h.UpdateFactQuantity)
		inventory.PATCH("/:id/detailed-flow", h.UpdateDetailedFactQuantity)
		inventory.POST("/confirm/:id", h.Confirm)
		inventory.POST("/send1c/:id", h.Send1C)
		inventory.POST("/cancel/:id", h.Cancel)
		inventory.DELETE("/:inventory_id", h.DeleteInventory)
	}
	detail := r.Group("inventory-detail")
	{
		detail.GET("/list", h.InventoryDetailList)
		detail.GET("/detailed-flow", h.InventoryDetailedFlow)
		detail.GET("/export-excel", h.InventoryDetailExport)
		detail.POST("/upload-excel", h.InventoryDetailUpload)
		detail.GET("/price-option", h.PriceOption)
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var inventoryRequest domain.InventoryRequest
	err := c.ShouldBindJSON(&inventoryRequest)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	inventoryRequest.CreatedBy = user.UserId
	// create inventory
	err = h.service.CreateInventory(ctx, &inventoryRequest)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE (ALL||surplus||shortage||zero_price)"
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/{id} [GET]
func (h *InventoryHandler) Get(c *gin.Context) {
	var params domain.InventoryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}

	// get inventory by id
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	params.InventoryId = id

	res, err := h.service.GetInventoryById(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.InventoryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	res, totalCount, err := h.service.GetInventories(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	data := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, data)
}

// InventoryStatus godoc
// @Summary Get inventory total stats
// @Description Get total sum and count stats for inventories
// @Tags Inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param   store_id query string false "Store ID"
// @Param   search query string false "Search Keyword"
// @Param   type query string false "Inventory Type"
// @Param   status query string false "Inventory Status (0->new|1->pending|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/list-status [get]
func (h *InventoryHandler) GetInventoryStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.InventoryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	res, err := h.service.GetInventoryStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, res)
}

// Get List Export excel
// @Summary Get inventory Export excel
// @Description Get inventory list Export excel
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
// @Router /inventory/export-excel [GET]
func (h *InventoryHandler) InventoryExportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var params domain.InventoryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}
	res, _, err := h.service.GetInventories(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименования", "Филиал", "Текущее Кол-во", "Факт Кол-во", "Разница Кол-во", "Текущие Сумма", "Факт Сумма", "Статус", "Создание", "Завершение", "Создал", "Завершил"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.PublicId)
		f.SetCellValue(sheetName, "B"+row, imp.Name)
		if imp.Store.Valid {
			f.SetCellValue(sheetName, "C"+row, imp.Store.Value.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		f.SetCellValue(sheetName, "D"+row, imp.CurrentCount)
		f.SetCellValue(sheetName, "E"+row, imp.FactCount)
		f.SetCellValue(sheetName, "F"+row, imp.DifferenceCount)
		f.SetCellValue(sheetName, "G"+row, imp.CurrentSum)
		f.SetCellValue(sheetName, "H"+row, imp.FactSum)
		f.SetCellValue(sheetName, "I"+row, imp.FactSum)
		f.SetCellValue(sheetName, "J"+row, helper.StatusToRussian(imp.Status))
		if imp.CreatedAt != nil {
			f.SetCellValue(sheetName, "K"+row, imp.CreatedAt.Format("2006-01-02 15:04:05"))
		} else {
			f.SetCellValue(sheetName, "K"+row, "N/A")
		}
		if imp.UpdatedAt != nil {
			f.SetCellValue(sheetName, "L"+row, imp.UpdatedAt.Format("2006-01-02 15:04:05"))
		} else {
			f.SetCellValue(sheetName, "L"+row, "N/A")
		}
		if imp.CreatedBy.Valid {
			f.SetCellValue(sheetName, "M"+row, imp.CreatedBy.Value.FullName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}
		if imp.UpdatedBy.Valid {
			f.SetCellValue(sheetName, "N"+row, imp.UpdatedBy.Value.FullName)
		} else {
			f.SetCellValue(sheetName, "N"+row, "N/A")
		}
	}
	saveExcelToUploads(c, f, *h.log, "inventory")
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
func (h *InventoryHandler) UpdateFactQuantity(c *gin.Context) {
	var (
		request     domain.InventoryAddProduct
		inventoryID = c.Param("id")
		err         error
	)
	// validate inventory id
	err = uuid.Validate(inventoryID)
	if err != nil {
		handleResponse(c, BadRequest, "Inventory id is invalid")
		return
	}
	// parse
	err = c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// update barcode
	if request.Barcode != "" && request.RetailPrice > 0 {
		err = h.db.Exec(`UPDATE products SET barcode = ? WHERE id = ?`, request.Barcode, request.Id).Error
		if err != nil {
			h.log.Warn("ERROR on updating product barcode: %v", err)
			handleResponse(c, InternalError, "Failed to update barcode")
			return
		}
		err = h.db.Exec(`UPDATE import_details SET retail_price_vat = ? WHERE retail_price_vat = 0 AND product_id = ? AND import_id = ?`,
			request.RetailPrice, request.Id, inventoryID).Error
		if err != nil {
			h.log.Warn("ERROR on updating retail_price_vat on inventory_details: %v", err)
			handleResponse(c, InternalError, "Failed to update retail_price")
			return
		}

		handleResponse(c, OK, "UPDATED")
		return
	}

	var res []domain.ImportDetail
	// find import_detail row
	err = h.db.Raw(`
	SELECT 
		import_details.*, 
		p.unit_per_pack
	FROM 
		import_details
	JOIN 
		products p ON p.id = import_details.product_id
	WHERE 
		product_id = ? AND import_id = ? ORDER BY imported_at
`, request.Id, inventoryID).Scan(&res).Error
	if err != nil {
		h.log.Warn("Failed to find import_detail row: %v", err)
		handleResponse(c, InternalError, "Not found import detail row")
		return
	}

	if len(res) == 0 {
		handleResponse(c, NotFound, "No import detail rows found")
		return
	}

	unitPerPack := res[0].UnitPerPack
	if unitPerPack <= 0 {
		handleResponse(c, BadRequest, "Invalid unit_per_pack value")
		return
	}

	if request.FactQuantity == 0 && request.FactUnit == 0 {
		err = h.db.Model(&domain.ImportDetail{}).
			Where("import_id = ? AND product_id = ?", inventoryID, request.Id).
			Update("scanned_count", 0).Error
		if err != nil {
			h.log.Warn("Error on updating scanned_count: %v", err)
			handleResponse(c, InternalError, "Failed to update scanned_count")
			return
		}
		handleResponse(c, OK, "Scanned count reset to zero")
		return
	}

	// Calculate total fact as quantity + (unit / unitPerPack)
	remainingFact := request.FactQuantity*float64(unitPerPack) + request.FactUnit
	for i := 0; i < len(res); i++ {
		if i == len(res)-1 {
			res[i].ScannedCount = remainingFact
		} else {
			available := res[i].ReceivedCount
			if remainingFact >= available {
				res[i].ScannedCount = available
				remainingFact -= available
			} else {
				res[i].ScannedCount = remainingFact
				remainingFact = 0
			}
		}
		// Update each row
		err := h.db.Exec(`
			UPDATE 
				import_details
			SET 
				scanned_count = scanned_count+?
			WHERE id = ?
		`, res[i].ScannedCount, res[i].Id).Error
		if err != nil {
			h.log.Warn("Failed to update scanned_count: %v", err)
			handleResponse(c, InternalError, "Failed to update scanned count")
			return
		}
	}

	handleResponse(c, OK, "UPDATED")
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
// @Router /inventory/{id}/detailed-flow [PATCH]
func (h *InventoryHandler) UpdateDetailedFactQuantity(c *gin.Context) {
	var (
		request     domain.InventoryAddProduct
		inventoryID = c.Param("id")
		err         error
	)
	// validate inventory id
	err = uuid.Validate(inventoryID)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.inventory.id")
		return
	}
	// bind request body
	err = c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	// update retail price
	if request.RetailPrice > 0 && request.ExpireDate != "" {
		err = h.db.Exec(`UPDATE import_details SET retail_price_vat = ? WHERE id = ?`,
			request.RetailPrice, request.Id).Error
		if err != nil {
			h.log.Warn("ERROR on updating retail_price_vat on inventory_details: %v", err)
			handleResponse(c, InternalError, "Failed to update retail_price")
			return
		}

		err = h.db.Exec(`UPDATE import_details SET expire_date = ? WHERE id = ?`, request.ExpireDate, request.Id).Error
		if err != nil {
			h.log.Warn("ERROR on updating expire_date on inventory details: %v", err)
			handleResponse(c, InternalError, "failed_to_update_expire_date")
			return
		}

		handleResponse(c, OK, "UPDATED")
		return
	}

	if request.FactQuantity == 0 && request.FactUnit == 0 {
		err = h.db.Exec(`
		UPDATE import_details
		SET scanned_count = 0
		WHERE id = ?
	`, request.Id).Error
		if err != nil {
			h.log.Warn("Error on updating scanned_count: %v", err)
			handleResponse(c, InternalError, "Failed to update scanned_count")
			return
		}
		handleResponse(c, OK, "Scanned count reset to zero")
		return
	}

	// update fact quantity
	if request.FactQuantity > 0 {
		err = h.db.Exec(`
		UPDATE 
			import_details
		SET 
			scanned_count = scanned_count+(?::numeric * p.unit_per_pack)
		FROM 
			products p
		WHERE
			import_details.product_id = p.id AND import_details.id = ?
	`, request.FactQuantity, request.Id).Error
		if err != nil {
			h.log.Warn("Error on updating scanned_count: %v", err)
			handleResponse(c, InternalError, "Failed to update scanned_count")
			return
		}
	}

	// update fact unit
	if request.FactUnit > 0 {
		err = h.db.Exec(`
		UPDATE 
			import_details
		SET 
			scanned_count = scanned_count + ?
		FROM 
			products p
		WHERE 
			import_details.product_id = p.id 
			AND 
			import_details.id = ?
	`, request.FactUnit, request.Id).Error
		if err != nil {
			h.log.Warn("Error on updating scanned_count: %v", err)
			handleResponse(c, InternalError, "Failed to update scanned_count")
			return
		}
	}

	handleResponse(c, OK, "UPDATED")
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
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var id = c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get inventory by id
	var res domain.Inventory
	err := h.db.WithContext(ctx).Raw(`SELECT * FROM imports WHERE id = ?`, id).Scan(&res).Error
	if err != nil {
		h.log.Errorf("could not get inventory: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// check if inventory is already confirmed
	if res.Status == constants.GeneralStatusCompleted {
		handleServiceResponse(c, CONFLICT, domain.AlreadyCompletedError)
		return
	}

	// confirm inventory service
	err = h.service.ConfirmInventory(ctx, id, user.UserId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// send1c inventory
// @Summary Send Inventory
// @Description Send Inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Inventory ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/send1c/{id} [POST]
func (h *InventoryHandler) Send1C(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid inventory id")
		return
	}
	// send inventory info to 1C
	err := h.service.SendInventory1C(id)
	if err != nil {
		h.log.Warn("ERROR on sending inventory to 1c: %v", err)
		handleResponse(c, InternalError, "failed_to_send_inventory")
		return
	}
	handleResponse(c, OK, "SENT")
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
	err := uuid.Validate(id)
	if err != nil {
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
	err = h.service.CancelInventory(id, userId.(string))
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
// @Param   type 	query string false "TYPE (ALL||surplus||shortage||zero_price||checking)"
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/list [GET]
func (h *InventoryHandler) InventoryDetailList(c *gin.Context) {
	var param domain.InventoryParam
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalData, totalCount, err := h.service.InventoryDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}

	data := map[string]any{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     param.Limit,
			CurrentPage: (param.Offset / param.Limit) + 1,
			PageCount:   int((totalCount + int64(param.Limit) - 1) / int64(param.Limit)),
		},
		"data":       res,
		"total_data": totalData,
	}

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
// @Param   order 	query string false "ORDER (+name|-name|+current_sum|-current_sum|+fact_sum|-fact_sum|+difference_sum|-difference_sum)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/export-excel [GET]
func (h *InventoryHandler) InventoryDetailExport(c *gin.Context) {
	var param domain.InventoryParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, _, err := h.service.InventoryDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Код", "Наименования", "УП", "Програм Кол-во", "Програм Кол-во", "Програм Сумма", "Факт Кол-во", "Факт Кол-во", "Факт Сумма", "Разница Кол-во", "Разница Кол-во", "Разница Сумма"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
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

	saveExcelToUploads(c, f, *h.log, "inventory_details")
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
	var param domain.InventoryParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalData, totalCount, err := h.service.InventoryDetailedFlow(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}

	data := map[string]any{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     param.Limit,
			CurrentPage: (param.Offset / param.Limit) + 1,
			PageCount:   int((totalCount + int64(param.Limit) - 1) / int64(param.Limit)),
		},
		"data":       res,
		"total_data": totalData,
	}

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
	err = c.ShouldBind(&file)
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
	UPDATE 
		import_details 
	SET 
		scanned_count = ?, 
		supply_price_vat = ?, 
		retail_price_vat = ?, 
		expire_date = ?, 
		updated_at = NOW()
	WHERE id = ?
	`

	var count = 0
	// Process rows
	for _, row := range rows[1:] {
		count++
		err := h.db.Exec(query, cast.ToFloat64(row[3]), cast.ToFloat64(row[5]), cast.ToFloat64(row[6]), row[4], row[0]).Error
		if err != nil {
			h.log.Warn("ERROR on updating products: %v", err)
		}
	}

	handleResponse(c, OK, "Successfully upload: "+cast.ToString(count))
}

// Price options
// @Summary Get Price options
// @Description Get Price options
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   product_id 	query string true "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory-detail/price-option [GET]
func (h *InventoryHandler) PriceOption(c *gin.Context) {
	var (
		param domain.InventoryParam
		res   []domain.InventoryPriceOption
		err   error
	)
	// bind query param
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}

	// validate product_id
	if err = uuid.Validate(param.ProductId); err != nil {
		handleResponse(c, BadRequest, "invalid.product_id")
		return
	}

	// get default limit, offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// execute query
	err = h.db.Table("import_details imd").
		Select(`
			imd.product_id,
    		imd.retail_price_vat AS retail_price,
    		imd.expire_date`,
		).
		Joins("JOIN imports im ON imd.import_id = im.id").
		Where("im.entry_type = 2 AND imd.product_id = ? ", param.ProductId).
		Order("imd.created_at DESC").
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		h.log.Warn("ERROR on getting price_options: %v", err)
		handleResponse(c, InternalError, "failed.get.price.options")
		return
	}

	handleResponse(c, OK, res)
}

// godoc DeleteInventory
// @Summary Delete inventory
// @Description Delete inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   inventory_id path string true "Inventory ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory/{inventory_id} [DELETE]
func (h *InventoryHandler) DeleteInventory(c *gin.Context) {
	var id = c.Param("inventory_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()
	err := uuid.Validate(id)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	err = h.service.DeleteInventory(ctx, id)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "DELETED")
}
