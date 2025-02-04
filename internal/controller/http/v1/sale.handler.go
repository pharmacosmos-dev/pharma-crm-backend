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
		p.id, p.name, p.barcode, sp.retail_price, p.bonus_percent, 
		p.bonus_amount, p.photos, ci.quantity, 
		ci.unit_quantity, ci.total_price, u.short_name
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
// @Param employee_id query string false "Employee ID"
// @Param store_id query string false "Store ID"
// @Param search query string false "Search"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list [get]
func (h *SaleHandler) List(c *gin.Context) {
	var (
		totalAmount int64
		startDate   = c.Query("start_date")
		endDate     = c.Query("end_date")
		employeeID  = c.Query("employee_id")
		storeID     = c.Query("store_id")
		search      = c.Query("search")
	)

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	res := []domain.SaleResponse{}
	query := h.db.Model(&domain.Sale{}).Table("sales s").
		Preload("Employee").
		Preload("Customer").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).
		Select("s.*, stores.name AS store_name, cash_boxes.name AS cash_box_name").
		Joins("JOIN employees ON s.employee_id = employees.id").
		Joins("JOIN stores ON employees.store_id = stores.id").
		Joins("JOIN cashbox_operations co ON s.cash_box_operation_id = co.id").
		Joins("JOIN cash_boxes ON co.cash_box_id = cash_boxes.id")

	if employeeID != "" {
		query = query.Where("s.employee_id = ?", employeeID)
	}
	if storeID != "" {
		query = query.Where("stores.id = ?", storeID)
	}
	if startDate != "" && endDate != "" {
		query = query.Where("s.completed_at::date >= ? AND s.completed_at::date <= ?  ", startDate, endDate)
	}
	if startDate != "" && endDate == "" {
		query = query.Where("s.completed_at::date = ?", startDate)
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("stores.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?", search, search)
	}

	err = query.Where("s.status = 'completed'").
		Count(&totalAmount).
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

	data := utils.ListResponse(res, totalAmount, limit, offset)
	handleResponse(c, OK, data)
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
// @Param input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	var (
		body domain.FinalSale
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
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
	if len(body.PaymentTypes) == 0 {
		handleResponse(c, BadRequest, "At least one payment type is required")
		return
	}

	// create transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// get sale
	var count int64
	err = tx.Model(&domain.Sale{}).Where("id = ? AND status = 'completed'", body.SaleID).Count(&count).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// check count > 0
	if count > 0 {
		handleResponse(c, CONFLICT, "Sale is already completed")
		return
	}

	// get store_id by employee_id
	err = tx.Raw(`SELECT store_id FROM employees WHERE id = ?`, userID).Scan(&body.StoreID).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	now := time.Now()
	var salePayment domain.SalePaymentRequest
	// iterate over payment types
	for _, item := range body.PaymentTypes {
		// err = cashboxOperationAmounts(tx, item)
		if item.Type == "app" && (item.AppType == config.CLICK || item.AppType == config.PAYME || item.AppType == config.UZUM) {
			var paymentService domain.PaymentService
			err = h.db.First(&paymentService, "store_id = ? AND type = ? AND is_active = true",
				body.StoreID, item.AppType).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					handleResponse(c, NotFound, "The Payment service is not active OR does not exist")
					return
				}
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			// payment handlers map for each payment type
			paymentHandlers := map[string]func(ctx context.Context, click *domain.PaymentService, data *domain.FinalPaymentType, CashOperationID string, transactionID string, saleID string) (map[string]interface{}, error){
				config.CLICK: h.ClickPass,
				config.PAYME: h.PaymeGo,
				config.UZUM:  h.UzumFastPay,
			}
			// get handler for the payment type
			handler, exists := paymentHandlers[item.AppType]
			// return error if payment type is not found
			if !exists {
				handleResponse(c, InternalError, "Invalid payment type")
				return
			}
			salePayment = domain.SalePaymentRequest{
				ID:                 uuid.New().String(),
				SaleID:             body.SaleID,
				CashBoxOperationID: body.CashBoxOperationId,
				PaymentServiceID:   &paymentService.ID,
				PaymentTypeID:      item.PaymentTypeID,
				Amount:             item.Amount,
				PaidAt:             &now,
				Status:             "pending",
			}
			// Insert sale payments
			err = tx.
				Table("sale_payments").
				Create(&salePayment).Error
			if err != nil {
				tx.Rollback()
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}

			resp, err := handler(c.Request.Context(), &paymentService, &item, body.CashBoxOperationId, salePayment.ID, body.SaleID)
			if err != nil {
				tx.Rollback()
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			if errCode, ok := resp["error_code"].(float64); ok && errCode == 0 {
				err = tx.
					Table("sale_payments").Where("id = ? ", salePayment.ID).Update("status", "paid").Error
				if err != nil {
					tx.Rollback()
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
				err = tx.Exec(`
				INSERT INTO sale_payment_summary (cash_box_operation_id, payment_type_id, total_amount) 
				VALUES(?, ?, ?)
				ON CONFLICT (cash_box_operation_id, payment_type_id) 
				DO UPDATE SET total_amount = EXCLUDED.total_amount + ?`, body.CashBoxOperationId, item.PaymentTypeID, item.Amount, item.Amount).Error
				if err != nil {
					tx.Rollback()
					h.log.Error(err)
					handleResponse(c, InternalError, err.Error())
					return
				}
				continue
			} else {
				tx.Rollback()
				t, _ := json.Marshal(resp)
				h.log.Info("Failed payment with %v", string(t))
				handleResponse(c, InternalError, "Failed payment with "+item.AppType)
				return
			}
		} else if item.Type == "cash" || item.Type == "card" {
			// Insert sale payments if payment is cash or card
			salePayment = domain.SalePaymentRequest{
				ID:                 uuid.New().String(),
				SaleID:             body.SaleID,
				CashBoxOperationID: body.CashBoxOperationId,
				PaymentTypeID:      item.PaymentTypeID,
				Amount:             item.Amount,
				PaidAt:             &now,
				Status:             "paid",
			}
			// Insert sale payments
			err = tx.
				Table("sale_payments").
				Create(&salePayment).Error
			if err != nil {
				tx.Rollback()
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
			err = tx.Exec(`
			INSERT INTO sale_payment_summary (cash_box_operation_id, payment_type_id, total_amount) 
			VALUES(?, ?, ?)
			ON CONFLICT (cash_box_operation_id, payment_type_id) 
			DO UPDATE SET total_amount = EXCLUDED.total_amount + ?`, body.CashBoxOperationId, item.PaymentTypeID, item.Amount, item.Amount).Error
			if err != nil {
				tx.Rollback()
				h.log.Error(err)
				handleResponse(c, InternalError, err.Error())
				return
			}
		} else {
			handleResponse(c, InternalError, "Invalid payment type")
			return
		}

	}

	// Update sale status
	err = updateSaleStatus(tx, body.SaleID, body.TotalAmount, body.CustomerID)
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Update cart items
	err = updateCartItemStatus(tx, body.SaleID)
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}

	newSale := domain.SaleRequest{
		ID:                 uuid.New().String(),
		EmployeeID:         cast.ToString(userID),
		CashBoxOperationId: body.CashBoxOperationId,
	}
	err = tx.
		WithContext(c.Request.Context()).
		Table("sales").
		Create(&newSale).Error
	if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, newSale.ID)
}

// Update sale status and total amount after the sale is completed
func updateSaleStatus(tx *gorm.DB, saleID string, totalAmount float64, customerID *string) error {
	return tx.
		Table("sales").
		Where("id = ?", saleID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"total_amount": totalAmount,
			"customer_id":  customerID,
			"completed_at": time.Now(),
		}).Error
}

// Update cart item status and store product quantities after the sale is completed
func updateCartItemStatus(tx *gorm.DB, saleID string) error {
	var cartItems []domain.CartItem
	err := tx.Raw(`
		SELECT 
			id, store_product_id,
			quantity, unit_price,
			total_price, status
		FROM cart_items WHERE sale_id = ?`, saleID).
		Scan(&cartItems).Error
	if err != nil {
		return err
	}

	for _, item := range cartItems {
		err = tx.Exec(`
		UPDATE store_products 
		SET 
			pack_quantity = pack_quantity - ?, 
			unit_quantity = unit_quantity - ? * unit_per_pack + ? 
		WHERE id = ?`,
			item.Quantity, item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			return err
		}
	}

	return tx.
		Table("cart_items").
		Where("sale_id = ?", saleID).
		Update("status", "sold").Error
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
