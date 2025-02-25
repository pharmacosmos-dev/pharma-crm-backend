package v1

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
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
		sale.GET("/:id", h.Get)
		sale.GET("/list", h.List)
		sale.PUT("/:id", h.Update)
		sale.POST("/final", h.FinalSale)
		sale.GET("/stats", h.SaleStats)
		sale.POST("/epos-result", h.EposRequest)
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
		body domain.SaleRequest
		res  domain.Sale
		err  error
	)
	user, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.EmployeeID = cast.ToString(user)
	err = h.db.
		WithContext(c.Request.Context()).
		Raw(`
		INSERT INTO sales (id, employee_id, cash_box_operation_id)
		VALUES (?, ?, ?) RETURNING *`,
			body.ID, body.EmployeeID, body.CashBoxOperationId).
		Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, res)
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
	var products []domain.ProductRes
	err = h.db.Raw(`
	SELECT 
		p.id, p.name, p.barcode, sp.retail_price, sp.bonus_percent, 
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
	var (
		totalCount    int64
		startDate     = c.Query("start_date")
		endDate       = c.Query("end_date")
		employeeID    = c.Query("vendor_id")
		cashBoxId     = c.Query("cashbox_id")
		paymentTypeId = c.Query("payment_type_id")
		storeID       = c.Query("store_id")
		search        = c.Query("search")
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	var res = []domain.SaleResponse{}
	query := h.db.Model(&domain.Sale{}).Table("sales s").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).
		Select(`
		s.*, em.full_name, em.phone,
		st.name AS store_name, customers.full_name as customer_name, customers.phone AS customer_phone,
		cash_boxes.name AS cash_box_name`).
		// Change INNER JOIN to LEFT JOIN to include sales without store_id
		Joins("LEFT JOIN stores st ON st.id = s.store_id").
		// Change INNER JOIN to LEFT JOIN to include sales without employee_id
		Joins("LEFT JOIN employees em ON em.id = s.employee_id").
		// Change INNER JOIN to LEFT JOIN to include sales without cashbox_operation_id
		Joins("LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id").
		// Ensure cash_boxes can be null
		Joins("LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id").
		Joins("LEFT JOIN customers ON s.customer_id = customers.id")

	// filter by payment type
	if paymentTypeId != "" {
		query = query.Joins("JOIN sale_payments sp ON s.id = sp.sale_id").
			Where("sp.payment_type_id = ?", paymentTypeId).
			Group("s.id, st.name, cash_boxes.name, em.full_name, em.phone, customers.full_name, customers.phone")
	}
	// filter by employee
	if employeeID != "" {
		query = query.Where("s.employee_id = ?", employeeID)
	} else {
		query = query.Where("s.employee_id IS NOT NULL OR s.employee_id IS NULL") // Include online sales
	}
	// filter by store id
	if storeID != "" {
		query = query.Where("s.store_id = ?", storeID)
	} else {
		query = query.Where("s.store_id IS NOT NULL OR s.store_id IS NULL") // Include online sales
	}
	// filter by cashbox id
	if cashBoxId != "" {
		query = query.Where("co.cash_box_id = ?", cashBoxId)
	} else {
		query = query.Where("s.cash_box_operation_id IS NULL OR co.cash_box_id IS NOT NULL") // Include online sales
	}
	// filter by start date and end date
	if startDate != "" && endDate != "" {
		query = query.Where("s.completed_at::date >= ? AND s.completed_at::date <= ?  ", startDate, endDate)
	}
	// filter by start date
	if startDate != "" && endDate == "" {
		query = query.Where("s.completed_at::date = ?", startDate)
	}
	// search condition
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("st.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?", search, search)
	}
	// complete query
	err = query.Where("s.status = 'completed'").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("s.completed_at DESC").
		Debug().
		Find(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	result := utils.ListResponse(res, totalCount, limit, offset)

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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list-excel [get]
func (h *SaleHandler) ListExcel(c *gin.Context) {
	var (
		totalCount    int64
		startDate     = c.Query("start_date")
		endDate       = c.Query("end_date")
		employeeID    = c.Query("vendor_id")
		cashBoxId     = c.Query("cashbox_id")
		paymentTypeId = c.Query("payment_type_id")
		storeID       = c.Query("store_id")
		search        = c.Query("search")
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	var res = []domain.SaleResponse{}
	query := h.db.Model(&domain.Sale{}).Table("sales s").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).
		Select(`
		s.*, em.full_name, em.phone,
		st.name AS store_name, customers.full_name as customer_name, customers.phone AS customer_phone,
		cash_boxes.name AS cash_box_name`).
		// Change INNER JOIN to LEFT JOIN to include sales without store_id
		Joins("LEFT JOIN stores st ON st.id = s.store_id").
		// Change INNER JOIN to LEFT JOIN to include sales without employee_id
		Joins("LEFT JOIN employees em ON em.id = s.employee_id").
		// Change INNER JOIN to LEFT JOIN to include sales without cashbox_operation_id
		Joins("LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id").
		// Ensure cash_boxes can be null
		Joins("LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id").
		Joins("LEFT JOIN customers ON s.customer_id = customers.id")

	// filter by payment type
	if paymentTypeId != "" {
		query = query.Joins("JOIN sale_payments sp ON s.id = sp.sale_id").
			Where("sp.payment_type_id = ?", paymentTypeId).
			Group("s.id, st.name, cash_boxes.name, em.full_name, em.phone, customers.full_name, customers.phone")
	}
	// filter by employee
	if employeeID != "" {
		query = query.Where("s.employee_id = ?", employeeID)
	} else {
		query = query.Where("s.employee_id IS NOT NULL OR s.employee_id IS NULL") // Include online sales
	}
	// filter by store id
	if storeID != "" {
		query = query.Where("s.store_id = ?", storeID)
	} else {
		query = query.Where("s.store_id IS NOT NULL OR s.store_id IS NULL") // Include online sales
	}
	// filter by cashbox id
	if cashBoxId != "" {
		query = query.Where("co.cash_box_id = ?", cashBoxId)
	} else {
		query = query.Where("s.cash_box_operation_id IS NULL OR co.cash_box_id IS NOT NULL") // Include online sales
	}
	// filter by start date and end date
	if startDate != "" && endDate != "" {
		query = query.Where("s.completed_at::date >= ? AND s.completed_at::date <= ?  ", startDate, endDate)
	}
	// filter by start date
	if startDate != "" && endDate == "" {
		query = query.Where("s.completed_at::date = ?", startDate)
	}
	// search condition
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("st.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?", search, search)
	}
	// complete query
	err = query.Where("s.status = 'completed'").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("s.completed_at DESC").
		Debug().
		Find(&res).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
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
		res domain.SaleStats
	)
	// Get query params
	var (
		startDate     = c.Query("start_date")
		endDate       = c.Query("end_date")
		vendorID      = c.Query("vendor_id")
		cashBoxId     = c.Query("cashbox_id")
		paymentTypeId = c.Query("payment_type_id")
		storeID       = c.Query("store_id")
		search        = c.Query("search")
	)
	var (
		args []interface{}
		// query for total transactions sum
		squery = `
		SELECT
			COALESCE(SUM(s.total_amount), 0) AS total_transactions_sum
		FROM sales s
		JOIN employees ON s.employee_id = employees.id
		JOIN stores st ON employees.store_id = st.id
		JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
		JOIN cash_boxes ON co.cash_box_id = cash_boxes.id
		LEFT JOIN sale_payments sp ON s.id = sp.sale_id 
		`
		// query for each payment types sum
		pquery = `
		SELECT
			pt.id,
			pt.name,
			pt.type,
			COALESCE(SUM(sp.amount), 0) AS sum
		FROM payment_types pt
		LEFT JOIN sale_payments sp ON pt.id = sp.payment_type_id
		LEFT JOIN sales s ON s.id = sp.sale_id
		LEFT JOIN employees e ON s.employee_id = e.id
		LEFT JOIN stores st ON e.store_id = st.id
		LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
		LEFT JOIN cash_boxes cb ON co.cash_box_id = cb.id
		`
		filter = `WHERE 1=1 `
		group  = ` GROUP BY pt.id, pt.name, pt.type `
	)

	if paymentTypeId != "" {
		args = append(args, paymentTypeId)
		filter += " AND sp.payment_type_id = ?"
	}

	if vendorID != "" {
		args = append(args, vendorID)
		filter += " AND s.employee_id = ?"
	}
	if storeID != "" {
		args = append(args, storeID)
		filter += "AND st.id = ?"
	}
	if cashBoxId != "" {
		args = append(args, cashBoxId)
		filter += "AND co.cash_box_id = ?"
	}

	if startDate != "" && endDate != "" {
		args = append(args, startDate, endDate)
		filter += " AND s.completed_at::date >= ? AND s.completed_at::date <= ?"
	}

	if startDate != "" && endDate == "" {
		args = append(args, startDate)
		filter += " AND s.completed_at::date >= ?"
	}

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		filter += fmt.Sprintf(" AND stores.name ILIKE %s OR CAST(s.sale_number AS TEXT) LIKE %s", search, search)
	}
	// collect total transactions query
	var q = squery + filter
	// replace with :param with ?
	err := h.db.Debug().Raw(q, args...).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect payment type sum query
	var pq = pquery + filter + group
	// replace with :param with ?
	err = h.db.Raw(pq, args...).Scan(&res.PaymentTypeStats).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
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
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Response, err = json.Marshal(&body.ResponseData)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// create epos response body
	err = h.db.WithContext(c.Request.Context()).Table("epos_responses").Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// FinalSale
// @Summary Final Sale
// @Description Final Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	var (
		body domain.FinalSale
		err  error
	)

	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	// validate payment types
	if err := validateFinalSaleRequest(body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// create transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// check sale is completed or no
	if isSaleCompleted(tx, body.SaleID) {
		handleResponse(c, CONFLICT, "Sale is already completed")
		return
	}

	// process payment types
	for _, item := range body.PaymentTypes {
		if err = processPaymentType(tx, h, body, item); err != nil {
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	// complete sale, cart_items, employee_bonus
	if err = h.completeSaleTransaction(tx, body, userID.(string)); err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// collect new sale data
	newSale := domain.SaleRequest{
		ID:                 uuid.New().String(),
		EmployeeID:         cast.ToString(userID),
		CashBoxOperationId: body.CashBoxOperationId,
	}
	// create new sale
	err = tx.Table("sales").Create(&newSale).Error
	if err != nil {
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
	handleResponse(c, OK, newSale.ID)
}

// Validate payment Type
func validateFinalSaleRequest(body domain.FinalSale) error {
	if len(body.PaymentTypes) == 0 {
		return errors.New("at least one payment type is required")
	}
	return nil
}

// Check sale is completed
func isSaleCompleted(tx *gorm.DB, saleID string) bool {
	var count int64
	err := tx.Model(&domain.Sale{}).Where("id = ? AND status = 'completed'", saleID).Count(&count).Error
	return err == nil && count > 0
}

// Process payment type
func processPaymentType(tx *gorm.DB, h *SaleHandler, body domain.FinalSale, item domain.FinalPaymentType) error {
	if item.Type == "app" && (item.AppType == config.CLICK || item.AppType == config.PAYME || item.AppType == config.UZUM) {
		paymentService, err := h.service.GetPaymentServiceByStoreId(body.StoreID, item.AppType)
		if err != nil {
			return errors.New("failed to get payment service")
		}
		paymentHandlers := map[string]func(ctx context.Context, service *domain.PaymentService, data *domain.FinalPaymentType, cashOpID string, transactionID string, saleID string) (map[string]interface{}, error){
			config.CLICK: h.ClickPass,
			config.PAYME: h.PaymeGo,
			config.UZUM:  h.UzumFastPay,
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
		if err != nil || resp["error_code"].(float64) != 0 {
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
	if err := h.service.UpdateSaleStatus(tx, body.SaleID, body.TotalAmount, body.CustomerID); err != nil {
		return err
	}
	if err := h.service.UpdateCartItemStatus(tx, body.SaleID); err != nil {
		return err
	}
	if err := h.service.CreateEmployeeBonus(tx, userID, body.SaleID, body.CashBoxOperationId); err != nil {
		return errors.New("error on adding bonus: " + err.Error())
	}
	return nil
}

// ClickPass implements PaymentService
func (h *SaleHandler) ClickPass(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error) {
	var cashBoxId string
	err := h.db.Raw(`SELECT cash_box_id FROM cashbox_operations WHERE id = ?`, CashOperationID).Scan(&cashBoxId).Error
	if err != nil {
		return nil, err
	}
	// Click Pass request body
	clickData := domain.ClickPassRequest{
		ServiceID:     click.ServiceID,
		OtpData:       data.OtpData,
		CashboxCode:   cashBoxId,
		Amount:        data.Amount,
		TransactionID: transactionID,
	}
	// Marshal click pass request
	t, _ := json.Marshal(clickData)
	// Save request of one click pass data
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
		Method:          "click_pass",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "click",
	})
	if err != nil {
		return nil, err
	}
	// generate click pass auth token
	token := h.generateClickAndUzumAuthToken(click.SecretKey, click.MerchantUserID)
	// send request to click pass
	res, err := h.ClickPassDoRequest(ctx, "/click_pass/payment", clickData, token)
	if err != nil {
		h.log.Info("ClickPassDoRequest error: %v", err.Error())
		return nil, err
	}
	// convert to json response of click pass
	t, _ = json.Marshal(res)
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      t,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Check click pass payment status
func (h *SaleHandler) ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}, token string) (map[string]interface{}, error) {
	fullUrl := h.cfg.ClickEndpointUrl + fmt.Sprintf("/payment/status/%v/%v", data["service_id"], data["payment_id"])
	res, err := h.ClickPassDoRequest(ctx, fullUrl, data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Generate click pass and uzum fast pay auth token
func (h *SaleHandler) generateClickAndUzumAuthToken(secretKey string, merchantUserId int) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	digest := sha1.Sum([]byte(timestamp + secretKey))
	digestStr := fmt.Sprintf("%x", digest)
	return fmt.Sprintf("%d:%s:%s", merchantUserId, digestStr, timestamp)
}

// DoRequest for Click Pass
func (h *SaleHandler) ClickPassDoRequest(ctx context.Context, url string, data interface{}, token string) (map[string]interface{}, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	// Construct request
	fullURL := h.cfg.ClickEndpointUrl + url
	fmt.Println(fullURL)
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Auth", token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response body
	var result map[string]interface{}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return result, nil
}

// Payme Go Handler functon
func (h *SaleHandler) PaymeGo(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error) {
	return h.PaymeGoDoRequest(ctx, data)
}

// DoRequest for Payme Go
func (h *SaleHandler) PaymeGoDoRequest(ctx context.Context, data interface{}) (map[string]interface{}, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Auth", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

// Uzum fast pay handler function
func (h *SaleHandler) UzumFastPay(ctx context.Context, paymentService *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error) {
	var cashBoxId string
	err := h.db.Raw(`SELECT cash_box_id FROM cashbox_operations WHERE id = ?`, CashOperationID).Scan(&cashBoxId).Error
	if err != nil {
		return nil, err
	}
	uzumData := domain.UzumRequest{
		OrderId:       saleID,
		TransactionID: transactionID,
		CashboxCode:   cashBoxId,
		ServiceID:     paymentService.ServiceID,
		Amount:        data.Amount,
		OtpData:       data.OtpData,
	}
	t, err := json.Marshal(uzumData)
	if err != nil {
		return nil, err
	}
	err = h.SaveRequest(ctx, &domain.PaymentRequest{
		Method:          "uzum_fast_pay",
		Payload:         t,
		TransactionID:   transactionID,
		PaymentProvider: "uzum",
	})
	if err != nil {
		h.log.Info("Error on saving uzum fast pay request: %v", err.Error())
		return nil, err
	}

	// Generate Uzum Fast Pay auth token
	token := h.generateClickAndUzumAuthToken(paymentService.SecretKey, paymentService.MerchantUserID)

	res, err := h.UzumFastPayDoRequest(ctx, "/v2/payment", uzumData, token)
	if err != nil {
		return nil, err
	}
	// convert to json response of click pass
	t, _ = json.Marshal(res)
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transactionID,
		Response:      t,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Uzum Fast Pay Check payment status
func (h *SaleHandler) UzumFastPayCheckPaymentStatus(ctx context.Context, paymentService domain.PaymentService, paymentId string) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"service_id": paymentService.ServiceID,
		"payment_id": paymentId,
	}
	token := h.generateClickAndUzumAuthToken(paymentService.SecretKey, paymentService.MerchantUserID)

	res, err := h.UzumFastPayDoRequest(ctx, "/payment/status", data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DoRequest for Uzum Fast Pay
func (h *SaleHandler) UzumFastPayDoRequest(ctx context.Context, url string, data interface{}, token string) (map[string]interface{}, error) {
	client := &http.Client{}
	buf := bytes.Buffer{}

	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	// Construct request
	fullURL := h.cfg.UzumEndpointUrl + url
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response body
	var result map[string]interface{}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return result, nil
}

// Save payment request to database
func (h *SaleHandler) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Create(&req).Error
	if err != nil {
		return err
	}
	return nil
}

// Save payment response to database
func (h *SaleHandler) SaveResponse(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Exec(
		`UPDATE payment_requests SET response = ? WHERE transaction_id = ?`,
		req.Response, req.TransactionID,
	).Error
	if err != nil {
		return err
	}
	return nil
}
