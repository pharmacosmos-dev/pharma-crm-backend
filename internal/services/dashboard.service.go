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
		sale    domain.DashboardCountStatsSale
		product domain.DashboardCountStatsProduct
		income  domain.DashboardCountStatsIncome
		res     domain.DashboardCountStats
	)
	// check end date for empty string
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	// calculate before start and before end date
	beforeStart, beforeEnd := utils.BeforeDates(param.StartDate, param.EndDate)
	// queries
	var (
		args []any
		// get sale stats information
		querys = fmt.Sprintf(`
		SELECT 
			COUNT(CASE WHEN (completed_at + interval '5 hours')::date BETWEEN '%s' AND '%s' THEN id END) sale_count,
    		COUNT(CASE WHEN (completed_at + interval '5 hours')::date BETWEEN '%s' AND '%s' THEN id END) before_sale_count,
			SUM(CASE WHEN (completed_at + interval '5 hours')::date BETWEEN '%s' AND '%s' THEN total_amount END) AS sale_amount,
			SUM(CASE WHEN (completed_at + interval '5 hours')::date BETWEEN '%s' AND '%s' THEN total_amount END) AS before_sale_amount
		FROM sales WHERE status = 'completed' AND sale_type = 'SALE' `,
			param.StartDate, param.EndDate, beforeStart, beforeEnd,
			param.StartDate, param.EndDate, beforeStart, beforeEnd)
		// get stock stats information
		queryp = fmt.Sprintf(`
		SELECT
			ROUND(SUM(pack_quantity::numeric+(sp.unit_quantity%sp.unit_per_pack)::numeric/p.unit_per_pack), 2) AS stock_count,
			ROUND(SUM(pack_quantity::numeric+(sp.unit_quantity%sp.unit_per_pack)::numeric/p.unit_per_pack+COALESCE(ci_sold.quantity, 0)), 2) AS before_stock_count,
			ROUND(SUM(pack_quantity*retail_price)+SUM((retail_price/p.unit_per_pack)*(sp.unit_quantity%sp.unit_per_pack)), 2) AS stock_amount,
			ROUND(SUM((pack_quantity*retail_price)+((retail_price/p.unit_per_pack)*(sp.unit_quantity%sp.unit_per_pack))+COALESCE(ci_sold.amount, 0)), 2) AS before_stock_amount,
			SUM(CASE WHEN expire_date <= NOW() + INTERVAL '3 month' THEN pack_quantity ELSE 0 END) AS expiring_count,
			SUM(CASE WHEN expire_date <= NOW() + INTERVAL '3 month' THEN (pack_quantity*retail_price)+(retail_price/p.unit_per_pack) ELSE 0 END) AS expiring_amount,
			SUM(CASE WHEN expire_date <= NOW() + INTERVAL '3 month' THEN COALESCE(ci_sold.amount, 0) ELSE 0 END) AS before_expiring_amount
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN (
			SELECT store_product_id, SUM(quantity) AS quantity, SUM(quantity*unit_price) AS amount
			FROM cart_items
			JOIN sales s ON cart_items.sale_id = s.id
			WHERE s.completed_at::date >= '%s'
			AND s.completed_at::date <= '%s'
			AND s.status = 'completed'
			GROUP BY store_product_id
		) AS ci_sold ON ci_sold.store_product_id = sp.id
		WHERE 1 = 1 `, "%", "%", "%", "%", beforeStart, beforeEnd)
		// get total net income
		queryc = fmt.Sprintf(`
		SELECT
			ROUND(SUM(CASE WHEN completed_at::date BETWEEN '%s' AND '%s' THEN (ci.unit_price-sp.supply_price)*ci.quantity+ (CASE WHEN p.unit_per_pack > 0 THEN ((ci.unit_price - sp.supply_price)/p.unit_per_pack)*ci.unit_quantity ELSE 0 END) END), 2) AS income_amount,
			ROUND(SUM(CASE WHEN completed_at::date BETWEEN '%s' AND '%s' THEN (ci.unit_price-sp.supply_price)*ci.quantity + (CASE WHEN p.unit_per_pack > 0 THEN ((ci.unit_price - sp.supply_price)/p.unit_per_pack)*ci.unit_quantity ELSE 0 END) END), 2) AS before_income_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN sales s ON ci.sale_id = s.id
		WHERE s.status = 'completed' AND s.sale_type = 'SALE'`, param.StartDate, param.EndDate, beforeStart, beforeEnd)
		filter  = ""
		filterc = ""
	)
	// filter by several store ids
	if len(param.StoreIds) > 0 {
		filter += " AND store_id IN (?)"
		filterc += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
	}

	// get total sale count and amount
	querys += filter
	err := s.db.Raw(querys, args...).Scan(&sale).Error
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

	res.TotalSaleCount = sale.SaleCount
	res.BeforeSaleCount = sale.BeforeSaleCount
	res.TotalSaleAmount = sale.SaleAmount
	res.BeforeSaleAmount = sale.BeforeSaleAmount
	res.TotalProductCount = product.StockCount
	res.BeforeProductCount = product.BeforeStockCount
	res.StockTotalAmount = product.StockAmount
	res.BeforeStockAmount = product.BeforeStockAmount
	res.ExpiringSoonCount = product.ExpiringCount
	res.ExpiringSoonAmount = product.ExpiringAmount
	res.BeforeExpiringSoonAmount = product.BeforeExpiringAmount
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
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}

	switch param.Type {
	case "HALF_HOURLY":
		interval = "30 minutes"
		timeTruncCol = `
		DATE_TRUNC('hour', s.completed_at) + 
		INTERVAL '30 minutes' * FLOOR(EXTRACT(minute FROM s.completed_at) / 30)`
	case "HOURLY":
		interval = "1 hour"
		timeTruncCol = "DATE_TRUNC('hour', s.completed_at)"
	case "DAILY":
		interval = "1 day"
		timeTruncCol = "s.completed_at::date"
	case "WEEKLY":
		interval = "1 week"
		timeTruncCol = "DATE_TRUNC('week', s.completed_at)"
	case "MONTHLY":
		interval = "1 month"
		timeTruncCol = "DATE_TRUNC('month', s.completed_at)"
	case "YEARLY":
		interval = "1 year"
		timeTruncCol = "DATE_TRUNC('year', s.completed_at)"
	default:
		interval = "1 hour"
		timeTruncCol = "DATE_TRUNC('hour', s.completed_at)"
	}
	// WEEKLY tanlangan bo‘lsa startDate ni truncate qilamiz
	layout := "2006-01-02 15:04:05"
	startTimeStr := param.StartDate + " 00:00:00"
	startTimeParsed, _ := time.Parse(layout, startTimeStr)

	if param.Type == "WEEKLY" {
		// haftaning boshiga truncate qilish (Dushanba)
		offset := (int(startTimeParsed.Weekday()) + 6) % 7 // Monday = 0
		startTimeParsed = startTimeParsed.AddDate(0, 0, -offset)
	}

	startTime := startTimeParsed.Format(layout)
	endTime := param.EndDate + " 23:59:59"

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
		ts.period AS id,
		ts.period AS created_at,
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
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting chart info: %v", err)
		return res, err
	}
	return res, nil
}

// get dashboard top stores
func (s *Services) DashboardTopStores(param *domain.DashboardQueryParam) ([]domain.TopStores, error) {
	// declaration
	var (
		res []domain.TopStores
	)
	// query
	var (
		args   []any
		query  = `SELECT stores.id, stores.name, COUNT(*) AS count, SUM(sales.total_amount) AS total_amount FROM sales INNER JOIN stores ON sales.store_id = stores.id`
		filter = " WHERE sales.status = 'completed'"
		group  = " GROUP BY stores.id"
		order  = " ORDER BY total_amount DESC"
	)
	if param.StoreId != "" {
		filter += " AND sales.store_id = ?"
		args = append(args, param.StoreId)
	}
	if param.StartDate != "" && param.EndDate == "" {
		filter += " AND sales.completed_at::date = ?"
		args = append(args, param.StartDate)
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND sales.completed_at::date >= ? AND sales.completed_at::date <= ?"
		args = append(args, param.StartDate, param.EndDate)
	}

	var q = query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	err := s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return res, nil
}

// get dashboard top products
func (s *Services) DashboardTopProducts(param *domain.DashboardQueryParam) ([]domain.TopProducts, error) {
	// declaration
	var (
		res []domain.TopProducts
	)
	// query
	var (
		args  []any
		query = `
		SELECT
			p.id, p.name,
			CAST(SUM(ci.quantity) AS TEXT) || ',' || CAST(SUM(ci.unit_quantity) AS TEXT) AS count,
			sum(ci.total_price) as total_amount
		FROM cart_items ci
			JOIN store_products sp ON ci.store_product_id = sp.id
			JOIN products p on sp.product_id = p.id`
		filter = " WHERE 1 = 1 "
		group  = " GROUP BY p.id, p.name"
		order  = " ORDER BY total_amount DESC"
	)
	if param.StoreId != "" {
		filter += " AND sp.store_id = ?"
		args = append(args, param.StoreId)
	}

	if len(param.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?)"
		args = append(args, param.StoreIds)
	}

	if len(param.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?)"
		args = append(args, param.StoreIds)
	}

	if param.StartDate != "" && param.EndDate == "" {
		filter += " AND ci.updated_at::date = ?"
		args = append(args, param.StartDate)
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND ci.updated_at::date >= ? AND ci.updated_at::date <= ?"
		args = append(args, param.StartDate, param.EndDate)
	}
	args = append(args, param.Limit, param.Offset)
	var q = query + filter + group + order + " LIMIT ? OFFSET ?"
	err := s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top products: ", err)
		return nil, err
	}

	return res, nil
}

// get dashboard bonus products
func (s *Services) DashboardBonusProducts(param *domain.DashboardQueryParam) ([]domain.BonusProducts, error) {
	// declaration
	var res []domain.BonusProducts
	// checking end date for empty
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	// query
	var (
		args  []any
		query = `
		SELECT
			p.id, p.name,
			SUM(eb.quantity) + SUM(eb.unit_quantity)/p.unit_per_pack || ',' || SUM(eb.unit_quantity)%p.unit_per_pack AS count,
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

	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (eb.created_at + interval '5 hours')::date BETWEEN ? AND ?"
		args = append(args, param.StartDate, param.EndDate)
	}
	query = query + join + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting bonus products: ", err)
		return nil, err
	}

	return res, nil
}

// get dashboard top seller
func (s *Services) DashboardTopSeller(param *domain.DashboardQueryParam) ([]domain.TopSeller, error) {
	// declaration
	var (
		res []domain.TopSeller
	)
	// query
	var (
		args  []any
		query = `
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
		filter = "	WHERE s.status = 'completed' AND s.sale_type = 'SALE' "
		group  = " GROUP BY e.id, st.id "
		order  = " ORDER BY total_amount DESC"
		offset = " LIMIT ? OFFSET ?"
	)
	if param.StoreId != "" {
		filter += " AND s.store_id = ?"
		args = append(args, param.StoreId)
	}
	// check store_ids
	if len(param.StoreIds) > 0 {
		filter += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
	}
	// check end_date for empty string
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND s.completed_at::date BETWEEN ? AND  ?"
		args = append(args, param.StartDate, param.EndDate)
	}
	args = append(args, param.Limit, param.Offset)
	var q = query + filter + group + order + offset
	err := s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting top seller: ", err)
		return nil, err
	}

	return res, nil
}

// get payment
func (s *Services) DashboardPayments(param *domain.DashboardQueryParam) ([]domain.DashboardPayment, error) {
	// check end date for empty string
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}
	var (
		res   []domain.DashboardPayment
		query = `
		SELECT
			pt.id, pt.name,
			SUM(sp.amount) AS amount,
			COUNT(sp.id) AS count
		FROM sale_payments sp
		JOIN payment_types pt ON sp.payment_type_id = pt.id
		`
		join   = "JOIN sales s ON sp.sale_id = s.id"
		filter = " WHERE 1 = 1 "
		group  = " GROUP BY pt.id "
		order  = " ORDER BY amount DESC;"
		args   = []any{}
	)
	if len(param.StoreIds) > 0 {
		filter += " AND s.store_id IN (?) "
		join = "JOIN sales s ON sp.sale_id = s.id"
		args = append(args, param.StoreIds)
	}

	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND (sp.created_at + interval '5 hours')::date BETWEEN ? AND ? "
		args = append(args, param.StartDate, param.EndDate)
	}
	query = query + join + filter + group + order
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting dashboard payment stats: %v", err)
		return res, err
	}

	return res, nil
}

// get dashboard transaction types
func (s *Services) DashboardTransaction(param *domain.DashboardQueryParam) ([]domain.DashboardTransaction, error) {
	if param.EndDate == "" {
		param.EndDate = param.StartDate
	}

	res := []domain.DashboardTransaction{}
	args := []any{param.StartDate, param.EndDate}
	whereClause := `s.status = 'completed' AND s.sale_type = 'SALE' AND (s.completed_at + interval '5 hours')::date BETWEEN ? AND ?`

	if len(param.StoreIds) > 0 {
		whereClause += ` AND s.store_id IN (?)`
		args = append(args, param.StoreIds)
	}

	saleQuery := fmt.Sprintf(`
	SELECT
		'Товары' AS name,
		SUM(sub.total_amount) as amount,
		COALESCE(SUM(sub.quantity) || ',' || SUM(sub.unit_quantity), '0') as count
	FROM (
		SELECT s.total_amount, SUM(ci.quantity) as quantity, SUM(ci.unit_quantity) as unit_quantity
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE %s
		GROUP BY s.id, s.total_amount
	) sub`, whereClause)

	// Reset args for returns
	argsReturn := []any{param.StartDate, param.EndDate}
	whereClauseReturn := `s.status = 'completed' AND s.sale_type = 'RETURN' AND (s.completed_at + interval '5 hours')::date BETWEEN ? AND ?`

	if len(param.StoreIds) > 0 {
		whereClauseReturn += ` AND s.store_id IN (?)`
		argsReturn = append(argsReturn, param.StoreIds)
	}

	returnQuery := fmt.Sprintf(`
	SELECT
		'Возвраты' AS name,
		SUM(sub.total_amount) as amount,
		COALESCE(SUM(sub.quantity) || ',' || SUM(sub.unit_quantity), '0') as count
	FROM (
		SELECT s.total_amount, SUM(ci.quantity) as quantity, SUM(ci.unit_quantity) as unit_quantity
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE %s
		GROUP BY s.id, s.total_amount
	) sub`, whereClauseReturn)

	fullQuery := fmt.Sprintf(`%s UNION ALL %s`, saleQuery, returnQuery)

	// Use gorm’s parameter binding
	err := s.db.Raw(fullQuery, append(args, argsReturn...)...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}
	return res, nil
}
