package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

// get cash box operation shift list
func (s *Storage) GetOperationShiftList(storeID, isOpen, search string, limit, offset int) ([]domain.CashboxOperationShift, int64, error) {
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
func (s *Storage) GetOperationStats(storeID, isOpen, search string) (domain.CashboxOperationStats, error) {
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
	query += filter
	err := s.db.Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return domain.CashboxOperationStats{}, err
	}
	return stats, err
}
