package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
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
		sale.POST("/final", h.ProccessingSale)
		sale.GET("/stats", h.SaleStats)
		sale.POST("/epos-result", h.EposResponse)
		sale.GET("/get-list", h.GetSaleList)
		sale.POST("/discount-card", h.AddDiscountCard)
		sale.DELETE("/discount-card", h.RemoveCustomerDiscount)
		sale.GET("/online-count", h.GetOnlineSaleCount)
		sale.POST("/online-accept", h.AcceptOnlineSale)
		sale.POST("online-cancel", h.CancelOnlineSale)
		sale.GET("/online-list", h.OnlineSaleList)
		sale.GET("/pending-list", h.PendingSaleList)
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
	if body.StoreId == "" {
		handleResponse(c, BadRequest, "Store ID is required")
		return
	}
	// get cashbox operation
	err = h.db.First(&cashboxOperation, "id = ?", body.CashBoxOperationId).Error
	if err != nil {
		h.log.Warn("ERROR on getting cashbox_operation: %v", err)
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
		INSERT INTO sales (id, employee_id, cash_box_operation_id, cashbox_id, store_id)
		VALUES (?, ?, ?, ?, ?) RETURNING *`,
			body.ID, body.EmployeeID, body.CashBoxOperationId, body.CashboxId, body.StoreId).
		Scan(&res).Error
	if err != nil {
		h.log.Warn("")
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
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "invalid.sale.id")
		return
	}
	// get sale info
	err := h.db.
		Table("sales").
		Preload("Employee", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "full_name", "first_name", "last_name", "phone") // keep it minimal
		}).
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "full_name", "first_name", "last_name") // keep it minimal
		}).
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name") // or whatever needed
			})
		}).First(&res, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, "Sale info not found")
			return
		}
		h.log.Warn("ERROR on getting sale: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// get products info
	var products []domain.ProductRes
	err = h.db.Raw(`
	SELECT
		p.id, sp.id AS store_product_id, p.name, p.barcode, p.is_marking,
        p.photos, p.mxik AS class_code, p.unit_label AS package_name, 
        ROUND(sp.vat_price * ci.quantity + (sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2) AS vat,
        sp.vat AS vat_percent,
        ci.quantity, ci.unit_price AS pack_price,
        ci.unit_quantity, ci.marking_count, ci.total_price, u.short_name,
        (ci.discount_price*ci.quantity) AS  total_discount,
        ROUND(ci.unit_price / p.unit_per_pack, 2) AS unit_price,
        pb.bonus_amount*ci.quantity+ ROUND((pb.bonus_amount/p.unit_per_pack)*ci.unit_quantity, 2) AS bonus_amount,
        ci.discount_amount
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN product_bonuses pb ON pb.product_id = p.id
	WHERE ci.sale_id = ?`, id).Scan(&products).Error
	if err != nil {
		h.log.Warn("ERROR on getting sold products : %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// get vat sum
	var vatSum float64
	err = h.db.Raw(`
	SELECT
			 COALESCE(SUM(ROUND(sp.vat_price * quantity +(sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2)), 0) AS vat_sum
	FROM cart_items ci
		JOIN store_products sp ON sp.id = ci.store_product_id
		JOIN products p ON sp.product_id = p.id
		WHERE  sale_id = ?;
	`, id).Scan(&vatSum).Error
	if err != nil {
		h.log.Warn("ERROR on getting vat_sum: %v", err)
		handleResponse(c, InternalError, "Can't calculate vat sum")
		return
	}
	if res.ParentId != "" {
		// get epos response
		err = h.db.Raw(`SELECT * FROM epos_responses WHERE sale_id = ?`, res.ParentId).Scan(&res.EposResponse).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				h.log.Error(err)
			}
		}
	}
	// get cart item products list
	res.Product = products
	res.VatSum = vatSum

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
// @Param total_amount_from query int false "Total Amount From"
// @Param total_amount_to query int false "Total Amount To"
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
// @Param total_amount_from query int false "Total Amount From"
// @Param total_amount_to query int false "Total Amount To"
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
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Филиал", "Наличный", "Humo", "Uzcard", "Payme", "Click", "AlifBank", "Обшая сумма", "Дата продажа", "Касса", "Продавец", "Клиент"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, sale := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, helper.SaleTypeToRussian(sale.SaleType, sale.SaleNumber))
		f.SetCellValue(sheetName, "B"+row, sale.StoreName)
		f.SetCellValue(sheetName, "C"+row, sale.Cash)
		f.SetCellValue(sheetName, "D"+row, sale.Humo)
		f.SetCellValue(sheetName, "E"+row, sale.Uzcard)
		f.SetCellValue(sheetName, "F"+row, sale.Payme)
		f.SetCellValue(sheetName, "G"+row, sale.Click)
		f.SetCellValue(sheetName, "H"+row, 0.00)
		if sale.SaleType == "RETURN" {
			f.SetCellValue(sheetName, "I"+row, sale.TotalAmount*(-1))
		} else {
			f.SetCellValue(sheetName, "I"+row, sale.TotalAmount)
		}
		f.SetCellValue(sheetName, "J"+row, sale.CompletedAt.Format(time.DateTime))
		f.SetCellValue(sheetName, "K"+row, sale.CashBoxName)
		f.SetCellValue(sheetName, "L"+row, sale.FullName)
		if sale.CustomerName != nil {
			f.SetCellValue(sheetName, "M"+row, *sale.CustomerName)
		} else {
			f.SetCellValue(sheetName, "M"+row, "N/A")
		}

	}

	saveExcelToUploads(c, f, *h.log, "Barcha_sotuvlar")
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
// @Param total_amount_from query int false "Total Amount From"
// @Param total_amount_to query int false "Total Amount To"
// @Param sale_type query string false "Sale Type (SALE, RETURN)"
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
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
	}
	var (
		args []any
		// query for total transactions sum
		squery = `
		SELECT
			SUM(CASE WHEN s.sale_type = 'SALE' THEN s.total_amount ELSE 0 END) - SUM(CASE WHEN s.sale_type = 'RETURN' THEN s.total_amount ELSE 0 END) AS total_transactions_sum,
        	SUM(CASE WHEN s.sale_type = 'RETURN' THEN s.total_amount ELSE 0 END) AS total_returnals_sum,
			COUNT(*) AS total_count
		FROM sales s
		JOIN stores st ON s.store_id = st.id
		`
		// query for each payment types sum
		pquery = `
		SELECT
			pt.id,
			pt.name,
			pt.type,
			COALESCE(SUM(CASE WHEN s.sale_type = 'SALE' THEN sp.amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN s.sale_type = 'RETURN' THEN sp.amount ELSE 0 END), 0) AS sum
		FROM payment_types pt
		LEFT JOIN sale_payments sp ON sp.payment_type_id = pt.id
		LEFT JOIN sales s ON sp.sale_id = s.id
		`
		filter = ` s.status = 'completed' `
		join   = ""
		group  = ` GROUP BY pt.id, pt.name, pt.type`
	)
	// filter by employee id
	if param.VendorID != "" {
		args = append(args, param.VendorID)
		filter += " AND s.employee_id = ?"
	}
	// filter by payment type
	if param.PaymentTypeID != "" {
		filter += " AND sp.payment_type_id = ? "
		join += " LEFT JOIN sale_payments sp ON sp.sale_id = s.id "
		args = append(args, param.PaymentTypeID)
	}
	// filter by store_id
	if param.StoreID != "" {
		args = append(args, param.StoreID)
		filter += " AND s.store_id = ?"
	}
	// filter by cashbox_id
	if param.CashBoxID != "" {
		args = append(args, param.CashBoxID)
		filter += " AND s.cashbox_id = ?"
	}
	// filter by start_date, end_date
	if param.StartDate != "" && param.EndDate != "" {
		args = append(args, param.StartDate, param.EndDate)
		filter += " AND (s.completed_at + interval '5 hours') BETWEEN ? AND ? "
	}

	// filter by start_date
	if param.StartDate != "" && param.EndDate == "" {
		args = append(args, param.StartDate)
		filter += " AND (s.completed_at + interval '5 hours') >= ?"
	}

	// filter by total amount for less
	if param.TotalAmountFrom > 0 {
		args = append(args, param.TotalAmountFrom)
		filter += " AND s.total_amount >= ? "
	}
	// filter by total amount for greater
	if param.TotalAmountTo > 0 {
		args = append(args, param.TotalAmountTo)
		filter += " AND s.total_amount <= ? "
	}
	// filter by search key
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		filter += fmt.Sprintf(" AND CAST(s.sale_number AS TEXT) LIKE '%s'", param.Search)
	}

	// filter by sale type
	if param.SaleType != "" {
		filter += " AND s.sale_type = ? "
		args = append(args, param.SaleType)
	}
	// collect total transactions query
	squery = squery + join + " WHERE " + filter
	// replace with :param with ?
	err = h.db.Raw(squery, args...).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect payment type sum query
	pquery = pquery + " AND " + filter + group + " ORDER BY sum DESC;"
	// replace with :param with ?
	err = h.db.Raw(pquery, args...).Scan(&res.PaymentTypeStats).Error
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
func (h *SaleHandler) ProccessingSale(c *gin.Context) {
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
	// lock parallel request
	mu := h.getOrderLock(body.SaleID)
	mu.Lock()
	defer mu.Unlock()

	// validate payment types
	if len(body.PaymentTypes) == 0 {
		handleResponse(c, BadRequest, "at least one payment type is required")
		return
	}

	// create transaction
	tx := h.db.Begin()
	defer recoverTransaction(tx, h.log)

	defer RollbackIfError(tx, &err)

	// get sale info
	err = tx.First(&sale, "id = ?", body.SaleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Sale not found")
			return
		}
		h.log.Error("ERROR on getting sale info: ", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// check sale is completed or no
	if sale.Status == config.COMPLETED {
		handleResponse(c, CONFLICT, "Sale is already completed")
		return
	}

	// add marking to cart_items
	err = h.service.AddMarkingCount(body.MarkingData)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// validate amounts
	if !h.service.ValidateSaleAmount(&body) {
		handleResponse(c, BadRequest, "invalid.calculate.amount")
		return
	}

	// delete sale_payments which depends on the sale
	err = tx.Exec(`DELETE FROM sale_payments WHERE sale_id = ?`, body.SaleID).Error
	if err != nil {
		h.log.Error("ERROR on deleting sale_payments: ", err)
		tx.Rollback()
		return
	}

	// process payment types
	for _, item := range body.PaymentTypes {
		err = processPaymentType(c.Request.Context(), tx, h, body, item)

		if err != nil {
			h.log.Warn("ERROR on payment process: %v", err.Error())
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	// complete sale
	err = h.service.CompleteSale(tx, &sale)
	if err != nil {
		h.log.Error("ERROR on completing sale: ", err)
		handleResponse(c, InternalError, "Failed to complete sale")
		return
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, "Can't commit transaction")
		return
	}

	handleResponse(c, OK, "COMPLETED")
}

// Process payment type
func processPaymentType(
	ctx context.Context,
	tx *gorm.DB, h *SaleHandler,
	body domain.FinalSale,
	item domain.FinalPaymentType) error {
	if item.Type == "app" && (item.AppType == config.CLICK || item.AppType == config.PAYME || item.AppType == config.UZUM || item.AppType == config.ALIF) {
		paymentService, err := h.service.GetPaymentServiceByStoreId(body.StoreID, item.AppType)
		if err != nil {
			return errors.New("failed to get payment service")
		}

		paymentHandlers := map[string]func(ctx context.Context, tx *gorm.DB, service *domain.PaymentService, data *domain.FinalPaymentType, cashOpID string, transactionID string, saleID string) (map[string]any, error){
			config.CLICK: h.service.ClickPass,
			config.PAYME: h.service.PaymeGo,
			config.UZUM:  h.service.UzumFastPay,
			config.ALIF:  h.service.AlifPay,
		}
		// get payment handlers for integration app services
		handler, exists := paymentHandlers[item.AppType]
		if !exists {
			return errors.New("invalid payment type")
		}
		// create new sale_payment
		salePayment, err := h.service.CreateSalePayment(tx, body, item, &paymentService.ID)
		if err != nil {
			return err
		}
		// check if sale_payment is created
		resp, err := handler(ctx, tx, paymentService, &item, body.CashBoxOperationId, salePayment.ID, body.SaleID)
		if err != nil || cast.ToString(resp["error_code"]) != "0" {
			return err
		}
		// update sale_payment if payment is success
		return h.service.UpdateSalePaymentStatus(tx, salePayment.ID)
	} else if item.Type == config.CASH || item.Type == config.CARD {
		// Insert sale payments if payment is cash or card
		_, err := h.service.CreateSalePayment(tx, body, item, nil)
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
func (h *SaleHandler) EposResponse(c *gin.Context) {
	var (
		body domain.EposResponseRequest
		sale domain.Sale
	)
	// get user_id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Error("user_id not found in context")
		handleResponse(c, InternalError, "user_id not found in context")
		return
	}

	// Bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
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
	// Transaction rollback if any error occurs
	defer recoverTransaction(tx, h.log)
	defer RollbackIfError(tx, &err)

	// get sale info
	err = h.db.First(&sale, "id = ?", body.SaleId).Error
	if err != nil {
		h.log.Warn("ERROR on getting sale info: %v", err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if body.Error {
		// delete sale_payments which depends on the sale
		err = tx.Exec(`DELETE FROM sale_payments WHERE sale_id = ?`, body.SaleId).Error
		if err != nil {
			h.log.Error("ERROR on deleting sale_payments: ", err)
			handleResponse(c, InternalError, "Failed remove sale_payments")
			return
		}
		// Save to epos_responses table
		err = tx.WithContext(c.Request.Context()).Table("epos_responses").Create(&body).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		// return sale status and quantities
		err = h.service.ReturnSale(tx, &sale)
		if err != nil {
			h.log.Warn("Failed to update sale status: %v", err)
			handleResponse(c, InternalError, "Failed to update sale status")
			return
		}
	} else {
		// Save to epos_responses table
		body.Status = 1
		err = tx.WithContext(c.Request.Context()).Table("epos_responses").Create(&body).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}

		// parse epos success json to structure
		var successResp domain.EposSuccessResponse
		if err = json.Unmarshal([]byte(responseDataStr), &successResp); err != nil {
			h.log.Error("failed to parse success response", err)
			handleResponse(c, BadRequest, "invalid success response format")
			return
		}
		// update sale status to completed
		err = h.service.SetFiscalId(sale.ID, successResp.Message.FiscalSign)
		if err != nil {
			h.log.Warn("Failed to complete sale status: %v", err)
			handleResponse(c, InternalError, "Failed to complete sale status")
			return
		}

		// check payme exists
		salePayment := h.service.GetPaymeSalePayment(sale.ID)
		// set fiscal data if payment completed with payme
		if salePayment.ReceiptId != "" {
			var paymentService domain.PaymentService
			err := h.db.First(&paymentService, "store_id = ?", sale.StoreId).Error
			if err != nil {
				h.log.Warn("ERROR on getting payment service: %v", err)
				handleResponse(c, InternalError, "failed_to_get_payment_service")
				return
			}
			err = h.service.PaymeGoSetFiscalData(c.Request.Context(), &domain.FiscalData{
				StatusCode: 0,
				Message:    "accepted",
				TerminalId: successResp.Message.TerminalId,
				ReceiptId:  cast.ToInt(successResp.Message.ReceiptSeq),
				Date:       successResp.Message.DateTime,
				FiscalSign: successResp.Message.FiscalSign,
				QrCodeUrl:  successResp.Message.QrCodeURL,
			}, salePayment, &paymentService)

			if err != nil {
				h.log.Warn("ERROR on set_fiscal_to_payme: %v", err)
				handleResponse(c, InternalError, "failed_to_set_fiscal_to_payme")
				return
			}
		}

		// create or get sale
		res, err := h.service.CreateOrGetSale(&domain.SaleRequest{
			EmployeeID:         userId.(string),
			StoreId:            sale.StoreId,
			CashBoxOperationId: sale.CashBoxOperationId,
			CashboxId:          sale.CashboxId,
		})
		if err != nil {
			h.log.Warn("ERROR on creating new sale: %v", err)
			handleResponse(c, InternalError, "Can't create new sale")
			return
		}

		// Commit transaction before responding
		err = tx.Commit().Error
		if err != nil {
			h.log.Warn("ERROR on commiting transaction: %v", err)
			handleResponse(c, InternalError, "Transaction not completed")
			return
		}
		handleResponse(c, CREATED, res)
		return
	}

	// Commit transaction before final response
	err = tx.Commit().Error
	if err != nil {
		h.log.Warn("ERROR on commiting transaction: %v", err)
		handleResponse(c, InternalError, "Can't commit transcation")
		return
	}

	handleResponse(c, BadRequest, "Sale not completed")
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

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.AddDiscountCard true "Add discount card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/discount-card [POST]
func (h *SaleHandler) AddDiscountCard(c *gin.Context) {
	var (
		body             domain.AddDiscountCard
		customerDiscount domain.SaleCustomerDiscount
		discountCard     domain.DiscountCard
	)
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// start transcation
	tx := h.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// get discount card info by card number
	err = tx.First(&discountCard, "barcode = ?", body.Barcode).Error
	if err != nil {
		handleResponse(c, NotFound, "discount.card.not.found")
		return
	}

	// delete sale_customer_discount
	err = tx.Exec(`DELETE FROM sale_customer_discounts WHERE sale_id = ?`, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on deleting sale_customer_discount: %v", err)
		handleResponse(c, InternalError, "not.deleted.sale_discount")
		return
	}

	// get discount card info by customer id
	err = tx.First(&customerDiscount, "customer_id = ? AND sale_id = ? AND discount_card_id = ? ", body.CustomerID, body.SaleID, discountCard.Id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		// create new customer_discounts
		err = tx.Raw(`INSERT INTO sale_customer_discounts(customer_id, sale_id, discount_card_id, discount_percent) VALUES(?, ?, ?, ?) RETURNING *`,
			body.CustomerID, body.SaleID, discountCard.Id, discountCard.Percent).Scan(&customerDiscount).Error
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				handleResponse(c, BadRequest, "duplicate.discount_cart.not.accepted")
				return
			}
			h.log.Warn("ERROR on creating sale discount: %v", err)
			handleResponse(c, InternalError, "failed.create.sale.discount")
			return
		}
	} else if err != nil {
		// if error is not record not found
		h.log.Warn("ERROR on getting discount card info: %v", err)
		handleResponse(c, NotFound, "failed.get.discount.card")
		return
	}
	// update cart_items discount amount with total_price
	err = tx.Exec(`
	UPDATE cart_items SET discount_type = ?, discount_value = ? WHERE sale_id = ?;
	`, config.PERCENT, discountCard.Percent, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on updating cart_item discount_value and type : %v", err)
		handleResponse(c, InternalError, "failed.set.discount")
		return
	}
	// set customer_id to sale
	err = tx.Exec(`
	UPDATE
		sales
	SET
		customer_id = ?
	WHERE id = ?`, body.CustomerID, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on updating sale: %v", err)
		handleResponse(c, InternalError, "failed.update.sale.customer_id")
		return
	}
	// commit transcation
	err = tx.Commit().Error
	if err != nil {
		h.log.Warn("ERROR on commiting transcation: %v", err)
		handleResponse(c, InternalError, "not.completed.transcation")
		return
	}

	handleResponse(c, OK, customerDiscount)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	body body domain.AddDiscountCard true "Add discount card"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/discount-card [DELETE]
func (h *SaleHandler) RemoveCustomerDiscount(c *gin.Context) {
	var (
		body domain.AddDiscountCard
	)
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer recoverTransaction(tx, h.log)
	defer RollbackIfError(tx, &err)

	// delete customer discount by customer id
	err = tx.Exec(`DELETE FROM sale_customer_discounts WHERE customer_id = ? AND sale_id = ?`,
		body.CustomerID, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on deleting customer discount: %v", err)
		handleResponse(c, InternalError, "Can't delete customer discount")
		return
	}
	// update sale customer_id to null
	err = tx.Exec(`UPDATE sales SET customer_id = NULL WHERE id = ?`, body.SaleID).Error
	if err != nil {
		h.log.Warn("ERROR on updating sale customer_id: %v", err)
		handleResponse(c, InternalError, "failed.update.sale.customer_id")
		return
	}
	// update discount_type and value to 0
	err = tx.Exec(`UPDATE cart_items SET discount_value = ?, discount_type = ? WHERE sale_id = ?`, 0, "percent", body.SaleID).Error
	if err != nil {
		handleResponse(c, InternalError, "failed.update.cart_items")
		return
	}

	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, "transcation.not.commited")
		return
	}

	handleResponse(c, OK, "DELETED")

}

// Get online pending sale count
// @Summary Get online pending sale count
// @Description Get online pending sale count
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param	store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-count [GET]
func (h *SaleHandler) GetOnlineSaleCount(c *gin.Context) {
	// get user_id from the set header
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, BadRequest, "user.not.found.header")
		return
	}
	// get employee info by set user_id
	var employee domain.Employee
	err := h.db.First(&employee, "id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "user.not.found")
			return
		}
		h.log.Warn("ERROR on getting employee info: %v", err)
		handleResponse(c, InternalError, "not.get.user")
		return
	}
	// get online order count
	var count int64
	err = h.db.Debug().Raw(`
	SELECT
		COUNT(*) AS count
	FROM sales
	WHERE store_id = ? AND
		(online_status = 1 OR online_status = 2);
	`, employee.StoreId).Scan(&count).Error

	if err != nil {
		h.log.Warn("ERROR on getting online sale count: %v", err)
		handleResponse(c, InternalError, "internal.server.error")
		return
	}

	handleResponse(c, OK, gin.H{"count": count})
}

// Confirm online sale
// @Summary Confirm online sale
// @Description Confirm online sale
// @Tags 	sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   body body domain.ConfirmOnlineSaleRequest true "confirm online sale"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router 	/sale/online-accept [POST]
func (h *SaleHandler) AcceptOnlineSale(c *gin.Context) {
	var body domain.ConfirmOnlineSaleRequest
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// get user_id from the set header
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, BadRequest, "user.not.found.header")
		return
	}
	body.EmployeeID = userID.(string)

	err = h.service.AcceptOnlineSale(&body)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "ACCEPTED")
}

// godoc cancel online sale
// @Summary Cancel online sale
// @Description Cancel online sale
// @Tags 	sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   body body domain.ConfirmOnlineSaleRequest true "cancel online sale"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-cancel [POST]
func (h *SaleHandler) CancelOnlineSale(c *gin.Context) {
	var body domain.ConfirmOnlineSaleRequest
	// bind request body
	err := c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}
	// get user_id from the set header
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, BadRequest, "user.not.found.header")
		return
	}
	body.EmployeeID = userID.(string)

	err = h.service.CancelOnlineSale(&body)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "CANCELED")
}

// Get online pending sale list
// @Summary Get online pending sale list
// @Description Get online pending sale list
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   limit query int false "Limit"
// @Param	offset query int false "Offset"
// @Param   store_id query string true "Store ID"
// @Param   search query string false "Search"
// @Param	start_date query string false "StartDate"
// @Param	end_date query string false "EndDate"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/online-list [GET]
func (h *SaleHandler) OnlineSaleList(c *gin.Context) {
	var param domain.QueryParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.query.param")
		return
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.OnlinePendingSaleList(&param)
	if err != nil {
		h.log.Warn("ERROR on getting online pending sale list: %v", err)
		handleResponse(c, InternalError, "failed.get.online.sale")
		return
	}
	// get response data with pagination _meta data
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// PendingSaleList godoc
// @Summary Get pending sales
// @Description Get all sales with status 'pending' filtered by store, date, search
// @Tags sales
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search (sale number or store name)"
// @Param start_date query string false "Created At Start Date (RFC3339)"
// @Param end_date query string false "Created At End Date (RFC3339)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/pending-list [get]
func (h *SaleHandler) PendingSaleList(c *gin.Context) {
	var param domain.QueryParam

	// get user_id from context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	if err := c.ShouldBindQuery(&param); err != nil {
		h.log.Error("bind query error: ", err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get pending sales
	res, totalCount, err := h.service.ListPendingSales(&param, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	result := utils.ListResponse(res, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, result)
}

// lock order for parallel request
func (h *SaleHandler) getOrderLock(orderId string) *sync.Mutex {
	lock, _ := h.ordersToMutexes.LoadOrStore(orderId, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
