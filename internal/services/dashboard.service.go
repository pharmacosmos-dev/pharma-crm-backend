package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/pharma-crm-backend/plagins"
)

// get dashboard count and amount data
func (s *Services) DashboardTotalCountStats(ctx context.Context, param *domain.DashboardQueryParam) (*domain.DashboardCountStats, error) {
	// declarations
	var (
		res            domain.DashboardCountStats
		startTimeInUTC = (*param.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*param.StartDate, param.EndDate).ToUTC()

		startStr       = startTimeInUTC.GetString()
		endStr         = endTimeInUTC.GetString()
		beforeStartStr = startTimeInUTC.PrevDay().GetString()
		beforeEndStr   = endTimeInUTC.PrevDay().GetString()
	)

	// queries
	var (
		args []any
		// get sale stats information
		querys = fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.id END) AS sale_count,
			COUNT(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.id END) AS before_sale_count,
			SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.total_amount ELSE 0 END) AS sale_amount,
			SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.total_amount ELSE 0 END) AS before_sale_amount,
			SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.total_discount ELSE 0 END) AS discount_amount,
			SUM(CASE WHEN completed_at BETWEEN '%s' AND '%s' THEN sales.total_discount ELSE 0 END) AS before_discount_amount
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
	if len(param.CompanyIds) > 0 {
		filter += " AND st.company_id IN (?)"
		filterc += " AND p.company_id IN (?)"
		query24h += " AND st.company_id IN (?)"
		args = append(args, param.CompanyIds)
	}

	// Execute queries
	var sale domain.DashboardCountStatsSale
	querys += filter
	err := s.db.WithContext(ctx).Debug().Raw(querys, args...).Scan(&sale).Error
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
		res       []domain.ChartResponse
		interval  string
		startTime = (*params.StartDate).GetTime()
		endTime   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).GetTime()
	)

	if params.EndDate == nil {
		endTime = startTime
	}

	// Group type
	switch params.Type {
	case "HALF_HOURLY":
		interval = "30 minutes"
		startTime = startTime.Truncate(30 * time.Minute)

	case "HOURLY":
		interval = "1 hour"
		startTime = startTime.Truncate(time.Hour)

	case "DAILY":
		interval = "1 day"
		startTime = time.Date(
			startTime.Year(), startTime.Month(), startTime.Day(),
			0, 0, 0, 0, startTime.Location(),
		)

	case "WEEKLY":
		interval = "1 week"
		// Move to start of the week (Monday)
		weekday := int(startTime.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday should move to previous Monday
		}
		startTime = time.Date(
			startTime.Year(), startTime.Month(), startTime.Day()-weekday+1,
			0, 0, 0, 0, startTime.Location(),
		)

	case "MONTHLY":
		interval = "1 month"
		startTime = time.Date(
			startTime.Year(), startTime.Month(), 1,
			0, 0, 0, 0, startTime.Location(),
		)

	case "YEARLY":
		interval = "1 year"
		startTime = time.Date(
			startTime.Year(), 1, 1,
			0, 0, 0, 0, startTime.Location(),
		)

	default:
		interval = "1 hour"
		startTime = startTime.Truncate(time.Hour)
	}
	// WEEKLY tanlangan bo‘lsa startDate ni truncate qilamiz

	args := []any{startTime, endTime, interval}

	// qo‘shimcha filterlar
	storeFilter := ""
	if len(params.StoreIds) > 0 {
		storeFilter += " AND s.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	companyFilter := ""
	if len(params.CompanyIds) > 0 {
		companyFilter += " AND st.company_id IN(?)"
		args = append(args, params.CompanyIds)
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
		COUNT(distinct s.id) filter ( where s.sale_type = 'SALE' ) AS count,
    	SUM(CASE WHEN s.sale_type = 'SALE' THEN ci.total_price - ci.discount_amount ELSE (ci.total_price - ci.discount_amount) * (-1) END) AS total_amount
	FROM time_series ts
	LEFT JOIN sales s ON
		(s.completed_at + INTERVAL '5 hours') >= ts.period AND (s.completed_at + INTERVAL '5 hours') < ts.period + INTERVAL '%s' 
		AND s.stage IN (9, 11)
		%s
	LEFT JOIN cart_items ci ON ci.sale_id = s.id
	LEFT JOIN stores st ON s.store_id = st.id
	%s
	WHERE s.stage IN(9, 11)
	GROUP BY ts.period
	ORDER BY ts.period;
	`, interval, storeFilter, companyFilter)

	// bajarish
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
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
		order  = utils.BuildTopStoreOrderClauseForDashBoard(params.Order)

		startTimeInUTCStr = (*params.StartDate).ToUTC().GetString()
		endTimeInUTCStr   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	// Apply filter
	filter += " AND sales.completed_at BETWEEN ? AND ?"
	args = append(args, startTimeInUTCStr, endTimeInUTCStr)

	// Store filter
	if len(params.StoreIds) > 0 {
		filter += " AND sales.store_id IN (?)"
		args = append(args, params.StoreIds)
	}

	// Company filter
	if len(params.CompanyIds) > 0 {
		filter += " AND stores.company_id IN (?)"
		args = append(args, params.CompanyIds)
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
	// declaration
	var (
		res  []domain.TopProducts
		args []any

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	query := `
	SELECT
		curr.id,
		curr.name,
		curr.producer_name,
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
				ps.name AS producer_name,
				s.company_id,
				SUM(ci.unit_quantity) / p.unit_per_pack AS count,
				SUM(ci.unit_quantity) % p.unit_per_pack AS unit_quantity,
				p.unit_per_pack,
				SUM(ci.total_price) as total_amount
			FROM cart_items ci
					JOIN store_products sp ON ci.store_product_id = sp.id
					JOIN products p ON sp.product_id = p.id
					JOIN producers ps ON p.producer_id = ps.id
					JOIN stores s ON sp.store_id = s.id
			WHERE ci.updated_at BETWEEN ? AND ?
			GROUP BY p.id, p.name, ps.name, p.unit_per_pack,s.company_id
		) AS curr
			left JOIN (
		SELECT
			p.id,
			s.company_id,
			SUM(ci.total_price) AS total_amount
		FROM cart_items ci
				JOIN store_products sp ON ci.store_product_id = sp.id
				JOIN products p ON sp.product_id = p.id
				JOIN stores s ON sp.store_id = s.id
		WHERE ci.updated_at BETWEEN ? AND ?
		GROUP BY p.id, s.company_id
	) AS prev ON curr.id = prev.id and curr.company_id = prev.company_id
`

	// Arguments for current and previous period
	args = append(args,
		startTimeStr,
		endTimeStr,
		beforeStartTimeStr,
		beforeEndTimeStr,
	)

	// Filters
	where := " WHERE 1 = 1"
	if params.Search != "" {
		where += " AND curr.name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.CompanyIds) > 0 {
		where += " AND curr.company_id IN (?) "
		args = append(args, params.CompanyIds)
	}
	if len(params.StoreIds) > 0 {
		where += " AND EXISTS (SELECT 1 FROM store_products sp3 WHERE sp3.product_id = curr.id AND sp3.store_id IN (?))"
		args = append(args, params.StoreIds)
	}

	// Sorting (replaced switch)
	order := utils.BuildTopProductOrderClause(params.Order)
	query += where + order

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	// Execute query
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top products: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard bonus products
func (s *Services) DashboardBonusProducts(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.BonusProducts, error) {
	// declaration
	var (
		res  []domain.BonusProducts
		args []any

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	query := `
	SELECT
		curr.id,
		curr.name,
		curr.count,
		curr.unit_quantity,
		curr.unit_per_pack,
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
    		p.unit_per_pack,
    		SUM(eb.unit_quantity) % p.unit_per_pack as unit_quantity,
			SUM(eb.quantity) + ROUND(SUM(eb.unit_quantity) / p.unit_per_pack,0) AS count,
			SUM(eb.bonus_amount) AS bonus_amount
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
	`

	// Dynamic JOIN and filters
	join := ""
	filter := " WHERE 1 = 1"
	if len(params.StoreIds) > 0 {
		join += " JOIN employees e ON eb.employee_id = e.id"
		filter += " AND e.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	if params.Search != "" {
		filter += " AND p.name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.CompanyIds) > 0 {
		filter += " AND p.company_id IN (?) "
		args = append(args, params.CompanyIds)
	}
	filter += " AND eb.created_at BETWEEN ? AND ?"
	args = append(args, startTimeStr, endTimeStr)

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
	if len(params.StoreIds) > 0 {
		prevJoin += " JOIN employees e ON eb.employee_id = e.id"
		prevFilter += " AND e.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	prevFilter += " AND eb.created_at BETWEEN ? AND ?"
	args = append(args, beforeStartTimeStr, beforeEndTimeStr)

	query += prevJoin + prevFilter + " GROUP BY p.id ) AS prev ON curr.id = prev.id"

	// New flexible order logic
	order := utils.BuildBonusProductOrderClause(params.Order)
	query += order

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	// Execute query
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting bonus products: ", err)
		return nil, err
	}

	return res, nil
}

// get dashboard top seller
func (s *Services) DashboardTopSeller(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopSeller, error) {
	var (
		res  []domain.TopSeller
		args []any

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

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
			st.company_id,
			COUNT(s.id) AS count,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		INNER JOIN stores st ON s.store_id = st.id
		WHERE s.stage IN(9, 11)
		AND s.sale_type = 'SALE'
		AND s.completed_at BETWEEN ? AND ?
		GROUP BY e.id, e.full_name, st.name, st.company_id
	) AS curr
	LEFT JOIN (
		SELECT
			e.id,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		WHERE s.stage IN(9, 11)
		AND s.sale_type = 'SALE'
		AND s.completed_at BETWEEN ? AND ?
		GROUP BY e.id
	) AS prev ON curr.id = prev.id
`

	// First 4 args: 2 for current, 2 for previous range
	args = append(args,
		startTimeStr, endTimeStr,
		beforeStartTimeStr, beforeEndTimeStr,
	)

	// Optional filters
	where := " WHERE 1 = 1"
	if params.Search != "" {
		where += " AND curr.full_name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.CompanyIds) > 0 {
		where += " AND curr.company_id IN (?) "
		args = append(args, params.CompanyIds)
	}
	// check store_ids
	if len(params.StoreIds) > 0 {
		where += " AND curr.store_name IN (SELECT name FROM stores WHERE id IN (?))"
		args = append(args, params.StoreIds)
	}

	// Apply flexible ordering
	order := utils.BuildTopSellerOrderClause(params.Order)

	// Pagination
	limitOffset := " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	finalQuery := query + where + order + limitOffset

	// Execute
	err := s.db.WithContext(ctx).Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top seller: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get payment
func (s *Services) DashboardPayments(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardPaymentDto, error) {
	var (
		res domain.DashboardPaymentDto

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

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
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at between ? and ? ", startTimeStr, endTimeStr)

	// filters
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		qb = qb.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyIds)
	}

	err := qb.Take(&res).Error
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
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at between ? and ? ", beforeStartTimeStr, beforeEndTimeStr)

	// previus filter
	if len(params.StoreIds) > 0 {
		qbPrev = qbPrev.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qbPrev = qbPrev.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyIds)
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
		err error

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	res := []domain.DashboardTransaction{}

	// ---- FILTERS ----
	storeFilter := ""
	if len(params.StoreIds) > 0 {
		storeFilter += " AND s.store_id IN ? "
	}
	if len(params.CompanyIds) > 0 {
		storeFilter += " AND s.store_id IN (SELECT id FROM stores WHERE company_id IN (?) ) "
	}

	// SALES query
	fullQuery := fmt.Sprintf(`
	WITH sales_data AS (
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
					COALESCE(ROUND(SUM(ci.quantity + (ci.unit_quantity / p.unit_per_pack)), 0), 0) AS count
				FROM sales s
						JOIN cart_items ci ON ci.sale_id = s.id
						JOIN store_products as sp on ci.store_product_id = sp.id
						JOIN products as p on sp.product_id = p.id
				WHERE s.stage IN(9, 11)
				AND s.sale_type = 'SALE'
				AND s.completed_at BETWEEN ? AND ?
				%s
			) curr
				LEFT JOIN (
			SELECT
				SUM(s.total_amount) AS amount
			FROM sales s
					JOIN cart_items ci ON ci.sale_id = s.id
			WHERE s.stage IN(9, 11)
			AND s.sale_type = 'SALE'
			AND s.completed_at BETWEEN ? AND ?
			%s
		) prev ON 1=1
	),
		returns_data AS (
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
						COALESCE(ROUND(SUM(ci.quantity + (ci.unit_quantity / p.unit_per_pack)), 0), 0) AS count
					FROM sales s
							JOIN cart_items ci ON ci.sale_id = s.id
							JOIN store_products as sp on ci.store_product_id = sp.id
							JOIN products as p on sp.product_id = p.id
					WHERE s.stage IN(9, 11)
						AND s.sale_type = 'RETURN'
						AND s.completed_at BETWEEN ? AND ?
						%s
				) curr
					LEFT JOIN (
				SELECT
					SUM(s.total_amount) AS amount
				FROM sales s
						JOIN cart_items ci ON ci.sale_id = s.id
				WHERE s.stage IN(9, 11)
				AND s.sale_type = 'RETURN'
				AND s.completed_at BETWEEN ? AND ?
				%s
			) prev ON 1=1
		)
	SELECT
		name,
		amount + COALESCE((SELECT amount FROM returns_data), 0) AS amount,
		count - COALESCE((SELECT count FROM returns_data), 0) AS count,
		previous_amount + COALESCE((SELECT previous_amount FROM returns_data), 0) AS previous_amount,
		ROUND(
				CASE
					WHEN COALESCE(previous_amount - COALESCE((SELECT previous_amount FROM returns_data), 0), 0) = 0 THEN 100
					ELSE (((amount - COALESCE((SELECT amount FROM returns_data), 0)) - (previous_amount - COALESCE((SELECT previous_amount FROM returns_data), 0))) * 100.0) /
						NULLIF(previous_amount - COALESCE((SELECT previous_amount FROM returns_data), 0), 0)
					END, 2
		) AS percent
	FROM sales_data
	UNION ALL
	SELECT * FROM returns_data
	`, storeFilter, storeFilter, storeFilter, storeFilter)

	// ---- ARGS ----
	finalArgs := []any{
		// Sales curr
		startTimeStr, endTimeStr,
	}
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		finalArgs = append(finalArgs, params.CompanyIds)
	}
	// sales prev
	finalArgs = append(finalArgs, beforeStartTimeStr, beforeEndTimeStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		finalArgs = append(finalArgs, params.CompanyIds)
	}
	// returns curr
	finalArgs = append(finalArgs, startTimeStr, endTimeStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		finalArgs = append(finalArgs, params.CompanyIds)
	}
	// returns prev
	finalArgs = append(finalArgs, beforeStartTimeStr, beforeEndTimeStr)
	if len(params.StoreIds) > 0 {
		finalArgs = append(finalArgs, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		finalArgs = append(finalArgs, params.CompanyIds)
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
	var tmpImport []struct {
		Id                string     `gorm:"id"`
		PublicId          int        `gorm:"public_id"`
		StoreId           string     `gorm:"store_id"`
		CreatedBy         string     `gorm:"created_by"`
		AcceptedBy        string     `gorm:"accepted_by"`
		DocumentNumber    string     `gorm:"document_number"`
		DocumentYear      int        `gorm:"document_year"`
		Status            string     `gorm:"status"`
		ImportDate        *time.Time `gorm:"import_date"`
		AcceptedAmount    float64    `gorm:"accepted_amount"`
		ReceivedAmount    float64    `gorm:"received_amount"`
		ReceivedCount     float64    `gorm:"received_count"`
		AcceptedCount     float64    `gorm:"accepted_count"`
		AcceptedAmountVat float64    `gorm:"accepted_amount_vat"`
		ReceivedAmountVat float64    `gorm:"received_amount_vat"`
		CreatedAt         *time.Time `gorm:"created_at"`
		UpdatedAt         *time.Time `gorm:"updated_at"`
		StoreName         string     `gorm:"store_name"`
		CreatedByName     string     `gorm:"created_by_name"`
		AcceptedByName    string     `gorm:"accepted_by_name"`
	}

	// Fetch imports with detailed data
	qb := s.db.
		WithContext(ctx).
		Table("imports im").
		Joins("JOIN stores st ON st.id = im.store_id").
		Where("im.entry_type = ?", constants.ProductMovementImport).
		Where("im.created_at <= NOW() - interval '24 hours'")

	qb = qb.Where("im.status = ?", constants.GeneralStatusNew)

	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.document_number ILIKE ? OR im.public_id::text LIKE ?", params.Search, params.Search)
	}
	if len(params.StoreIds) > 0 {
		qb = qb.Where("im.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN (?) ", params.CompanyIds)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get imports total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	err := qb.
		Select(
			"im.id",
			"im.public_id",
			"im.store_id",
			"im.name",
			"im.document_number",
			"im.document_year",
			"im.status",
			"im.import_date",
			"im.received_count AS received_count",
			"im.received_sum AS received_amount_vat",
			"im.scanned_count AS accepted_count",
			"im.scanned_sum AS accepted_amount_vat",
			"im.created_by",
			"im.accepted_by",
			"im.created_at",
			"im.updated_at",

			"st.name AS store_name",
		).
		Order("im.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmpImport).Error
	if err != nil {
		s.log.Errorf("could not get imports: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res = []domain.Import{}
	for _, item := range tmpImport {
		res = append(res, domain.Import{
			Id:                item.Id,
			PublicId:          item.PublicId,
			StoreId:           item.StoreId,
			DocumentNumber:    item.DocumentNumber,
			DocumentYear:      item.DocumentYear,
			Status:            item.Status,
			ReceivedCount:     item.ReceivedCount,
			ReceivedAmountVat: item.ReceivedAmountVat,
			AcceptedCount:     item.AcceptedCount,
			AcceptedAmountVat: item.AcceptedAmountVat,
			CreatedBy:         item.CreatedBy,
			AcceptedBy:        item.AcceptedBy,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
			ImportDate:        item.ImportDate,
			Store: domain.NewNullStruct(domain.ImportStore{
				Id:   item.StoreId,
				Name: item.StoreName,
			}, item.StoreId != ""),
			Sender: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.CreatedBy,
				FullName: item.CreatedByName,
			}, item.CreatedBy != ""),
			Receiver: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.AcceptedBy,
				FullName: item.AcceptedByName,
			}, item.AcceptedBy != ""),
		})
	}

	return res, totalCount, nil
}

// get dashboard count and amount data
func (s *Services) DashboardSaleStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardSaleStatistic, error) {
	var (
		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	qb := s.db.
		WithContext(ctx).
		Select(
			"COUNT(distinct s.id) filter ( where s.sale_type = 'SALE' ) AS sale_count",
			"SUM(CASE WHEN s.sale_type = 'SALE' THEN ci.total_price - ci.discount_amount ELSE (ci.total_price - ci.discount_amount) * (-1) END) AS sale_amount",
		).
		Table("sales s").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN stores st ON s.store_id = st.id").
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)

	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardSaleStatistic
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get total sale amounts: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardNetProfitStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardCountStatsIncome, error) {
	var (
		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	qb := s.db.WithContext(ctx).
		Select(
			`ROUND(
                 SUM(
                     case
                         when s.sale_type = 'SALE'
                         then ((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity - ci.discount_amount
                         else (((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity - ci.discount_amount)*(-1)
                     end
                 ), 2) AS income_amount`,
			`ROUND(
                 SUM(
                     case
                         when s.sale_type = 'SALE'
                         then (sp.supply_price / p.unit_per_pack) * ci.unit_quantity
                         else ((sp.supply_price / p.unit_per_pack) * ci.unit_quantity) * (-1)
                     end
                 ), 2) AS production_cost`,
		).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardCountStatsIncome
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get total income: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardImportStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardImportStatistic, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"SUM(im.received_sum) AS import_amount",
			"SUM(CASE WHEN im.created_at < NOW() - interval '24 hour' THEN im.received_sum ELSE 0 END) AS expired_import_amount",
		).
		Table("imports im").
		Joins("JOIN stores st ON im.store_id = st.id").
		Where("im.entry_type = ?", constants.ProductMovementImport).
		Where("im.status = ?", constants.GeneralStatusNew)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		qb = qb.Where("im.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardImportStatistic
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get import_count for_24: %v", err)
		return nil, domain.InternalServerError
	}

	res.NotLast24HImportAmount = res.ExpiredImportAmount

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardProductStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardProductStatistic, error) {
	// declarations
	var res domain.DashboardProductStatistic

	// queries
	var (
		args []any
		// get sale stats information
		query = `
		SELECT
			ROUND(SUM(sp.unit_quantity), 2) AS total_product_count,
			ROUND(SUM((retail_price / p.unit_per_pack) * sp.unit_quantity), 2) AS stock_total_amount,
			ROUND(SUM(CASE WHEN expire_date BETWEEN NOW() AND NOW() + INTERVAL '3 month' THEN sp.unit_quantity ELSE 0 END), 2) AS expiring_soon_count,
			ROUND(SUM(CASE WHEN expire_date BETWEEN NOW() AND NOW() + INTERVAL '3 month' THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END), 2) AS expiring_soon_amount,
			ROUND(SUM(CASE WHEN expire_date < NOW() THEN sp.unit_quantity ELSE 0 END), 2) AS expired_soon_count,
			ROUND(SUM(CASE WHEN expire_date < NOW() THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END),2) AS expired_soon_amount
		FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			JOIN stores st ON sp.store_id = st.id
		WHERE sp.unit_quantity > 0
		`
		filter = ""
	)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?)"
		args = append(args, params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		filter += " AND st.company_id IN(?)"
		args = append(args, params.CompanyIds)
	}

	// Execute queries
	// get total product count
	query += filter
	err := s.db.WithContext(ctx).Raw(query, args...).Debug().Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get total product_amounts: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) LoyaltyCardStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardLoyaltyCardStatistic, error) {
	var (
		res domain.DashboardLoyaltyCardStatistic

		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = plagins.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	var query = `
select
    sum(case when loyalty_card_barcode is not null THEN 1 ELSE 0 END) as total_loyalty_card_count,
    sum(balance) as total_loyalty_card_balance,
    sum(case when loyalty_card_created_at between ? and ? then 1 else 0 end) as today_created_loyalty_card_count
from
    customers
	`

	err := s.db.WithContext(ctx).Raw(query, startTimeInUTC, endTimeInUTC).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get loyalty card stats: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}
