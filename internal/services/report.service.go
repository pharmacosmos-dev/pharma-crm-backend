package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/pkg/utils"

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
			SUM(CASE WHEN s.completed
		::date BETWEEN '%s' AND '%s' THEN s.total_amount ELSE 0 END) AS total_amount
		FROM products p
		JOIN store_products sp ON p.id = sp.product_id
		LEFT JOIN cart_items ci ON sp.id = ci.store_product_id
		LEFT JOIN sales s ON ci.sale_id = s.id
	`, datesQuery, param.StartDate, param.EndDate)

	query = query + filter + group + order
	// Queryni bajarish
	err = s.db.Raw(query, args...).Scan(&res).Error
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
		args       []any
	)

	// Bazaviy query
	query := `
		SELECT
			e.id AS id,
			e.public_id,
			e.full_name,
			e.phone,
			s.name AS store_name,
			STRING_AGG(DISTINCT r.name, '/' ORDER BY r.name) AS role,
			SUM(eb.bonus_amount) AS amount,
			SUM(eb.quantity + (eb.unit_quantity / 10.0)) AS count,
			COUNT(*) OVER() AS total_count
		FROM employee_bonus eb
		JOIN employees e ON eb.employee_id = e.id
		JOIN stores s ON e.store_id = s.id
		JOIN employee_roles er ON e.id = er.employee_id
		JOIN roles r ON er.role_id = r.id
	`

	filter := " WHERE 1 = 1 "
	group := " GROUP BY e.id, s.id "
	order := " ORDER BY e.full_name"

	// Filtrlash: do‘konlar
	if len(param.StoreIds) > 0 {
		filter += " AND s.id IN (?)"
		args = append(args, param.StoreIds)
	}

	// Filtrlash: qidiruv (public_id, full_name, phone)
	if param.Search != "" {
		search := "%" + param.Search + "%"
		filter += " AND (e.full_name ILIKE ? OR e.phone LIKE ? OR CAST(e.public_id AS TEXT) LIKE ?)"
		args = append(args, search, search, search)
	}

	// Sana bo‘yicha filter (5 soat farq bilan)
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
		args = append(args, param.StartDate, param.EndDate)
	} else if param.EndDate == "" && param.StartDate != "" {
		filter += " AND (eb.created_at + interval '5 hours') = ?"
		args = append(args, param.StartDate)
	}

	// Sortlash: parametrga qarab
	switch param.Order {
	case "min_count":
		order = " ORDER BY count ASC"
	case "max_count":
		order = " ORDER BY count DESC"
	case "min_amount":
		order = " ORDER BY amount ASC"
	case "max_amount":
		order = " ORDER BY amount DESC"
	}

	// Yakuniy query
	finalQuery := query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// So‘rovni bajarish
	err := s.db.Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting bonus report: %v", err)
		return res, 0, nil
	}

	// Total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// get product report service
func (s *Services) ProductReport(ctx context.Context, param *domain.ReportQueryParam) ([]domain.ProductReport, int64, error) {
	var (
		res        []domain.ProductReport
		totalCount int64
		filter     = " WHERE sl.status = 'completed' "
		args       = []any{}
		order      = utils.BuildProductReport(param.Order)
		pagination = fmt.Sprintf(" LIMIT %d OFFSET %d ", param.Limit, param.Offset)
	)

	query := `
	SELECT
		ci.id AS cart_item_id,
		p.material_code,
		s.name AS store_name,
		p.name AS product_name,
		COALESCE(pr.name, '') AS producer_name,
		sp.serial_number,
		sp.expire_date,
		ROUND(ci.quantity::numeric + ci.unit_quantity::numeric/p.unit_per_pack, 4) AS quantity,
		sp.supply_price,
		sp.retail_price,
		ROUND((ci.quantity * sp.supply_price) + (ci.unit_quantity * (sp.supply_price / p.unit_per_pack)), 2) AS supply_price_sum,
		ROUND((ci.quantity * sp.retail_price) + (ci.unit_quantity * (sp.retail_price / p.unit_per_pack)), 2) AS retail_price_sum,
		ROUND((ci.quantity * (sp.retail_price-sp.supply_price)) + (ci.unit_quantity * ((sp.retail_price-sp.supply_price) / p.unit_per_pack)), 2) AS markup_sum,
		ROUND((ci.quantity * sp.vat_price) + (ci.unit_quantity * (sp.vat_price / p.unit_per_pack)), 2) AS vat_sum,
		(sl.completed_at + interval '5 hours') AS completed_at,
		e.full_name,
		sl.sale_number,
		sl.sale_type,
		ci.marking_count
	FROM
		sales sl
		INNER JOIN stores s ON sl.store_id = s.id
		INNER JOIN employees e ON sl.employee_id = e.id
		INNER JOIN cart_items ci ON sl.id = ci.sale_id
		INNER JOIN store_products sp ON ci.store_product_id = sp.id
		INNER JOIN products p ON sp.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	tquery := `
	SELECT
		COUNT(*) AS total_count
	FROM
		sales sl
		INNER JOIN stores s ON sl.store_id = s.id
		INNER JOIN employees e ON sl.employee_id = e.id
		INNER JOIN cart_items ci ON sl.id = ci.sale_id
		INNER JOIN store_products sp ON ci.store_product_id = sp.id
		INNER JOIN products p ON sp.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	// search filter
	if param.Search != "" {
		filter += " AND (p.name ILIKE ? OR s.name ILIKE ?) "
		args = append(args, param.Search, param.Search)
	}
	// store_ids filter
	if len(param.StoreIds) > 0 {
		filter += " AND sl.store_id IN (?) "
		args = append(args, param.StoreIds)
	}
	// producer filter
	if param.ProducerId != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerId)
	}
	// employee filter
	if param.EmployeeId != "" {
		filter += " AND sl.employee_id = ? "
		args = append(args, param.EmployeeId)
	}

	// Apply date filter with full datetime support
	if param.StartDate != "" && param.EndDate != "" {
		startTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Warn("Invalid start_date format: %v", err)
			return res, 0, err
		}
		endTime, err := time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Warn("Invalid end_date format: %v", err)
			return res, 0, err
		}
		startStr := startTime.Format("2006-01-02 15:04:05")
		endStr := endTime.Format("2006-01-02 15:04:05")
		filter += " AND (sl.completed_at + interval '5 hours') BETWEEN ? AND ? "
		args = append(args, startStr, endStr)
	} else if param.EndDate == "" && param.StartDate != "" {
		endTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Warn("Invalid start_date format: %v", err)
			return res, 0, err
		}
		endStr := endTime.Format("2006-01-02 15:04:05")
		filter += " AND (sl.completed_at + interval '5 hours') <= ? "
		args = append(args, endStr)
	}

	// Total count query
	tquery += filter
	err := s.db.Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting product report total count: %v", err)
		return res, 0, nil
	}

	// Main query
	query += filter + order + pagination
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product report: %v", err)
		return res, 0, nil
	}

	return res, totalCount, nil
}

func (s *Services) ProductStatusReport(ctx context.Context, param *domain.ReportQueryParam) (domain.ProductStatusReport, error) {
	var (
		res   domain.ProductStatusReport
		args  []any
		joins = []string{
			"INNER JOIN cart_items ci ON sl.id = ci.sale_id",
			"INNER JOIN store_products sp ON ci.store_product_id = sp.id",
			"INNER JOIN products p ON sp.product_id = p.id",
		}
		filter = " WHERE sl.status = 'completed' "
	)

	// Conditionally add joins
	if param.Search != "" || len(param.StoreIds) > 0 {
		joins = append([]string{"INNER JOIN stores s ON sl.store_id = s.id"}, joins...)
	}
	if param.EmployeeId != "" {
		joins = append(joins, "INNER JOIN employees e ON sl.employee_id = e.id")
	}
	if param.ProducerId != "" {
		joins = append(joins, "LEFT JOIN producers pr ON p.producer_id = pr.id")
	}

	// Filters
	if param.Search != "" {
		filter += " AND (p.name ILIKE ? OR s.name ILIKE ?) "
		args = append(args, param.Search, param.Search)
	}
	if len(param.StoreIds) > 0 {
		filter += " AND sl.store_id IN (?) "
		args = append(args, param.StoreIds)
	}
	if param.ProducerId != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerId)
	}
	if param.EmployeeId != "" {
		filter += " AND sl.employee_id = ? "
		args = append(args, param.EmployeeId)
	}
	if param.StartDate != "" && param.EndDate != "" {
		startTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Warn("Invalid start_date format: %v", err)
			return res, err
		}
		endTime, err := time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Warn("Invalid end_date format: %v", err)
			return res, err
		}
		startStr := startTime.Format("2006-01-02 15:04:05")
		endStr := endTime.Format("2006-01-02 15:04:05")

		filter += " AND (sl.completed_at + interval '5 hours') BETWEEN ? AND ? "
		args = append(args, startStr, endStr)
	} else if param.EndDate == "" && param.StartDate != "" {
		endTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Warn("Invalid start_date format: %v", err)
			return res, err
		}
		endStr := endTime.Format("2006-01-02 15:04:05")
		filter += " AND (sl.completed_at + interval '5 hours') <= ? "
		args = append(args, endStr)
	}

	// Build and run query
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN sl.sale_type = 'SALE' THEN (ci.quantity + ci.unit_quantity / p.unit_per_pack) ELSE 0 END), 0) AS total_quantity,
			COALESCE(SUM(CASE WHEN sl.sale_type = 'RETURN' THEN (ci.quantity + ci.unit_quantity / p.unit_per_pack) ELSE 0 END), 0) AS total_quantity_returned,
			ROUND(COALESCE(SUM(CASE
				WHEN sl.sale_type = 'SALE' THEN (ci.quantity * sp.retail_price) + (ci.unit_quantity * (sp.retail_price / p.unit_per_pack))
				WHEN sl.sale_type = 'RETURN' THEN ((ci.quantity * sp.retail_price) + (ci.unit_quantity * (sp.retail_price / p.unit_per_pack))) * (-1)
				ELSE 0
			END), 0), 2) AS total_retail_price_sum,
			ROUND(COALESCE(SUM(CASE 
				WHEN sl.sale_type = 'RETURN' THEN (ci.quantity * sp.retail_price) + (ci.unit_quantity * (sp.retail_price / p.unit_per_pack)) 
				ELSE 0 END), 0), 2) AS total_retail_price_sum_returned
		FROM sales sl
		%s
		%s
	`, strings.Join(joins, "\n"), filter)

	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product status report: %v", err)
		return res, err
	}
	return res, nil
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

// get store report amount
func (s *Services) StoreReportAmount(param *domain.ReportQueryParam) ([]domain.StoreAmount, int64, error) {

	var (
		res        []domain.StoreAmount
		totalCount int64
		filter     = " WHERE sa.status = 'completed' "
		args       []any
		group      = " GROUP BY s.id, s.name, sale_date"
		order      = utils.BuildStoreReportOrderClause(param.Order)
	)

	// Filters
	if param.StoreId != "" {
		filter += " AND s.id = ?"
		args = append(args, param.StoreId)
	}
	if param.Search != "" {
		filter += " AND s.name ILIKE ?"
		args = append(args, "%"+param.Search+"%")
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (sa.completed_at + interval '5 hours')::date BETWEEN ? AND ?"
		args = append(args, param.StartDate, param.EndDate)
	} else if param.EndDate == "" && param.StartDate != "" {
		filter += " AND (sa.completed_at + interval '5 hours')::date = ?"
		args = append(args, param.StartDate)
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) AS total_count FROM (
			SELECT s.id, s.name, (sa.completed_at + interval '5 hours')::date AS sale_date
			FROM stores s
			JOIN sales sa ON s.id = sa.store_id
			JOIN sale_payments sp ON sa.id = sp.sale_id
			JOIN payment_types pt ON sp.payment_type_id = pt.id
			%s
			%s
		) AS total_count_table
	`, filter, group)

	if err := s.db.Raw(countQuery, args...).Scan(&totalCount).Error; err != nil {
		s.log.Warn("ERROR on getting store report total count: %v", err)
		return res, 0, err
	}

	// Main query
	mainQuery := fmt.Sprintf(`
		SELECT
			row_number() OVER (ORDER BY s.name, (sa.completed_at + interval '5 hours')::date) AS uid,
			s.id,
			s.store_code,
			s.name AS store_name,
			(sa.completed_at + interval '5 hours')::date AS sale_date,
			SUM(CASE WHEN pt.name = 'Naqd' AND sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Naqd' AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS cash,
			SUM(CASE WHEN pt.name = 'Uzcard' AND sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Uzcard' AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS uzcard,
			SUM(CASE WHEN pt.name = 'Humo' AND sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Humo' AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS humo,
			SUM(CASE WHEN pt.name = 'Click' AND sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Click' AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS click,
			SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS return_amount,
			SUM(CASE WHEN sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) - 
			SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS total_amount
		FROM stores s
		JOIN sales sa ON s.id = sa.store_id
		JOIN sale_payments sp ON sa.id = sp.sale_id
		JOIN payment_types pt ON sp.payment_type_id = pt.id
		%s
		%s
		%s
		LIMIT ? OFFSET ?
	`, filter, group, order)

	argsWithPagination := append(args, param.Limit, param.Offset)

	if err := s.db.Raw(mainQuery, argsWithPagination...).Scan(&res).Error; err != nil {
		s.log.Warn("ERROR on getting store payment amounts: %v", err)
		return res, 0, err
	}

	return res, totalCount, nil
}

// get store report stats
func (s *Services) ReportByStoreStats(param *domain.ReportQueryParam) (domain.StoreReportStats, error) {
	var (
		res    domain.StoreReportStats
		filter = " WHERE sa.status = 'completed' "
		args   []any
	)

	// Filterlar
	if param.StoreId != "" {
		filter += " AND s.id = ?"
		args = append(args, param.StoreId)
	}
	if param.Search != "" {
		filter += " AND s.name ILIKE ?"
		args = append(args, "%"+param.Search+"%")
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (sa.completed_at + interval '5 hours')::date BETWEEN ? AND ?"
		args = append(args, param.StartDate, param.EndDate)
	} else if param.EndDate == "" && param.StartDate != "" {
		filter += " AND (sa.completed_at + interval '5 hours')::date >= ?"
		args = append(args, param.StartDate)
	}

	query := fmt.Sprintf(`
		SELECT
			SUM(CASE WHEN pt.name = 'Naqd'   AND sa.sale_type = 'SALE'   THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Naqd'   AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS cash,

			SUM(CASE WHEN pt.name = 'Uzcard' AND sa.sale_type = 'SALE'   THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Uzcard' AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS uzcard,

			SUM(CASE WHEN pt.name = 'Humo'   AND sa.sale_type = 'SALE'   THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Humo'   AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS humo,

			SUM(CASE WHEN pt.name = 'Click'  AND sa.sale_type = 'SALE'   THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Click'  AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS click,

			SUM(CASE WHEN pt.name = 'Payme'  AND sa.sale_type = 'SALE'   THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN pt.name = 'Payme'  AND sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS payme,

			SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS return_amount,

			SUM(CASE WHEN sa.sale_type = 'SALE' THEN sp.amount ELSE 0 END) -
			SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sp.amount ELSE 0 END) AS total_amount
		FROM stores s
		JOIN sales sa ON s.id = sa.store_id
		JOIN sale_payments sp ON sa.id = sp.sale_id
		JOIN payment_types pt ON sp.payment_type_id = pt.id
		%s
	`, filter)

	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Warn("ERROR on getting store report: %v", err)
		return res, err
	}

	return res, nil
}

// get report top products
func (s *Services) ReportTopProducts(param *domain.ReportQueryParam) ([]domain.TopProducts, int64, error) {
	// declaration
	var (
		res        []domain.TopProducts
		args       []any
		totalCount int64
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Warn("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Warn("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime, err = time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Warn("Invalid end_date format: %v", err)
			return nil, 0, err
		}
		endTime = endTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	query := `
	SELECT
		curr.id,
		curr.name,
		curr.count,
		curr.unit_quantity,
		curr.unit_per_pack,
		curr.total_amount,
		prev.total_amount AS previous_total_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.total_amount, 0) = 0 THEN 100
				ELSE ((curr.total_amount - prev.total_amount) * 100.0) / NULLIF(prev.total_amount, 0)
			END, 2
		) AS percent,
		COUNT(*) OVER() AS total_count
	FROM (
		SELECT
			p.id,
			p.name,
			SUM(ci.quantity) + FLOOR(SUM(ci.unit_quantity)::decimal / p.unit_per_pack) AS count,
			(SUM(ci.unit_quantity) % p.unit_per_pack) AS unit_quantity,
			p.unit_per_pack,
			SUM(ci.total_price) as total_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		WHERE (ci.updated_at+ interval '5 hours') BETWEEN ? AND ?
		GROUP BY p.id, p.name, p.unit_per_pack
	) AS curr
	LEFT JOIN (
		SELECT
			p.id,
			SUM(ci.total_price) AS total_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		WHERE (ci.updated_at+ interval '5 hours') BETWEEN ? AND ?
		GROUP BY p.id
	) AS prev ON curr.id = prev.id
`

	// Arguments for current and previous period
	args = append(args,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		beforeStart.Format(time.RFC3339),
		beforeEnd.Format(time.RFC3339),
	)

	// Filters
	where := " WHERE 1 = 1"
	if param.Search != "" {
		where += " AND curr.name ILIKE ?"
		args = append(args, "%"+param.Search+"%")
	}
	if param.StoreId != "" {
		where += " AND EXISTS (SELECT 1 FROM store_products sp2 WHERE sp2.product_id = curr.id AND sp2.store_id = ?)"
		args = append(args, param.StoreId)
	}
	if len(param.StoreIds) > 0 {
		where += " AND EXISTS (SELECT 1 FROM store_products sp3 WHERE sp3.product_id = curr.id AND sp3.store_id IN (?))"
		args = append(args, param.StoreIds)
	}

	// Sorting
	switch param.Order {
	case "min_count":
		query += where + " ORDER BY count ASC"
	case "max_count":
		query += where + " ORDER BY count DESC"
	case "min_amount":
		query += where + " ORDER BY total_amount ASC"
	case "max_amount":
		query += where + " ORDER BY total_amount DESC"
	default:
		query += where + " ORDER BY total_amount DESC"
	}

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top products: ", err)
		return nil, 0, err
	}

	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// get report top seller
func (s *Services) ReportTopSeller(param *domain.ReportQueryParam) ([]domain.TopSeller, int64, error) {
	var (
		res        []domain.TopSeller
		totalCount int64
		args       []any
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Warn("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		param.EndDate = endTime.Format(time.RFC3339)
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	// Main query
	query := `
	SELECT
		curr.id,
		curr.full_name,
		curr.store_name,
		curr.count,
		curr.total_amount,
		prev.total_amount AS previous_total_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.total_amount, 0) = 0 THEN 100
				ELSE ((curr.total_amount - prev.total_amount) * 100.0) / NULLIF(prev.total_amount, 0)
			END, 2
		) AS percent,
		COUNT(*) OVER() AS total_count
	FROM (
		SELECT
			e.id,
			e.full_name,
			st.name AS store_name,
			COUNT(s.id) AS count,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		INNER JOIN stores st ON s.store_id = st.id
		WHERE s.status = 'completed'
		AND s.sale_type = 'SALE'
		AND (s.completed_at + interval '5 hours')::date BETWEEN ? AND ?
		GROUP BY e.id, e.full_name, st.name
	) AS curr
	LEFT JOIN (
		SELECT
			e.id,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		WHERE s.status = 'completed'
		AND s.sale_type = 'SALE'
		AND (s.completed_at + interval '5 hours')::date BETWEEN ? AND ?
		GROUP BY e.id
	) AS prev ON curr.id = prev.id
`

	// First 4 args: 2 for current, 2 for previous range
	args = append(args,
		param.StartDate, param.EndDate,
		beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339),
	)

	// Optional filters
	where := " WHERE 1 = 1"
	if param.Search != "" {
		where += " AND curr.full_name ILIKE ?"
		args = append(args, "%"+param.Search+"%")
	}
	if param.StoreId != "" {
		where += " AND curr.store_name = (SELECT name FROM stores WHERE id = ?)"
		args = append(args, param.StoreId)
	}
	// check store_ids
	if len(param.StoreIds) > 0 {
		where += " AND curr.store_name IN (SELECT name FROM stores WHERE id IN (?))"
		args = append(args, param.StoreIds)
	}

	// Sorting
	switch param.Order {
	case "min_count":
		where += " ORDER BY curr.count ASC"
	case "max_count":
		where += " ORDER BY curr.count DESC"
	case "min_amount":
		where += " ORDER BY curr.total_amount ASC"
	case "max_amount":
		where += " ORDER BY curr.total_amount DESC"
	default:
		where += " ORDER BY curr.total_amount DESC"
	}

	// Pagination
	where += " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Final query
	finalQuery := query + where

	// Execute
	err = s.db.Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top seller: ", err)
		return nil, 0, err
	}
	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

func (s *Services) ReportTopStores(param *domain.ReportQueryParam) ([]domain.TopStores, int64, error) {
	var (
		res        []domain.TopStores
		totalCount int64
		args       []any
		startTime  time.Time
		endTime    time.Time
	)

	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		param.EndDate = endTime.Format(time.RFC3339)
	}

	// Get previous date range
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	// Main query with subqueries
	query := `
		SELECT
			curr.store_id AS id,
			stores.name,
			curr.count,
			curr.total_amount,
			COALESCE(prev.total_amount, 0) AS previous_total_amount,
			ROUND(
				CASE 
					WHEN COALESCE(prev.total_amount, 0) = 0 THEN 100
					ELSE ((curr.total_amount - prev.total_amount) * 100.0) / NULLIF(prev.total_amount, 0)
				END, 2
			) AS percent,
			COUNT(*) OVER() AS total_count
		FROM (
			SELECT
				sales.store_id,
				COUNT(*) AS count,
				SUM(sales.total_amount) AS total_amount
			FROM sales
			WHERE sales.status = 'completed'
			AND (sales.completed_at + interval '5 hours')::date BETWEEN ? AND ?
			GROUP BY sales.store_id
		) AS curr
		LEFT JOIN (
			SELECT
				sales.store_id,
				SUM(sales.total_amount) AS total_amount
			FROM sales
			WHERE sales.status = 'completed'
			AND (sales.completed_at + interval '5 hours')::date BETWEEN ? AND ?
			GROUP BY sales.store_id
		) AS prev ON curr.store_id = prev.store_id
		INNER JOIN stores ON curr.store_id = stores.id
	`

	// Add base arguments: current and previous period
	args = append(args,
		param.StartDate, param.EndDate,
		beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339),
	)

	// Dynamic filters
	whereClauses := []string{}
	if param.Search != "" {
		whereClauses = append(whereClauses, "stores.name ILIKE ?")
		args = append(args, "%"+param.Search+"%")
	}
	if param.StoreId != "" {
		whereClauses = append(whereClauses, "curr.store_id = ?")
		args = append(args, param.StoreId)
	}

	// Append filters
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Sorting
	switch param.Order {
	case "min_count":
		query += " ORDER BY count ASC"
	case "max_count":
		query += " ORDER BY count DESC"
	case "min_amount":
		query += " ORDER BY total_amount ASC"
	case "max_amount":
		query += " ORDER BY total_amount DESC"
	default:
		query += " ORDER BY total_amount DESC"
	}

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	// Extract total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}
	return res, totalCount, nil
}

// get dashboard bonus products
func (s *Services) ReportBonusProducts(param *domain.ReportQueryParam) ([]domain.BonusProducts, int64, error) {
	// declaration
	var (
		res        []domain.BonusProducts
		totalCount int64
		args       []any
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		param.EndDate = endTime.Format(time.RFC3339)
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	query := `
	SELECT
		curr.id,
		curr.name,
		curr.count,
		curr.bonus_amount,
		prev.bonus_amount AS previous_bonus_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.bonus_amount, 0) = 0 THEN 100
				ELSE ((curr.bonus_amount - prev.bonus_amount) * 100.0) / NULLIF(prev.bonus_amount, 0)
			END, 2
		) AS percent,
		COUNT(*) OVER() AS total_count
	FROM (
		SELECT
			p.id,
			p.name,
			SUM(eb.quantity) + SUM(eb.unit_quantity)/p.unit_per_pack || ',' || SUM(eb.unit_quantity)%p.unit_per_pack AS count,
			SUM(eb.bonus_amount) AS bonus_amount
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
	`

	// Dynamic JOIN and filters
	join := ""
	filter := " WHERE 1 = 1"
	if len(param.StoreIds) > 0 {
		join += " JOIN employees e ON eb.employee_id = e.id"
		filter += " AND e.store_id IN (?)"
		args = append(args, param.StoreIds)
	}
	if param.Search != "" {
		filter += " AND p.name ILIKE ?"
		args = append(args, "%"+param.Search+"%")
	}
	filter += " AND (eb.created_at + interval '5 hours')::date BETWEEN ? AND ?"
	args = append(args, param.StartDate, param.EndDate)

	// Close current subquery
	group := " GROUP BY p.id, p.name, p.unit_per_pack ) AS curr"
	query += join + filter + group

	// Add previous subquery
	query += `
	LEFT JOIN (
		SELECT
			p.id,
			SUM(eb.bonus_amount) AS bonus_amount
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
	`
	prevJoin := ""
	prevFilter := " WHERE 1 = 1"
	if len(param.StoreIds) > 0 {
		prevJoin += " JOIN employees e ON eb.employee_id = e.id"
		prevFilter += " AND e.store_id IN (?)"
		args = append(args, param.StoreIds)
	}
	prevFilter += " AND (eb.created_at + interval '5 hours')::date BETWEEN ? AND ?"
	args = append(args, beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339))

	query += prevJoin + prevFilter + " GROUP BY p.id ) AS prev ON curr.id = prev.id"

	// Sorting
	switch param.Order {
	case "min_count":
		query += " ORDER BY curr.count ASC"
	case "max_count":
		query += " ORDER BY curr.count DESC"
	case "min_amount":
		query += " ORDER BY curr.bonus_amount ASC"
	case "max_amount":
		query += " ORDER BY curr.bonus_amount DESC"
	default:
		query += " ORDER BY curr.bonus_amount DESC"
	}

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting bonus products: ", err)
		return nil, 0, err
	}
	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}
