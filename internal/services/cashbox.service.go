package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

// get cash box operation shift list
func (s *Storage) GetOperationShiftList(storeID, isOpen, search string, limit, offset int) ([]domain.CashboxOperationShift, int64, error) {
	var (
		shifts     []domain.CashboxOperationShift
		args       = []interface{}{}
		totalCount int64
		baseQuery  = `
			FROM cashbox_operations co
			JOIN cash_boxes cb ON co.cash_box_id = cb.id
			JOIN stores s ON s.id = cb.store_id
			LEFT JOIN sale_payments sp ON co.id = sp.cash_box_operation_id
			LEFT JOIN payment_types pt ON sp.payment_type_id = pt.id
		`
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
		filter += " AND (cb.name ILIKE ? OR s.name ILIKE ? OR CAST(co.operation_id AS TEXT) LIKE ?)"
		args = append(args, search, search, search)
	}

	// Get total count (excluding pagination)
	countQuery := "SELECT COUNT(DISTINCT co.id) " + baseQuery + filter
	err := s.db.Raw(countQuery, args...).Scan(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// Query for paginated results
	dataQuery := `
		SELECT
			co.id,
			co.operation_id,
			cb.name AS cashbox_name,
			s.name AS store_name,
			(co.end_time IS NULL) AS is_open,
			COALESCE(SUM(CASE WHEN pt.type = 'cash' THEN sp.amount ELSE 0 END), 0) AS cash_amount,
			COALESCE(SUM(CASE WHEN pt.type IN ('card', 'app') THEN sp.amount ELSE 0 END), 0) AS cashless_amount,
			co.start_time,
			co.end_time
		` + baseQuery + filter + `
		GROUP BY co.id, co.operation_id, cb.name, s.name, co.end_time
		ORDER BY co.operation_id DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)
	err = s.db.Debug().Raw(dataQuery, args...).Scan(&shifts).Error
	if err != nil {
		return nil, 0, err
	}

	return shifts, totalCount, nil
}
