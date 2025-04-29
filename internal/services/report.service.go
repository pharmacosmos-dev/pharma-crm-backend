package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) ProductReportWithDate(param *domain.ReportQueryParam) ([]map[string]any, error) {
	var res []map[string]any

	// Date range ichida har bir kun uchun dinamik columnlar
	var dateColumns []string
	// var dateList []time.Time
	startDate, err := time.Parse("2006-01-02", param.StartDate)
	if err != nil {
		return res, fmt.Errorf("invalid start date")
	}
	endDate, err := time.Parse("2006-01-02", param.EndDate)
	if err != nil {
		return res, fmt.Errorf("invalid end date")
	}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateColumns = append(dateColumns, fmt.Sprintf(
			`SUM(CASE WHEN s.completed_at::date = '%s' THEN s.total_amount ELSE 0 END) AS "%s"`,
			dateStr, dateStr,
		))
	}

	// Dinamik columnlarni JOIN qilish
	datesQuery := strings.Join(dateColumns, ",\n")

	// Asosiy query template
	query := fmt.Sprintf(`
		SELECT
			p.name AS product_name,
			%s,
			SUM(CASE WHEN s.completed_at::date BETWEEN '%s' AND '%s' THEN s.total_amount ELSE 0 END) AS total_amount
		FROM products p
		JOIN store_products sp ON p.id = sp.product_id
		LEFT JOIN cart_items ci ON sp.id = ci.store_product_id
		LEFT JOIN sales s ON ci.sale_id = s.id
		WHERE
			sp.store_id = ?
			AND sp.product_id IN (?)
		GROUP BY p.name
		ORDER BY p.name;
	`, datesQuery, param.StartDate, param.EndDate)

	// Queryni bajarish
	err = s.db.Raw(query, param.StoreId, param.ProductIds).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on getting product report: %v", err)
		return nil, err
	}

	return res, nil
}

// get employee bonuses report service
func (s *Services) BonusReport(param *domain.ReportQueryParam) ([]domain.BonusReport, int64, error) {
	var (
		res        []domain.BonusReport
		totalCount int64
		filter     = "WHERE 1 = 1 "
		args       = []any{}
		order      = " ORDER BY e.full_name"
		group      = " GROUP BY e.id, s.id"
	)
	// build query
	query := `
	SELECT
		e.id AS id,
		e.public_id,
		e.full_name,
		e.phone,
		s.name AS store_name,
		STRING_AGG(DISTINCT r.name, '/' ORDER BY r.name) AS role,
		SUM(eb.bonus_amount) as amount,
		SUM(eb.quantity+(eb.unit_quantity / 10.0)) AS count, 
		COUNT(*) OVER() AS total_count
	FROM employee_bonus eb
		JOIN employees e ON eb.employee_id = e.id
		JOIN stores s ON e.store_id = s.id
		JOIN employee_roles er ON e.id = er.employee_id
		JOIN roles r ON er.role_id = r.id
	`
	// filter with store_id
	if len(param.StoreIds) > 0 {
		filter += " AND s.id IN (?) "
		args = append(args, param.StoreIds)
	}
	// filter with search PublicID, fullName, phone
	if param.Search != "" {
		search := "%" + param.Search + "%"
		filter += " AND (e.full_name ILIKE ? OR e.phone LIKE ? OR CAST(e.public_id AS TEXT) LIKE ?)"
		args = append(args, search, search, search)
	}
	// checking end time with empty string
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	// filter with start date
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND eb.created_at::date BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}

	// sort by order type
	if param.Order != "" {
		switch param.Order {
		case "min_count":
			order = " ORDER BY count "
		case "max_count":
			order = " ORDER BY count DESC "
		case "min_amount":
			order = " ORDER BY amount "
		case "max_amount":
			order = " ORDER BY amount DESC "
		}
	}
	// add pagination
	pagination := " LIMIT ? OFFSET ?;"
	args = append(args, param.Limit, param.Offset)
	// collect query with filter, group by and order
	query = query + filter + group + order + pagination
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting bonus report: %v", err)
		return res, 0, nil
	}
	// get total count
	if len(res) > 0 {
		// Read from the first row — all rows have the same total_count
		totalCount = res[0].TotalCount
	}
	return res, totalCount, nil
}

// get product report with sale_number
func (s *Services) ProductReport(param *domain.ReportQueryParam) ([]domain.ProductReport, int64, error) {
	var (
		res        []domain.ProductReport
		totalCount int64
		filter     = " WHERE sl.status = 'completed' AND ci.status = 'sold' "
		args       = []any{}
		order      = " ORDER BY sl.completed_at DESC "
		pagination = fmt.Sprintf(" LIMIT %d OFFSET %d ", param.Limit, param.Offset)
	)

	query := `
	SELECT
		ci.id AS cart_item_id,
		p.material_code,
		s.name AS store_name,
		p.name AS product_name,
		pr.name AS producer_name,
		sp.serial_number,
		sp.expire_date,
		ci.quantity || ',' || ci.unit_quantity AS quantity,
		sp.supply_price,
		sp.retail_price,
		ROUND(CASE
			WHEN ci.unit_quantity > 0 THEN (ci.quantity * sp.supply_price) + (ci.unit_quantity * (sp.supply_price / p.unit_per_pack))
			ELSE ci.quantity * sp.supply_price
		END, 2) AS supply_price_sum,
		ROUND(CASE
			WHEN ci.unit_quantity > 0 THEN (ci.quantity * sp.retail_price) + (ci.unit_quantity * (sp.retail_price / p.unit_per_pack))
			ELSE ci.quantity * sp.retail_price
		END, 2) AS retail_price_sum,
		ROUND(CASE
			WHEN ci.unit_quantity > 0 THEN (ci.quantity * (sp.retail_price-sp.supply_price)) + (ci.unit_quantity * ((sp.retail_price-sp.supply_price) / p.unit_per_pack))
			ELSE ci.quantity * sp.retail_price
		END, 2) AS markup_sum,
		ROUND(CASE
			WHEN ci.unit_quantity > 0 THEN (ci.quantity * sp.vat_price) + (ci.unit_quantity * (sp.vat_price / p.unit_per_pack))
			ELSE ci.quantity * sp.vat_price
		END, 2) AS vat_sum,
		sl.completed_at,
		e.full_name,
		sl.sale_number,
		ci.marking_count,
		COUNT(*) OVER() AS total_count
	FROM
		sales sl
		INNER JOIN stores s ON sl.store_id = s.id
		INNER JOIN employees e ON sl.employee_id = e.id
		INNER JOIN cart_items ci ON sl.id = ci.sale_id
		INNER JOIN store_products sp ON ci.store_product_id = sp.id
		INNER JOIN products p ON sp.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	`
	// filter by search key
	if param.Search != "" {
		filter += " (p.name ILIKE ? OR s.name ILIKE ?) "
		args = append(args, param.Search, param.Search)
	}
	// filter by store_ids
	if len(param.StoreIds) > 0 {
		filter += " sl.store_id IN (?) "
		args = append(args, param.StoreIds)
	}
	// filter by producers
	if len(param.ProducerIds) > 0 {
		filter += " p.producer_id IN (?) "
		args = append(args, param.ProducerIds)
	}
	// filter by employee
	if param.EmployeeId != "" {
		filter += " sl.employee_id = ? "
		args = append(args, param.EmployeeId)
	}
	// check end_date
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	// filter by start_date, end_date
	if param.StartDate != "" && param.EndDate != "" {
		filter += " sl.completed_at BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}

	query = query + filter + order + pagination
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product report: %v", err)
		return res, 0, nil
	}

	// get total_count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}
