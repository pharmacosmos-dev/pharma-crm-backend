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
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// region Create

// create new sale
func (s *Services) CreateSale(ctx context.Context, tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	// check cashbox_id
	if req.CashboxId == "" {
		operation, err := s.GetCashboxOperationByID(ctx, req.CashBoxOperationId)
		if err != nil {
			return nil, err
		}
		req.CashboxId = operation.CashBoxID
	}

	var res domain.Sale
	query := "INSERT INTO sales(employee_id, cash_box_operation_id, store_id, cashbox_id, display_id) VALUES(?, ?, ?, ?, ?) RETURNING *"
	err := tx.WithContext(ctx).Raw(query,
		req.EmployeeId,
		req.CashBoxOperationId,
		req.StoreId,
		req.CashboxId,
		s.generateDisplayId(),
	).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not create new sale: %v", err)
		return &res, domain.InternalServerError
	}
	return &res, nil
}

// create return sale
func (s *Services) CreateReturnSale(ctx context.Context, req *domain.SaleReturnRequest) (*domain.Sale, error) {

	// get cashbox operation
	if req.CashboxId == "" {
		operation, err := s.GetCashboxOperationByID(ctx, req.CashBoxOperationId)
		if err != nil {
			return nil, err
		}
		req.CashboxId = operation.CashBoxID
	}

	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// build create sale query
	query := `
	INSERT INTO sales (
		employee_id, 
		cash_box_operation_id, 
		cashbox_id, 
		store_id, 
		customer_id, 
		sale_number, 
		parent_id, 
		sale_type, 
		type,
		display_id,
		stage
		)
	SELECT 
		?, 
		?, 
		?, 
		store_id, 
		customer_id, 
		sale_number,
		id, 
		?, 
		type,
		display_id,
		?
	FROM sales
	WHERE id = ? RETURNING *`

	// execute new return sale query
	var sale domain.Sale
	err := tx.Raw(query,
		req.EmployeeId,
		req.CashBoxOperationId,
		req.CashboxId,
		constants.SaleTypeReturn,
		constants.SaleStageReturning,
		req.SaleId,
	).Scan(&sale).Error
	if err != nil {
		s.log.Errorf("could not create new return sale: %v", err)
		_ = tx.Rollback()
		return nil, domain.InternalServerError
	}
	// cart item create query
	cquery := `
	INSERT INTO cart_items(
		sale_id,
		store_product_id,
		unit_quantity,
		unit_price,
		total_price
		)
	SELECT
		?,
		ci.store_product_id,
		ci.unit_quantity * (-1),
		ci.unit_price,
		ci.total_price * (-1)
	FROM cart_items ci
	WHERE ci.sale_id = ? AND ci.store_product_id = ?;
	`

	for _, item := range req.Items {
		err = tx.Exec(cquery,
			sale.Id,
			req.SaleId,
			item.StoreProductId,
		).Error
		if err != nil {
			s.log.Errorf("could not create return sale items: %v", err)
			_ = tx.Rollback()
			return nil, err
		}

	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return nil, err
	}
	return &sale, nil
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
		saleId, storeID, constants.SaleTypeOnline, constants.SaleOnlineStageNew, constants.ServiceTypeNoor, customer.Id).Scan(&sale).Error
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
		s.log.Errorf("could not save epos response: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// region Update

// finalize sale
func (s *Services) FinalizeSale(ctx context.Context, req *domain.FinalSale) (*domain.MarkingItemsResponse, error) {
	sale, err := s.GetSaleById(ctx, req.SaleID)
	if err != nil {
		return nil, err
	}
	// check if sale is already completed
	if utils.In(sale.Stage, constants.FinishedSaleStages...) {
		return nil, domain.SaleIsClosedError
	}

	// check payment types
	if len(req.PaymentTypes) == 0 {
		return nil, domain.PaymentTypeRequiredError
	}

	// check sale amount and validate payment types
	req, err = s.matchingPaymentTypeSum(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.ServiceType != nil && *req.ServiceType == constants.ServiceTypeDmed {
		var cartItems []*domain.CartItemForDMED
		cartItems, err = s.GetCartItems(ctx, sale.Id)
		if err != nil {
			return nil, err
		}
		// send req dmed
		err = s.DmedGiveReceipt(cartItems, req.MarkingData, sale.Employee.FullName, req.PrescriptionID, "check-issue")
		if err != nil {
			return nil, err
		}
		err = s.DmedGiveReceipt(cartItems, req.MarkingData, sale.Employee.FullName, req.PrescriptionID, "issue")
		if err != nil {
			return nil, err
		}
	} else {
		req.ServiceType = nil
	}
	if !req.TaxFree {
		val := constants.GeneralStatusTaxFree
		req.ServiceType = &val
	}

	// start transaction
	tx := s.db.Begin()

	// add marking to cart_items
	err = s.updateCartItemsMarkingCount(ctx, tx, req.MarkingData)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	updates := map[string]any{}
	updates["tax_free"] = req.TaxFree

	if req.TaxFree {
		sale = s.getSalePayAmounts(sale, req)
		if sale.Stage < constants.SaleStagePayFinished {
			err = s.Payment(ctx, tx, sale, nil)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}
			updates["cash"] = req.Cash - req.ReturnAmount
			updates["humo"] = req.Humo
			updates["uzcard"] = req.Uzcard
			updates["click"] = req.Click
			updates["payme"] = req.Payme
			updates["alif"] = req.Alif
			updates["total_amount"] = gorm.Expr("(SELECT COALESCE(SUM(total_price) - SUM(discount_amount), 0) FROM cart_items WHERE sale_id = ?)", req.SaleID)
			updates["total_discount"] = gorm.Expr("(SELECT COALESCE(SUM(discount_amount), 0) FROM cart_items WHERE sale_id = ?)", req.SaleID)
			updates["stage"] = constants.SaleStagePayFinished
			updates["updated_at"] = time.Now()
		}

		if sale.Stage < constants.SaleStageFinished {
			err = s.ApplySaleInventoryUpdate(ctx, tx, sale)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}
			updates["stage"] = constants.SaleStageFinished
			updates["updated_at"] = time.Now()
			updates["completed_at"] = time.Now()
		}
	} else {
		if sale.Stage < constants.SaleStagePayFinished {
			if req.OtpCode != "" {
				updates["otp_code"] = req.OtpCode
			}

			updates["cash"] = req.Cash - req.ReturnAmount
			updates["humo"] = req.Humo
			updates["uzcard"] = req.Uzcard
			updates["click"] = req.Click
			updates["payme"] = req.Payme
			updates["alif"] = req.Alif
			updates["total_amount"] = gorm.Expr("(SELECT COALESCE(SUM(total_price) - SUM(discount_amount), 0) FROM cart_items WHERE sale_id = ?)", req.SaleID)
			updates["total_discount"] = gorm.Expr("(SELECT COALESCE(SUM(discount_amount), 0) FROM cart_items WHERE sale_id = ?)", req.SaleID)
			updates["stage"] = constants.SaleStageOfdWaiting
			updates["updated_at"] = time.Now()
		}
	}

	// update sale data
	if len(updates) > 0 {
		err = s.updateSaleFields(ctx, tx, req.SaleID, updates)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	res, err := s.GetDatasByMarkings(ctx, tx, req.MarkingData)
	if err != nil {
		_ = tx.Rollback()
		return res, err
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Error("could not commit transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// epos result
func (s *Services) EposResult(ctx context.Context, req *domain.EposResponseRequest, user *domain.EmployeeClaims) (*domain.Sale, error) {
	// Ensure response_data is a string
	responseDataStr, ok := req.ResponseData.(string)
	if !ok {
		s.log.Error("response_data is not a valid string")
		return nil, domain.BadRequestError
	}

	// Convert string to []byte and store in Response field
	req.Response = []byte(responseDataStr)

	// Get sale by ID
	sale, err := s.GetSaleById(ctx, req.SaleId)
	if err != nil {
		return nil, err
	}

	// Check sale finish stages
	if utils.In(sale.Stage, constants.FinishedSaleStages...) {
		return nil, domain.SaleIsClosedError
	}

	// Epos response error
	if req.Error {
		if sale.Stage < constants.SaleStageOfdCancelled {
			err = s.SaveEposResponse(ctx, req)
			if err != nil {
				return nil, err
			}
			updates := map[string]any{
				"stage":      constants.SaleStageOfdCancelled,
				"updated_at": time.Now(),
			}

			err = s.updateSaleFields(ctx, s.db, sale.Id, updates)
			if err != nil {
				return nil, err
			}
		}

		return sale, nil
	}

	// Decode Fiscal Data
	fiscal, err := s.DecodeFiscalData(responseDataStr)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			s.log.Errorf("Panic recovered in EposResult: %v", r)
		}
	}()

	updates := map[string]any{}

	if sale.Stage < constants.SaleStageOfdSent {
		req.Status = 1
		err = s.SaveEposResponse(ctx, req)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		updates["stage"] = constants.SaleStageOfdSent
		updates["fiscal_sign"] = fiscal.FiscalSign
		updates["check_url"] = fiscal.QrCodeUrl
		updates["is_sent_to_tax"] = true
		updates["updated_at"] = time.Now()

		// Save fiscal data immediately within transaction
		err = s.updateSaleFields(ctx, tx, sale.Id, updates)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		// Update sale object for next stages
		sale.Stage = constants.SaleStageOfdSent

		// Clear updates for next stages
		updates = map[string]any{}

	} else {
		fiscal, err = s.getFiscalDataBySaleId(ctx, sale.Id)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	// Stage 2: Handle Payment (stages 7-8)
	if sale.Stage < constants.SaleStagePayFinished {
		err = s.Payment(ctx, tx, sale, &fiscal)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		updates["stage"] = constants.SaleStagePayFinished
		updates["updated_at"] = time.Now()
	} else {
		s.log.Infof("Payment already processed for sale %s, skipping", sale.Id)
	}

	// Stage 3: Handle Inventory (stage 9)
	if sale.Stage < constants.SaleStageFinished {
		err = s.ApplySaleInventoryUpdate(ctx, tx, sale)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		updates["stage"] = constants.SaleStageFinished
		updates["updated_at"] = time.Now()
		updates["completed_at"] = time.Now()
	} else {
		s.log.Infof("Inventory already applied for sale %s, skipping", sale.Id)
	}

	if len(updates) > 0 {
		err = s.updateSaleFields(ctx, tx, req.SaleId, updates)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	// Create new sale for next operation
	res, err := s.CreateSale(ctx, tx, &domain.SaleRequest{
		EmployeeId:         user.UserId,
		StoreId:            sale.StoreId,
		CashBoxOperationId: sale.CashBoxOperationId,
		CashboxId:          sale.CashboxId,
	})
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit epos-result transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) ApplySaleInventoryUpdate(ctx context.Context, tx *gorm.DB, sale *domain.Sale) error {
	// Single query with JOIN to get both cart items and store product quantities
	type CartItemWithProduct struct {
		domain.CartItem
		StoreProductUnitQuantity int `gorm:"column:sp_unit_quantity"`
	}

	var cartItemsWithProducts []CartItemWithProduct
	err := tx.WithContext(ctx).
		Table("cart_items ci").
		Select("ci.*, sp.unit_quantity as sp_unit_quantity").
		Joins("JOIN store_products sp ON sp.id = ci.store_product_id").
		Where("ci.sale_id = ?", sale.Id).
		Find(&cartItemsWithProducts).Error

	if err != nil {
		s.log.Errorf("could not get cart_items with products: %v", err)
		return domain.InternalServerError
	}

	for _, item := range cartItemsWithProducts {
		if item.UnitQuantity > item.StoreProductUnitQuantity {
			s.log.Warnf("Product not enough cart_item(%s) -> %d, store_product(%s) -> %d", item.Id, item.UnitQuantity, item.StoreProductId, item.StoreProductUnitQuantity)
			return domain.NotEnoughProductError
		}
	}

	query := `
	UPDATE store_products sp
	SET unit_quantity = sp.unit_quantity - ci.unit_quantity
	FROM cart_items ci
	WHERE sp.id = ci.store_product_id 
	AND ci.sale_id = ?
	AND sp.unit_quantity >= ci.unit_quantity
	`

	qb := tx.WithContext(ctx).Exec(query, sale.Id)

	if qb.Error != nil {
		s.log.Errorf("could not update store_product unit_quantity: %v", qb.Error)
		return domain.InternalServerError
	}

	if qb.RowsAffected != int64(len(cartItemsWithProducts)) {
		return domain.NotEnoughProductError
	}

	return nil
}

func (s *Services) matchingPaymentTypeSum(ctx context.Context, req *domain.FinalSale) (*domain.FinalSale, error) {
	var sum float64
	for _, item := range req.PaymentTypes {
		sum += item.Amount - item.ReturnAmount
		if item.Type == constants.PaymentTypeCash {
			req.Cash = item.Amount - item.ReturnAmount
			req.ReturnAmount = item.ReturnAmount
		} else if item.Type == constants.PaymentTypeCard && item.AppType == constants.PaymentTypeHumo {
			req.Humo = item.Amount
		} else if item.Type == constants.PaymentTypeCard && item.AppType == constants.PaymentTypeUzcard {
			req.Uzcard = item.Amount
		} else if item.Type == constants.PaymentTypeApp && item.AppType == constants.PaymentTypeClick {
			req.Click = item.Amount
			req.OtpCode = item.OtpData
		} else if item.Type == constants.PaymentTypeApp && item.AppType == constants.PaymentTypePayme {
			req.Payme = item.Amount
			req.OtpCode = item.OtpData
		} else if item.Type == constants.PaymentTypeApp && item.AppType == constants.PaymentTypeAlif {
			req.Alif = item.Amount
			req.OtpCode = item.OtpData
		} else {
			return req, domain.InvalidPaymentTypeError
		}
	}
	// get cart item sum
	cartItemSum, err := s.cartItemsSumBySaleId(ctx, req.SaleID)
	if err != nil {
		return req, err
	}
	if sum != cartItemSum || req.TotalAmount != cartItemSum || req.TotalAmount != sum {
		s.log.Warn("cartItemSum: %v, paymentTypeSum: %v, req.TotalAmount: %v", cartItemSum, sum, req.TotalAmount)
		return req, domain.InvalidSaleAmount
	}

	return req, nil
}

func (s *Services) updateFinalizeSale(ctx context.Context, tx *gorm.DB, req *domain.FinalSale, stage int) (*domain.Sale, error) {
	var res domain.Sale

	query := `
	UPDATE sales
		SET
			total_amount = (SELECT SUM(total_price)-SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
			total_discount = (SELECT SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
			status = ?,
			stage = ?,
			cash = ?,
			humo = ?,
			uzcard = ?,
			click = ?,
			payme = ?,
			alif = ?,
			tax_free = ?,
			otp_code = ?,
			service_type = ?,
			updated_at = NOW()
	WHERE id = ?;
	`
	err := tx.WithContext(ctx).Raw(query,
		req.SaleID,
		req.SaleID,
		constants.GeneralStatusCompleted,
		stage,
		req.Cash,
		req.Humo,
		req.Uzcard,
		req.Click,
		req.Payme,
		req.Alif,
		req.TaxFree,
		req.OtpCode,
		req.ServiceType,
		req.SaleID,
	).Scan(&res).Error
	if err != nil {
		s.log.Error("could not complete sale(%s) error: %v", req.SaleID, err)
		return &res, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) SetFiscalId(ctx context.Context, tx *gorm.DB, saleId string, fiscalId string) error {
	err := tx.WithContext(ctx).Exec(`UPDATE sales SET fiscal_sign = ?, updated_at = NOW() WHERE id = ?`, fiscalId, saleId).Error
	if err != nil {
		s.log.Errorf("could not set fiscal_id: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) AttachDiscountCardToSale(ctx context.Context, req *domain.AddDiscountCard) (*domain.SaleCustomerDiscount, error) {
	// get discount card info by barcode
	discountCard, err := s.GetDiscountCardByBarcode(ctx, req.Barcode)
	if err != nil {
		return nil, err
	}

	// start transcation
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// create new customer_discounts
	customerDiscount, err := s.CreateSaleCustomerDiscount(ctx, tx, req, discountCard)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// update cart_items discount amount with total_price
	err = s.updateCartItemDiscountValue(ctx, tx, discountCard.Percent, req.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// set customer_id to sale
	_, err = s.updateSaleField(ctx, tx, "customer_id", req.CustomerId, req.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// commit transcation
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit attach discount transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return customerDiscount, nil

}

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
		constants.SaleOnlineStagePending,
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
	err = s.DoRequest(&response, "PATCH", url, requestData, headers)
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
			_ = tx.Rollback()
		}
	}()

	var sale *domain.Sale
	err = tx.First(&sale, "id = ?", req.SaleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = tx.Rollback()
			return errors.New("sale.not.found")
		}
		_ = tx.Rollback()
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
		constants.SaleOnlineStageCanceled,
		req.SaleID).Error

	if err != nil {
		_ = tx.Rollback()
		s.log.Warn("ERROR on getting online sale count: %v", err)
		return err
	}
	// Prepare Headers
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.cfg.NoorApiToken),
		"Content-Type":  "application/json",
	}
	requestData, err := json.Marshal(gin.H{"order_id": sale.SaleNumber})
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not parse online sale response: %v", err)
	}
	var response *http.Response
	url := s.cfg.NoorApiUrl + fmt.Sprintf("/orders/vendor/%d/cancel", sale.SaleNumber)
	err = s.DoRequest(&response, "PATCH", url, requestData, headers)
	if err != nil {
		_ = tx.Rollback()
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
	err := tx.WithContext(ctx).Exec(query, constants.GeneralStatusPending, sale.Id).Error
	if err != nil {
		s.log.Warn("ERROR on update sale to returned: %v", err)
		return err
	}
	return nil
}

func (s *Services) updateSaleFields(ctx context.Context, tx *gorm.DB, saleId string, updates map[string]any) error {
	err := tx.WithContext(ctx).Model(&domain.Sale{}).Where("id = ?", saleId).Updates(&updates).Error
	if err != nil {
		s.log.Errorf("could not update sale fields: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) updateSaleField(ctx context.Context, tx *gorm.DB, field string, value any, saleId string) (*domain.Sale, error) {
	var sale domain.Sale

	query := fmt.Sprintf("UPDATE sales SET %s = ? WHERE id = ? RETURNING *", field)
	err := tx.WithContext(ctx).
		Raw(query, value, saleId).
		Scan(&sale).Error
	if err != nil {
		s.log.Errorf("could not update sale field %s: %v", field, err)
		return nil, domain.InternalServerError
	}
	return &sale, nil
}

func (s *Services) UpdateSalePaymentType(ctx context.Context, req *domain.ChangePaymentTypeRequest) error {

	// Should write update payment type logic
	// ...

	return nil
}

// region Get

func (s *Services) GetSaleOne(ctx context.Context, saleId string) (*domain.SaleResponse, error) {
	var tempSale struct {
		Id                 string     `gorm:"id"`
		DisplayId          int        `gorm:"display_id"`
		ParentId           string     `gorm:"parent_id"`
		EmployeeId         string     `gorm:"employee_id"`
		CashBoxOperationId string     `gorm:"cash_box_operation_id"`
		CustomerId         string     `gorm:"customer_id"`
		SaleNumber         int        `gorm:"sale_number"`
		TotalDiscount      float64    `gorm:"total_discount"`
		TotalAmount        float64    `gorm:"total_amount"`
		VatSum             float64    `gorm:"vat_sum"`
		ReturnedAmount     float64    `gorm:"returned_amount"`
		Status             string     `gorm:"status"`
		Stage              int        `gorm:"stage"`
		OnlineStatus       int        `gorm:"online_status"`
		Type               string     `gorm:"type"`
		SaleType           string     `gorm:"sale_type"`
		Cash               float64    `gorm:"cash"`
		Uzcard             float64    `gorm:"uzcard"`
		Humo               float64    `gorm:"humo"`
		Click              float64    `gorm:"click"`
		Payme              float64    `gorm:"payme"`
		Alif               float64    `gorm:"alif"`
		IsDelivered        bool       `gorm:"is_delivered"`
		TaxFree            bool       `gorm:"tax_free"`
		FiscalSign         string     `gorm:"fiscal_sign"`
		CheckUrl           string     `gorm:"check_url"`
		OtpCode            string     `gorm:"otp_code"`
		IsSentToTax        string     `gorm:"is_sent_to_tax"`
		CreatedAt          *time.Time `gorm:"created_at"`
		UpdatedAt          *time.Time `gorm:"updated_at"`
		CompletedAt        *time.Time `gorm:"completed_at"`

		StoreName   string `gorm:"store_name"`
		CashBoxName string `gorm:"cash_box_name"`

		EmFullName  string `gorm:"em_full_name"`
		EmFirstName string `gorm:"em_first_name"`
		EmLastname  string `gorm:"em_last_name"`
		EmPhone     string `gorm:"em_phone"`

		CFullName  string `gorm:"c_full_name"`
		CFirstName string `gorm:"c_first_name"`
		CLastName  string `gorm:"c_last_name"`
		CPhone     string `gorm:"c_phone"`
	}

	// get sale info
	err := s.db.
		Select(
			"s.id",
			"s.display_id",
			"s.parent_id",
			"s.employee_id",
			"s.cash_box_operation_id",
			"s.store_id",
			"s.customer_id",
			"s.cashbox_id",
			"s.sale_number",
			"s.total_amount",
			"s.total_discount",
			"s.returned_amount",
			"s.cash",
			"s.humo",
			"s.uzcard",
			"s.payme",
			"s.click",
			"s.alif",
			"s.status",
			"s.stage",
			"s.online_status",
			"s.sale_type",
			"s.type",
			"s.fiscal_sign",
			"s.check_url",
			"s.is_sent_to_tax",
			"s.tax_free",
			"s.otp_code",
			"s.created_at",
			"s.updated_at",
			"s.completed_at",

			"st.name AS store_name",
			"ca.name AS cash_box_name",

			"em.first_name AS em_first_name",
			"em.last_name AS em_last_name",
			"em.full_name AS em_full_name",
			"em.phone AS em_phone",

			"c.first_name AS c_first_name",
			"c.last_name AS c_last_name",
			"c.full_name AS c_full_name",
			"c.phone AS c_phone",
		).
		Table("sales s").
		Joins("LEFT JOIN stores st ON s.store_id = st.id").
		Joins("LEFT JOIN cash_boxes ca ON s.cashbox_id = ca.id").
		Joins("LEFT JOIN employees em ON s.employee_id = em.id").
		Joins("LEFT JOIN customers c ON s.customer_id = c.id").
		Take(&tempSale, "s.id = ?", saleId).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get sale: %v", err)
		return nil, domain.InternalServerError
	}

	res := domain.SaleResponse{
		Id:                 tempSale.Id,
		DisplayId:          tempSale.DisplayId,
		ParentId:           tempSale.ParentId,
		EmployeeId:         tempSale.EmployeeId,
		CustomerId:         tempSale.CustomerId,
		CashBoxOperationId: tempSale.CashBoxOperationId,
		SaleNumber:         tempSale.SaleNumber,
		TotalAmount:        tempSale.TotalAmount,
		TotalDiscount:      tempSale.TotalDiscount,
		Cash:               tempSale.Cash,
		Humo:               tempSale.Humo,
		Uzcard:             tempSale.Uzcard,
		Click:              tempSale.Click,
		Payme:              tempSale.Payme,
		Alif:               tempSale.Alif,
		Status:             tempSale.Status,
		Stage:              tempSale.Stage,
		OnlineStatus:       tempSale.OnlineStatus,
		SaleType:           tempSale.SaleType,
		Type:               tempSale.Type,
		FiscalSign:         tempSale.FiscalSign,
		CheckUrl:           tempSale.CheckUrl,
		IsSentToTax:        tempSale.IsSentToTax,
		TaxFree:            tempSale.TaxFree,
		OtpCode:            tempSale.OtpCode,
		CreatedAt:          tempSale.CreatedAt,
		UpdatedAt:          tempSale.UpdatedAt,
		CompletedAt:        tempSale.CompletedAt,

		StoreName:   tempSale.StoreName,
		CashBoxName: tempSale.CashBoxName,

		Customer: &domain.CustomerForSale{
			Id:        tempSale.CustomerId,
			FirstName: tempSale.CFirstName,
			LastName:  tempSale.CLastName,
			FullName:  tempSale.CFullName,
			Phone:     tempSale.CPhone,
		},

		Employee: &domain.EmployeeForSale{
			Id:        tempSale.EmployeeId,
			FirstName: tempSale.EmFirstName,
			LastName:  tempSale.EmLastname,
			FullName:  tempSale.EmFullName,
			Phone:     tempSale.EmPhone,
		},
	}

	res.Product, err = s.GetSoldProductsBySaleId(ctx, saleId)
	if err != nil {
		return nil, err
	}

	res.VatSum, err = s.getSaleVatSum(ctx, saleId)
	if err != nil {
		return nil, err
	}

	if res.ParentId != "" {
		// get epos response
		err = s.db.Raw(`SELECT * FROM epos_responses WHERE sale_id = ? AND status = 1`, res.ParentId).Scan(&res.EposResponse).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				s.log.Errorf("could not get epos_response by sale: %v", err)
			}
		}
	}

	return &res, nil
}

func (s *Services) GetSaleById(ctx context.Context, saleId string) (*domain.Sale, error) {
	var sale domain.Sale
	err := s.db.WithContext(ctx).First(&sale, "id = ?", saleId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &sale, domain.NotFoundError
		}
		s.log.Error("could not get sale(%s) info: %v", saleId, err)
		return &sale, domain.InternalServerError
	}

	return &sale, nil
}

func (s *Services) GetSales(ctx context.Context, params *domain.SaleQueryParams, user *domain.EmployeeClaims) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
	var res []domain.SaleResponse

	if utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// query builder
	qb := s.db.
		WithContext(ctx).
		Table("sales s").
		Joins("LEFT JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN employees em ON em.id = s.employee_id").
		Joins("LEFT JOIN cash_boxes ON s.cashbox_id = cash_boxes.id").
		Joins("LEFT JOIN customers ON s.customer_id = customers.id")

	// filters
	qb = qb.Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.Cash {
		qb = qb.Where("s.cash > 0")
	}
	if params.Humo {
		qb = qb.Where("s.humo > 0")
	}
	if params.Uzcard {
		qb = qb.Where("s.uzcard > 0")
	}
	if params.Click {
		qb = qb.Where("s.click > 0")
	}
	if params.Payme {
		qb = qb.Where("s.payme > 0")
	}
	if params.Alif {
		qb = qb.Where("s.alif > 0")
	}
	if params.VendorId != "" {
		qb = qb.Where("s.employee_id = ?", params.VendorId)
	}
	if params.StoreId != "" {
		qb = qb.Where("s.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.CashboxId != "" {
		qb = qb.Where("s.cashbox_id = ?", params.CashboxId)
	}

	if params.StartDate != "" && params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}

	if params.StartDate != "" && params.EndDate == "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND (?::timestamp + interval '24 hours')", params.StartDate, params.StartDate)
	}

	if params.Search != "" {
		if num, err := strconv.Atoi(params.Search); err == nil {
			// If will be digit
			qb = qb.Where("s.sale_number = ?", num)
		} else {
			// otherwise text
			qb = qb.Where("st.name ILIKE ?", "%"+params.Search+"%")
		}
	}
	if params.TotalAmountFrom > 0 {
		qb = qb.Where("s.total_amount >= ?", params.TotalAmountFrom)
	}
	if params.TotalAmountTo > 0 {
		qb = qb.Where("s.total_amount <= ?", params.TotalAmountTo)
	}
	if params.SaleType != "" {
		qb = qb.Where("s.sale_type = ?", params.SaleType)
	}

	// 1) get total count without (LIMIT/OFFSET)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// 2) get data with (LIMIT/OFFSET bilan)
	err := qb.
		Select(
			"s.id",
			"s.sale_number",
			"s.sale_type",
			"s.type",
			"s.total_amount",
			"s.return_amount",
			"s.total_discount",
			"s.cash",
			"s.uzcard",
			"s.humo",
			"s.click",
			"s.payme",
			"s.alif",
			"s.status",
			"s.check_url",
			"s.fiscal_sign",
			"s.is_sent_to_tax",
			"s.is_paid",
			"s.created_at",
			"s.completed_at",
			"em.full_name",
			"em.phone",
			"st.name AS store_name",
			"customers.full_name as customer_name",
			"customers.phone AS customer_phone",
			"cash_boxes.name AS cash_box_name",
		).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("s.completed_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetSalesStats(ctx context.Context, params *domain.SaleQueryParams, user *domain.EmployeeClaims) (*domain.SaleStats, error) {
	// check user role
	if utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// query builder
	qb := s.db.
		Select(
			"SUM(s.total_amount) AS total_transactions_sum",
			"SUM(CASE WHEN s.sale_type = 'RETURN' THEN s.total_amount ELSE 0 END) AS total_returnals_sum",
			"SUM(s.total_discount) AS total_discount_amount",
			"SUM(s.cash) AS total_cash",
			"SUM(s.humo) AS total_humo",
			"SUM(s.uzcard) AS total_uzcard",
			"SUM(s.click) AS total_click",
			"SUM(s.payme) AS total_payme",
			"SUM(s.alif) AS total_alif",
			"COUNT(*) AS total_count",
		).Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id")

	if params.Cash {
		qb = qb.Where("s.cash > 0")
	}
	if params.Humo {
		qb = qb.Where("s.humo > 0")
	}
	if params.Uzcard {
		qb = qb.Where("s.uzcard > 0")
	}
	if params.Click {
		qb = qb.Where("s.click > 0")
	}
	if params.Payme {
		qb = qb.Where("s.payme > 0")
	}
	if params.Alif {
		qb = qb.Where("s.alif > 0")
	}
	if params.VendorId != "" {
		qb = qb.Where("s.employee_id = ?", params.VendorId)
	}
	if params.StoreId != "" {
		qb = qb.Where("s.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.CashboxId != "" {
		qb = qb.Where("s.cashbox_id = ?", params.CashboxId)
	}

	if params.StartDate != "" && params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}

	if params.StartDate != "" && params.EndDate == "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND (?::timestamp + interval '24 hours')", params.StartDate, params.StartDate)
	}

	if params.Search != "" {
		if num, err := strconv.Atoi(params.Search); err == nil {
			// If will be digit
			qb = qb.Where("s.sale_number = ?", num)
		} else {
			// otherwise text
			qb = qb.Where("st.name ILIKE ?", "%"+params.Search+"%")
		}
	}
	if params.TotalAmountFrom > 0 {
		qb = qb.Where("s.total_amount >= ?", params.TotalAmountFrom)
	}
	if params.TotalAmountTo > 0 {
		qb = qb.Where("s.total_amount <= ?", params.TotalAmountTo)
	}
	if params.SaleType != "" {
		qb = qb.Where("s.sale_type = ?", params.SaleType)
	}

	var res domain.SaleStats
	err := qb.WithContext(ctx).Take(&res).Debug().Error
	if err != nil {
		s.log.Errorf("could not get sale_stats: %v", err)
		return nil, domain.InternalServerError
	}

	if res.PaymentTypeStats == nil {
		res.PaymentTypeStats = []domain.PaymentTypeStats{}
	}

	return &res, nil
}

func (s *Services) GetSaleList(ctx context.Context, params *domain.SaleQueryParams, user *domain.EmployeeClaims) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
	var res []domain.SaleResponse

	if utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// query builder
	qb := s.db.
		WithContext(ctx).
		Table("sales s").
		Joins("LEFT JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN employees em ON em.id = s.employee_id").
		Joins("LEFT JOIN cash_boxes ON s.cashbox_id = cash_boxes.id").
		Joins("LEFT JOIN customers ON s.customer_id = customers.id")

	// filters
	qb = qb.Where("s.status = ?", constants.GeneralStatusCompleted)

	if params.Cash {
		qb = qb.Where("s.cash > 0")
	}
	if params.Humo {
		qb = qb.Where("s.humo > 0")
	}
	if params.Uzcard {
		qb = qb.Where("s.uzcard > 0")
	}
	if params.Click {
		qb = qb.Where("s.click > 0")
	}
	if params.Payme {
		qb = qb.Where("s.payme > 0")
	}
	if params.Alif {
		qb = qb.Where("s.alif > 0")
	}
	if params.VendorId != "" {
		qb = qb.Where("s.employee_id = ?", params.VendorId)
	}
	if params.StoreId != "" {
		qb = qb.Where("s.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.CashboxId != "" {
		qb = qb.Where("s.cashbox_id = ?", params.CashboxId)
	}

	if params.StartDate != "" && params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}

	if params.StartDate != "" && params.EndDate == "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND (?::timestamp + interval '24 hours')", params.StartDate, params.StartDate)
	}

	if params.Search != "" {
		if num, err := strconv.Atoi(params.Search); err == nil {
			// If will be digit
			qb = qb.Where("s.sale_number = ?", num)
		} else {
			// otherwise text
			qb = qb.Where("st.name ILIKE ?", "%"+params.Search+"%")
		}
	}
	if params.TotalAmountFrom > 0 {
		qb = qb.Where("s.total_amount >= ?", params.TotalAmountFrom)
	}
	if params.TotalAmountTo > 0 {
		qb = qb.Where("s.total_amount <= ?", params.TotalAmountTo)
	}
	if params.SaleType != "" {
		qb = qb.Where("s.sale_type = ?", params.SaleType)
	}

	// 1) get total count without (LIMIT/OFFSET)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// 2) get data with (LIMIT/OFFSET bilan)
	err := qb.
		Select(
			"s.id",
			"s.sale_number",
			"s.sale_type",
			"s.type",
			"s.total_amount",
			"s.return_amount",
			"s.total_discount",
			"s.cash",
			"s.uzcard",
			"s.humo",
			"s.click",
			"s.payme",
			"s.alif",
			"s.status",
			"s.check_url",
			"s.fiscal_sign",
			"s.is_sent_to_tax",
			"s.is_paid",
			"s.created_at",
			"s.completed_at",
			"em.full_name",
			"em.phone",
			"st.name AS store_name",
			"customers.full_name as customer_name",
			"customers.phone AS customer_phone",
			"cash_boxes.name AS cash_box_name",
		).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("s.completed_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// Get Payment service with store id and payment type  if status is active
func (s *Services) GetPaymentServiceByStoreId(ctx context.Context, tx *gorm.DB, storeId, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := tx.WithContext(ctx).
		Where("store_id = ?", storeId).
		Where("type = ? AND is_active = true", paymentType).
		First(&res).Error
	if err != nil {
		s.log.Error("could not get payment_service by store(%s) error: %v", storeId, err)
		return &res, domain.InternalServerError
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
		return sum, domain.InternalServerError
	}
	return sum, nil
}

// get online pending sale list
func (s *Services) GetOnlinePendingSaleList(ctx context.Context, params *domain.QueryParam) ([]domain.Sale, int64, error) {
	var (
		res        []domain.Sale
		filter     = " WHERE s.store_id = ? AND (s.online_status = 1 OR s.online_status = 2) "
		args       = []any{params.StoreID}
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

	if params.StartDate != "" {
		filter += " AND (s.created_at + interval '5 hours')::date >= ? "
		args = append(args, params.StartDate)
	}

	if params.EndDate != "" {
		filter += " AND (s.created_at + interval '5 hours')::date <= ? "
		args = append(args, params.EndDate)
	}

	if params.Search != "" {
		filter += " AND CAST(s.sale_number AS TEXT) LIKE ? "
		args = append(args, "%"+params.Search+"%")
	}
	// collect and execute totalCount query
	totalCountQuery += filter + group
	err := s.db.WithContext(ctx).Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get online sale count: %v", err)
		return res, totalCount, domain.InternalServerError
	}

	// collect and execute query
	query += filter + group + order + " LIMIT ? OFFSET ?;"
	args = append(args, params.Limit, params.Offset)
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get online sale list: %v", err)
		return res, totalCount, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetPendingSales(ctx context.Context, params *domain.SaleQueryParams, user *domain.EmployeeClaims) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
	var res []domain.SaleResponse

	if utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// query builder
	qb := s.db.
		WithContext(ctx).
		Table("sales s").
		Joins("LEFT JOIN stores st ON st.id = s.store_id").
		Joins("LEFT JOIN employees em ON em.id = s.employee_id").
		Joins("LEFT JOIN cash_boxes ON s.cashbox_id = cash_boxes.id").
		Joins("LEFT JOIN customers ON s.customer_id = customers.id")

	// filters
	qb = qb.Where("s.status = ?", constants.GeneralStatusPending)

	if params.Cash {
		qb = qb.Where("s.cash > 0")
	}
	if params.Humo {
		qb = qb.Where("s.humo > 0")
	}
	if params.Uzcard {
		qb = qb.Where("s.uzcard > 0")
	}
	if params.Click {
		qb = qb.Where("s.click > 0")
	}
	if params.Payme {
		qb = qb.Where("s.payme > 0")
	}
	if params.Alif {
		qb = qb.Where("s.alif > 0")
	}
	if params.VendorId != "" {
		qb = qb.Where("s.employee_id = ?", params.VendorId)
	}
	if params.StoreId != "" {
		qb = qb.Where("s.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.CashboxId != "" {
		qb = qb.Where("s.cashbox_id = ?", params.CashboxId)
	}

	if params.StartDate != "" && params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}

	if params.StartDate != "" && params.EndDate == "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND (?::timestamp + interval '24 hours')", params.StartDate, params.StartDate)
	}

	if params.Search != "" {
		if num, err := strconv.Atoi(params.Search); err == nil {
			// If will be digit
			qb = qb.Where("s.sale_number = ?", num)
		} else {
			// otherwise text
			qb = qb.Where("st.name ILIKE ?", "%"+params.Search+"%")
		}
	}
	if params.TotalAmountFrom > 0 {
		qb = qb.Where("s.total_amount >= ?", params.TotalAmountFrom)
	}
	if params.TotalAmountTo > 0 {
		qb = qb.Where("s.total_amount <= ?", params.TotalAmountTo)
	}
	if params.SaleType != "" {
		qb = qb.Where("s.sale_type = ?", params.SaleType)
	}

	// 1) get total count without (LIMIT/OFFSET)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// 2) get data with (LIMIT/OFFSET bilan)
	err := qb.
		Select(
			"s.id",
			"s.sale_number",
			"s.sale_type",
			"s.type",
			"s.total_amount",
			"s.return_amount",
			"s.total_discount",
			"s.cash",
			"s.uzcard",
			"s.humo",
			"s.click",
			"s.payme",
			"s.alif",
			"s.status",
			"s.check_url",
			"s.fiscal_sign",
			"s.is_sent_to_tax",
			"s.is_paid",
			"s.created_at",
			"s.completed_at",
			"em.full_name",
			"em.phone",
			"st.name AS store_name",
			"customers.full_name as customer_name",
			"customers.phone AS customer_phone",
			"cash_boxes.name AS cash_box_name",
		).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("s.completed_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
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

func (s *Services) getSaleVatSum(ctx context.Context, saleId string) (float64, error) {
	// get vat sum
	var vatSum float64
	err := s.db.
		WithContext(ctx).
		Raw(`
			SELECT
				COALESCE(SUM(ROUND((sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2)), 0) AS vat_sum
			FROM 
				cart_items ci
			JOIN 
				store_products sp ON sp.id = ci.store_product_id
			JOIN 
				products p ON sp.product_id = p.id
			WHERE  
				sale_id = ?;
	`, saleId).Scan(&vatSum).Error
	if err != nil {
		s.log.Errorf("could not get sale: %v", err)
		return 0, err
	}
	return vatSum, nil
}

func (s *Services) GetDatasByMarkings(ctx context.Context, tx *gorm.DB, markings []domain.MarkingData) (*domain.MarkingItemsResponse, error) {
	var (
		err   error
		items []domain.BarcodeResponse
	)

	for _, m := range markings {
		// 1. cart_items ni olib kelamiz
		var cartItem domain.CartItem
		err = tx.Table("cart_items").
			Select("id, store_product_id, quantity, unit_quantity").
			Where("id = ?", m.Id).
			Scan(&cartItem).Error
		if err != nil {
			return nil, err
		}

		// 2. Agar MarkingList bo‘lsa → markinglardan barcode chiqaramiz
		if len(m.MarkingList) > 0 {
			for _, mark := range m.MarkingList {
				if len(mark) >= 16 {
					barcode := mark[3:16]

					var br domain.BarcodeResponse
					err = tx.Table("product_barcodes pb").
						Select("pb.id, pb.barcode, pb.mxik, pb.unit_code").
						Where("pb.barcode = ? AND pb.status = 'completed' AND pb.mxik is not null AND pb.unit_code is not null ", barcode).
						Order("pb.created_at desc").
						Limit(1).
						Scan(&br).Error
					if err != nil {
						return nil, err
					}
					br.CartItemId = m.Id
					items = append(items, br)
				}
			}
		} else {
			// 3. Agar MarkingList bo‘lmasa → store_product_id orqali barcode topamiz
			var productID string
			err = tx.Table("store_products").
				Select("product_id").
				Where("id = ?", cartItem.StoreProductId).
				Scan(&productID).Error
			if err != nil {
				return nil, err
			}

			var br domain.BarcodeResponse
			err = tx.Table("product_barcodes pb").
				Select("pb.id, pb.barcode, pb.mxik, pb.unit_code").
				Where("pb.product_id = ? AND pb.status = 'completed' AND pb.mxik is not null AND pb.unit_code is not null ", productID).
				Order("pb.created_at desc").
				Limit(1).
				Scan(&br).Error
			if err != nil {
				return nil, err
			}
			br.CartItemId = m.Id

			// quantity + unit_quantity hisoblash
			total := cartItem.Quantity
			if cartItem.UnitQuantity > 0 {
				total += 1
			}

			for i := 0; i < total; i++ {
				items = append(items, br)
			}
		}
	}

	return &domain.MarkingItemsResponse{Items: items}, nil
}

// region Delete

func (s *Services) DeleteSalePayments(ctx context.Context, tx *gorm.DB, saleId string) error {
	err := tx.WithContext(ctx).Exec(`DELETE FROM sale_payments WHERE sale_id = ?`, saleId).Error
	if err != nil {
		tx.Rollback()
		s.log.Error("could not delete sale_payments: %v", err)
		return err
	}
	return nil

}

func (s *Services) DeleteDiscountCardFromSale(ctx context.Context, req *domain.AddDiscountCard) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// delete sale_customer_discount
	err := s.DeleteSaleCustomerDiscount(ctx, tx, req)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// update sale customer_id to null
	_, err = s.updateSaleField(ctx, tx, "customer_id", nil, req.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	// update discount_type and value to 0
	err = s.updateCartItemDiscountValue(ctx, tx, 0, req.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
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
				return domain.SerialOrMarkingRequiredError
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

func (s *Services) DecodeFiscalData(reqJsonStr string) (domain.FiscalData, error) {
	var fiscal domain.FiscalData
	// parse epos success json to structure
	var successResp domain.EposSuccessResponse
	if err := json.Unmarshal([]byte(reqJsonStr), &successResp); err != nil {
		s.log.Error("could not parse epos success response: %v", err)
		return fiscal, domain.BadRequestError
	}
	if successResp.Info.FiscalSign != "" {
		fiscal = domain.FiscalData{
			StatusCode: 0,
			Message:    "accepted",
			TerminalId: successResp.Info.TerminalId,
			ReceiptId:  cast.ToInt(successResp.Info.ReceiptSeq),
			Date:       successResp.Info.DateTime,
			FiscalSign: successResp.Info.FiscalSign,
			QrCodeUrl:  successResp.Info.QrCodeURL,
		}
	} else {
		fiscal = domain.FiscalData{
			StatusCode: 0,
			Message:    "accepted",
			TerminalId: successResp.Message.TerminalId,
			ReceiptId:  cast.ToInt(successResp.Message.ReceiptSeq),
			Date:       successResp.Message.DateTime,
			FiscalSign: successResp.Message.FiscalSign,
			QrCodeUrl:  successResp.Message.QrCodeURL,
		}
	}

	return fiscal, nil
}

func (s *Services) getFiscalDataBySaleId(ctx context.Context, saleId string) (domain.FiscalData, error) {
	var res domain.FiscalData

	return res, nil
}

func (s *Services) getSalePayAmounts(sale *domain.Sale, req *domain.FinalSale) *domain.Sale {
	sale.Cash = req.Cash
	sale.Humo = req.Humo
	sale.Uzcard = req.Uzcard
	sale.Click = req.Click
	sale.Payme = req.Payme
	sale.Alif = req.Alif

	return sale
}

func (s *Services) generateDisplayId() int {
	displayId := utils.GenerateRandomValue(1000, 9999)
	return displayId
}
