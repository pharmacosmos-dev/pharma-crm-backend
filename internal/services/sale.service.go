package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"gorm.io/gorm"
)

// create new sale
func (s *Services) CreateSale(tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	var res domain.Sale
	err := tx.Raw(`INSERT INTO sales(id, employee_id, cash_box_operation_id, store_id) VALUES(?, ?, ?, ?) RETURNING *`,
		req.ID, req.EmployeeID, req.CashBoxOperationId, req.StoreId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &res, nil
}

// create return sale
func (s *Services) CreateReturnSale(req *domain.SaleReturnRequest) (*domain.Sale, error) {
	var sale domain.Sale
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
		employee_id, cash_box_operation_id, store_id, customer_id, sale_number, parent_id, sale_type, type)
	SELECT ?, ?, store_id, customer_id, sale_number, id, ?, type FROM sales where id = ?
	RETURNING *;`
	err := tx.Raw(query, req.EmployeeID, req.CashBoxOperationId, req.SaleType, req.SaleId).Scan(&sale).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return nil, err
	}
	// cart item create query
	cquery := `
	INSERT INTO cart_items(sale_id, store_product_id, quantity, unit_quantity, unit_price, total_price, status)
	SELECT ?, sp.id, ?, ?, retail_price, (?*retail_price+(CASE WHEN p.unit_per_pack > 0 THEN retail_price / p.unit_per_pack ELSE 0 END) * ?)*(-1), ?
	FROM store_products sp JOIN products p ON p.id = sp.product_id WHERE sp.id = ?`
	for _, item := range req.Items {
		item.SaleId = sale.ID
		// complete cart item create query
		err = tx.Exec(cquery, item.SaleId, item.Quantity, item.UnitQuantity, item.Quantity, item.UnitQuantity, config.PENDING, item.StoreProductId).Error
		if err != nil {
			s.log.Error(err)
			tx.Rollback()
			return nil, err
		}
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error(err)
		tx.Rollback()
		return nil, err
	}
	return &sale, nil
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
func (s *Services) CreateSalePayment(tx *gorm.DB, req domain.FinalSale, item domain.FinalPaymentType, paymentServiceId *string, status string) (*domain.SalePayment, error) {
	var now = time.Now()
	salePayment := domain.SalePayment{}
	// Insert sale payments
	err := tx.Raw(`
	INSERT INTO sale_payments(
		id, sale_id, cash_box_operation_id, 
		payment_service_id, payment_type_id, 
		amount, paid_at, status) 
	VALUES(?, ?, ?, ?, ?, ?, ?, ?) RETURNING *`,
		uuid.New().String(), req.SaleID, req.CashBoxOperationId,
		paymentServiceId, item.PaymentTypeID, item.Amount, now, status).
		Scan(&salePayment).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &salePayment, nil
}

// Get Payment service with store id and payment type  if status is active
func (s *Services) GetPaymentServiceByStoreId(storeId string, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := s.db.Raw(`SELECT * FROM payment_services WHERE store_id = ? AND type = ? AND is_active = true`,
		storeId, paymentType).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// update sale payment status
func (s *Services) UpdateSalePaymentStatus(tx *gorm.DB, salePaymentID string) error {
	err := tx.Exec(`UPDATE sale_payments SET status = 'paid' WHERE id = ?`, salePaymentID).Error
	if err != nil {
		s.log.Error(err)
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

// Update sale status and total amount after the sale is completed
func (s *Services) UpdateSaleStatus(tx *gorm.DB, saleID string, totalAmount float64, customerID *string) error {
	return tx.Exec(`
	UPDATE sales
	SET
		status = 'completed', total_amount = ?,
		customer_id = ?, completed_at = ?,
		total_discount = (SELECT SUM(discount_amount*quantity) FROM cart_items WHERE sale_id = ?)
	WHERE id = ?`, totalAmount, customerID, time.Now(), saleID, saleID).Error
}

// Update cart item status and store product quantities after the sale is completed
func (s *Services) UpdateCartItemStatus(tx *gorm.DB, saleID string, employeeID string, cashBoxOperationId string) error {
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
		WHERE sale_id = ?`, saleID).
		Scan(&cartItems).Error
	if err != nil {
		s.log.Error(err)
		return err
	}

	// Store productlarni yangilash uchun batch update query
	updateQuery := `UPDATE store_products sp
	SET pack_quantity = pack_quantity - data.new_pack_quantity,
		unit_quantity = unit_quantity - data.new_unit_quantity
	FROM (
		VALUES
	`
	values := []any{}

	for _, item := range cartItems {
		// Pack va unit miqdorlarini hisoblash
		newPackQuantity := item.Quantity
		newUnitQuantity := item.Quantity*item.UnitPerPack + item.UnitQuantity
		values = append(values, item.StoreProductID, newPackQuantity, newUnitQuantity)
		updateQuery += " (CAST(? AS UUID), ?::INTEGER, ?::INTEGER),"
	}

	// So‘rovni yakunlash
	updateQuery = updateQuery[:len(updateQuery)-1] + `
	) AS data(id, new_pack_quantity, new_unit_quantity)
	WHERE sp.id = data.id;`

	// Batch update bajarish
	err = tx.Exec(updateQuery, values...).Error
	if err != nil {
		return err
	}

	// Bonuslarni batch insert qilish
	insertBonusQuery := `INSERT INTO employee_bonus (
		employee_id, sale_id, product_id, cashbox_operation_id, bonus_amount, quantity
	) VALUES `

	bonusValues := []any{}
	for _, item := range cartItems {
		if item.BonusAmount > 0 {
			unitBonus := (item.BonusAmount / float64(item.UnitPerPack)) * float64(item.UnitQuantity)
			totalBonus := item.BonusAmount*float64(item.Quantity) + unitBonus
			bonusValues = append(bonusValues, employeeID, saleID, item.ProductId, cashBoxOperationId, totalBonus, item.Quantity)
			insertBonusQuery += "(?, ?, ?, ?, ?, ?),"
		}
	}

	// Agar bonuslar mavjud bo‘lsa, batch insert qilamiz
	if len(bonusValues) > 0 {
		insertBonusQuery = insertBonusQuery[:len(insertBonusQuery)-1] // oxirgi vergulni olib tashlash
		err = tx.Exec(insertBonusQuery, bonusValues...).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// update return sale cart items
func (s *Services) UpdateReturnSaleCartItems(tx *gorm.DB, saleID string) error {
	var cartItems []domain.CartItem
	// get cart items
	err := tx.Raw(`
		SELECT
			id, store_product_id,
			quantity, unit_quantity, unit_price,
			total_price, status
		FROM cart_items WHERE sale_id = ?`, saleID).
		Scan(&cartItems).Error
	if err != nil {
		return err
	}
	// update store product quantities
	for _, item := range cartItems {
		err = tx.Exec(`
		UPDATE store_products
		SET
			pack_quantity = store_products.pack_quantity + ?,
			unit_quantity = store_products.unit_quantity + (? * products.unit_per_pack + ?)
		FROM products
		WHERE products.id = store_products.product_id AND  store_products.id = ?`,
			item.Quantity, item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			return err
		}
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
func (s *Services) ListSale(param *domain.QueryParam) ([]domain.SaleResponse, int64, error) {
	var totalCount int64
	// get employee info
	var employee domain.Employee
	err := s.db.First(&employee, "id = ?", param.VendorID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, errors.New("employee not found")
		}
		s.log.Error(err)
		return nil, 0, err
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, s.cfg) {
		param.StoreID = employee.StoreId
		param.VendorID = employee.Id
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
		Joins("JOIN stores st ON st.id = s.store_id").
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
	} else {
		query = query.Where("s.employee_id IS NOT NULL OR s.employee_id IS NULL") // Include online sales
	}
	// filter by store id
	if param.StoreID != "" {
		query = query.Where("s.store_id = ?", param.StoreID)
	} else {
		query = query.Where("s.store_id IS NOT NULL OR s.store_id IS NULL") // Include online sales
	}
	// filter by cashbox id
	if param.CashBoxID != "" {
		query = query.Where("co.cash_box_id = ?", param.CashBoxID)
	} else {
		query = query.Where("s.cash_box_operation_id IS NULL OR co.cash_box_id IS NOT NULL") // Include online sales
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
		Debug().
		Find(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return res, totalCount, nil
}
