package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// region Create

// create new sale
func (s *Services) CreateSale(tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	var res domain.Sale
	err := tx.Raw(`INSERT INTO sales(employee_id, cash_box_operation_id, store_id, cashbox_id) VALUES(?, ?, ?, ?) RETURNING *`,
		req.EmployeeID, req.CashBoxOperationId, req.StoreId, req.CashboxId).Scan(&res).Error
	if err != nil {
		s.log.Warn("could not create new sale: %v", err)
		return &res, errors.New(constants.InternalServerError)
	}
	return &res, nil
}

// create return sale
func (s *Services) CreateReturnSale(req *domain.SaleReturnRequest) (*domain.Sale, error) {
	var (
		sale             domain.Sale
		cashboxOperation domain.CashboxOperation
		err              error
	)

	// start transaction
	tx := s.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// get cashbox operation
	err = tx.First(&cashboxOperation, "id = ?", req.CashBoxOperationId).Error
	if err != nil {
		s.log.Error("ERROR on getting cashbox_operation: ", err)
		return nil, err
	}
	req.CashboxId = cashboxOperation.CashBoxID

	// build query
	query := `
	INSERT INTO sales (
		employee_id, cash_box_operation_id, cashbox_id, store_id, customer_id, sale_number, parent_id, sale_type, type)
	SELECT ?, ?, ?, store_id, customer_id, sale_number, id, ?, type FROM sales where id = ?
	RETURNING *;`
	err = tx.Raw(query, req.EmployeeID, req.CashBoxOperationId, req.CashboxId, config.SALE_TYPE_RETURN, req.SaleId).Scan(&sale).Error
	if err != nil {
		s.log.Error("ERROR on creating return sale: ", err)
		return nil, err
	}
	// cart item create query
	cquery := `
	INSERT INTO cart_items(sale_id, store_product_id, quantity, unit_quantity, unit_price, total_price, status)
	SELECT ?, sp.id, ?, ?, retail_price, (?*retail_price+(CASE WHEN p.unit_per_pack > 0 THEN retail_price / p.unit_per_pack ELSE 0 END) * ?), ?
	FROM store_products sp JOIN products p ON p.id = sp.product_id WHERE sp.id = ?`
	for _, item := range req.Items {
		item.SaleId = sale.ID
		// complete cart item create query
		err = tx.Exec(cquery, item.SaleId, item.Quantity, item.UnitQuantity, item.Quantity, item.UnitQuantity, config.PENDING, item.StoreProductId).Error
		if err != nil {
			s.log.Error("ERROR on creating return sale items: ", err)
			return nil, err
		}
	}

	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Error("ERROR on commiting transaction: ", err)
		return nil, err
	}
	return &sale, nil
}

// create new sale or get pending sale
func (s *Services) CreateOrGetSale(ctx context.Context, tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	var res *domain.Sale

	// getting pending sale with no cart items
	err := tx.Raw(`
			SELECT * FROM sales 
			WHERE 
				store_id = ? AND 
				employee_id = ? AND 
				cash_box_operation_id = ? AND 
				cashbox_id = ? AND 
				status = ? AND 
				type = ? AND 
				online_status = ?  AND 
				sale_type = ?
			AND NOT EXISTS (
				SELECT 1 FROM cart_items WHERE cart_items.sale_id = sales.id
			)
			LIMIT 1`,
		req.StoreId,
		req.EmployeeID,
		req.CashBoxOperationId,
		req.CashboxId,
		config.PENDING,
		config.SALE_TYPE_OFFLINE,
		config.ONLINE_STATUS_DEFAULT,
		config.SALE_TYPE_SALE).Scan(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) || res == nil || res.ID == "" {
		res, err = s.CreateSale(tx, req)
		if err != nil {
			s.log.Warn("ERROR on creating sale: %v", err)
			return res, err
		}
		return res, nil // return new sale info
	} else if err != nil {
		s.log.Warn("ERROR on getting sale: %v", err)
		return res, err
	}

	return res, nil
}

// create new sale or get pending sale
func (s *Services) CreateOrGetSalePending(tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	var res *domain.Sale

	// getting pending sale with no cart items
	err := s.db.
		Raw(`
			SELECT * FROM sales 
			WHERE store_id = ? AND employee_id = ? AND cash_box_operation_id = ? AND cashbox_id = ?
			AND status = ?  AND sale_type = ?
			LIMIT 1
		`, req.StoreId, req.EmployeeID, req.CashBoxOperationId, req.CashboxId, config.PENDING, config.SALE_TYPE_SALE).
		Scan(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) || res == nil || res.ID == "" {
		res, err = s.CreateSale(tx, req)
		if err != nil {
			s.log.Warn("ERROR on creating sale: %v", err)
			return res, err
		}
		return res, nil // return new sale info
	} else if err != nil {
		s.log.Warn("ERROR on getting sale: %v", err)
		return res, err
	}

	return res, nil
}

// Create sale payment
func (s *Services) CreateSalePayment(tx *gorm.DB, req domain.FinalSale, item domain.FinalPaymentType, paymentServiceId *string) (*domain.SalePayment, error) {
	var (
		now         = time.Now()
		salePayment = domain.SalePayment{}
	)
	query := `
	INSERT INTO sale_payments(
		sale_id, 
		cash_box_operation_id, 
		payment_service_id, 
		payment_type_id, 
		amount, 
		return_amount, 
		paid_at
		) 
	VALUES(?, ?, ?, ?, ?, ?, ?) 
	RETURNING *`
	// Insert sale payments
	err := tx.Raw(query,
		req.SaleID,
		req.CashBoxOperationId,
		paymentServiceId,
		item.PaymentTypeID,
		item.Amount-item.ReturnAmount,
		item.ReturnAmount, now).Scan(&salePayment).Error
	if err != nil {
		s.log.Error("ERROR on creating new sale payment: %w", err)
		return &salePayment, err
	}
	return &salePayment, nil
}

// create sale for online order
func (s *Services) CreateOnlineSale(saleId string, storeID string, customer *domain.Customer, cartItems []domain.CartItemOnlineRequest) (*domain.Sale, error) {
	var (
		sale domain.Sale
		err  error
	)
	tx := s.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// create new sale
	err = tx.Raw(`
	INSERT INTO sales(
		id,
		store_id,
		type,
		online_status,
		service_type,
		customer_id
		) 
	VALUES(?, ?, ?, ?, ?, ?) RETURNING *`,
		saleId, storeID, config.SALE_TYPE_ONLINE, config.ONLINE_STATUS_NEW, config.NOOR, customer.Id).Scan(&sale).Error
	if err != nil {
		return &sale, errors.New("not.created.new.order")
	}
	// create cart_items
	err = tx.Table("cart_items").Create(&cartItems).Error
	if err != nil {
		return &sale, errors.New("not.created.cart_items")
	}

	err = tx.Commit().Error
	if err != nil {
		s.log.Error(err)
		return &sale, err
	}

	return &sale, nil
}

func (s *Services) SaveEposResponse(ctx context.Context, req *domain.EposResponseRequest) error {
	err := s.db.WithContext(ctx).Table("epos_responses").Create(req).Error
	if err != nil {
		s.log.Error("could not save epos response: %v", err)
		return errors.New(constants.InternalServerError)
	}
	return nil
}

// region Update

// update sale payment status
func (s *Services) UpdateSalePaymentStatus(tx *gorm.DB, salePaymentID string) error {
	err := tx.Exec(`UPDATE sale_payments SET status = 'paid' WHERE id = ?`, salePaymentID).Error
	if err != nil {
		s.log.Error("ERROR on updating sale payment status: ", err)
		return err
	}
	return nil
}

// update sales one item with field and value
func (s *Services) UpdateSaleFieldValue(saleID string, field, value string) error {
	err := s.db.Exec(`UPDATE sales SET `+field+` = ? WHERE id = ?`, value, saleID).Error
	if err != nil {
		s.log.Warn("ERROR on updating sale status: %v", err)
		return err
	}
	return nil
}

// update sale with receiving field
func (s *Services) UpdateSaleField(field string, value string, idField string, idValue string) (*domain.Sale, error) {
	var res domain.Sale
	err := s.db.Raw(`UPDATE sales SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// finalize sale
func (s *Services) FinalizeSale(ctx context.Context, req *domain.FinalSale) (*domain.Sale, error) {

	sale, err := s.GetSaleById(ctx, req.SaleID)
	if err != nil {
		return nil, err
	}
	// check if sale is already completed
	if sale.Status == constants.COMPLETED {
		return nil, errors.New(constants.SaleIsClosedError)
	}
	// check
	if len(req.PaymentTypes) == 0 {
		return nil, errors.New(constants.PaymentTypeRequiredError)
	}

	// check sale amount and validate payment types
	req, err = s.matchingPaymentTypeSum(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.ServiceType != nil && *req.ServiceType == config.DMED {
		var cartItems []*domain.CartItemForDMED
		cartItems, err = s.GetCartItems(ctx, sale.ID)
		if err != nil {
			return nil, err
		}
		// send req dmed
		err = s.DmedGiveReceipt(cartItems, req.MarkingData, sale.Employee.FullName, req.PrescriptionID, "check-issue")
		if err != nil {
			return sale, err
		}
		err = s.DmedGiveReceipt(cartItems, req.MarkingData, sale.Employee.FullName, req.PrescriptionID, "issue")
		if err != nil {
			return sale, err
		}
	} else {
		req.ServiceType = nil
	}
	if !req.TaxFree {
		val := config.TAX_FREE
		req.ServiceType = &val
	}

	// start transaction
	tx := s.db.Begin()

	// add marking to cart_items
	err = s.updateCartItemsMarkingCount(ctx, tx, req.MarkingData)
	if err != nil {
		_ = tx.Rollback()
		return sale, err
	}

	for _, item := range req.PaymentTypes {
		err = s.processPayment(ctx, tx, sale, item)
		if err != nil {
			s.log.Warn("ERROR on payment process: %v", err.Error())
			_ = tx.Rollback()
			return sale, err
		}
	}
	// update sale data
	sale, err = s.updateSaleToComplete(ctx, tx, req)
	if err != nil {
		_ = tx.Rollback()
		return sale, err
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Error("could not commit transaction: %v", err)
		return sale, errors.New(constants.InternalServerError)
	}

	return sale, nil
}

// epos result
func (s *Services) EposResult(ctx context.Context, req *domain.EposResponseRequest, user *domain.EmployeeClaims) (*domain.Sale, error) {
	// Ensure response_data is a string
	responseDataStr, ok := req.ResponseData.(string)
	if !ok {
		s.log.Error("response_data is not a valid string")
		return nil, errors.New(constants.BadRequestError)
	}

	// Convert string to []byte and store in Response field
	req.Response = []byte(responseDataStr)

	// start transaction
	tx := s.db.Begin()

	// Get sale by ID
	sale, err := s.GetSaleById(ctx, req.SaleId)
	if err != nil {
		return nil, err
	}

	if req.Error {
		err = s.SaveEposResponse(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	// Save to epos_responses table
	req.Status = 1
	err = s.SaveEposResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	// parse epos success json to structure
	var successResp domain.EposSuccessResponse
	if err = json.Unmarshal([]byte(responseDataStr), &successResp); err != nil {
		s.log.Error("could not parse epos success response: %v", err)
		return nil, errors.New(constants.BadRequestError)
	}

	if successResp.Message.FiscalSign == "" {
		successResp.Message.FiscalSign = successResp.Info.FiscalSign
	}

	// set fiscal data if payment completed with payme
	if sale.PaymentReceiptId != "" {
		var paymentService domain.PaymentService
		err = s.db.First(&paymentService, "store_id = ?", sale.StoreId).Error
		if err != nil {
			s.log.Error("could not get payment service: %v", err)
			return nil, errors.New(constants.InternalServerError)
		}
		fiscalData := domain.FiscalData{
			StatusCode: 0,
			Message:    "accepted",
			TerminalId: successResp.Message.TerminalId,
			ReceiptId:  cast.ToInt(successResp.Message.ReceiptSeq),
			Date:       successResp.Message.DateTime,
			FiscalSign: successResp.Message.FiscalSign,
			QrCodeUrl:  successResp.Message.QrCodeURL,
		}
		err = s.PaymeGoSetFiscalData(ctx, &fiscalData, sale, &paymentService)
		if err != nil {
			return nil, err
		}

	}

	// create or get sale
	res, err := s.CreateSale(tx, &domain.SaleRequest{
		EmployeeID:         user.UserId,
		StoreId:            sale.StoreId,
		CashBoxOperationId: sale.CashBoxOperationId,
		CashboxId:          sale.CashboxId,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Process payment type
func (s *Services) processPayment(
	ctx context.Context,
	tx *gorm.DB,
	sale *domain.Sale,
	item domain.FinalPaymentType,
) error {
	if item.Type == constants.APP && utils.In(item.AppType, constants.AppPayments...) {
		var paymentService *domain.PaymentService
		paymentService, err := s.GetPaymentServiceByStoreId(tx, sale.StoreId, item.AppType)
		if err != nil {
			s.log.Error("could not get payment service by store id: (%v)", sale.StoreId)
			return err
		}

		paymentHandlers := map[string]func(ctx context.Context, tx *gorm.DB, service *domain.PaymentService, data *domain.FinalPaymentType, sale *domain.Sale) (map[string]any, error){
			constants.CLICK: s.ClickPass,
			constants.PAYME: s.PaymeGo,
			constants.UZUM:  s.UzumFastPay,
			constants.ALIF:  s.AlifPay,
		}

		// get payment handlers for integration app services
		handler, exists := paymentHandlers[item.AppType]
		if !exists {
			return errors.New(constants.InvalidPaymentTypeError)
		}

		// check if sale_payment is created
		var resp map[string]any
		resp, err = handler(ctx, tx, paymentService, &item, sale)
		if err != nil || cast.ToString(resp["error_code"]) != "0" {
			return err
		}
	} else if !utils.In(item.Type, constants.PaymentTypes...) {
		return errors.New(constants.InvalidPaymentTypeError)
	}

	return nil
}

func (s *Services) matchingPaymentTypeSum(ctx context.Context, req *domain.FinalSale) (*domain.FinalSale, error) {
	var sum float64
	for _, item := range req.PaymentTypes {
		sum += item.Amount - item.ReturnAmount
		if item.Type == constants.CASH {
			req.Cash = item.Amount - item.ReturnAmount
		} else if item.Type == constants.CARD && item.AppType == constants.HUMO {
			req.Humo = item.Amount
		} else if item.Type == constants.CARD && item.AppType == constants.UZCARD {
			req.Uzcard = item.Amount
		} else if item.Type == constants.APP && item.AppType == constants.CLICK {
			req.Click = item.Amount
		} else if item.Type == constants.APP && item.AppType == constants.PAYME {
			req.Payme = item.Amount
		} else if item.Type == constants.APP && item.AppType == constants.ALIF {
			req.Alif = item.Amount
		} else {
			return req, errors.New(constants.InvalidPaymentTypeError)
		}
	}
	// get cart item sum
	cartItemSum, err := s.cartItemsSumBySaleId(ctx, req.SaleID)
	if err != nil {
		return req, err
	}
	if sum != cartItemSum || req.TotalAmount != cartItemSum || req.TotalAmount != sum {
		s.log.Warn("cartItemSum: %v, paymentTypeSum: %v, req.TotalAmount: %v", cartItemSum, sum, req.TotalAmount)
		return req, errors.New(constants.InvalidSaleAmount)
	}

	return req, nil
}

func (s *Services) updateSaleToComplete(ctx context.Context, tx *gorm.DB, req *domain.FinalSale) (*domain.Sale, error) {
	var res domain.Sale
	query := `
	UPDATE sales
		SET
			total_amount = (SELECT SUM(total_price)-SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
			total_discount = (SELECT SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
			status = ?,
			cash = ?, 
			humo = ?, 
			uzcard = ?,
			click = ?, 
			payme = ?, 
			alif = ?,
			completed_at = NOW(),
			updated_at = NOW()
	WHERE id = ?;
	`
	err := tx.WithContext(ctx).Raw(query,
		req.SaleID,
		req.SaleID,
		constants.COMPLETED,
		req.Cash,
		req.Humo,
		req.Uzcard,
		req.Click,
		req.Payme,
		req.Alif,
		req.SaleID,
	).Scan(&res).Error
	if err != nil {
		s.log.Error("could not complete sale(%s) error: %v", req.SaleID, err)
		return &res, errors.New(constants.InternalServerError)
	}

	return &res, nil
}

// return sale to pending status and reset quantities
func (s *Services) ReturnSale(ctx context.Context, tx *gorm.DB, sale *domain.Sale) error {
	err := s.RestoreStoreProductQuantities(ctx, tx, sale)
	if err != nil {
		s.log.Warn("ERROR on restoring store_product quantity: %v", err)
		return err
	}

	// build query for update sale status to return
	query := `
	UPDATE sales
	SET
		total_amount = 0,
		total_discount = 0,
		status = ?, completed_at = NULL, updated_at = NOW()
	WHERE id = ?;
	`
	// complete the query
	err = tx.WithContext(ctx).Exec(query, config.PENDING, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on update sale to returned: %v", err)
		return err
	}

	return nil
}

// Update cart item status and reduce store product quantities and add employee bonus after completed the sale
func (s *Services) DeductStoreProductQuantities(ctx context.Context, tx *gorm.DB, sale *domain.Sale) error {
	var cartItems []domain.CartItem
	err := tx.WithContext(ctx).Raw(`
		SELECT 
			ci.id, 
			ci.store_product_id, 
			sp.product_id,
			ci.quantity, 
			ci.unit_quantity, 
			unit_price,
			total_price, 
			ci.status, 
			pb.bonus_amount,
			p.unit_per_pack
		FROM 
			cart_items ci
		JOIN 
			store_products sp ON sp.id = ci.store_product_id
		JOIN 
			products p ON sp.product_id = p.id
		LEFT JOIN 
			product_bonuses pb ON pb.product_id = sp.product_id
		WHERE 
			sale_id = ?`, sale.ID).Scan(&cartItems).Error
	if err != nil {
		s.log.Error("", err)
		return err
	}
	var bonusAmount float64
	for _, item := range cartItems {
		err = tx.WithContext(ctx).Exec(`
		UPDATE store_products
		SET
			pack_quantity = GREATEST(CASE WHEN ? > 0 THEN (unit_quantity - ?)/products.unit_per_pack - ? ELSE pack_quantity - ? END, 0),
			unit_quantity = GREATEST(unit_quantity - (? * products.unit_per_pack + ?), 0),
			updated_at = NOW()
		FROM products
		WHERE products.id = store_products.product_id AND  store_products.id = ?`,
			item.UnitQuantity, item.UnitQuantity, item.Quantity, item.Quantity,
			item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			return err
		}
		// add employee bonus
		if item.BonusAmount > 0 && sale.SaleType == config.SALE_TYPE_SALE {
			bonusAmount += item.BonusAmount * float64(item.Quantity)
			if item.UnitPerPack > 0 && item.UnitQuantity > 0 {
				bonusAmount += item.BonusAmount / float64(item.UnitPerPack) * float64(item.UnitQuantity)
			}
			// add employee bonus service
			err = s.AddEmployeeBonus(ctx, tx, &domain.EmployeeBonusRequest{
				EmployeeId:         sale.EmployeeID,
				CashboxOperationId: sale.CashBoxOperationId,
				SaleId:             sale.ID,
				ProductId:          item.ProductId,
				Quantity:           item.Quantity,
				UnitQuantity:       item.UnitQuantity,
				BonusAmount:        bonusAmount,
			})
			if err != nil {
				s.log.Error("ERROR on adding bonus to employee: ", err)
				return err
			}
		}
	}
	return nil
}

// update return sale cart items
func (s *Services) RestoreStoreProductQuantities(ctx context.Context, tx *gorm.DB, sale *domain.Sale) error {
	var cartItems []domain.CartItem
	// get cart items
	err := tx.WithContext(ctx).Raw(`
		SELECT
			id, 
			store_product_id,
			quantity, 
			unit_quantity, 
			unit_price,
			total_price, 
			status
		FROM 
			cart_items 
		WHERE 
			sale_id = ?`,
		sale.ID).Scan(&cartItems).Error
	if err != nil {
		s.log.Warn("ERROR on getting cart_items: %v", err)
		return err
	}
	// update store product quantities
	for _, item := range cartItems {
		err = tx.WithContext(ctx).Exec(`
		UPDATE 
			store_products
		SET
			pack_quantity = FLOOR((? + store_products.unit_quantity + (? * products.unit_per_pack)) / products.unit_per_pack),
			unit_quantity = store_products.unit_quantity + (? * products.unit_per_pack + ?), 
			updated_at = NOW()
		FROM 
			products
		WHERE 
			products.id = store_products.product_id AND  
			store_products.id = ?`,
			item.UnitQuantity,
			item.Quantity,
			item.Quantity,
			item.UnitQuantity,
			item.StoreProductID).Error
		if err != nil {
			s.log.Warn("ERROR on restoring store_products quantity: %v", err)
			return err
		}
	}
	// delete employee bonus for return sale
	err = tx.WithContext(ctx).Exec(`DELETE FROM employee_bonus WHERE sale_id = ?`, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on deleting employee_bonus: %v", err)
		return err
	}
	return nil
}

// set ficalsign to sale
func (s *Services) SetFiscalId(ctx context.Context, tx *gorm.DB, saleID string, fiscalID string) error {
	err := tx.WithContext(ctx).Exec(`UPDATE sales SET fiscal_sign = ?, updated_at = NOW() WHERE id = ?`, fiscalID, saleID).Error
	if err != nil {
		s.log.Warn("ERROR on setting fiscal_id: %v", err)
		return err
	}
	return nil
}

// region Get

// get sale list data
func (s *Services) ListSale(param *domain.QueryParam, userId string) ([]domain.SaleResponse, int64, error) {
	var (
		totalCount int64
		filter     = " WHERE s.status = 'completed' "
		args       = []any{}
		groupBy    = " GROUP BY s.id, em.id, st.id, customers.id, cash_boxes.id, dc.barcode "
		orderBy    = " ORDER BY s.completed_at DESC "
	)
	// get employee info
	var employee domain.Employee
	err := s.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, errors.New("employee not found")
		}
		s.log.Error(err)
		return nil, 0, err
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, s.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyId = employee.CompanyId
	}
	var res = []domain.SaleResponse{}
	query := `
	SELECT
		s.*,
		em.full_name, em.phone,
		st.name AS store_name,
		COALESCE(customers.full_name, '') as customer_name,
		CONCAT(REPEAT('*', GREATEST(LENGTH(dc.barcode) - 4, 0)),RIGHT(dc.barcode, 4)) AS discount_barcode,
		COALESCE(customers.phone, '') AS customer_phone,
		cash_boxes.name AS cash_box_name,
		COALESCE(SUM(CASE WHEN pt.name = 'Naqd' THEN sp.amount ELSE 0 END), 0.00) AS cash,
		COALESCE(SUM(CASE WHEN pt.name = 'Uzcard' THEN sp.amount ELSE 0 END), 0.00) AS uzcard,
		COALESCE(SUM(CASE WHEN pt.name = 'Humo' THEN sp.amount ELSE 0 END), 0.00) AS humo,
		COALESCE(SUM(CASE WHEN pt.name = 'Click' THEN sp.amount ELSE 0 END), 0.00) AS click,
		COALESCE(SUM(CASE WHEN pt.name = 'Payme' THEN sp.amount ELSE 0 END), 0.00) AS payme,
		COALESCE(SUM(CASE WHEN pt.name = 'Alif' THEN sp.amount ELSE 0 END), 0.00) AS alif
	FROM sales s
		LEFT JOIN stores st ON st.id = s.store_id
		LEFT JOIN employees em ON em.id = s.employee_id
		LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
		LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id
		LEFT JOIN customers ON s.customer_id = customers.id
		LEFT JOIN sale_payments sp ON sp.sale_id = s.id
		LEFT JOIN payment_types pt ON sp.payment_type_id = pt.id
		LEFT JOIN discount_cards dc ON customers.id = dc.customer_id
	`
	totalCountQuery := `
		SELECT
			COUNT(DISTINCT s.id) AS total_count
		FROM sales s
			LEFT JOIN stores st ON st.id = s.store_id
			LEFT JOIN employees em ON em.id = s.employee_id
			LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
			LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id
			LEFT JOIN customers ON s.customer_id = customers.id
			LEFT JOIN sale_payments sp ON sp.sale_id = s.id
			LEFT JOIN payment_types pt ON sp.payment_type_id = pt.id
	`

	// filter by payment type
	if param.PaymentTypeID != "" {
		filter += " AND sp.payment_type_id = ? "
		args = append(args, param.PaymentTypeID)
	}
	// filter by employee
	if param.VendorID != "" {
		filter += " AND s.employee_id = ? "
		args = append(args, param.VendorID)
	}
	// filter by store id
	if param.StoreID != "" {
		filter += " AND s.store_id = ? "
		args = append(args, param.StoreID)
	}
	if param.CompanyId != "" {
		filter += " AND st.company_id = ? "
		args = append(args, param.CompanyId)
	}
	// filter by cashbox id
	if param.CashBoxID != "" {
		filter += " AND co.cash_box_id = ? "
		args = append(args, param.CashBoxID)
	}
	// filter by start date and end date
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (s.completed_at + interval '5 hours') BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}
	// filter by start date
	if param.StartDate != "" && param.EndDate == "" {
		filter += " AND (s.completed_at + interval '5 hours') BETWEEN ? AND (?::timestamp + interval '24 hours') "
		args = append(args, param.StartDate, param.StartDate)
	}
	// search condition
	if param.Search != "" {
		filter += " AND (st.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?) "
		args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
	}

	if param.TotalAmountFrom > 0 {
		filter += " AND s.total_amount >= ? "
		args = append(args, param.TotalAmountFrom)
	}

	if param.TotalAmountTo > 0 {
		filter += " AND s.total_amount <= ? "
		args = append(args, param.TotalAmountTo)
	}
	// filter by sale type (SALE || RETURN)
	if param.SaleType != "" {
		filter += " AND s.sale_type = ? "
		args = append(args, param.SaleType)
	}
	// collect total count query
	totalCountQuery += filter
	err = s.db.Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on gettig total count: %v", err)
		return nil, 0, err
	}

	// collect query
	query += filter + groupBy + orderBy + " LIMIT ? OFFSET ?;"
	args = append(args, param.Limit, param.Offset)

	// complete query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	return res, totalCount, nil
}

// Get sale by Id
func (s *Services) GetSaleById(ctx context.Context, saleId string) (*domain.Sale, error) {
	var (
		err  error
		sale domain.Sale
	)

	err = s.db.WithContext(ctx).Preload("Employee").First(&sale, "id = ?", saleId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &sale, errors.New(constants.NotFoundError)
		}
		s.log.Error("could not get sale(%s) info: %v", saleId, err)
		return &sale, errors.New(constants.InternalServerError)
	}

	return &sale, nil
}

// Get sale list
func (s *Services) GetSaleList(param *domain.QueryParam) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
	// build sale get list query
	var res = []domain.SaleResponse{}
	query := s.db.Model(&domain.Sale{}).Table("sales s").
		Preload("SalePayments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("PaymentType")
		}).
		Select(`s.*,st.name AS store_name, c.full_name as customer_name`).
		// Change INNER JOIN to LEFT JOIN to include sales without store_id
		Joins("JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN customers c ON s.customer_id = c.id")

	// filter by employee
	if param.VendorID != "" {
		query = query.Where("s.employee_id = ?", param.VendorID)
	}
	// filter by store id
	if param.StoreID != "" {
		query = query.Where("s.store_id = ?", param.StoreID)
	}

	// filter by start date and end date
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("s.completed_at::date >= ? AND s.completed_at::date <= ?  ", param.StartDate, param.EndDate)
	}
	// filter by start date
	if param.StartDate != "" && param.EndDate == "" {
		query = query.Where("s.completed_at::date >= ?", param.StartDate)
	}
	// search condition
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("st.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?", param.Search, param.Search)
	}
	// complete query
	err := query.
		Where("s.status = 'completed'").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("s.completed_at DESC").
		Find(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return res, totalCount, nil
}

// get sale payments by sale_id
func (s *Services) GetPaymeSalePayment(ctx context.Context, tx *gorm.DB, saleID string) (*domain.SalePayment, error) {
	var salePayment domain.SalePayment
	query := `
	SELECT sp.*
	FROM sale_payments sp
	JOIN payment_types pt ON pt.id = sp.payment_type_id
	WHERE sp.sale_id = ? AND pt.type = ?
	`
	err := tx.Raw(query, saleID, config.PAYME).Scan(&salePayment).Error
	if err != nil {
		s.log.Warn("ERROR on getting sale_payments by saleID: %v", err)
		return &salePayment, err
	}
	return &salePayment, nil
}

// Get Payment service with store id and payment type  if status is active
func (s *Services) GetPaymentServiceByStoreId(tx *gorm.DB, storeId, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := tx.
		Where("store_id = ?", storeId).
		Where("type = ? AND is_active = true", paymentType).
		First(&res).Error
	if err != nil {
		s.log.Error("could not get payment_service by store(%s) error: %v", storeId, err)
		return &res, errors.New(constants.InternalServerError)
	}
	return &res, nil
}

// cart items sum of the sale
func (s *Services) cartItemsSumBySaleId(ctx context.Context, saleID string) (float64, error) {
	var sum float64
	err := s.db.
		WithContext(ctx).
		Raw(`SELECT SUM(total_price) - SUM(discount_amount) AS sum FROM cart_items WHERE sale_id = ?`, saleID).Scan(&sum).Error
	if err != nil {
		s.log.Error("could not calculate cart_items sum: %v", err)
		return sum, errors.New(constants.InternalServerError)
	}
	return sum, nil
}

// sum sale payment amounts
func (s *Services) collectSalePaymentAmount(typeAmounts []domain.FinalPaymentType) float64 {
	var sum float64
	for _, v := range typeAmounts {
		sum += (v.Amount - v.ReturnAmount)
	}
	return sum
}

// online pending sale list
func (s *Services) OnlinePendingSaleList(param *domain.QueryParam) ([]domain.Sale, int64, error) {
	var (
		res        []domain.Sale
		filter     = " WHERE s.store_id = ? AND (s.online_status = 1 OR s.online_status = 2) "
		args       = []any{param.StoreID}
		group      = " GROUP BY s.id "
		order      = " ORDER BY s.created_at DESC "
		totalCount int64
	)
	query := `
	SELECT
		s.id,
		s.store_id,
		s.employee_id,
		s.cashbox_id,
		s.cash_box_operation_id,
		s.sale_number,
		s.status,
		s.online_status,
		s.type,
		s.customer_id,
		s.sale_type,
		s.created_at,
		s.updated_at,
		COALESCE(SUM(ci.total_price), 0.00) AS total_amount,
		COALESCE(COUNT(ci.id), 0) AS count
	FROM sales s
	LEFT JOIN cart_items ci ON s.id = ci.sale_id
	`

	totalCountQuery := `
	SELECT
		COUNT(*) as total_count
	FROM sales s
	LEFT JOIN cart_items ci ON s.id = ci.sale_id
	`

	if param.StartDate != "" {
		filter += " AND (s.created_at + interval '5 hours')::date >= ? "
		args = append(args, param.StartDate)
	}

	if param.EndDate != "" {
		filter += " AND (s.created_at + interval '5 hours')::date <= ? "
		args = append(args, param.EndDate)
	}

	if param.Search != "" {
		filter += " AND CAST(s.sale_number AS TEXT) LIKE ? "
		args = append(args, "%"+param.Search+"%")
	}
	// collect and execute totalCount query
	totalCountQuery += filter + group
	err := s.db.Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Error("Error on getting total_count online sale: %v", err)
		return res, totalCount, err
	}

	// collect and execute query
	query += filter + group + order + " LIMIT ? OFFSET ?;"
	args = append(args, param.Limit, param.Offset)
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("Error on getting online pending sale list: %v", err)
		return res, totalCount, err
	}

	return res, totalCount, nil
}

func (s *Services) ListPendingSales(param *domain.QueryParam, userId string) ([]domain.SaleResponse, int64, error) {
	var (
		totalCount int64
		filter     = " WHERE s.status = 'pending' "
		args       []any
		groupBy    = " GROUP BY s.id, em.id, st.id, customers.id, cash_boxes.id"
		orderBy    = " ORDER BY s.created_at DESC"
		having     = " HAVING COALESCE(SUM(ci.quantity + (ci.unit_quantity::decimal / NULLIF(products.unit_per_pack, 0))), 0) > 0"
	)

	// SELECT query
	query := `
	SELECT
		s.*,
		em.full_name, em.phone,
		st.name AS store_name,
		COALESCE(customers.full_name, '') AS customer_name,
		COALESCE(customers.phone, '') AS customer_phone,
		cash_boxes.name AS cash_box_name,
		round(COALESCE(SUM(ci.quantity + (ci.unit_quantity::decimal / NULLIF(products.unit_per_pack, 0))), 0), 2) AS product_count,
		COUNT(*) OVER() AS total_count
	FROM sales s
		LEFT JOIN stores st ON st.id = s.store_id
		LEFT JOIN employees em ON em.id = s.employee_id
		LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
		LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id
		LEFT JOIN customers ON s.customer_id = customers.id
		LEFT JOIN cart_items ci ON ci.sale_id = s.id
		LEFT JOIN store_products sp ON ci.store_product_id = sp.id
		LEFT JOIN products ON sp.product_id = products.id
	`

	// COUNT query
	totalCountQuery := `
	SELECT COUNT(*) FROM (
		SELECT s.id
		FROM sales s
			LEFT JOIN stores st ON st.id = s.store_id
			LEFT JOIN employees em ON em.id = s.employee_id
			LEFT JOIN cashbox_operations co ON s.cash_box_operation_id = co.id
			LEFT JOIN cash_boxes ON co.cash_box_id = cash_boxes.id
			LEFT JOIN customers ON s.customer_id = customers.id
			LEFT JOIN cart_items ci ON ci.sale_id = s.id
		    LEFT JOIN store_products sp ON ci.store_product_id = sp.id
			LEFT JOIN products ON sp.product_id = products.id
	`

	// Filters
	if param.StoreID != "" {
		filter += " AND s.store_id = ?"
		args = append(args, param.StoreID)
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (s.created_at + interval '5 hours') BETWEEN ? AND ?"
		args = append(args, param.StartDate, param.EndDate)
	} else if param.StartDate != "" {
		filter += " AND (s.created_at + interval '5 hours') >= ?"
		args = append(args, param.StartDate)
	}
	if param.Search != "" {
		filter += " AND (st.name ILIKE ? OR CAST(s.sale_number AS TEXT) ILIKE ?)"
		args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
	}

	// Finalize total count query
	totalCountQuery += filter + groupBy + having + ") AS temp"
	err := s.db.Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Error("count query error: %v", err)
		return nil, 0, err
	}

	// Final SELECT query
	query += filter + groupBy + having + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	var res []domain.SaleResponse
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("list query error: %v", err)
		return nil, 0, err
	}

	return res, totalCount, nil
}

// accept order
func (s *Services) AcceptOnlineSale(req *domain.ConfirmOnlineSaleRequest) error {
	var err error
	tx := s.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	defer recoverTransaction(tx, s.log)
	defer RollbackIfError(tx, &err)
	var sale *domain.Sale
	err = tx.First(&sale, "id = ?", req.SaleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("sale.not.found")
		}
		return err
	}
	// accepted online sale
	err = tx.Exec(`
	UPDATE 
		sales
	SET
		cash_box_operation_id = ?,
		cashbox_id = ?,
		employee_id = ?,
		online_status = ?
	WHERE id = ?`,
		req.CashBoxOperationID,
		req.CashboxID,
		req.EmployeeID,
		config.ONLINE_STATUS_PENDING,
		req.SaleID).Error

	if err != nil {
		s.log.Warn("ERROR on getting online sale count: %v", err)
		return err
	}
	// Prepare Headers
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.cfg.NoorApiToken),
		"Content-Type":  "application/json",
	}
	requestData, err := json.Marshal(gin.H{"order_id": sale.SaleNumber})
	var response *http.Response
	url := s.cfg.NoorApiUrl + fmt.Sprintf("/orders/vendor/%d/confirm", sale.SaleNumber)
	err = DoRequest(&response, "PATCH", url, requestData, headers)
	if err != nil {
		s.log.Warn("ERROR on sending confirm request: %v", err)
		return err
	}
	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		return err
	}

	return nil
}

// accept order
func (s *Services) CancelOnlineSale(req *domain.ConfirmOnlineSaleRequest) error {
	var err error
	tx := s.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	defer recoverTransaction(tx, s.log)
	defer RollbackIfError(tx, &err)
	var sale *domain.Sale
	err = tx.First(&sale, "id = ?", req.SaleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("sale.not.found")
		}
		return err
	}
	// accepted online sale
	err = tx.Exec(`
	UPDATE 
		sales
	SET
		cash_box_operation_id = ?,
		cashbox_id = ?,
		employee_id = ?,
		online_status = ?
	WHERE id = ?`,
		req.CashBoxOperationID,
		req.CashboxID,
		req.EmployeeID,
		config.ONLINE_STATUS_CANCELED,
		req.SaleID).Error

	if err != nil {
		s.log.Warn("ERROR on getting online sale count: %v", err)
		return err
	}
	// Prepare Headers
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.cfg.NoorApiToken),
		"Content-Type":  "application/json",
	}
	requestData, err := json.Marshal(gin.H{"order_id": sale.SaleNumber})
	var response *http.Response
	url := s.cfg.NoorApiUrl + fmt.Sprintf("/orders/vendor/%d/cancel", sale.SaleNumber)
	err = DoRequest(&response, "PATCH", url, requestData, headers)
	if err != nil {
		s.log.Warn("ERROR on sending confirm request: %v", err)
		return err
	}
	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		return err
	}

	return nil
}

// delete sale_payments by sale_id
func (s *Services) DeleteSalePayments(ctx context.Context, tx *gorm.DB, saleId string) error {
	err := tx.WithContext(ctx).Exec(`DELETE FROM sale_payments WHERE sale_id = ?`, saleId).Error
	if err != nil {
		tx.Rollback()
		s.log.Error("could not delete sale_payments: %v", err)
		return err
	}
	return nil

}

func (s *Services) GetPrescriptionsFromDMED(patientID, safeCode string) ([]domain.Prescription, error) {
	url := fmt.Sprintf("/prescriptions?patient_id=%s&safe_code=%s", patientID, safeCode)

	respBody, err := s.doRequestToDMED("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var rawResp = domain.PrescriptionResponse{}
	if err = json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return rawResp.Data, nil
}

// Internal reusable method
func (s *Services) doRequestToDMED(method, url string, data any) ([]byte, error) {
	var (
		body       []byte
		bodyReader io.Reader
		err        error
	)

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		fmt.Printf("Request body DMEDD: %s\n", string(body))
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, s.cfg.DmedApiUrl+url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.DmedApiToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	fmt.Println(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DMED API error: %s", string(respBody))
	}

	return respBody, nil
}

func (s *Services) DmedGiveReceipt(cartItems []*domain.CartItemForDMED, markingData []domain.MarkingData, employeeName, prescriptionID, action string) error {
	for i, cartItem := range cartItems {
		q := cartItem.Quantity
		uq := cartItem.UnitQuantity
		j := 0

		for q > 0 || uq > 0 {
			var drugAmount int
			if q > 0 {
				drugAmount = cartItem.UnitPerPack
				q--
			} else if uq > 0 {
				drugAmount = uq
				uq = 0
			}

			payload := map[string]any{
				"drug_amount":         drugAmount,
				"price":               int(cartItem.UnitPrice),
				"issued_by_full_name": employeeName,
				// //   "pharmacy_id": 123,
			}
			if j < len(markingData[i].MarkingList) && markingData[i].MarkingList[j] != "" {
				payload["marking_code"] = markingData[i].MarkingList[j]
			} else if cartItem.SerialNumber != "" && cartItem.Barcode != "" {
				payload["serial_number"] = cartItem.SerialNumber
				payload["gtin"] = "010" + cartItem.Barcode
			} else {
				s.log.Error("could not find serial number or marking code for dmed")
				return errors.New(constants.SerialOrMarkingRequiredError)
			}

			url := fmt.Sprintf("/prescriptions/%d/%s", markingData[i].DmedId, action)
			method := http.MethodPost
			if action == "issue" {
				method = http.MethodPut
			}
			if _, err := s.doRequestToDMED(method, url, payload); err != nil {
				s.log.Error("could not send dmed %s request: %v", action, err)
				return fmt.Errorf("DMED %s failed: %w", action, err)
			}
			j++
		}
	}
	return nil
}

func (s *Services) ReturnStatusPending(ctx context.Context, tx *gorm.DB, sale *domain.Sale) error {
	// build query for update sale status to return
	query := `
	UPDATE sales
	SET
		total_amount = 0,
		total_discount = 0,
		status = ?, completed_at = NULL, updated_at = NOW()
	WHERE id = ?;
	`
	// complete the query
	err := tx.WithContext(ctx).Exec(query, config.PENDING, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on update sale to returned: %v", err)
		return err
	}
	return nil
}

func (s *Services) GetStoreProductsDifference(ctx context.Context, tx *gorm.DB, saleId string) error {
	var (
		err   error
		diffs domain.SaleDifference
	)
	defer RollbackIfError(tx, &err)

	err = tx.WithContext(ctx).Raw(`
        SELECT
            ROUND(SUM(sp.retail_price * (ci.quantity + (ci.unit_quantity::numeric / p.unit_per_pack))) - SUM(ci.total_price), 2) as difference,
            ROUND(SUM(sp.retail_price * (ci.quantity + (ci.unit_quantity::numeric / p.unit_per_pack))) - s.total_amount + s.total_discount, 2) as total_difference
        FROM cart_items ci
        LEFT JOIN sales s ON ci.sale_id = s.id
        LEFT JOIN store_products sp ON ci.store_product_id = sp.id
        JOIN products p ON sp.product_id = p.id
        WHERE ci.sale_id = ?
        GROUP BY s.id
    `, saleId).Scan(&diffs).Error

	if err != nil {
		s.log.Error("", err)
		return err
	}
	if math.Abs(diffs.Difference) > 0.01 {
		return errors.New("invalid.sale.amount")
	}
	if math.Abs(diffs.TotalDifference) > 0.01 {
		return errors.New("invalid.sale.amount")
	}
	return nil
}
