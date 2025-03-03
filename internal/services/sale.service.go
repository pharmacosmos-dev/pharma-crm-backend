package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"gorm.io/gorm"
)

// create new sale
func (s *Storage) CreateSale(tx *gorm.DB, req *domain.SaleRequest) (*domain.Sale, error) {
	var res domain.Sale
	err := tx.Raw(`INSERT INTO sales(id, employee_id, cash_box_operation_id) VALUES(?, ?, ?) RETURNING *`,
		req.ID, req.EmployeeID, req.CashBoxOperationId).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// update sale with receiving field
func (s *Storage) UpdateSaleField(field string, value string, idField string, idValue string) (*domain.Sale, error) {
	var res domain.Sale
	err := s.db.Raw(`UPDATE sales SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Create sale payment
func (s *Storage) CreateSalePayment(tx *gorm.DB, req domain.FinalSale, item domain.FinalPaymentType, paymentServiceId *string, status string) (*domain.SalePayment, error) {
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
func (s *Storage) GetPaymentServiceByStoreId(storeId string, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := s.db.Raw(`SELECT * FROM payment_services WHERE store_id = ? AND type = ? AND is_active = true`,
		storeId, paymentType).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// update sale payment status
func (s *Storage) UpdateSalePaymentStatus(tx *gorm.DB, salePaymentID string) error {
	err := tx.Exec(`UPDATE sale_payments SET status = 'paid' WHERE id = ?`, salePaymentID).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}

// Create or update sale payment summary with on conflict do update
func (s *Storage) CreateOrUpdateSalePaymentSummary(tx *gorm.DB, cashBoxOperationId string, paymentTypeId string, amount float64) error {
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
func (s *Storage) UpdateSaleStatus(tx *gorm.DB, saleID string, totalAmount float64, customerID *string, storeID string) error {
	return tx.Exec(`
	UPDATE sales
	SET
		status = 'completed', total_amount = ?,
		customer_id = ?, completed_at = ?,
		store_id = ?,
		total_discount = (SELECT SUM(discount_amount*quantity) FROM cart_items WHERE sale_id = ?)
	WHERE id = ?`, totalAmount, customerID, time.Now(), storeID, saleID, saleID).Error
}

// Update cart item status and store product quantities after the sale is completed
func (s *Storage) UpdateCartItemStatus(tx *gorm.DB, saleID string) error {
	var cartItems []domain.CartItem
	err := tx.Raw(`
		SELECT
			ci.id, ci.store_product_id,
			ci.quantity, ci.unit_quantity, ci.unit_price,
			ci.total_price, ci.status
		FROM cart_items ci WHERE sale_id = ?`, saleID).
		Scan(&cartItems).Error
	if err != nil {
		return err
	}

	for _, item := range cartItems {
		err = tx.Debug().Exec(`
		UPDATE store_products
		SET
			pack_quantity = CASE WHEN ? > 0 THEN (unit_quantity - ?)/products.unit_per_pack - ? ELSE pack_quantity - ? END,
			unit_quantity = unit_quantity - (? * products.unit_per_pack + ?)
		FROM products
		WHERE products.id = store_products.product_id AND  store_products.id = ?`,
			item.UnitQuantity, item.UnitQuantity, item.Quantity, item.Quantity,
			item.Quantity, item.UnitQuantity, item.StoreProductID).Error
		if err != nil {
			return err
		}
	}

	err = tx.
		Table("cart_items").
		Where("sale_id = ?", saleID).
		Update("status", "sold").Error
	if err != nil {
		return err
	}
	return nil
}

// create sale for online order
func (s *Storage) CreateOnlineSale(tx *gorm.DB, saleId string, totalAmount int64) error {
	err := tx.Exec(`
	INSERT INTO sales(id, total_amount, type, is_delivered, status, completed_at) VALUES(?, ?, ?, ?, ?, ?)`,
		saleId, totalAmount, "online", false, config.COMPLETED, time.Now()).Error
	if err != nil {
		return err
	}
	return nil
}

// get sale list data
func (s *Storage) ListSale(c *gin.Context, param *domain.QueryParam) ([]domain.SaleResponse, int64, error) {
	var totalCount int64

	// get user id from header
	userId, ok := c.Get("user_id")
	if !ok {
		return nil, 0, errors.New("user not found in context")
	}
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
		param.StoreID = employee.StoreId
		param.StoreID = userId.(string)
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
		Find(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return res, totalCount, nil
}
