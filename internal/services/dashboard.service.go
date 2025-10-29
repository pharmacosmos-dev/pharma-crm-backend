package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

// get dashboard count and amount data
func (s *Services) DashboardTotalCountStats(ctx context.Context, param *domain.DashboardQueryParam) (*domain.DashboardCountStats, error) {
	// declarations
	var (
		res       domain.DashboardCountStats
		startTime time.Time
		endTime   time.Time
	)

	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Errorf("could not parse start_date format: %v", err)
		return nil, domain.InvalidTimeFormatError
	}

	if param.EndDate == "" { // get end time if end_date will be empty string 23 hour and 59 minute
		endTime = startTime.Add(time.Minute * 1439)
	}

	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Errorf("could not parse end_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
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
		args []any
		// get sale stats information
		querys = fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.id END) AS sale_count,
			COUNT(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.id END) AS before_sale_count,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.total_amount ELSE 0 END) AS sale_amount,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.total_amount ELSE 0 END) AS before_sale_amount,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.total_discount ELSE 0 END) AS discount_amount,
			SUM(CASE WHEN (completed_at + interval '5 hours') BETWEEN '%s' AND '%s' THEN sales.total_discount ELSE 0 END) AS before_discount_amount
		FROM sales
		LEFT JOIN stores st on sales.store_id = st.id
		WHERE stage IN(9, 11)
		`,
			startStr, endStr, beforeStartStr, beforeEndStr,
			startStr, endStr, beforeStartStr, beforeEndStr,
			startStr, endStr, beforeStartStr, beforeEndStr)

		queryp = fmt.Sprintf(`
		SELECT
			ROUND(SUM(sp.unit_quantity / p.unit_per_pack), 2) AS stock_count,
			ROUND(SUM(sp.unit_quantity / p.unit_per_pack + COALESCE(ci_sold.quantity, 0)), 2) AS before_stock_count,
			ROUND(SUM((retail_price / p.unit_per_pack) * sp.unit_quantity), 2) AS stock_amount,
			ROUND(SUM((retail_price / p.unit_per_pack) * sp.unit_quantity  + COALESCE(ci_sold.amount, 0)), 2) AS before_stock_amount,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN (sp.unit_quantity/p.unit_per_pack) ELSE 0 END), 2) AS expiring_count,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END), 2) AS expiring_amount,
			ROUND(SUM(CASE WHEN expire_date > NOW() AND expire_date <= NOW() + INTERVAL '3 month' THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) + COALESCE(ci_sold.amount, 0) ELSE 0 END), 2) AS before_expiring_amount,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN (sp.unit_quantity/p.unit_per_pack) ELSE 0 END), 2) AS expired_count,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END),2) AS expired_amount,
			ROUND(SUM(CASE WHEN expire_date <= NOW() THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) + COALESCE(ci_sold.amount, 0) ELSE 0 END), 2) AS before_expired_amount
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN stores st ON sp.store_id = st.id
		LEFT JOIN (
			SELECT store_product_id, SUM(quantity) AS quantity, SUM(quantity * unit_price) AS amount
			FROM cart_items
			JOIN sales s ON cart_items.sale_id = s.id
			WHERE s.completed_at BETWEEN '%s' AND '%s'
			AND s.stage IN(9, 11)
			GROUP BY store_product_id
		) AS ci_sold ON ci_sold.store_product_id = sp.id
		WHERE 1 = 1
		`, beforeStartStr, beforeEndStr)

		queryc = fmt.Sprintf(`
		SELECT
			ROUND(SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN ((ci.unit_price - sp.supply_price)/p.unit_per_pack) * ci.unit_quantity ELSE 0 END), 2) AS income_amount,
			ROUND(SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN ((ci.unit_price - sp.supply_price)/p.unit_per_pack) * ci.unit_quantity ELSE 0 END), 2) AS before_income_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN sales s ON ci.sale_id = s.id
		WHERE s.stage IN(9, 11) AND s.sale_type = 'SALE'`,
			startStr, endStr, beforeStartStr, beforeEndStr)

		query24h = `
		SELECT
			-- 24 soatdan eski (hammasi)
			COALESCE(SUM(
				CASE 
					WHEN im.created_at < NOW() - interval '24 hour'
					THEN imd.received_count * imd.retail_price_vat 
					ELSE 0 
				END
			), 0) AS import_amount,
		
			-- 24–48 soat oralig‘i
			COALESCE(SUM(
				CASE
					WHEN im.created_at BETWEEN NOW() - interval '48 hour' AND NOW() - interval '24 hour'
					THEN imd.received_count * imd.retail_price_vat 
					ELSE 0 
				END
			), 0) AS not_last_24h_import_amount
		
		FROM import_details imd
		JOIN imports im ON imd.import_id = im.id
		LEFT JOIN stores st ON im.store_id = st.id
		WHERE im.status = 'new'
		  AND im.entry_type = 1`

		queryImportCountNot24 = `
		SELECT COUNT(*)
		FROM imports im
		LEFT JOIN stores st ON im.store_id = st.id
		WHERE im.status = 'new'
 		 AND im.entry_type = 1
  		 AND im.created_at < NOW() - interval '24 hour'
`

		filter  = ""
		filterc = ""
	)

	// filter by several store ids
	if len(param.StoreIds) > 0 {
		filter += " AND store_id IN (?)"
		filterc += " AND s.store_id IN (?)"
		args = append(args, param.StoreIds)
		query24h += " AND im.store_id IN (?)"
	}

	// filter by company_id
	if param.CompanyId != "" {
		filter += " AND st.company_id = ?"
		filterc += " AND p.company_id = ?"
		args = append(args, param.CompanyId)
		query24h += " AND st.company_id = ?"
	}

	// Execute queries
	var sale domain.DashboardCountStatsSale
	querys += filter
	err = s.db.WithContext(ctx).Debug().Raw(querys, args...).Scan(&sale).Error
	if err != nil {
		s.log.Errorf("could not get total sale amounts: %v", err)
		return nil, domain.InternalServerError
	}
	// get total product count
	var product domain.DashboardCountStatsProduct
	queryp += filter
	err = s.db.WithContext(ctx).Raw(queryp, args...).Scan(&product).Error
	if err != nil {
		s.log.Errorf("could not get total product_amounts: %v", err)
		return nil, domain.InternalServerError
	}
	// get total net income
	var income domain.DashboardCountStatsIncome
	queryc += filterc
	err = s.db.WithContext(ctx).Raw(queryc, args...).Scan(&income).Error
	if err != nil {
		s.log.Errorf("could not get total income: %v", err)
		return nil, domain.InternalServerError
	}
	var imported domain.DashboardImport
	err = s.db.WithContext(ctx).Raw(query24h, args...).Scan(&imported).Error
	if err != nil {
		s.log.Errorf("could not get import_count for_24: %v", err)
		return nil, domain.InternalServerError
	}
	var notLast24HImportCount int
	queryImportCountNot24 += filter
	err = s.db.WithContext(ctx).Raw(queryImportCountNot24, args...).Scan(&notLast24HImportCount).Error

	if err != nil {
		s.log.Errorf("could not get import_count for_not_24: %v", err)
		return nil, domain.InternalServerError
	}
	// Map results
	res.ImportAmount = imported.ImportAmount
	res.NotLast24HImportCount = float64(notLast24HImportCount)
	res.NotLast24HImportAmount = imported.NotLast24hImportAmount
	res.TotalSaleCount = sale.SaleCount
	res.BeforeSaleCount = sale.BeforeSaleCount
	res.TotalSaleAmount = sale.SaleAmount
	res.BeforeSaleAmount = sale.BeforeSaleAmount
	res.DiscountAmount = sale.DiscountAmount
	res.BeforeDiscountAmount = sale.BeforeDiscountAmount
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
func (s *Services) DashboardChartStats(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.ChartResponse, error) {
	var (
		res          []domain.ChartResponse
		interval     string
		timeTruncCol string
	)

	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Errorf("could not parse start_date format: %v", err)
		return nil, domain.InvalidTimeFormatError
	}

	endTime := startTime
	if params.EndDate == "" { // get end time if end_date will be empty string, so add  23 hour and 59 minute
		endTime = startTime.Add(time.Minute * 1439)
	}

	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Errorf("could not parse end_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
	}

	// Group type
	switch params.Type {
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

	if params.Type == "WEEKLY" {
		// haftaning boshiga truncate qilish (Dushanba)
		offset := (int(startTime.Weekday()) + 6) % 7 // Monday = 0
		startTime = startTime.AddDate(0, 0, -offset)
	}

	args := []any{startTime, endTime, interval}
	fmt.Println("--->>> ", params.StoreIds)
	// qo‘shimcha filterlar
	storeFilter := ""
	if len(params.StoreIds) > 0 {
		storeFilter += " AND s.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	companyFilter := ""
	if params.CompanyId != "" {
		companyFilter += " AND st.company_id = ?"
		args = append(args, params.CompanyId)
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
		AND s.stage IN (9, 11)
		AND s.sale_type = 'SALE'
		%s
	LEFT JOIN stores st ON s.store_id = st.id
	%s
	GROUP BY ts.period
	ORDER BY ts.period;
	`, timeTruncCol, storeFilter, companyFilter)

	// bajarish
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get chart info: %v", err)
		return res, domain.InternalServerError
	}
	return res, nil
}

// get dashboard top stores
func (s *Services) DashboardTopStores(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopStores, error) {
	var res []domain.TopStores

	var (
		args   []any
		query  = `SELECT stores.id, stores.name, COUNT(*) AS count, SUM(sales.total_amount) AS total_amount FROM sales JOIN stores ON sales.store_id = stores.id`
		filter = ` WHERE sales.stage IN (9, 11)`
		group  = ` GROUP BY stores.id`
		order  = ` ORDER BY total_amount DESC`
	)

	// Parse and apply date filters
	var startStr, endStr string
	if params.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, params.StartDate)
		if err != nil {
			s.log.Errorf("Invalid start_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
		startStr = startTime.Format("2006-01-02 15:04:05")

		// if end_date is empty → use start_date
		var endTime time.Time
		if params.EndDate != "" {
			endTime, err = time.Parse(time.RFC3339, params.EndDate)
			if err != nil {
				s.log.Errorf("Invalid end_date format: %v", err)
				return nil, domain.InvalidTimeFormatError
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
	if params.StoreId != "" {
		filter += " AND sales.store_id = ?"
		args = append(args, params.StoreId)
	}

	// Company filter
	if params.CompanyId != "" {
		filter += " AND stores.company_id = ?"
		args = append(args, params.CompanyId)
	}

	// Limit & Offset
	var q = query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	// Execute query
	err := s.db.WithContext(ctx).Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("Failed to get top stores: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard top products
func (s *Services) DashboardTopProducts(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopProducts, error) {
	var res []domain.TopProducts

	var (
		args  []any
		query = `
		SELECT
			p.id, 
			p.name,
			SUM(ci.unit_quantity) / p.unit_per_pack AS count,
			SUM(ci.unit_quantity) % p.unit_per_pack AS unit_quantity,
			p.unit_per_pack,
			sum(ci.total_price) as total_amount
		FROM cart_items ci
			JOIN store_products sp ON ci.store_product_id = sp.id
			JOIN products p on sp.product_id = p.id`
		filter = ` WHERE 1 = 1`
		group  = ` GROUP BY p.id, p.name, p.unit_per_pack`
		order  = ` ORDER BY total_amount DESC`
	)

	// Filter by company_id
	if params.CompanyId != "" {
		filter += ` AND sp.company_id = ?`
		args = append(args, params.CompanyId)
	}

	// Filter by one store
	if params.StoreId != "" {
		filter += ` AND sp.store_id = ?`
		args = append(args, params.StoreId)
	}

	// Filter by multiple stores
	if len(params.StoreIds) > 0 {
		filter += ` AND sp.store_id IN (?)`
		args = append(args, params.StoreIds)
	}

	// Parse RFC3339 date-time range
	if params.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, params.StartDate)
		if err != nil {
			s.log.Errorf("Invalid start_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
		startStr := startTime.Format("2006-01-02 15:04:05")

		var endTime time.Time
		if params.EndDate != "" {
			endTime, err = time.Parse(time.RFC3339, params.EndDate)
			if err != nil {
				s.log.Errorf("Invalid end_date format: %v", err)
				return nil, domain.InvalidTimeFormatError
			}
		} else {
			endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endStr := endTime.Format("2006-01-02 15:04:05")

		filter += " AND ci.updated_at BETWEEN ? AND ?"
		args = append(args, startStr, endStr)
	}

	// Add pagination
	args = append(args, params.Limit, params.Offset)
	query = query + filter + group + order + " LIMIT ? OFFSET ?"

	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top products: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard bonus products
func (s *Services) DashboardBonusProducts(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.BonusProducts, error) {
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
		JOIN employees e ON eb.employee_id = e.id
		`
		filter = " WHERE 1=1 "
		group  = " GROUP BY p.id "
		order  = " ORDER BY count DESC"
	)

	// company_id
	if params.CompanyId != "" {
		filter += " AND e.company_id = ? "
		args = append(args, params.CompanyId)
	}

	// check store_ids
	if len(params.StoreIds) > 0 {
		filter += " AND e.store_id IN (?) "
		args = append(args, params.StoreIds)
	}

	// Parse RFC3339 start va end vaqtlar
	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Errorf("Invalid start_date format: %v", err)
		return nil, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Errorf("Invalid end_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	filter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, startStr, endStr)

	// Limit / Offset
	query = query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	// Execute
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting bonus products: ", err)
		return nil, err
	}

	return res, nil
}

// get dashboard top seller
func (s *Services) DashboardTopSeller(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopSeller, error) {
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
		filter = " WHERE s.stage IN(9, 11) AND s.sale_type = 'SALE'"
		group  = " GROUP BY e.id, st.id"
		order  = " ORDER BY total_amount DESC"
		offset = " LIMIT ? OFFSET ?"
	)

	// Company filter
	if params.CompanyId != "" {
		filter += " AND st.company_id = ?"
		args = append(args, params.CompanyId)
	}

	// Filter by one store
	if params.StoreId != "" {
		filter += " AND s.store_id = ?"
		args = append(args, params.StoreId)
	}

	// Filter by multiple stores
	if len(params.StoreIds) > 0 {
		filter += " AND s.store_id IN (?)"
		args = append(args, params.StoreIds)
	}

	// Date filter — RFC3339 parse
	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("could not parse start_date: %v", err)
		return nil, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Errorf("could not parse end_date: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")

	filter += " AND s.completed_at BETWEEN ? AND ?"
	args = append(args, startStr, endStr)

	// Pagination
	args = append(args, params.Limit, params.Offset)

	// Build and run query
	var q = query + filter + group + order + offset
	err = s.db.WithContext(ctx).Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("ERROR on getting top seller: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get payment
func (s *Services) DashboardPayments(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardPaymentDto, error) {
	var startTime, endTime time.Time
	// Parse datetimes
	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Oldingi davrni hisoblash
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	var res domain.DashboardPaymentDto
	qb := s.db.WithContext(ctx).
		Select(
			"SUM(s.cash) AS cash",
			"COUNT(1) FILTER (WHERE s.cash > 0) AS cash_count",
			"SUM(s.humo) AS humo",
			"COUNT(1) FILTER (WHERE s.humo > 0) AS humo_count",
			"SUM(s.uzcard) AS uzcard",
			"COUNT(1) FILTER (WHERE s.uzcard > 0) AS uzcard_count",
			"SUM(s.click) AS click",
			"COUNT(1) FILTER (WHERE s.click > 0) AS click_count",
			"SUM(s.payme) AS payme",
			"COUNT(1) FILTER (WHERE s.payme > 0) AS payme_count",
			"SUM(s.alif) AS alif",
			"COUNT(1) FILTER (WHERE s.alif > 0) AS alif_count",
		).
		Table("sales s").
		Where("s.stage IN(?)", constants.FinishedSaleStages)

	// filters
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}
	if params.CompanyId != "" {
		qb = qb.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyId)
	}
	if params.StartDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') >= ?", startTime)
	}
	if params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') <= ?", endTime)
	}

	err = qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard payment stats: %v", err)
		return &res, domain.InternalServerError
	}

	// previus data
	var tmpPreviues struct {
		CashPrevius   float64 `gorm:"cash_previus"`
		HumoPrevius   float64 `gorm:"humo_previus"`
		UzcardPrevius float64 `gorm:"uzcard_previus"`
		ClickPrevius  float64 `gorm:"click_previus"`
		PaymePrevius  float64 `gorm:"payme_previus"`
		AlifPrevius   float64 `gorm:"alif_previus"`
	}

	qbPrev := s.db.WithContext(ctx).
		Select(
			"SUM(s.cash) AS cash_previus",
			"SUM(s.humo) AS humo_previus",
			"SUM(s.uzcard) AS uzcard_previus",
			"SUM(s.click) AS click_previus",
			"SUM(s.payme) AS payme_previus",
			"SUM(s.alif) AS alif_previus",
		).
		Table("sales s").
		Where("s.stage IN(?)", constants.FinishedSaleStages)

	// previus filter
	if len(params.StoreIds) > 0 {
		qbPrev = qbPrev.Where("s.store_id IN(?)", params.StoreIds)
	}
	if params.CompanyId != "" {
		qbPrev = qbPrev.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyId)
	}
	if params.StartDate != "" {
		qbPrev = qbPrev.Where("(s.completed_at + interval '5 hours') >= ?", beforeStart)
	}
	if params.EndDate != "" {
		qbPrev = qbPrev.Where("(s.completed_at + interval '5 hours') <= ?", beforeEnd)
	}
	err = qbPrev.Take(&tmpPreviues).Error
	if err != nil {
		s.log.Errorf("could not get dashboard payment stats for previus: %v", err)
		return &res, domain.InternalServerError
	}
	// cash
	if tmpPreviues.CashPrevius != 0 {
		res.CashPercent = math.Round((((res.Cash - tmpPreviues.CashPrevius) * 100) / tmpPreviues.CashPrevius) * 100)
	}
	// humo
	if tmpPreviues.HumoPrevius != 0 {
		res.HumoPercent = math.Round((((res.Humo - tmpPreviues.HumoPrevius) * 100) / tmpPreviues.HumoPrevius) * 100)
	}
	// uzcard
	if tmpPreviues.UzcardPrevius != 0 {
		res.UzcardPercent = math.Round((((res.Uzcard - tmpPreviues.UzcardPrevius) * 100) / tmpPreviues.UzcardPrevius) * 100)
	}
	// click
	if tmpPreviues.ClickPrevius != 0 {
		res.ClickPercent = math.Round((((res.Cash - tmpPreviues.ClickPrevius) * 100) / tmpPreviues.ClickPrevius) * 100)
	}
	// payme
	if tmpPreviues.PaymePrevius != 0 {
		res.PaymePercent = math.Round((((res.Cash - tmpPreviues.PaymePrevius) * 100) / tmpPreviues.PaymePrevius) * 100)
	}
	// alif
	if tmpPreviues.AlifPrevius != 0 {
		res.AlifPercent = math.Round((((res.Cash - tmpPreviues.AlifPrevius) * 100) / tmpPreviues.AlifPrevius) * 100)
	}

	return &res, nil
}

func (s *Services) DashboardTransaction(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.DashboardTransaction, error) {
	var (
		startTime, endTime time.Time
		err                error
	)

	// Parse datetimes
	startTime, err = time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, domain.InvalidTimeFormatError
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Oldingi davrni hisoblash
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	// Format for SQL
	startStr := startTime.Format("2006-01-02 15:04:05")
	endStr := endTime.Format("2006-01-02 15:04:05")
	beforeStartStr := beforeStart.Format("2006-01-02 15:04:05")
	beforeEndStr := beforeEnd.Format("2006-01-02 15:04:05")

	res := []domain.DashboardTransaction{}

	// ---- FILTERS ----
	storeFilter := ""
	if len(params.StoreIds) > 0 {
		storeFilter += " AND s.store_id IN ? "
	}
	if params.CompanyId != "" {
		storeFilter += " AND s.store_id IN (SELECT id FROM stores WHERE company_id = ?) "
	}

	// SALES query
	saleQuery := fmt.Sprintf(`
	SELECT
		'Товары' AS name,
		curr.amount,
		curr.count,
		prev.amount AS previous_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.amount, 0) = 0 THEN 100
				ELSE ((curr.amount - prev.amount) * 100.0) / NULLIF(prev.amount, 0)
			END, 2
		) AS percent
	FROM (
		SELECT
			SUM(s.total_amount) AS amount,
			COALESCE(ROUND(SUM(ci.quantity + (ci.unit_quantity / 100.0)), 0), 0) AS count
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE s.stage IN(9, 11)
			AND s.sale_type = 'SALE' 
			AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
			%s
	) curr
	LEFT JOIN (
		SELECT
			SUM(s.total_amount) AS amount
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE s.stage IN(9, 11)
			AND s.sale_type = 'SALE' 
			AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
			%s
	) prev ON 1=1
	`, storeFilter, storeFilter)

	// RETURN query
	returnQuery := fmt.Sprintf(`
	SELECT
		'Возвраты' AS name,
		curr.amount,
		curr.count,
		prev.amount AS previous_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.amount, 0) = 0 THEN 100
				ELSE ((curr.amount - prev.amount) * 100.0) / NULLIF(prev.amount, 0)
			END, 2
		) AS percent
	FROM (
		SELECT
			SUM(s.total_amount) AS amount,
			COALESCE(ROUND(SUM(ci.quantity + (ci.unit_quantity / 100.0)), 0), 0) AS count
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE s.stage IN(9, 11) 
			AND s.sale_type = 'RETURN'
			AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
			%s
	) curr
	LEFT JOIN (
		SELECT
			SUM(s.total_amount) AS amount
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE s.stage IN(9, 11) 
			AND s.sale_type = 'RETURN'
			AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
			%s
	) prev ON 1=1
	`, storeFilter, storeFilter)

	// Combine queries
	fullQuery := fmt.Sprintf(`%s UNION ALL %s`, saleQuery, returnQuery)

	// ---- ARGS ----
	finalArgs := []any{
		// Sales curr
		startStr, endStr,
	}
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if params.CompanyId != "" {
		finalArgs = append(finalArgs, params.CompanyId)
	}
	// sales prev
	finalArgs = append(finalArgs, beforeStartStr, beforeEndStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if params.CompanyId != "" {
		finalArgs = append(finalArgs, params.CompanyId)
	}
	// returns curr
	finalArgs = append(finalArgs, startStr, endStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if params.CompanyId != "" {
		finalArgs = append(finalArgs, params.CompanyId)
	}
	// returns prev
	finalArgs = append(finalArgs, beforeStartStr, beforeEndStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if params.CompanyId != "" {
		finalArgs = append(finalArgs, params.CompanyId)
	}

	// Execute
	err = s.db.WithContext(ctx).Raw(fullQuery, finalArgs...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("Error fetching dashboard transaction stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) DashboardOldImports(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.Import, int64, error) {
	var (
		imports    []domain.Import
		totalCount int64

		err error
	)

	// Main query
	query := s.db.WithContext(ctx).Model(&domain.Import{}).
		Preload("Store").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*,
			ROUND(SUM(import_details.retail_price * import_details.received_count)::numeric, 2) AS received_amount,
			ROUND(SUM(import_details.retail_price * import_details.accepted_count)::numeric, 2) AS accepted_amount,
			ROUND(SUM(import_details.retail_price_vat * import_details.received_count)::numeric, 2) AS received_amount_vat,
			ROUND(SUM(import_details.retail_price_vat * import_details.accepted_count)::numeric, 2) AS accepted_amount_vat,
			ROUND(SUM(import_details.received_count)::numeric, 2) AS received_count,
			ROUND(SUM(import_details.accepted_count)::numeric, 2) AS accepted_count
		`).Joins("LEFT JOIN import_details ON imports.id = import_details.import_id").
		Where("imports.entry_type = ?", 1).
		Where("imports.created_at < NOW() - interval '24 hours'").
		Where("imports.status = ?", "new")

	// Apply filters
	if params.CompanyId != "" {
		query = query.Joins("JOIN stores ON imports.store_id = stores.id").
			Where("stores.company_id = ?", params.CompanyId)
	}
	if params.Search != "" {
		search := fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where(`
			imports.document_number ILIKE ? OR 
			CAST(imports.public_id AS TEXT) LIKE ?`, search, search)
	}
	if params.StoreId != "" {
		query = query.Where("imports.store_id = ?", params.StoreId)
	}

	// Grouping, count, pagination
	err = query.Group("imports.id").
		Order("imports.created_at DESC").
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&imports).Error
	if err != nil {
		s.log.Errorf("could not get old imports: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return imports, totalCount, nil
}

func (s *Services) GetTotalSaleStats(ctx context.Context, params *domain.DashboardQueryParam) (*domain.TotalSaleDto, error) {
	// qb := s.db.WithContext(ctx).Error

	var res domain.TotalSaleDto

	return &res, nil
}
