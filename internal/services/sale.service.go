package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"gorm.io/gorm"
)

// create new sale
func (s *Services) CreateSale(req *domain.SaleRequest) (*domain.Sale, error) {
	var res domain.Sale
	err := s.db.Raw(`INSERT INTO sales(employee_id, cash_box_operation_id, store_id, cashbox_id) VALUES(?, ?, ?, ?) RETURNING *`,
		req.EmployeeID, req.CashBoxOperationId, req.StoreId, req.CashboxId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on creating new sale: %v", err)
		return nil, err
	}
	return &res, nil
}

// create return sale
func (s *Services) CreateReturnSale(req *domain.SaleReturnRequest) (*domain.Sale, error) {
	var (
		sale             domain.Sale
		cashboxOperation domain.CashboxOperation
	)
	// get cashbox operation
	err := s.db.First(&cashboxOperation, "id = ?", req.CashBoxOperationId).Error
	if err != nil {
		s.log.Error("ERROR on getting cashbox_operation: ", err)
		return nil, err
	}
	req.CashboxId = cashboxOperation.CashBoxID
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// build query
	query := `
	INSERT INTO sales (
		employee_id, cash_box_operation_id, cashbox_id, store_id, customer_id, sale_number, parent_id, sale_type, type)
	SELECT ?, ?, ?, store_id, customer_id, sale_number, id, ?, type FROM sales where id = ?
	RETURNING *;`
	err = tx.Raw(query, req.EmployeeID, req.CashBoxOperationId, req.CashboxId, config.SALE_TYPE_RETURN, req.SaleId).Scan(&sale).Error
	if err != nil {
		s.log.Error("ERROR on creating return sale: ", err)
		tx.Rollback()
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
			tx.Rollback()
			return nil, err
		}
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error("ERROR on commiting transaction: ", err)
		tx.Rollback()
		return nil, err
	}
	return &sale, nil
}

// create new sale or get pending sale
func (s *Services) CreateOrGetSale(req *domain.SaleRequest) (*domain.Sale, error) {
	var res *domain.Sale

	// getting pending sale with no cart items
	err := s.db.
		Raw(`
			SELECT * FROM sales 
			WHERE store_id = ? AND employee_id = ? AND cash_box_operation_id = ? AND cashbox_id = ?
			AND status = ?  AND sale_type = ?
			AND NOT EXISTS (
				SELECT 1 FROM cart_items WHERE cart_items.sale_id = sales.id
			)
			LIMIT 1
		`, req.StoreId, req.EmployeeID, req.CashBoxOperationId, req.CashboxId, config.PENDING, config.SALE_TYPE_SALE).
		Scan(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) || res == nil || res.ID == "" {
		res, err = s.CreateSale(req)
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
func (s *Services) CreateOrGetSalePending(req *domain.SaleRequest) (*domain.Sale, error) {
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
		res, err = s.CreateSale(req)
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

// update sale with receiving field
func (s *Services) UpdateSaleField(field string, value string, idField string, idValue string) (*domain.Sale, error) {
	var res domain.Sale
	err := s.db.Raw(`UPDATE sales SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Create sale payment
func (s *Services) CreateSalePayment(tx *gorm.DB, req domain.FinalSale, item domain.FinalPaymentType, paymentServiceId *string) (*domain.SalePayment, error) {
	var (
		now         = time.Now()
		salePayment = domain.SalePayment{}
	)
	query := `
	INSERT INTO sale_payments(
		sale_id, cash_box_operation_id, 
		payment_service_id, payment_type_id, 
		amount, return_amount, paid_at) 
	VALUES(?, ?, ?, ?, ?, ?, ?) RETURNING *`
	// Insert sale payments
	err := tx.Raw(query,
		req.SaleID, req.CashBoxOperationId,
		paymentServiceId, item.PaymentTypeID, item.Amount-item.ReturnAmount, item.ReturnAmount, now).
		Scan(&salePayment).Error
	if err != nil {
		s.log.Error("ERROR on creating new sale payment: ", err)
		return nil, err
	}
	return &salePayment, nil
}

// Get Payment service with store id and payment type  if status is active
func (s *Services) GetPaymentServiceByStoreId(storeId string, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := s.db.Raw(`SELECT * FROM payment_services WHERE store_id = ? AND type = ? AND is_active = true`,
		storeId, paymentType).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// update sale payment status
func (s *Services) UpdateSalePaymentStatus(tx *gorm.DB, salePaymentID string) error {
	err := tx.Exec(`UPDATE sale_payments SET status = 'paid' WHERE id = ?`, salePaymentID).Error
	if err != nil {
		s.log.Error("ERROR on updating sale payment status: ", err)
		return err
	}
	return nil
}

// Create or update sale payment summary with on conflict do update
func (s *Services) CreateOrUpdateSalePaymentSummary(tx *gorm.DB, cashBoxOperationId string, paymentTypeId string, amount float64) error {
	err := tx.Exec(`
				INSERT INTO sale_payment_summary (
					cash_box_operation_id, payment_type_id, total_amount
					) 
				VALUES (?, ?, ?)
				ON CONFLICT (cash_box_operation_id, payment_type_id) 
				DO UPDATE SET total_amount = EXCLUDED.total_amount + ?`, cashBoxOperationId, paymentTypeId, amount, amount).Error
	if err != nil {
		s.log.Error(err)
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

// complete sale
func (s *Services) CompleteSale(tx *gorm.DB, sale *domain.Sale) error {
	var err error
	if sale.SaleType == config.SALE_TYPE_SALE {
		// reduce store_product quantities and add employee bonus
		err = s.DeductStoreProductQuantities(tx, sale)
		if err != nil {
			s.log.Warn("ERROR on reducing store_product quantity: %v", err)
			return err
		}
	} else if sale.SaleType == config.SALE_TYPE_RETURN {
		err = s.RestoreStoreProductQuantities(tx, sale)
		if err != nil {
			s.log.Warn("ERROR on restore store_product quantity: %v", err)
			return err
		}
	}
	// build query for update sale status to complete
	query := `
	UPDATE sales
	SET
		total_amount = (SELECT SUM(total_price)-SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
		total_discount = (SELECT SUM(discount_amount) FROM cart_items WHERE sale_id = ?),
		status = ?, completed_at = NOW(), updated_at = NOW()
	WHERE id = ?
	`
	// complete the query
	err = tx.Exec(query, sale.ID, sale.ID, config.COMPLETED, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on update sale to completed: %v", err)
		return err
	}
	return nil
}

// return sale to pending status and reset quantities
func (s *Services) ReturnSale(tx *gorm.DB, sale *domain.Sale) error {
	err := s.RestoreStoreProductQuantities(tx, sale)
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
	WHERE id = ?
	`
	// complete the query
	err = tx.Exec(query, config.PENDING, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on update sale to returned: %v", err)
		return err
	}

	return nil
}

// Update cart item status and reduce store product quantities and add employee bonus after completed the sale
func (s *Services) DeductStoreProductQuantities(tx *gorm.DB, sale *domain.Sale) error {
	var cartItems []domain.CartItem
	err := tx.Raw(`
		SELECT 
			ci.id, ci.store_product_id, sp.product_id,
			ci.quantity, ci.unit_quantity, unit_price,
			total_price, ci.status, pb.bonus_amount,
			p.unit_per_pack
		FROM cart_items ci
		JOIN store_products sp ON sp.id = ci.store_product_id
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN product_bonuses pb ON pb.product_id = sp.product_id
		WHERE sale_id = ?`, sale.ID).
		Scan(&cartItems).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	var bonusAmount float64
	for _, item := range cartItems {
		err = tx.Exec(`
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
		if item.BonusAmount > 0 {
			bonusAmount += item.BonusAmount * float64(item.Quantity)
			if item.UnitPerPack > 0 && item.UnitQuantity > 0 {
				bonusAmount += item.BonusAmount / float64(item.UnitPerPack) * float64(item.UnitQuantity)
			}
			// add employee bonus service
			err = s.AddEmployeeBonus(tx, &domain.EmployeeBonusRequest{
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
func (s *Services) RestoreStoreProductQuantities(tx *gorm.DB, sale *domain.Sale) error {
	var cartItems []domain.CartItem
	// get cart items
	err := tx.Raw(`
		SELECT
			id, store_product_id,
			quantity, unit_quantity, unit_price,
			total_price, status
		FROM cart_items WHERE sale_id = ?`,
		sale.ID).Scan(&cartItems).Error
	if err != nil {
		s.log.Warn("ERROR on getting cart_items: %v", err)
		return err
	}
	// update store product quantities
	for _, item := range cartItems {
		err = tx.Exec(`
		UPDATE store_products
		SET
			pack_quantity = store_products.pack_quantity + ?,
			unit_quantity = store_products.unit_quantity + (? * products.unit_per_pack + ?), 
			updated_at = NOW()
		FROM products
		WHERE products.id = store_products.product_id AND  store_products.id = ?`,
			item.Quantity, item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			s.log.Warn("ERROR on restoring store_products quantity: %v", err)
			return err
		}
	}
	// delete employee bonus for return sale
	err = tx.Exec(`DELETE FROM employee_bonus WHERE sale_id = ?`, sale.ID).Error
	if err != nil {
		s.log.Warn("ERROR on deleting employee_bonus: %v", err)
		return err
	}
	return nil
}

// create sale for online order
func (s *Services) CreateOnlineSale(tx *gorm.DB, saleId string, totalAmount int64) error {
	err := tx.Exec(`
	INSERT INTO sales(id, total_amount, type, is_delivered, status, completed_at) VALUES(?, ?, ?, ?, ?, ?)`,
		saleId, totalAmount, "online", false, config.COMPLETED, time.Now()).Error
	if err != nil {
		return err
	}
	return nil
}

// get sale list data
func (s *Services) ListSale(param *domain.QueryParam, userId string) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
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
	}

	// build sale get list query
	var res = []domain.SaleResponse{}
	query := s.db.Model(&domain.Sale{}).Table("sales s").
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
	if param.PaymentTypeID != "" {
		query = query.Joins("JOIN sale_payments sp ON s.id = sp.sale_id").
			Where("sp.payment_type_id = ?", param.PaymentTypeID).
			Group("s.id, st.name, cash_boxes.name, em.full_name, em.phone, customers.full_name, customers.phone")
	}
	// filter by employee
	if param.VendorID != "" {
		query = query.Where("s.employee_id = ?", param.VendorID)
	}
	// filter by store id
	if param.StoreID != "" {
		query = query.Where("s.store_id = ?", param.StoreID)
	}
	// filter by cashbox id
	if param.CashBoxID != "" {
		query = query.Where("co.cash_box_id = ?", param.CashBoxID)
	}
	// filter by start date and end date
	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("(s.completed_at + interval '5 hours') BETWEEN ? AND ? ", param.StartDate, param.EndDate)
	}
	// filter by start date
	if param.StartDate != "" && param.EndDate == "" {
		query = query.Where("(s.completed_at + interval '5 hours') >= ?", param.StartDate)
	}
	// search condition
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("st.name ILIKE ? OR CAST(s.sale_number AS TEXT) LIKE ?", param.Search, param.Search)
	}

	if param.TotalAmountFrom > 0 {
		query = query.Where("s.total_amount >= ?", param.TotalAmountFrom)
	}

	if param.TotalAmountTo > 0 {
		query = query.Where("s.total_amount <= ?", param.TotalAmountTo)
	}

	if param.SaleType != "" {
		query = query.Where("s.sale_type = ?", param.SaleType)
	}

	// complete query
	err = query.Where("s.status = 'completed'").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("s.completed_at DESC").
		Debug().
		Find(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return res, totalCount, nil
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
func (s *Services) GetPaymeSalePayment(saleID string) *domain.SalePayment {
	var salePayment domain.SalePayment
	query := `
	SELECT sp.*
	FROM sale_payments sp
	JOIN payment_types pt ON pt.id = sp.payment_type_id
	WHERE sp.sale_id = ? AND pt.type = ?
	`
	err := s.db.Raw(query, saleID, config.PAYME).Scan(&salePayment).Error
	if err != nil {
		s.log.Warn("ERROR on getting sale_payments by saleID: %v", err)
		return &salePayment
	}
	return &salePayment
}

func (s *Services) SetFiscalId(saleID string, fiscalID string) error {
	err := s.db.Exec(`UPDATE sales SET fiscal_sign = ?, updated_at = NOW() WHERE id = ?`, fiscalID, saleID).Error
	if err != nil {
		s.log.Warn("ERROR on setting fiscal_id: %v", err)
		return err
	}
	return nil
}
