package services

import (
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// get dashboard count and amount data
func (s *Services) DashboardTotalCountStats(param *domain.DashboardQueryParam) (*domain.DashboardCountStats, error) {
	// declarations
	var (
		sale      domain.DashboardCountStatsSale
		product   domain.DashboardCountStatsProduct
		income    domain.DashboardCountStatsIncome
		imported  domain.DashboardImport
		res       domain.DashboardCountStats
		startTime time.Time
		endTime   time.Time
	)

	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, err
	}

	if param.EndDate == "" { // get end time if end_date will be empty string 23 hour and 59 minute
		endTime = startTime.Add(time.Minute * 1439)
	}

	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, err
		}
	}

	// Calculate before period
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	// Format all timestamps for SQL
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")
	beforeStartStr := beforeStart.Format("2006-01-02 15:04:05")
	beforeEndStr := beforeEnd.Format("2006-01-02 15:04:05")

	// queries
	var (
		args  []any
		args1 = []any{startStr, endStr, beforeStartStr, beforeEndStr}

		// get sale stats information
		querys = fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN sale_type = 'SALE' AND (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN id END) AS sale_count,
			COUNT(CASE WHEN sale_type = 'SALE' AND (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN id END) AS before_sale_count,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN
				CASE WHEN sale_type = 'SALE' THEN total_amount
					 WHEN sale_type = 'RETURN' THEN -total_amount
				END ELSE 0 END) AS sale_amount,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN
				CASE WHEN sale_type = 'SALE' THEN total_amount
					 WHEN sale_type = 'RETURN' THEN -total_amount
				END ELSE 0 END) AS before_sale_amount
		FROM sales
		WHERE status = 'completed'`,
			startStr, endStr, beforeStartStr, beforeEndStr,
			startStr, endStr, beforeStartStr, beforeEndStr)

		queryp = fmt.Sprintf(`
		SELECT
			ROUND(SUM(pack_quantity::numeric + (sp.unit_quantity %% p.unit_per_pack)::numeric / p.unit_per_pack), 2) AS stock_count,
			ROUND(SUM(pack_quantity::numeric + (sp.unit_quantity %% p.unit_per_pack)::numeric / p.unit_per_pack + COALESCE(ci_sold.quantity, 0)), 2) AS before_stock_count,
			ROUND(SUM(pack_quantity * retail_price) + SUM((retail_price / p.unit_per_pack) * (sp.unit_quantity %% p.unit_per_pack)), 2) AS stock_amount,
			ROUND(SUM((pack_quantity * retail_price) + ((retail_price / p.unit_per_pack) * (sp.unit_quantity %% p.unit_per_pack)) + COALESCE(ci_sold.amount, 0)), 2) AS before_stock_amount,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN pack_quantity ELSE 0 END), 2) AS expiring_count,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN (pack_quantity * retail_price) + (retail_price / p.unit_per_pack) ELSE 0 END), 2) AS expiring_amount,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN COALESCE(ci_sold.amount, 0) ELSE 0 END), 2) AS before_expiring_amount,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN pack_quantity ELSE 0 END), 2) AS expired_count,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN (pack_quantity * retail_price) + (retail_price / p.unit_per_pack) ELSE 0 END), 2) AS expired_amount,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN COALESCE(ci_sold.amount, 0) ELSE 0 END), 2) AS before_expired_amount
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN (
			SELECT store_product_id, SUM(quantity) AS quantity, SUM(quantity * unit_price) AS amount
			FROM cart_items
			JOIN sales s ON cart_items.sale_id = s.id
			WHERE s.completed_at BETWEEN '%s' AND '%s'
			  AND s.status = 'completed'
			GROUP BY store_product_id
		) AS ci_sold ON ci_sold.store_product_id = sp.id
		WHERE 1 = 1`, beforeStartStr, beforeEndStr)

		queryc = fmt.Sprintf(`
		SELECT
			ROUND(SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN (ci.unit_price - sp.supply_price) * ci.quantity +
				(CASE WHEN p.unit_per_pack > 0 THEN ((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity ELSE 0 END) END), 2) AS income_amount,
			ROUND(SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN (ci.unit_price - sp.supply_price) * ci.quantity +
				(CASE WHEN p.unit_per_pack > 0 THEN ((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity ELSE 0 END) END), 2) AS before_income_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN sales s ON ci.sale_id = s.id
		WHERE s.status = 'completed' AND s.sale_type = 'SALE'`,
			startStr, endStr, beforeStartStr, beforeEndStr)

		query1 = `
		SELECT
			COALESCE(SUM(CASE
				WHEN im.created_at BETWEEN ? AND ?
				THEN imd.received_count * imd.retail_price_vat ELSE 0
			END), 0) AS import_amount,
			COALESCE(SUM(CASE
				WHEN im.created_at BETWEEN ? AND ?
				THEN imd.received_count * imd.retail_price_vat ELSE 0
			END), 0) AS before_import_amount
		FROM import_details imd
		JOIN imports im ON imd.import_id = im.id
		WHERE im.status = 'new' AND im.entry_type = 1`

		filter  = ""
		filterc = ""
	)
	// filter by several store ids
	if len(param.StoreIds) > 0 {
		filter += " AND store_id IN (?)"
		filterc += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
		query1 += " AND im.store_id IN (?)"
		args1 = append(args1, param.StoreIds)
	}

	// Execute queries
	querys += filter
	err = s.db.Raw(querys, args...).Scan(&sale).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	// get total product count
	queryp += filter
	err = s.db.Raw(queryp, args...).Scan(&product).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	// get total net income
	queryc += filterc
	err = s.db.Raw(queryc, args...).Scan(&income).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	err = s.db.Raw(query1, args1...).Scan(&imported).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	// Map results
	res.ImportAmount = imported.ImportAmount
	res.BeforeImportAmount = imported.BeforeImportAmount
	res.TotalSaleCount = sale.SaleCount
	res.BeforeSaleCount = sale.BeforeSaleCount
	res.TotalSaleAmount = sale.SaleAmount
	res.BeforeSaleAmount = sale.BeforeSaleAmount
	res.TotalProductCount = product.StockCount
	res.BeforeProductCount = product.BeforeStockCount
	res.StockTotalAmount = product.StockAmount
	res.BeforeStockAmount = product.BeforeStockAmount
	res.ExpiringSoonCount = product.ExpiringCount
	res.ExpiredSoonCount = product.ExpiredCount
	res.ExpiringSoonAmount = product.ExpiringAmount
	res.ExpiredSoonAmount = product.ExpiredAmount
	res.BeforeExpiringSoonAmount = product.BeforeExpiringAmount
	res.BeforeExpiredSoonAmount = product.BeforeExpiredAmount
	res.TotalNetIncome = income.IncomeAmount
	res.BeforeTotalNetIncome = income.BeforeIncomeAmount

	return &res, nil
}

// get dashboard chart stats data list
func (s *Services) DashboardChartStats(param *domain.DashboardQueryParam) ([]domain.ChartResponse, error) {
	var res []domain.ChartResponse
	// vaqt formatlarini aniqlash
	var (
		interval     string
		timeTruncCol string
	)
	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, err
	}

	endTime := startTime
	if param.EndDate == "" { // get end time if end_date will be empty string, so add  23 hour and 59 minute
		endTime = startTime.Add(time.Minute * 1439)
	}

	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, err
		}
	}

	switch param.Type {
	case "HALF_HOURLY":
		interval = "30 minutes"
		timeTruncCol = `
		DATE_TRUNC('hour', s.completed_at + INTERVAL '5 hours') +
		INTERVAL '30 minutes' * FLOOR(EXTRACT(minute FROM s.completed_at + INTERVAL '5 hours') / 30)`
	case "HOURLY":
		interval = "1 hour"
		timeTruncCol = "DATE_TRUNC('hour', s.completed_at + INTERVAL '5 hours')"
	case "DAILY":
		interval = "1 day"
		timeTruncCol = "(s.completed_at + INTERVAL '5 hours')::date"
	case "WEEKLY":
		interval = "1 week"
		timeTruncCol = "DATE_TRUNC('week', s.completed_at + INTERVAL '5 hours')"
	case "MONTHLY":
		interval = "1 month"
		timeTruncCol = "DATE_TRUNC('month', s.completed_at + INTERVAL '5 hours')"
	case "YEARLY":
		interval = "1 year"
		timeTruncCol = "DATE_TRUNC('year', s.completed_at + INTERVAL '5 hours')"
	default:
		interval = "1 hour"
		timeTruncCol = "DATE_TRUNC('hour', s.completed_at + INTERVAL '5 hours')"
	}
	// WEEKLY tanlangan bo‘lsa startDate ni truncate qilamiz

	if param.Type == "WEEKLY" {
		// haftaning boshiga truncate qilish (Dushanba)
		offset := (int(startTime.Weekday()) + 6) % 7 // Monday = 0
		startTime = startTime.AddDate(0, 0, -offset)
	}

	args := []any{startTime, endTime, interval}

	// qo‘shimcha filterlar
	filter := ""
	if len(param.StoreIds) > 0 {
		filter += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
	}

	// yakuniy query
	query := fmt.Sprintf(`
	WITH time_series AS (
		SELECT generate_series(
			?::timestamp,
			?::timestamp,
			?::interval
		) AS period
	)
	SELECT
		ts.period - INTERVAL '5 hours' AS id,
		ts.period - INTERVAL '5 hours' AS created_at,
		COUNT(s.id) AS count,
		COALESCE(SUM(s.total_amount), 0) AS total_amount
	FROM time_series ts
	LEFT JOIN sales s ON
		%s = ts.period
		AND s.status = 'completed'
		AND s.sale_type = 'SALE'
	%s
	GROUP BY ts.period
	ORDER BY ts.period;
	`, timeTruncCol, filter)

	// bajarish
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting chart info: %v", err)
		return res, err
	}
	return res, nil
}

// get dashboard top stores
func (s *Services) DashboardTopStores(param *domain.DashboardQueryParam) ([]domain.TopStores, error) {
	var res []domain.TopStores

	var (
		args   []any
		query  = `SELECT stores.id, stores.name, COUNT(*) AS count, SUM(sales.total_amount) AS total_amount FROM sales INNER JOIN stores ON sales.store_id = stores.id`
		filter = ` WHERE sales.status = 'completed'`
		group  = ` GROUP BY stores.id`
		order  = ` ORDER BY total_amount DESC`
	)

	// Parse and apply date filters
	var startStr, endStr string
	if param.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Error("Invalid start_date format: %v", err)
			return nil, err
		}
		startStr = startTime.Format("2006-01-02 15:04:05")

		// if end_date is empty → use start_date
		var endTime time.Time
		if param.EndDate != "" {
			endTime, err = time.Parse(time.RFC3339, param.EndDate)
			if err != nil {
				s.log.Error("Invalid end_date format: %v", err)
				return nil, err
			}
		} else {
			endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endStr = endTime.Format("2006-01-02 15:04:05")

		// Apply filter
		filter += " AND sales.completed_at BETWEEN ? AND ?"
		args = append(args, startStr, endStr)
	}

	// Store filter
	if param.StoreId != "" {
		filter += " AND sales.store_id = ?"
		args = append(args, param.StoreId)
	}

	// Limit & Offset
	var q = query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute query
	err := s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("Failed to get top stores: %v", err)
		return nil, err
	}

	return res, nil
}

// get dashboard top products
func (s *Services) DashboardTopProducts(param *domain.DashboardQueryParam) ([]domain.TopProducts, error) {
	var res []domain.TopProducts

	var (
		args  []any
		query = `
		SELECT
			p.id, p.name,
			SUM(ci.quantity) + FLOOR(SUM(ci.unit_quantity)::decimal / p.unit_per_pack) AS count,
			(SUM(ci.unit_quantity) % p.unit_per_pack) AS unit_quantity,
			p.unit_per_pack,
			sum(ci.total_price) as total_amount
		FROM cart_items ci
			JOIN store_products sp ON ci.store_product_id = sp.id
			JOIN products p on sp.product_id = p.id`
		filter = ` WHERE 1 = 1`
		group  = ` GROUP BY p.id, p.name, p.unit_per_pack`
		order  = ` ORDER BY total_amount DESC`
	)

	// Filter by one store
	if param.StoreId != "" {
		filter += ` AND sp.store_id = ?`
		args = append(args, param.StoreId)
	}

	// Filter by multiple stores
	if len(param.StoreIds) > 0 {
		filter += ` AND sp.store_id IN (?)`
		args = append(args, param.StoreIds)
	}

	// Parse RFC3339 date-time range
	if param.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, param.StartDate)
		if err != nil {
			s.log.Error("Invalid start_date format: %v", err)
			return nil, err
		}
		startStr := startTime.Format("2006-01-02 15:04:05")

		var endTime time.Time
		if param.EndDate != "" {
			endTime, err = time.Parse(time.RFC3339, param.EndDate)
			if err != nil {
				s.log.Error("Invalid end_date format: %v", err)
				return nil, err
			}
		} else {
			endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endStr := endTime.Format("2006-01-02 15:04:05")

		filter += " AND ci.updated_at BETWEEN ? AND ?"
		args = append(args, startStr, endStr)
	}

	// Add pagination
	args = append(args, param.Limit, param.Offset)
	query = query + filter + group + order + " LIMIT ? OFFSET ?"

	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top products: %v", err)
		return nil, err
	}

	return res, nil
}

// get dashboard bonus products
func (s *Services) DashboardBonusProducts(param *domain.DashboardQueryParam) ([]domain.BonusProducts, error) {
	var res []domain.BonusProducts

	// query
	var (
		startTime, endTime time.Time
		args               []any
		query              = `
		SELECT
			p.id, p.name,
			SUM(eb.quantity) AS count,
			SUM(eb.bonus_amount) AS bonus_amount
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
		`
		join   = ""
		filter = " WHERE 1 = 1 "
		group  = " GROUP BY p.id "
		order  = " ORDER BY count DESC"
	)

	// check store_ids
	if len(param.StoreIds) > 0 {
		filter += " AND e.store_id IN (?) "
		join = " JOIN employees e ON eb.employee_id = e.id "
		args = append(args, param.StoreIds)
	}

	// Parse RFC3339 start va end vaqtlar
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	filter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, startStr, endStr)

	// Limit / Offset
	query = query + join + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting bonus products: ", err)
		return nil, err
	}

	return res, nil
}

// get dashboard top seller
func (s *Services) DashboardTopSeller(param *domain.DashboardQueryParam) ([]domain.TopSeller, error) {
	var res []domain.TopSeller

	var (
		startTime, endTime time.Time
		args               []any
		query              = `
		SELECT
			e.id,
			e.full_name,
			st.name AS store_name,
			COUNT(s.id) AS count,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		INNER JOIN stores st ON s.store_id = st.id
		`
		filter = " WHERE s.status = 'completed' AND s.sale_type = 'SALE'"
		group  = " GROUP BY e.id, st.id"
		order  = " ORDER BY total_amount DESC"
		offset = " LIMIT ? OFFSET ?"
	)

	// Filter by one store
	if param.StoreId != "" {
		filter += " AND s.store_id = ?"
		args = append(args, param.StoreId)
	}

	// Filter by multiple stores
	if len(param.StoreIds) > 0 {
		filter += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
	}

	// Date filter — RFC3339 parse
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	filter += " AND s.completed_at BETWEEN ? AND ?"
	args = append(args, startStr, endStr)

	// Pagination
	args = append(args, param.Limit, param.Offset)

	// Build and run query
	var q = query + filter + group + order + offset
	err = s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top seller: %v", err)
		return nil, err
	}

	return res, nil
}

// get payment
func (s *Services) DashboardPayments(param *domain.DashboardQueryParam) ([]domain.DashboardPayment, error) {
	var res []domain.DashboardPayment

	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return res, err
	}
	endTime := startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return res, err
		}
	}

	// Format timestamps for SQL
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	query := `
		SELECT
			pt.id, pt.name,
			SUM(sp.amount) AS amount,
			COUNT(sp.id) AS count
		FROM sale_payments sp
		JOIN payment_types pt ON sp.payment_type_id = pt.id
	`

	join := "JOIN sales s ON sp.sale_id = s.id"
	filter := " WHERE 1 = 1 "
	group := " GROUP BY pt.id "
	order := " ORDER BY amount DESC;"
	args := []any{}

	// Store filter
	if len(param.StoreIds) > 0 {
		filter += " AND s.store_id IN (?) "
		join = "JOIN sales s ON sp.sale_id = s.id"
		args = append(args, param.StoreIds)
	}

	// Date range filter (with time)
	filter += " AND (sp.created_at + interval '5 hours') BETWEEN ? AND ? "
	args = append(args, startStr, endStr)

	query = query + join + filter + group + order
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting dashboard payment stats: %v", err)
		return res, err
	}

	return res, nil
}

func (s *Services) DashboardTransaction(param *domain.DashboardQueryParam) ([]domain.DashboardTransaction, error) {
	var startTime, endTime time.Time
	// Parse datetimes
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, err
	}
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Format for SQL
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	res := []domain.DashboardTransaction{}

	// Build WHERE clause for sales
	args := []any{startStr, endStr}
	whereClause := `s.status = 'completed' AND s.sale_type = 'SALE' AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?`

	if len(param.StoreIds) > 0 {
		whereClause += ` AND s.store_id IN (?)`
		args = append(args, param.StoreIds)
	}

	saleQuery := fmt.Sprintf(`
	SELECT
		'Товары' AS name,
		SUM(sub.total_amount) as amount,
		COALESCE(ROUND(SUM(sub.quantity + (sub.unit_quantity / 100.0)), 0),0) AS count
	FROM (
		SELECT s.total_amount, SUM(ci.quantity) as quantity, SUM(ci.unit_quantity) as unit_quantity
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE %s
		GROUP BY s.id, s.total_amount
	) sub`, whereClause)

	// Reset args for returns
	argsReturn := []any{startStr, endStr}
	whereClauseReturn := `s.status = 'completed' AND s.sale_type = 'RETURN' AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?`

	if len(param.StoreIds) > 0 {
		whereClauseReturn += ` AND s.store_id IN (?)`
		argsReturn = append(argsReturn, param.StoreIds)
	}

	returnQuery := fmt.Sprintf(`
	SELECT
		'Возвраты' AS name,
		SUM(sub.total_amount) as amount,
		COALESCE(ROUND(SUM(sub.quantity + (sub.unit_quantity / 100.0)), 0),0) AS count
	FROM (
		SELECT s.total_amount, SUM(ci.quantity) as quantity, SUM(ci.unit_quantity) as unit_quantity
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE %s
		GROUP BY s.id, s.total_amount
	) sub`, whereClauseReturn)

	// Combine both queries
	fullQuery := fmt.Sprintf(`%s UNION ALL %s`, saleQuery, returnQuery)

	// Execute query with both sets of arguments
	err = s.db.Raw(fullQuery, append(args, argsReturn...)...).Scan(&res).Error
	if err != nil {
		s.log.Error("Error fetching dashboard transaction stats: %v", err)
		return res, err
	}

	return res, nil
}
