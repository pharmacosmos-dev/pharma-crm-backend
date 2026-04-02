package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// region Create

// create cashbox operation
func (s *Services) CreateCashboxOperation(ctx context.Context, req *domain.CashboxOperationRequest, userId string) (*domain.Sale, error) {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	type result struct {
		ID string
	}

	var res result
	err := tx.WithContext(ctx).
		Raw(`
		SELECT id FROM cashbox_operations 
		WHERE is_open = TRUE AND current_employee_id = ? 
		LIMIT 1;
	`, userId).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Error("failed to check open cashbox:", err)
		return nil, domain.InternalServerError
	}
	if res.ID != "" {
		_ = tx.Rollback()
		return nil, domain.AlreadyHaveOpenCashboxOperationError
	}

	err = tx.WithContext(ctx).
		Raw(`
	INSERT INTO cashbox_operations (
			cash_box_id, 
			employee_id, 
			current_employee_id, 
			opened_amount, 
			open_cashless_amount, 
			is_open, 
			start_time, 
			description
			) 
	VALUES (
			?, ?, ?, ?,?, ?, ?, ?
	) RETURNING id
	`, req.CashBoxID,
			userId,
			userId,
			req.OpenedAmount,
			req.OpenCashlessAmount,
			true,
			time.Now(),
			req.Description,
		).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create cashbox operation: %v", err)
		return nil, domain.InternalServerError
	}

	var sale domain.Sale
	// create new sale
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO sales (employee_id, store_id, cash_box_operation_id, cashbox_id, display_id) 
		VALUES (?, ?, ?, ?, ?) RETURNING *`,
		userId, req.StoreID, res.ID, req.CashBoxID, s.generateDisplayId()).Scan(&sale).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create new sale: %v", err)
		return nil, domain.InternalServerError
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return nil, domain.InternalServerError
	}
	return &sale, nil
}

// region Get

func (s *Services) GetCashboxOperationByID(ctx context.Context, id string) (*domain.CashboxOperation, error) {
	var res domain.CashboxOperation
	err := s.db.Where("id = ?", id).First(&res).Error
	if err != nil {
		s.log.Errorf("could not get cashbox_operation: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

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
			cb.id, cb.name as cashbox_name, cb.terminal_id,
			s.name AS store_name,
			SUM(co.opened_amount) as opened_amount,
			SUM(co.open_cashless_amount) as opened_cashless_amount,
			COALESCE(SUM(CASE WHEN pt.type = 'cash' THEN sps.total_amount ELSE 0 END), 0) AS cash_amount,
			COALESCE(SUM(CASE WHEN pt.type IN ('card', 'app') THEN sps.total_amount ELSE 0 END), 0) AS cashless_amount,
			(MAX(co.end_time) IS NULL) AS is_open,
			MAX(co.start_time) AS start_time,
			MAX(co.end_time) AS end_time
		` + baseQuery + filter + `
		GROUP BY cb.id, cb.name, cb.terminal_id, s.id
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)
	err = s.db.Raw(dataQuery, args...).Scan(&shifts).Error
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
func (s *Services) GetOperationHistory(storeID, isOpen, search string, limit, offset int) ([]domain.CashBoxOperationHistory, int64, error) {
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

// region Update

// close cashbox operation
func (s *Services) CloseCashBoxOperation(ctx context.Context, cashBoxOperationId string, req *domain.CloseCashboxOperation, senderId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// update cash box operation
	err := tx.WithContext(ctx).Exec(`
	UPDATE cashbox_operations SET 
		closed_amount = ?, 
		close_cashless_amount = ?, 
		end_time = NOW(), 
		is_open = FALSE 
	WHERE id = ?`,
		req.ClosedAmount,
		req.CloseCashlessAmount,
		cashBoxOperationId,
	).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update cashbox operation on closing: %v", err)
		return domain.InternalServerError
	}
	// get total net amount
	var totalNetAmount float64
	err = tx.WithContext(ctx).
		Raw(`
	SELECT COALESCE(SUM(total_net_amount), 0) AS total_net_amount 
	FROM sale_payment_summary 
	WHERE cash_box_operation_id = ?`, cashBoxOperationId).
		Scan(&totalNetAmount).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get total net amount: %v", err)
		return domain.InternalServerError
	}

	// create cashbox closure
	err = tx.WithContext(ctx).Exec(`
	INSERT INTO cashbox_closures (
		cashbox_operation_id, received_amount, sender_id, status) 
	VALUES (?, ?, ?, ?)`,
		cashBoxOperationId,
		totalNetAmount,
		senderId,
		constants.GeneralStatusPending,
	).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create cashbox closure: %v", err)
		return domain.InternalServerError
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}
