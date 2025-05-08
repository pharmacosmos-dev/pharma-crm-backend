package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
)

// get cash box operation shift list
func (s *Services) GetOperationShiftList(storeID, isOpen, search string, limit, offset int) ([]domain.CashboxOperationShift, int64, error) {
	var (
		shifts     []domain.CashboxOperationShift
		args       = []any{}
		totalCount int64
		baseQuery  = `
		FROM cash_boxes cb
		JOIN cashbox_operations co ON cb.id = co.cash_box_id
		JOIN sale_payment_summary sps ON co.id = sps.cash_box_operation_id
		JOIN payment_types pt ON pt.id = sps.payment_type_id
		JOIN stores s ON s.id = cb.store_id`
		filter = " WHERE 1 = 1 "
	)

	// Apply filters
	if storeID != "" {
		filter += " AND s.id = ?"
		args = append(args, storeID)
	}

	if isOpen != "" {
		filter += " AND co.end_time IS NULL"
	}

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		filter += " AND s.name ILIKE ? OR cb.name ILIKE ? "
		args = append(args, search, search)
	}

	// Get total count (excluding pagination)
	countQuery := "SELECT COUNT(DISTINCT cb.id) " + baseQuery + filter
	err := s.db.Raw(countQuery, args...).Scan(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// Query for paginated results
	dataQuery := `
		SELECT
			cb.id, cb.name as cashbox_name,
			s.name AS store_name,
			SUM(co.opened_amount) as opened_amount,
			SUM(co.open_cashless_amount) as opened_cashless_amount,
			COALESCE(SUM(CASE WHEN pt.type = 'cash' THEN sps.total_amount ELSE 0 END), 0) AS cash_amount,
			COALESCE(SUM(CASE WHEN pt.type IN ('card', 'app') THEN sps.total_amount ELSE 0 END), 0) AS cashless_amount,
			(MAX(co.end_time) IS NULL) AS is_open,
			MAX(co.start_time) AS start_time,
			MAX(co.end_time) AS end_time
		` + baseQuery + filter + `
		GROUP BY cb.id, cb.name, s.id
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)
	err = s.db.Debug().Raw(dataQuery, args...).Scan(&shifts).Error
	if err != nil {
		return nil, 0, err
	}

	return shifts, totalCount, nil
}

// GetOperationStats godoc
func (s *Services) GetOperationStats(storeID, isOpen, search string) (domain.CashboxOperationStats, error) {
	var (
		stats domain.CashboxOperationStats
		query = `
		SELECT
			-- 1. Total cash and cashless amount across all cashboxes
			SUM(CASE WHEN pt.type = 'cash' THEN sps.total_amount ELSE 0 END) AS total_cash_amount,
			SUM(CASE WHEN pt.type IN ('card', 'app') THEN sps.total_amount ELSE 0 END) AS total_cashless_amount,

			-- 2. Total opened cash and cashless amount across all cashboxes
			SUM(co.opened_amount) AS total_opened_cash_amount,
			SUM(co.open_cashless_amount) AS total_opened_cashless_amount,

			-- 3. Cash and cashless amounts from currently open cashboxes
			SUM(CASE WHEN co.end_time IS NULL AND pt.type = 'cash' THEN sps.total_amount ELSE 0 END) AS current_cash_amount,
			SUM(CASE WHEN co.end_time IS NULL AND pt.type IN ('card', 'app') THEN sps.total_amount ELSE 0 END) AS current_cashless_amount
		FROM cashbox_operations co
		LEFT JOIN sale_payment_summary sps ON co.id = sps.cash_box_operation_id
		LEFT JOIN payment_types pt ON sps.payment_type_id = pt.id
		LEFT JOIN cash_boxes cb ON co.cash_box_id = cb.id`
		filter = " WHERE 1 = 1 "
		args   = []any{}
	)
	// add filter
	if storeID != "" {
		filter += " AND cb.store_id = ?"
		args = append(args, storeID)
	}
	if isOpen == "true" { // Fix: Apply this filter correctly
		filter += " AND co.end_time IS NULL"
	}
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		filter += " AND (cb.name ILIKE ?)"
		args = append(args, search)
	}
	// collect query
	query += filter
	err := s.db.Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return domain.CashboxOperationStats{}, err
	}
	return stats, err
}

// GetOperationHistory godoc
func (s *Services) OperationHistory(storeID, isOpen, search string, limit, offset int) ([]domain.CashBoxOperationHistory, int64, error) {
	var (
		group      = `co.id, co.operation_id, cb.name, s.name, co.start_time, e.full_name, co.end_time, em.full_name`
		order      = "co.operation_id DESC"
		res        []domain.CashBoxOperationHistory
		totalCount int64
	)

	// Build base query without GROUP BY for count
	baseQuery := s.db.
		Model(&domain.CashboxOperation{}).
		Table("cashbox_operations co").
		Joins("JOIN cash_boxes cb ON co.cash_box_id = cb.id").
		Joins("JOIN stores s ON s.id = cb.store_id").
		Joins("JOIN employees e ON e.id = co.employee_id").
		Joins("LEFT JOIN employees em ON em.id = co.current_employee_id").
		Joins("LEFT JOIN sale_payment_summary sps ON co.id = sps.cash_box_operation_id")

	// Apply filters to base query
	if storeID != "" {
		baseQuery = baseQuery.Where("cb.store_id = ?", storeID)
	}
	if isOpen == "true" {
		baseQuery = baseQuery.Where("co.end_time IS NULL")
	}
	if isOpen == "false" {
		baseQuery = baseQuery.Where("co.end_time IS NOT NULL")
	}
	if search != "" {
		baseQuery = baseQuery.Where("cb.name ILIKE ? OR s.name ILIKE ?", search, search)
	}

	// **Step 1: Get the correct count**
	err := baseQuery.Distinct("co.id").Count(&totalCount).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	// **Step 2: Get the paginated results**
	err = baseQuery.
		Select(`
			co.id, co.operation_id, cb.name AS cashbox_name, 
			s.name AS store_name, co.start_time, 
			e.full_name AS opened_by, co.end_time, 
			em.full_name AS closed_by, 
			COALESCE(SUM(sps.total_expense_amount), 0) AS total_expense_amount, 
			CASE WHEN co.end_time IS NULL THEN TRUE ELSE FALSE END AS is_open`).
		Group(group).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return res, totalCount, nil
}

// close cashbox operation
func (s *Services) CloseCashBoxOperation(cashBoxOperationID string, req *domain.CloseCashboxOperation, senderId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update cash box operation
	err := tx.Exec(`
	UPDATE cashbox_operations SET 
		closed_amount = ?, close_cashless_amount = ?, end_time = NOW(), is_open = FALSE 
	WHERE id = ?`, req.ClosedAmount, req.CloseCashlessAmount, cashBoxOperationID).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return err
	}
	// get total net amount
	var totalNetAmount float64
	err = tx.Raw(`
	SELECT COALESCE(SUM(total_net_amount), 0) AS total_net_amount 
	FROM sale_payment_summary 
	WHERE cash_box_operation_id = ?`, cashBoxOperationID).
		Scan(&totalNetAmount).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return err
	}

	// create cashbox closure
	err = tx.Exec(`
	INSERT INTO cashbox_closures (
		cashbox_operation_id, received_amount, sender_id, status) 
	VALUES (?, ?, ?, ?)`, cashBoxOperationID, totalNetAmount, senderId, config.PENDING).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return err
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error(err)
		tx.Rollback()
		return err
	}
	return nil
}

// create cashbox operation
func (s *Services) CreateCashboxOperation(req *domain.CashboxOperationRequest, userId any) (*domain.Sale, error) {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var id string
	// open cashbox_operation
	err := tx.Raw(`
	INSERT INTO cashbox_operations (
			cash_box_id, employee_id, current_employee_id, opened_amount, open_cashless_amount, is_open, start_time, description
			) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, req.CashBoxID, userId, userId,
		req.OpenedAmount, req.OpenCashlessAmount, true, time.Now(),
		req.Description).Scan(&id).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return nil, err
	}

	var sale domain.Sale
	// create new sale
	err = tx.Raw(`
		INSERT INTO sales (employee_id, store_id, cash_box_operation_id, cashbox_id) 
		VALUES (?, ?, ?, ?) RETURNING *`,
		userId, req.StoreID, id, req.CashBoxID).Scan(&sale).Error
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return nil, err
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error(err)
		tx.Rollback()
		return nil, err
	}
	return &sale, nil
}

// send expense products to 1C
func (s *Services) SendExpenseTo1C(storeID string) error {
	// get cashbox operation info with store

	var store domain.Store
	err := s.db.First(&store, "id = ?", storeID).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info with operation: %v", err)
		return err
	}
	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name

	// get expense docs number
	docNumberQuery := `
	SELECT 'NP-' || LPAD(store_code::TEXT, 5, '0') || '-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MI') AS docs_number
	FROM stores
	WHERE id = ?;`
	err = s.db.Raw(docNumberQuery, storeID).Scan(&expenseData.Document.NumberDok).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense docs number: %v", err)
		return err
	}

	// create new shift expense
	err = s.CreateNewExpense(storeID, expenseData.Document.NumberDok)
	if err != nil {
		s.log.Warn("ERROR on creating shift expense: %v", err)
		return err
	}

	// get expense dok time with adding 5 hours
	expenseData.Document.DocumentDate = time.Now().Add(time.Hour * 5).Format(time.DateTime)
	// get expense products query
	expenseProductQuery := `
	SELECT
		sp.product_id,
		p.material_code,
		p.name,
		p.barcode,
		p.mxik AS ikpu,
		COALESCE(pr.name, '') AS manufacturer,
		COALESCE(sp.serial_number, '') AS product_series_number,
		sp.expire_date::date,
		SUM(ci.quantity) AS quantity,
		SUM(ci.unit_quantity) AS unit_quantity,
		sp.supply_price AS supply_price_vat,
		sp.retail_price AS retail_price_vat,
		id.supply_price,
		id.retail_price,
		sp.vat,
		ROUND((sp.vat_price*SUM(ci.quantity)) + ((sp.vat_price/p.unit_per_pack)*SUM(ci.unit_quantity)), 2) AS vat_sum,
		ROUND((id.retail_price*SUM(ci.quantity)) + ((id.retail_price/p.unit_per_pack)*SUM(ci.unit_quantity)), 2) AS sum,
		SUM(ci.total_price) AS sum_vat
	FROM sales s
	JOIN cart_items ci ON s.id = ci.sale_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	AND s.status = 'completed' AND s.sale_type = 'SALE'
	GROUP BY
		p.id, pr.id, sp.id, id.id
	`
	// complete get expense product list
	err = s.db.Raw(expenseProductQuery, storeID).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense products: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}

	t, _ := json.Marshal(expenseData)
	fmt.Println("--->>> ", string(t))

	// send fakt to 1C
	err = s.DoRequest(context.Background(), expenseData, "/rasxod")
	if err != nil {
		s.log.Warn("ERROR on send rasxod request: %v", err)
		return err
	}
	// update expense status to 1 after successfully sent
	err = s.UpdateExpenseStatusByDocNumber(1, expenseData.Document.NumberDok)
	if err != nil {
		return err
	}
	return nil
}
