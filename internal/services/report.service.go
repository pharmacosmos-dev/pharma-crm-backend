package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) ProductReportWithDate(param *domain.ReportQueryParam) ([]map[string]any, error) {

	var (
		dateColumns []string
		res         []map[string]any
		filter      = "WHERE 1 = 1 "
		group       = " GROUP BY p.name "
		order       = " ORDER BY p.name "
		args        = []any{}
	)
	// filter by store ids
	if len(param.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?) "
		args = append(args, param.StoreIds)
	}
	// filter with producer_id
	if param.ProducerId != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerId)
	}
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
	`, datesQuery, param.StartDate, param.EndDate)

	query = query + filter + group + order
	// Queryni bajarish
	err = s.db.Debug().Raw(query, args...).Scan(&res).Error
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
		filter     = " WHERE sl.status = 'completed' "
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
			ELSE ci.quantity * (sp.retail_price-sp.supply_price)
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
		filter += " AND (p.name ILIKE ? OR s.name ILIKE ?) "
		args = append(args, param.Search, param.Search)
	}
	// filter by store_ids
	if len(param.StoreIds) > 0 {
		filter += " AND sl.store_id IN (?) "
		args = append(args, param.StoreIds)
	}
	// filter by producers
	if param.ProducerId != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerId)
	}
	// filter by employee
	if param.EmployeeId != "" {
		filter += " AND sl.employee_id = ? "
		args = append(args, param.EmployeeId)
	}
	// check end_date
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	// filter by start_date, end_date
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND sl.completed_at::date BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}

	query = query + filter + order + pagination
	err := s.db.Debug().Raw(query, args...).Scan(&res).Error
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

// get lfl report service
func (s *Services) LflReport(param *domain.ReportQueryParam) (domain.LflReport, int64, error) {
	var (
		res        domain.LflReport
		totalCount int64
	)

	query := `
	WITH BranchCount AS (
		SELECT COUNT(*) AS branch_count
		FROM stores
		WHERE is_active = true  -- Include only active stores
	),
	CategorySales AS (
		SELECT
			EXTRACT(WEEK FROM sl.completed_at) AS week_number,
			MIN(sl.completed_at::date) AS week_start_date, -- First date of the week
			TO_CHAR(MIN(sl.completed_at), 'DD.MM.YYYY') week_date,
			TO_CHAR(MIN(sl.completed_at), 'Dy') AS weekname,
			EXTRACT(DOW FROM MIN(sl.completed_at)) AS weekday,
			c.name AS category_name,
			SUM(ci.total_price) AS category_total
		FROM sales sl
		INNER JOIN stores s ON sl.store_id = s.id
		INNER JOIN cart_items ci ON sl.id = ci.sale_id
		INNER JOIN store_products sp ON ci.store_product_id = sp.id
		INNER JOIN products p ON sp.product_id = p.id
		INNER JOIN category_products cp ON p.id = cp.product_id
		INNER JOIN categories c ON cp.category_id = c.id
		WHERE
			sl.status = 'completed'
			AND c.name IN ('Лекарственные средства', 'Парафармасевтика')
			AND c.category_id IS NULL
			AND TO_CHAR(sl.completed_at, 'YYYY-MM') = ?
		GROUP BY
			EXTRACT(WEEK FROM sl.completed_at),
			c.name
	),
	PivotedSales AS (
		SELECT
			ROW_NUMBER() OVER (ORDER BY week_number, weekday) AS id,
			cs.week_number,
			cs.week_start_date AS sale_date,
			cs.week_date,
			cs.weekname,
			cs.weekday,
			bc.branch_count,
			MAX(CASE WHEN cs.category_name = 'Лекарственные средства' THEN cs.category_total ELSE 0 END) AS lc_total,
			MAX(CASE WHEN cs.category_name = 'Парафармасевтика' THEN cs.category_total ELSE 0 END) AS parapharma_total,
			SUM(cs.category_total) AS total
		FROM CategorySales cs
		CROSS JOIN BranchCount bc
		GROUP BY
			cs.week_number,
			cs.week_start_date,
			cs.week_date,
			cs.weekname,
			cs.weekday,
			bc.branch_count
	)
	SELECT
		id,
		week_date AS weekdate,
		weekname AS weekname,
		branch_count,
		lc_total AS lc_sum,
		parapharma_total AS parapharma_sum,
		total AS total_sum,
		week_number,
		weekday
	FROM PivotedSales
	ORDER BY week_number, weekday;
	`
	// get first month
	err := s.db.Raw(query, param.StartDate).Scan(&res.FirstMonth).Error
	if err != nil {
		s.log.Warn("ERROR on getting lfl first month: %v", err)
		return res, 0, err
	}
	// get second month
	err = s.db.Raw(query, param.EndDate).Scan(&res.SecondMonth).Error
	if err != nil {
		s.log.Warn("ERROR on getting lfl second month: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}

func (s *Services) StoreReportAmount(param *domain.ReportQueryParam) ([]domain.StoreAmount, int64, error) {
	if param.EndDate == "" {
		param.StartDate = param.EndDate
	}
	var (
		res        []domain.StoreAmount
		totalCount int64
		filter     = " WHERE sa.status = 'completed' "
		args       = []any{}
		group      = " GROUP BY s.id, s.name "
		order      = " ORDER BY total_amount DESC "
	)
	query := `
	SELECT
		s.id,
		s.name AS store_name,
		SUM(CASE WHEN pt.name = 'Naqd' AND sa.sale_type != 'RETURN' THEN sp.amount ELSE 0 END) AS cash,
		SUM(CASE WHEN pt.name = 'Uzcard' AND sa.sale_type != 'RETURN' THEN sp.amount ELSE 0 END) AS uzcard,
		SUM(CASE WHEN pt.name = 'Humo' AND sa.sale_type != 'RETURN' THEN sp.amount ELSE 0 END) AS humo,
		SUM(CASE WHEN pt.name = 'Click' AND sa.sale_type != 'RETURN' THEN sp.amount ELSE 0 END) AS click,
		SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS return_amount,
		SUM(CASE WHEN sa.sale_type != 'RETURN' THEN sp.amount ELSE 0 END) AS total_amount
	FROM
		stores s
	JOIN
		sales sa ON s.id = sa.store_id
	JOIN
		sale_payments sp ON sa.id = sp.sale_id
	JOIN
		payment_types pt ON sp.payment_type_id = pt.id
	`
	if param.StoreId != "" {
		filter += " AND s.id = ? "
		args = append(args, param.StoreId)
	}

	if param.Search != "" {
		filter += " AND s.name ILIKE ? "
		args = append(args, "%"+param.Search+"%")
	}

	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (sa.completed_at + interval '5 hours')::date BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}
	query = query + filter + group + order + " LIMIT ? OFFSET ?;"
	args = append(args, param.Limit, param.Offset)
	err := s.db.Debug().Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting store payment amounts: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}
