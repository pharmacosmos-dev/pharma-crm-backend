package v1

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type SaleHandler struct {
	*Handler
}

func (h *Handler) NewSaleHandler(r *gin.RouterGroup) {
	sale := &SaleHandler{h}
	sale.SaleRoutes(r)
}

func (h *SaleHandler) SaleRoutes(r *gin.RouterGroup) {
	sale := r.Group("/sale")
	{
		sale.POST("", h.Create)
		sale.POST("/return", h.CreateReturn)
		sale.GET("/:id", h.Get)
		sale.GET("/list", h.List)
		sale.GET("/export-excel", h.ExportSaleExcel)
		sale.PUT("/:id", h.Update)
		sale.POST("/final", h.FinalSale)
		sale.GET("/stats", h.SaleStats)
		sale.POST("/epos-result", h.EposRequest)
		sale.GET("/get-list", h.GetSaleList)
	}
}

// Create godoc
// @Summary Create a sale
// @Description Create a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.SaleRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale [post]
func (h *SaleHandler) Create(c *gin.Context) {
	var (
		body             domain.SaleRequest
		res              domain.Sale
		cashboxOperation domain.CashboxOperation
		err              error
	)
	// get user id from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// check store id
	if body.StoreId == nil {
		handleResponse(c, BadRequest, "Store ID is required")
		return
	}
	// get cashbox operation
	err = h.db.First(&cashboxOperation, "id = ?", body.CashBoxOperationId).Error
	if err != nil {
		h.log.Error("ERROR on getting cashbox_operation: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	body.ID = uuid.New().String()
	body.EmployeeID = userId.(string)
	body.CashboxId = cashboxOperation.CashBoxID
	// create sale
	err = h.db.
		WithContext(c.Request.Context()).
		Raw(`
		INSERT INTO sales (id, employee_id, cash_box_operation_id, cashbox_id)
		VALUES (?, ?, ?, ?) RETURNING *`,
			body.ID, body.EmployeeID, body.CashBoxOperationId, body.CashboxId).
		Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, res)
}

// Create return sale godoc
// @Summary Create a return sale
// @Description Create a return sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.SaleReturnRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/return [post]
func (h *SaleHandler) CreateReturn(c *gin.Context) {
	var (
		body domain.SaleReturnRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get user id in context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	body.EmployeeID = userId.(string)
	body.SaleType = config.SALE_TYPE_RETURN
	// create sale return
	sale, err := h.service.CreateReturnSale(&body)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, sale)
}

// Get godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [get]
func (h *SaleHandler) Get(c *gin.Context) {
	var (
		res domain.SaleResponse
		id  = c.Param("id")
	)
	// get sale info
	err := h.db.
		Table("sales").
		Preload("Employee").
		Preload("Customer").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).First(&res, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, "Sale info not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get products info
	var products []domain.ProductRes
	err = h.db.Raw(`
	SELECT 
		p.id, sp.id AS store_product_id, p.name, p.barcode, sp.retail_price, sp.bonus_percent, 
		((sp.bonus_percent*sp.retail_price)/100)  as  bonus_amount,
		p.photos, ci.quantity,
		ci.unit_quantity, ci.total_price, u.short_name, 
		(ci.discount_price*ci.quantity) AS  total_discount
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	WHERE ci.sale_id = ?`, id).Scan(&products).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if res.ParentId != "" {
		// get epos response
		err = h.db.Raw(`SELECT * FROM epos_responses WHERE sale_id = ?`, res.ParentId).Scan(&res.EposResponse).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) { // Faqat mavjud bo'lmagan yozuv emas, boshqa xatoliklarni logga yozish
				h.log.Error(err)
			}
		}
	}

	res.Product = products
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param vendor_id query string false "Vendor ID"
// @Param store_id query string false "Store ID"
// @Param cashbox_id query string false "Cash Box ID"
// @Param payment_type_id query string false "Payment Type ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list [get]
func (h *SaleHandler) List(c *gin.Context) {
	var param domain.QueryParam

	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	// bind query params
	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error("bind query params error: ", err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get limit offset with checking default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get sale list data
	res, totalCount, err := h.service.ListSale(&param, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	// added _meta section to response
	result := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, result)
}

// List godoc
// @Summary Get a sale list excel
// @Description Get a sale list excel
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param vendor_id query string false "Vendor ID"
// @Param store_id query string false "Store ID"
// @Param cashbox_id query string false "Cash Box ID"
// @Param payment_type_id query string false "Payment Type ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/export-excel [get]
func (h *SaleHandler) ExportSaleExcel(c *gin.Context) {
	var param domain.QueryParam
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// bind query params
	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get limit offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get sale list data
	res, _, err := h.service.ListSale(&param, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "Sales"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Касса", "Обшая сумма", "Тип продажа", "Доставлено", "Наличный", "Humo", "Uzcard", "Visa", "Payme", "Click", "UzumBank", "Баланс", "Дата продажа", "Филиал", "Продавец", "Клиент"}

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
	for i, sale := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, sale.SaleNumber)
		f.SetCellValue(sheetName, "B"+row, sale.CashBoxName)
		f.SetCellValue(sheetName, "C"+row, sale.TotalAmount)
		f.SetCellValue(sheetName, "D"+row, sale.Type)
		f.SetCellValue(sheetName, "E"+row, sale.IsDelivered)
		f.SetCellValue(sheetName, "F"+row, helper.SalePaymentAmount(sale.SalePayments, "cash"))
		f.SetCellValue(sheetName, "G"+row, helper.SalePaymentAmount(sale.SalePayments, "card"))
		f.SetCellValue(sheetName, "H"+row, helper.SalePaymentAmount(sale.SalePayments, "card"))
		f.SetCellValue(sheetName, "I"+row, helper.SalePaymentAmount(sale.SalePayments, "card"))
		f.SetCellValue(sheetName, "J"+row, helper.SalePaymentAmount(sale.SalePayments, "app"))
		f.SetCellValue(sheetName, "K"+row, helper.SalePaymentAmount(sale.SalePayments, "app"))
		f.SetCellValue(sheetName, "L"+row, helper.SalePaymentAmount(sale.SalePayments, "app"))
		f.SetCellValue(sheetName, "M"+row, helper.SalePaymentAmount(sale.SalePayments, "balance"))
		f.SetCellValue(sheetName, "N"+row, sale.CompletedAt.Format(time.DateTime))
		f.SetCellValue(sheetName, "O"+row, sale.StoreName)
		f.SetCellValue(sheetName, "P"+row, sale.FullName)
		if sale.CustomerName != nil {
			f.SetCellValue(sheetName, "Q"+row, *sale.CustomerName)
		} else {
			f.SetCellValue(sheetName, "Q"+row, "N/A")
		}

	}

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=sales.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}

}

// List godoc
// @Summary Get a sale stats
// @Description Get a sale stats from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param vendor_id query string false "Vendor ID"
// @Param store_id query string false "Store ID"
// @Param cashbox_id query string false "Cash Box ID"
// @Param payment_type_id query string false "Payment Type ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/stats [get]
func (h *SaleHandler) SaleStats(c *gin.Context) {
	var (
		res   domain.SaleStats
		param domain.QueryParam
		err   error
	)
	// bind query param
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get userid from header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, BadRequest, "User not found")
		return
	}
	var employee domain.Employee
	// get employee info
	err = h.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "User not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check user role
	if !helper.IsAdmin(employee, h.cfg) {
		param.StoreID = employee.StoreId
	}
	var (
		args []any
		// query for total transactions sum
		squery = `
		SELECT
			COALESCE(SUM(s.total_amount), 0) AS total_transactions_sum
		FROM sales s
		LEFT JOIN stores st ON s.store_id = st.id 
		LEFT JOIN sale_payments sp ON s.id = sp.sale_id
		`
		// query for each payment types sum
		pquery = `
		SELECT
			pt.id,
			pt.name,
			pt.type,
			SUM(sp.amount) AS sum
		FROM sale_payments sp
		JOIN payment_types pt ON sp.payment_type_id = pt.id
		JOIN sales s ON sp.sale_id = s.id
		`
		filter = `WHERE 1=1 `
		group  = ` GROUP BY pt.id, pt.name`
	)

	if param.PaymentTypeID != "" {
		args = append(args, param.PaymentTypeID)
		filter += " AND sp.payment_type_id = ?"
	}

	if param.VendorID != "" {
		args = append(args, param.VendorID)
		filter += " AND s.employee_id = ?"
	}
	if param.StoreID != "" {
		args = append(args, param.StoreID)
		filter += "AND s.store_id = ?"
	}
	if param.CashBoxID != "" {
		args = append(args, param.CashBoxID)
		filter += "AND s.cashbox_id = ?"
	}

	if param.StartDate != "" && param.EndDate != "" {
		args = append(args, param.StartDate, param.EndDate)
		filter += " AND s.completed_at::date >= ? AND s.completed_at::date <= ?"
	}

	if param.StartDate != "" && param.EndDate == "" {
		args = append(args, param.StartDate)
		filter += " AND s.completed_at::date >= ?"
	}

	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		filter += fmt.Sprintf(" AND CAST(s.sale_number AS TEXT) LIKE '%s'", param.Search)
	}
	// collect total transactions query
	var q = squery + filter
	// replace with :param with ?
	err = h.db.Debug().Raw(q, args...).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect payment type sum query
	var pq = pquery + filter + group
	// replace with :param with ?
	err = h.db.Debug().Raw(pq, args...).Scan(&res.PaymentTypeStats).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if res.PaymentTypeStats == nil {
		res.PaymentTypeStats = []domain.PaymentTypeStats{}
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a sale
// @Description Update a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Param input body domain.SaleUpdateRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [put]
func (h *SaleHandler) Update(c *gin.Context) {
	var (
		body domain.SaleUpdateRequest
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("sales").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, body)
}

// EposRequest godoc
// @Summary Epos Request
// @Description Epos Request from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.EposResponseRequest true "Epos Response info as json {}"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/epos-result [post]
func (h *SaleHandler) EposRequest(c *gin.Context) {
	var (
		body domain.EposResponseRequest
		sale domain.Sale
		err  error
	)

	// Bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Ensure response_data is a string
	responseDataStr, ok := body.ResponseData.(string)
	if !ok {
		h.log.Error("response_data is not a valid string")
		handleResponse(c, BadRequest, "response_data must be a string")
		return
	}

	// Convert string to []byte and store in Response field
	body.Response = []byte(responseDataStr)

	// Start transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.log.Error(tx.Error)
		handleResponse(c, InternalError, "failed to start transaction")
		return
	}

	// Transaction rollback if any error occurs
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save to epos_responses table
	err = tx.WithContext(c.Request.Context()).Table("epos_responses").Create(&body).Error
	if err != nil {
		h.log.Error(err)
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get sale info
	err = h.db.First(&sale, "id = ?", body.SaleId).Error
	if err != nil {
		h.log.Error(err)
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	//
	if body.Error {
		// Update sales status
		err = tx.Exec(`UPDATE sales SET status = ? WHERE id = ?`, config.PENDING, body.SaleId).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}

		// Update cart_items status
		err = tx.Exec(`UPDATE cart_items SET status = ? WHERE sale_id = ?`, config.PENDING, body.SaleId).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}

		// return products to store_product
		err = h.service.UpdateReturnSaleCartItems(tx, body.SaleId)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}
	}

	if !body.Error {
		// get cashbox operation
		var cashboxOperation domain.CashboxOperation
		// get cashbox operation
		err = h.db.First(&cashboxOperation, "id = ?", sale.CashBoxOperationId).Error
		if err != nil {
			h.log.Error("ERROR on getting cashbox_operation: ", err)
			handleResponse(c, InternalError, err.Error())
			return
		}

		// Create new sale
		newSale := domain.SaleRequest{
			ID:                 uuid.New().String(),
			EmployeeID:         sale.EmployeeID,
			StoreId:            &sale.StoreId,
			CashBoxOperationId: sale.CashBoxOperationId,
			CashboxId:          cashboxOperation.CashBoxID,
		}
		// Insert new sale
		_, err = h.service.CreateSale(tx, &newSale)
		if err != nil {
			h.log.Error("ERROR on creating new sale: %w", err)
			handleResponse(c, InternalError, "ERROR on creating new sale: "+err.Error())
			tx.Rollback()
			return
		}

		// Commit transaction before responding
		if err = tx.Commit().Error; err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}

		handleResponse(c, CREATED, newSale)
		return
	}

	// Commit transaction before final response
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, BadRequest, "Sale not completed")
}

// FinalSale
// @Summary Final Sale
// @Description Final Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	var (
		body domain.FinalSale
		sale domain.Sale
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get user id from context
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	// validate payment types
	if len(body.PaymentTypes) == 0 {
		handleResponse(c, BadRequest, "at least one payment type is required")
		return
	}

	// create transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// get sale info
	err = h.db.First(&sale, "id = ?", body.SaleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Sale not found")
			tx.Rollback()
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}

	// check sale is completed or no
	if sale.Status == config.COMPLETED {
		handleResponse(c, CONFLICT, "Sale is already completed")
		tx.Rollback()
		return
	}
	// var sum float64
	// for _, item := range body.PaymentTypes {
	// 	sum += item.Amount
	// }
	// // get total amount from cart items
	// totalAmount, err := h.service.GetCartItemsTotalAmount(body.SaleID)
	// if err != nil {
	// 	h.log.Error("ERROR on getting total amount from cart items: ", err.Error())
	// 	handleResponse(c, InternalError, err.Error())
	// 	tx.Rollback()
	// 	return
	// }
	// // validate total amount
	// if sum < totalAmount {
	// 	h.log.Info("Invalid payment amount")
	// 	handleResponse(c, BadRequest, "Invalid payment amount")
	// 	tx.Rollback()
	// 	return
	// }

	// process payment types
	for _, item := range body.PaymentTypes {
		if item.Type == "cash" && item.Amount > body.TotalAmount {
			body.ReturnedAmount = item.Amount - body.TotalAmount
			item.Amount = body.TotalAmount
		}
		if err = processPaymentType(tx, h, body, item); err != nil {
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}
	}

	// complete sale, cart_items, employee_bonus
	if sale.SaleType == config.SALE_TYPE_RETURN {
		if err = h.returnSaleTransaction(tx, &sale, &body); err != nil {
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}
	} else if sale.SaleType == config.SALE_TYPE_SALE {
		if err = h.completeSaleTransaction(tx, body, userID.(string)); err != nil {
			handleResponse(c, InternalError, err.Error())
			tx.Rollback()
			return
		}
	} else {
		handleResponse(c, BadRequest, "Invalid sale type")
		tx.Rollback()
		return
	}
	// collect new sale info
	newSale := domain.SaleRequest{
		ID:                 uuid.New().String(),
		StoreId:            &body.StoreID,
		EmployeeID:         sale.EmployeeID,
		CashBoxOperationId: sale.CashBoxOperationId,
		CashboxId:          sale.CashboxId,
	}

	// create new sale
	err = tx.Table("sales").Create(&newSale).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		tx.Rollback()
		return
	}
	handleResponse(c, OK, newSale)
}

// Process payment type
func processPaymentType(tx *gorm.DB, h *SaleHandler, body domain.FinalSale, item domain.FinalPaymentType) error {
	if item.Type == "app" && (item.AppType == config.CLICK || item.AppType == config.PAYME || item.AppType == config.UZUM) {
		paymentService, err := h.service.GetPaymentServiceByStoreId(body.StoreID, item.AppType)
		if err != nil {
			return errors.New("failed to get payment service")
		}
		paymentHandlers := map[string]func(ctx context.Context, service *domain.PaymentService, data *domain.FinalPaymentType, cashOpID string, transactionID string, saleID string) (map[string]interface{}, error){
			config.CLICK: h.service.ClickPass,
			config.PAYME: h.service.PaymeGo,
			config.UZUM:  h.service.UzumFastPay,
		}
		// get payment handlers for integration app services
		handler, exists := paymentHandlers[item.AppType]
		if !exists {
			return errors.New("invalid payment type")
		}
		// create new sale_payment
		salePayment, err := h.service.CreateSalePayment(tx, body, item, &paymentService.ID, "pending")
		if err != nil {
			return err
		}

		resp, err := handler(context.Background(), paymentService, &item, body.CashBoxOperationId, salePayment.ID, body.SaleID)
		if err != nil || cast.ToString(resp["error_code"]) != "0" {
			return errors.New("failed payment with " + item.AppType)
		}
		// update sale_payment if payment is success
		return h.service.UpdateSalePaymentStatus(tx, salePayment.ID)
	} else if item.Type == config.CASH || item.Type == config.CARD {
		// Insert sale payments if payment is cash or card
		_, err := h.service.CreateSalePayment(tx, body, item, nil, "paid")
		if err != nil {
			return err
		}
		// insert or update sale payment summary
		err = h.service.CreateOrUpdateSalePaymentSummary(tx, body.CashBoxOperationId, item.PaymentTypeID, item.Amount)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid payment type")
	}

	return nil
}

// Completed sale transaction
func (h *SaleHandler) completeSaleTransaction(tx *gorm.DB, body domain.FinalSale, userID string) error {
	// update sale status and total amount, returned_amount
	err := h.service.UpdateSaleStatus(tx, body.SaleID, body.TotalAmount, body.CustomerID)
	if err != nil {
		return err
	}
	// update cart items and store_products
	if err = h.service.UpdateCartItemStatus(tx, body.SaleID, userID, body.CashBoxOperationId); err != nil {
		return err
	}
	return nil
}

// Return sale transaction
func (h *SaleHandler) returnSaleTransaction(tx *gorm.DB, req *domain.Sale, body *domain.FinalSale) error {
	if err := h.service.UpdateSaleStatus(tx, req.ID, body.TotalAmount, body.CustomerID); err != nil {
		return err
	}
	if err := h.service.UpdateReturnSaleCartItems(tx, req.ID); err != nil {
		return err
	}
	return nil
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param vendor_id query string false "Vendor ID"
// @Param store_id query string true "Store ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/get-list [get]
func (h *SaleHandler) GetSaleList(c *gin.Context) {
	var (
		param domain.QueryParam
	)
	// bind query params
	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error("ERROR on binding query params: ", err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get sale list data
	res, totalCount, err := h.service.GetSaleList(&param)
	if err != nil {
		h.log.Error("ERROR on getting sale list: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// added _meta section to response
	result := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, result)
}
