package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) ProductReportWithDate(ctx context.Context, params *domain.ReportQueryParam) ([]map[string]any, error) {

	var (
		dateColumns []string
		res         []map[string]any
		filter      = "WHERE 1 = 1 "
		group       = " GROUP BY p.name "
		order       = " ORDER BY p.name "
		args        = []any{}
	)
	// filter by store ids
	if len(params.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?) "
		args = append(args, params.StoreIds)
	}
	if params.CompanyId != "" {
		filter += " AND st.company_id = ? "
		args = append(args, params.CompanyId)
	}
	// filter with producer_id
	if params.ProducerId != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, params.ProducerId)
	}
	// var dateList []time.Time
	startDate, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return res, fmt.Errorf("invalid start date")
	}
	endDate, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return res, fmt.Errorf("invalid end date")
	}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateColumns = append(dateColumns, fmt.Sprintf(
			`SUM(CASE WHEN s.completed_at = '%s' THEN s.total_amount ELSE 0 END) AS "%s"`,
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
		 BETWEEN '%s' AND '%s' THEN s.total_amount ELSE 0 END) AS total_amount
		FROM products p
		JOIN store_products sp ON p.id = sp.product_id
		LEFT JOIN cart_items ci ON sp.id = ci.store_product_id
		LEFT JOIN sales s ON ci.sale_id = s.id
		LEFT JOIN stores st ON s.store_id = st.id
	`, datesQuery, params.StartDate, params.EndDate)

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
func (s *Services) BonusReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.BonusReport, int64, error) {
	var (
		res        []domain.BonusReport
		totalCount int64
	)

	// Base query for filtering
	baseQuery := s.db.WithContext(ctx).
		Table("employee_bonus eb").
		Joins("JOIN employees e ON eb.employee_id = e.id").
		Joins("JOIN products p ON eb.product_id = p.id").
		Joins("LEFT JOIN stores s ON e.store_id = s.id")

	// Apply filters to base query
	if len(params.StoreIds) > 0 {
		baseQuery = baseQuery.Where("e.store_id IN(?)", params.StoreIds)
	}
	if params.CompanyId != "" {
		baseQuery = baseQuery.Where("s.company_id = ?", params.CompanyId)
	}
	if params.Search != "" {
		baseQuery = baseQuery.Where("e.full_name ILIKE ?", "%"+params.Search+"%")
	}

	// Date filter
	if params.StartDate != "" && params.EndDate != "" {
		baseQuery = baseQuery.Where("eb.created_at BETWEEN ? AND ?", params.StartDate, params.EndDate)
	} else if params.StartDate != "" {
		baseQuery = baseQuery.Where("DATE(eb.created_at) = DATE(?)", params.StartDate)
	}

	// Count unique employees (before grouping)
	countQuery := baseQuery.
		Select("DISTINCT e.id").
		Group("e.id")

	if err := s.db.Table("(?) as sub", countQuery).Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get bonus totalCount: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// Main query with roles aggregation
	qb := baseQuery.
		Joins(`
			LEFT JOIN (
				SELECT
					er.employee_id,
					STRING_AGG(DISTINCT r.name, '/' ORDER BY r.name) AS role
				FROM employee_roles er
				JOIN roles r ON er.role_id = r.id
				GROUP BY er.employee_id
			) roles_agg ON e.id = roles_agg.employee_id
		`).
		Select(
			"e.id",
			"e.public_id",
			"e.full_name",
			"e.phone",
			"s.name AS store_name",
			"roles_agg.role",
			"SUM(eb.bonus_amount) AS amount",
			"ROUND(SUM(eb.quantity::numeric + eb.unit_quantity::numeric/p.unit_per_pack), 2) AS count",
		).
		Group("e.id, s.id, s.name, roles_agg.role")

	// Apply ordering
	order := utils.BuildBonusReportOrderClause(params.Order)
	qb = qb.Order(order)

	err := qb.
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get employee_bonuses: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get product report service
func (s *Services) GetProductsReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.ProductReport, int64, error) {
	var order = utils.BuildProductReport(params.Order)

	qb := s.db.WithContext(ctx).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Joins("JOIN employees e ON s.employee_id = e.id").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id")

	// filters
	qb = qb.Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.Search != "" {
		if _, err := strconv.Atoi(params.Search); err == nil {
			qb = qb.Where("s.sale_number::text LIKE ?", params.Search+"%")
		} else {
			qb = qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
		}
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN (?)", params.StoreIds)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.ProducerId != "" {
		qb = qb.Where("p.producer_id = ?", params.ProducerId)
	}
	if params.EmployeeId != "" {
		qb = qb.Where("s.employee_id = ?", params.EmployeeId)
	}
	if params.StartDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') <= ?", params.EndDate)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get products report total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.ProductReport
	err := qb.Select(
		"p.material_code",
		"p.name AS product_name",
		"p.unit_per_pack",
		"pr.name AS producer_name",

		"st.name AS store_name",

		"sp.id AS store_product_id",
		"sp.serial_number",
		"sp.expire_date",
		"sp.supply_price",
		"sp.retail_price",
		"(sp.supply_price / p.unit_per_pack) * ci.unit_quantity AS supply_price_sum",
		"(sp.retail_price / p.unit_per_pack) * ci.unit_quantity AS retail_price_sum",
		"((sp.retail_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity AS markup_sum",
		"(sp.vat_price / p.unit_per_pack) * ci.unit_quantity AS vat_sum",

		"ci.id AS cart_item_id",
		"ci.unit_quantity",
		"ci.marking_count",

		"s.sale_number",
		"s.sale_type",
		"s.total_discount",
		"s.completed_at",

		"e.full_name",
	).
		Limit(params.Limit).
		Offset(params.Offset).
		Order(order).
		Find(&res).Error

	if err != nil {
		s.log.Errorf("could not get products report: %v", err)
		return nil, 0, domain.InternalServerError
	}

	for i := range res {
		if res[i].UnitQuantity%res[i].UnitPerPack > 0 {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)",
				res[i].UnitQuantity/res[i].UnitPerPack,
				res[i].UnitQuantity%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].UnitQuantity/res[i].UnitPerPack)
		}
	}

	return res, totalCount, nil
}

func (s *Services) GetProductsReportStats(ctx context.Context, params *domain.ReportQueryParam) (*domain.ProductStatusReport, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"COALESCE(SUM(CASE WHEN s.sale_type ='SALE' THEN (ci.unit_quantity / p.unit_per_pack) ELSE 0 END), 0) AS total_quantity",
			"COALESCE(SUM(CASE WHEN s.sale_type = 'RETURN' THEN (ci.unit_quantity / p.unit_per_pack) ELSE 0 END), 0) AS total_quantity_returned",
			"ROUND(COALESCE(SUM(CASE WHEN s.sale_type = 'SALE' THEN (ci.total_price - ci.discount_amount)  END), 0), 2) AS total_retail_price_sum",
			"ROUND(COALESCE(SUM(CASE WHEN s.sale_type = 'RETURN' THEN (ci.total_price - ci.discount_amount)  ELSE 0 END), 0), 2) AS total_retail_price_sum_returned",
			"ROUND(COALESCE(SUM(ci.discount_amount), 0), 2) AS total_discount_sum",
		).
		Table("sales s").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
		Joins("JOIN products p ON sp.product_id = p.id")

	// filters
	qb = qb.Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.Search != "" {
		if _, err := strconv.Atoi(params.Search); err == nil {
			qb = qb.Where("s.sale_number::text LIKE ?", params.Search+"%")
		} else {
			qb = qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
		}
	}
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}
	if params.CompanyId != "" {
		qb = qb.Joins("JOIN stores st ON s.store_id = st.id").Where("st.company_id = ?", params.CompanyId)
	}
	if params.EmployeeId != "" {
		qb = qb.Where("s.employee_id = ?", params.EmployeeId)
	}
	if params.ProducerId != "" {
		qb = qb.Where("p.producer_id = ?", params.ProducerId)
	}
	if params.StartDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(s.completed_at + interval '5 hours') <= ?", params.EndDate)
	}

	var res domain.ProductStatusReport
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("coudl not get get products report stats: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get lfl report service
func (s *Services) LflReport(ctx context.Context, params *domain.ReportQueryParam) (domain.LflReport, int64, error) {
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
			sl.stage IN(9, 11)
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
	err := s.db.WithContext(ctx).Raw(query, params.StartDate).Scan(&res.FirstMonth).Error
	if err != nil {
		s.log.Errorf("could get lfl first month: %v", err)
		return res, 0, domain.InternalServerError
	}
	// get second month
	err = s.db.WithContext(ctx).Raw(query, params.EndDate).Scan(&res.SecondMonth).Error
	if err != nil {
		s.log.Errorf("could not get lfl second month: %v", err)
		return res, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get store report amount
func (s *Services) GetStoreAmountReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.StoreAmount, int64, error) {

	qb := s.db.WithContext(ctx).
		Table("stores s").
		Joins("JOIN sales sa ON sa.store_id = s.id").
		Where("sa.stage IN (?, ?)", constants.SaleStageFinished, constants.SaleStageReturnedFinish).
		Group("s.id, s.name")

	// Filters
	if params.StoreId != "" {
		qb = qb.Where("s.id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("s.company_id = ?", params.CompanyId)
	}
	if params.Search != "" {
		qb = qb.Where("s.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.StartDate != "" {
		qb = qb.Where("(sa.completed_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(sa.completed_at + interval '5 hours') <= ?", params.EndDate)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get store_amount report total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}
	var (
		res   []domain.StoreAmount
		order = utils.BuildStoreReportOrderClause(params.Order)
	)

	err := qb.
		Select(
			"row_number() OVER (ORDER BY s.name) AS uid",
			"s.id",
			"s.store_code",
			"s.name AS store_name",
			"(sa.completed_at + interval '5 hours')::date AS sale_date",
			"SUM(sa.cash) AS cash",
			"SUM(sa.uzcard) AS uzcard",
			"SUM(sa.humo) AS humo",
			"SUM(sa.click) AS click",
			"SUM(sa.payme) AS payme",
			"SUM(sa.alif) AS alif",
			"SUM(sa.total_amount) AS total_amount",
			"SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sa.total_amount * (-1) ELSE 0 END) AS return_amount",
			"SUM(sa.total_discount) AS total_discount",
			"COUNT(DISTINCT sa.id) AS cheque_count",
		).
		Limit(params.Limit).
		Offset(params.Offset).
		Group("sale_date").
		Order(order).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get store report amount: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get store report stats
func (s *Services) ReportByStoreStats(ctx context.Context, params *domain.ReportQueryParam) (domain.StoreReportStats, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"SUM(sa.cash) AS cash",
			"SUM(sa.uzcard) AS uzcard",
			"SUM(sa.humo) AS humo",
			"SUM(sa.click) AS click",
			"SUM(sa.payme) AS payme",
			"SUM(sa.alif) AS alif",
			"SUM(sa.total_amount) AS total_amount",
			"SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sa.total_amount * (-1) ELSE 0 END) AS return_amount",
			"SUM(sa.total_discount) AS total_discount",
		).
		Table("stores s").
		Joins("JOIN sales sa ON sa.store_id = s.id")

	// Filters
	qb = qb.Where("sa.stage IN (?)", constants.FinishedSaleStages)

	if params.StoreId != "" {
		qb = qb.Where("s.id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("s.company_id = ?", params.CompanyId)
	}
	if params.Search != "" {
		qb = qb.Where("s.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.StartDate != "" {
		qb = qb.Where("(sa.completed_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(sa.completed_at + interval '5 hours') <= ?", params.EndDate)
	}
	var res domain.StoreReportStats
	err := qb.Take(&res).Error

	if err != nil {
		s.log.Errorf("could not get store amount report stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

// get report top products
func (s *Services) GetTopProductsReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.TopProducts, int64, error) {
	// declaration
	var (
		res        []domain.TopProducts
		args       []any
		totalCount int64
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Errorf("coluld not parse start_date in get top_products: %v", err)
		return nil, 0, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Errorf("coluld not parse end_date in get top_products: %v", err)
			return nil, 0, domain.InvalidTimeFormatError
		}
	} else {
		endTime, err = time.Parse(time.RFC3339, params.StartDate)
		if err != nil {
			s.log.Errorf("coluld not parse start_date in get top_products: %v", err)
			return nil, 0, domain.InvalidTimeFormatError
		}
		endTime = endTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

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
			(SUM(ci.unit_quantity) % p.unit_per_pack) AS unit_quantity,
			p.unit_per_pack,
			SUM(ci.total_price) as total_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN producers ps ON p.producer_id = ps.id
		JOIN stores s ON sp.store_id = s.id
		WHERE (ci.updated_at+ interval '5 hours') BETWEEN ? AND ?
		GROUP BY p.id, p.name, ps.name, p.unit_per_pack,s.company_id
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
	if params.Search != "" {
		where += " AND curr.name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if params.StoreId != "" {
		where += " AND EXISTS (SELECT 1 FROM store_products sp2 WHERE sp2.product_id = curr.id AND sp2.store_id = ?)"
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		where += " AND curr.company_id = ? "
		args = append(args, params.CompanyId)
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
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// get report top seller
func (s *Services) GetTopSellersReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.TopSeller, int64, error) {
	var (
		res        []domain.TopSeller
		totalCount int64
		args       []any
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Errorf("coluld not parse start_date in get top_products: %v", err)
		return nil, 0, domain.InvalidTimeFormatError
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Errorf("coluld not parse end_date in get top_products: %v", err)
			return nil, 0, domain.InvalidTimeFormatError
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		params.EndDate = endTime.Format(time.RFC3339)
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
			st.company_id,
			COUNT(s.id) AS count,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		INNER JOIN stores st ON s.store_id = st.id
		WHERE s.stage IN(9, 11)
		AND s.sale_type = 'SALE'
		AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
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
		AND (s.completed_at + interval '5 hours') BETWEEN ? AND ?
		GROUP BY e.id
	) AS prev ON curr.id = prev.id
`

	// First 4 args: 2 for current, 2 for previous range
	args = append(args,
		params.StartDate, params.EndDate,
		beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339),
	)

	// Optional filters
	where := " WHERE 1 = 1"
	if params.Search != "" {
		where += " AND curr.full_name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if params.StoreId != "" {
		where += " AND curr.store_name = (SELECT name FROM stores WHERE id = ?)"
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		where += " AND curr.company_id = ? "
		args = append(args, params.CompanyId)
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
	err = s.db.WithContext(ctx).Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top seller: %v", err)
		return nil, 0, domain.InternalServerError
	}
	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

func (s *Services) GetTopStoresReport(ctx context.Context, param *domain.ReportQueryParam) ([]domain.TopStores, int64, error) {
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
			WHERE sales.stage IN(9, 11)
			AND (sales.completed_at + interval '5 hours') BETWEEN ? AND ?
			GROUP BY sales.store_id
		) AS curr
		LEFT JOIN (
			SELECT
				sales.store_id,
				SUM(sales.total_amount) AS total_amount
			FROM sales
			WHERE sales.stage IN(9, 11)
			AND (sales.completed_at + interval '5 hours') BETWEEN ? AND ?
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
	if param.CompanyId != "" {
		whereClauses = append(whereClauses, " stores.company_id = ? ")
		args = append(args, param.CompanyId)
	}

	// Append filters
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Apply flexible ordering
	query += utils.BuildTopStoreOrderClause(param.Order)

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)

	// Execute
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top stores report: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// Extract total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}
	return res, totalCount, nil
}

// get dashboard bonus products
func (s *Services) GetBonusProductsReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.BonusProducts, int64, error) {
	// declaration
	var (
		res        []domain.BonusProducts
		totalCount int64
		args       []any
		startTime  time.Time
		endTime    time.Time
	)

	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		params.EndDate = endTime.Format(time.RFC3339)
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

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
    		SUM(eb.unit_quantity)-ROUND(SUM(eb.unit_quantity)::decimal / p.unit_per_pack,0)*p.unit_per_pack as unit_quantity,
    		SUM(eb.quantity) + ROUND(SUM(eb.unit_quantity)::decimal / p.unit_per_pack,0) AS count,
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
	if params.CompanyId != "" {
		filter += " AND p.company_id = ? "
		args = append(args, params.CompanyId)
	}
	filter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, startTime, endTime)

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
	prevFilter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339))

	query += prevJoin + prevFilter + " GROUP BY p.id ) AS prev ON curr.id = prev.id"

	// New flexible order logic
	order := utils.BuildBonusProductOrderClause(params.Order)
	query += order

	// Pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

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

// get bonus products stats
func (s *Services) GetBonusProductsReportStats(ctx context.Context, params *domain.ReportQueryParam) (domain.BonusProductsStats, error) {
	var (
		res       domain.BonusProductsStats
		args      []any
		startTime time.Time
		endTime   time.Time
	)

	// parse start date
	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		return res, err
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			return res, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)

	query := `
	SELECT
		COUNT(curr.id) AS documents_count,  -- nechta product (document) borligini sanaydi
		COALESCE(SUM(curr.count), 0) + COALESCE(SUM(curr.unit_quantity * curr.unit_per_pack), 0)  AS total_count,
		COALESCE(SUM(curr.unit_quantity), 0) AS total_unit_quantity,
		COALESCE(SUM(curr.bonus_amount), 0) AS total_bonus_amount,
		COALESCE(SUM(prev.bonus_amount), 0) AS previous_bonus_amount,
		ROUND(
			CASE 
				WHEN COALESCE(SUM(prev.bonus_amount), 0) = 0 THEN 100
				ELSE ((SUM(curr.bonus_amount) - SUM(prev.bonus_amount)) * 100.0) / NULLIF(SUM(prev.bonus_amount), 0)
			END, 2
		) AS percent
	FROM (
		SELECT
			p.id,
    		SUM(eb.unit_quantity) - ROUND(SUM(eb.unit_quantity)::decimal / p.unit_per_pack,0) * p.unit_per_pack AS unit_quantity,
    		SUM(eb.quantity) + ROUND(SUM(eb.unit_quantity)::decimal / p.unit_per_pack,0) AS count,
			SUM(eb.bonus_amount) AS bonus_amount,
			p.unit_per_pack
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
	`

	join := ""
	filter := " WHERE 1=1 "
	if len(params.StoreIds) > 0 {
		join += " JOIN employees e ON eb.employee_id = e.id"
		filter += " AND e.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	if params.CompanyId != "" {
		filter += " AND p.company_id = ? "
		args = append(args, params.CompanyId)
	}
	filter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, startTime, endTime)

	group := " GROUP BY p.id, p.unit_per_pack ) AS curr"

	query += join + filter + group

	// prev
	query += `
	LEFT JOIN (
		SELECT
			p.id,
			SUM(eb.bonus_amount) AS bonus_amount
		FROM employee_bonus eb
		JOIN products p ON eb.product_id = p.id
	`

	prevJoin := ""
	prevFilter := " WHERE 1=1 "
	if len(params.StoreIds) > 0 {
		prevJoin += " JOIN employees e ON eb.employee_id = e.id"
		prevFilter += " AND e.store_id IN (?)"
		args = append(args, params.StoreIds)
	}
	prevFilter += " AND (eb.created_at + interval '5 hours') BETWEEN ? AND ?"
	args = append(args, beforeStart, beforeEnd)

	query += prevJoin + prevFilter + " GROUP BY p.id ) AS prev ON curr.id = prev.id"

	// execute
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get bonus_products report stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetStoreSummaryReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.StoreSummary, int64, error) {
	var (
		res       []domain.StoreSummary
		args      []any
		startTime time.Time
		endTime   time.Time
		err       error
	)

	startTime, err = time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return nil, 0, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		params.EndDate = endTime.Format(time.RFC3339)
	}
	var total int64
	// Count query (total store count)
	countQuery := `SELECT COUNT(*) FROM stores WHERE is_active = true`
	err = s.db.Raw(countQuery).Scan(&total).Error
	if err != nil {
		s.log.Error("Failed to count stores: ", err)
		return nil, 0, err
	}

	query := `
	WITH sale_cte AS (
		SELECT
			store_id,
			SUM(CASE
					WHEN (completed_at + interval '5 hours') BETWEEN ? AND ?
						THEN CASE
								WHEN sale_type = 'SALE' THEN total_amount
								WHEN sale_type = 'RETURN' THEN -total_amount
							END
					ELSE 0
				END) AS sale_amount,
			SUM(CASE
					WHEN (completed_at + interval '5 hours') BETWEEN ? AND ?
						THEN CASE
								WHEN sale_type = 'SALE' THEN total_discount
								WHEN sale_type = 'RETURN' THEN -total_discount
							END
					ELSE 0
				END) AS discount_amount
		FROM sales
		WHERE stage IN(9, 11)
		GROUP BY store_id
	),
	import_cte AS (
		SELECT
			im.store_id,
			COALESCE(SUM(imd.received_count * imd.retail_price_vat), 0) AS import_amount
		FROM import_details imd
				 JOIN imports im ON imd.import_id = im.id
		WHERE im.status = 'new' AND im.entry_type = 1
		GROUP BY im.store_id
	),
	stock_cte AS (
		SELECT
			sp.store_id,
			ROUND(SUM(sp.pack_quantity * sp.retail_price) +
				  SUM((sp.retail_price / p.unit_per_pack) * (sp.unit_quantity % p.unit_per_pack)), 2) AS stock_amount
		FROM store_products sp
				 JOIN products p ON sp.product_id = p.id
		GROUP BY sp.store_id
	)
	SELECT
		st.name AS name,
		COALESCE(s.sale_amount, 0) AS sale_amount,
		COALESCE(s.discount_amount, 0) AS discount_amount,
		COALESCE(i.import_amount, 0) AS import_amount,
		COALESCE(k.stock_amount, 0) AS stock_amount,
		ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(k.stock_amount, 0), 2) AS total
	FROM stores st
	LEFT JOIN sale_cte s ON st.id = s.store_id
	LEFT JOIN import_cte i ON st.id = i.store_id
	LEFT JOIN stock_cte k ON st.id = k.store_id
	WHERE st.is_active = true
	`

	// 4 timestamps for 2 BETWEENs (sales & imports)
	args = append(args, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)) // sales
	if params.Order != "" {
		order := utils.BuildStoreSummaryOrderClause(params.Order)
		query += order
	}
	if params.Search != "" {
		query += " AND st.name LIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.StoreIds) > 0 {
		query += " AND st.id = ?"
		args = append(args, params.StoreIds)
	}
	if params.CompanyId != "" {
		query += " AND st.company_id = ? "
		args = append(args, params.CompanyId)
	}
	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}
	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}
	// Execute query
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store summary report: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, total, nil
}

func (s *Services) GetStoreSummaryReportStats(ctx context.Context, params *domain.ReportQueryParam) (domain.StoreSummaryStats, error) {
	var (
		res       domain.StoreSummaryStats
		args      []any
		startTime time.Time
		endTime   time.Time
		err       error
	)

	startTime, err = time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return res, err
	}
	if params.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, params.EndDate)
		if err != nil {
			s.log.Error("Invalid end_date format: %v", err)
			return res, err
		}
	} else {
		endTime = startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		params.EndDate = endTime.Format(time.RFC3339)
	}

	query := `
	WITH sale_cte AS (
		SELECT
			store_id,
			SUM(CASE
					WHEN (completed_at + interval '5 hours') BETWEEN ? AND ?
						THEN CASE
								WHEN sale_type = 'SALE' THEN total_amount
								WHEN sale_type = 'RETURN' THEN -total_amount
							END
					ELSE 0
				END) AS sale_amount,
			SUM(CASE
					WHEN (completed_at + interval '5 hours') BETWEEN ? AND ?
						THEN CASE
								WHEN sale_type = 'SALE' THEN total_discount
								WHEN sale_type = 'RETURN' THEN -total_discount
							END
					ELSE 0
				END) AS discount_amount
		FROM sales
		WHERE stage IN(9, 11)
		GROUP BY store_id
	),
	import_cte AS (
		SELECT
			im.store_id,
			COALESCE(SUM(imd.received_count * imd.retail_price_vat), 0) AS import_amount
		FROM import_details imd
				 JOIN imports im ON imd.import_id = im.id
		WHERE im.status = 'new' AND im.entry_type = 1
		GROUP BY im.store_id
	),
	stock_cte AS (
		SELECT
			sp.store_id,
			ROUND(SUM(sp.pack_quantity * sp.retail_price) +
				  SUM((sp.retail_price / p.unit_per_pack) * (sp.unit_quantity % p.unit_per_pack)), 2) AS stock_amount
		FROM store_products sp
				 JOIN products p ON sp.product_id = p.id
		GROUP BY sp.store_id
	),
	store_summary AS (
		SELECT
			st.name AS name,
			COALESCE(s.sale_amount, 0) AS sale_amount,
			COALESCE(s.discount_amount, 0) AS discount_amount,
			COALESCE(i.import_amount, 0) AS import_amount,
			COALESCE(k.stock_amount, 0) AS stock_amount,
			ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(k.stock_amount, 0), 2) AS total
		FROM stores st
		LEFT JOIN sale_cte s ON st.id = s.store_id
		LEFT JOIN import_cte i ON st.id = i.store_id
		LEFT JOIN stock_cte k ON st.id = k.store_id
		WHERE st.is_active = true
	)
	SELECT
		SUM(sale_amount) AS total_sale_amount,
		SUM(discount_amount) AS total_discount_amount,
		SUM(import_amount) AS total_import_amount,
		SUM(stock_amount) AS total_stock_amount,
		SUM(total) AS total
	FROM store_summary
	`

	args = append(args, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)) // sales

	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store summary stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetStoreProductsGivenDay(ctx context.Context, params *domain.ReportQueryParam) ([]domain.StoreProductsReport, int64, error) {
	var (
		res       []domain.StoreProductsReport
		args      []any
		countArgs []any
		total     int64
		err       error
	)

	startTime, err := time.Parse(time.RFC3339, params.StartDate)
	if err != nil {
		s.log.Error("Invalid start_date format: %v", err)
		return nil, 0, err
	}
	params.StartDate = startTime.Format(time.RFC3339)

	// Count total products for store
	countQuery := `
        SELECT COUNT(DISTINCT sp.product_id)
        FROM store_products sp
        JOIN products p ON p.id = sp.product_id
        JOIN stores st ON st.id = sp.store_id
        WHERE sp.store_id = ?
    `
	countArgs = append(countArgs, params.StoreId)

	if params.Search != "" {
		countQuery += " AND p.name ILIKE ?"
		countArgs = append(countArgs, "%"+params.Search+"%")
	}

	if params.CompanyId != "" {
		countQuery += " AND st.company_id = ?"
		countArgs = append(countArgs, params.CompanyId)
	}

	if err = s.db.Raw(countQuery, countArgs...).Scan(&total).Error; err != nil {
		s.log.Error("Failed to count store products: ", err)
		return nil, 0, err
	}

	// Main backward stock query
	query := `
	WITH vars AS (
	    SELECT ?::timestamp AS target_date,
	           ?::uuid AS target_store
	),

	-- 1. Base stock
	base_stock AS (
	    SELECT
	        sp.product_id,
	        sp.store_id,
			st.name AS store_name,
	        SUM(sp.pack_quantity) AS pack_qty,
	        SUM(sp.unit_quantity) % p.unit_per_pack AS unit_qty,
	        p.unit_per_pack,
	        p.name,
	        st.company_id
	    FROM store_products sp
	    JOIN products p ON p.id = sp.product_id
	    JOIN stores st ON st.id = sp.store_id
	    WHERE sp.store_id = (SELECT target_store FROM vars)
	    GROUP BY sp.product_id, sp.store_id, st.name, p.unit_per_pack, p.name, st.company_id
	),

	-- 2. Future imports
	future_imports AS (
	    SELECT
	        imd.product_id,
	        COALESCE(SUM(imd.accepted_count),0) AS pack_change,
	        COALESCE(SUM((imd.accepted_count * p.unit_per_pack ) % p.unit_per_pack),0) AS unit_change
	    FROM imports im
	    JOIN import_details imd ON im.id = imd.import_id
	    JOIN products p ON p.id = imd.product_id
	    WHERE im.store_id = (SELECT target_store FROM vars)
	      AND im.entry_type = 1
	      AND im.status = 'completed'
	      AND im.import_date > (SELECT target_date FROM vars)
	    GROUP BY imd.product_id
	),

	-- 3. Future sales
	future_sales AS (
	    SELECT
	        sp.product_id,
	        COALESCE(SUM(ci.quantity),0) AS pack_change,
	        COALESCE(SUM(ci.quantity * p.unit_per_pack + ci.unit_quantity) % p.unit_per_pack,0)  AS unit_change
	    FROM sales s
	    JOIN cart_items ci ON ci.sale_id = s.id
	    JOIN store_products sp ON sp.id = ci.store_product_id
	    JOIN products p ON p.id = sp.product_id
	    WHERE s.store_id = (SELECT target_store FROM vars)
	      AND s.stage IN(9, 11)
	      AND s.sale_type = 'SALE'
	      AND s.completed_at > (SELECT target_date FROM vars)
	    GROUP BY sp.product_id, p.unit_per_pack
	),

	-- 4. Future returns
	future_returns AS (
	    SELECT
	        sp.product_id,
	        COALESCE(SUM(ci.quantity),0) AS pack_change,
	        COALESCE(SUM(ci.quantity * p.unit_per_pack + ci.unit_quantity) % p.unit_per_pack,0) AS unit_change
	    FROM sales s
	    JOIN cart_items ci ON ci.sale_id = s.id
	    JOIN store_products sp ON sp.id = ci.store_product_id
	    JOIN products p ON p.id = sp.product_id
	    WHERE s.store_id = (SELECT target_store FROM vars)
	      AND s.stage IN(9, 11)
	      AND s.sale_type = 'RETURN'
	      AND s.completed_at > (SELECT target_date FROM vars)
	    GROUP BY sp.product_id, p.unit_per_pack
	),

	-- 5. Future transfers
	future_transfers AS (
	    SELECT
	        td.product_id,
	        SUM(
	            CASE
	                WHEN t.from_store_id = (SELECT target_store FROM vars)
	                    THEN td.accepted_count
	                WHEN t.to_store_id = (SELECT target_store FROM vars)
	                    THEN -td.accepted_count
	                ELSE 0
	            END
	        ) AS pack_change,
	        SUM(
	            CASE
	                WHEN t.from_store_id = (SELECT target_store FROM vars)
	                    THEN (td.accepted_count * p.unit_per_pack) % p.unit_per_pack
	                WHEN t.to_store_id = (SELECT target_store FROM vars)
	                    THEN -(td.accepted_count * p.unit_per_pack) % p.unit_per_pack
	                ELSE 0
	            END
	        ) AS unit_change
	    FROM transfers t
	    JOIN transfer_details td ON t.id = td.transfer_id
	    JOIN products p ON p.id = td.product_id
	    WHERE (t.from_store_id = (SELECT target_store FROM vars) OR t.to_store_id = (SELECT target_store FROM vars))
	      AND t.status IN ('completed','sent_to_1c')
	      AND t.created_at > (SELECT target_date FROM vars)
	    GROUP BY td.product_id
	),

	-- 6. Future inventory
	future_inventory AS (
	    SELECT
	        imd.product_id,
	        (SUM(imd.scanned_count - imd.received_count) / p.unit_per_pack)::int AS pack_change,
	        (SUM(imd.scanned_count - imd.received_count) % p.unit_per_pack)      AS unit_change
	    FROM imports im
	    JOIN import_details imd ON im.id = imd.import_id
	    JOIN products p ON p.id = imd.product_id
	    WHERE im.store_id = (SELECT target_store FROM vars)
	      AND im.entry_type = 2
	      AND im.status = 'completed'
	      AND im.import_date > (SELECT target_date FROM vars)
	    GROUP BY imd.product_id, p.unit_per_pack
	)

	-- Final calculation
	SELECT
	    b.product_id,
	    b.store_id,
		b.store_name,
	    b.name,
	    (b.pack_qty
	         - COALESCE(fi.pack_change,0)
	         + COALESCE(fs.pack_change,0)
	         - COALESCE(fr.pack_change,0)
	         + COALESCE(ft.pack_change,0)
	        - COALESCE(finv.pack_change,0)
	        ) AS final_pack_quantity,
	    ((b.unit_qty
	        - COALESCE(fi.unit_change,0)
	        + COALESCE(fs.unit_change,0)
	        - COALESCE(fr.unit_change,0)
	        + COALESCE(ft.unit_change,0)
	        - COALESCE(finv.unit_change,0)) % b.unit_per_pack
	    ) AS final_unit_quantity,
	    b.pack_qty,
	    b.unit_qty,
	    COALESCE(fi.pack_change,0)   AS import_pack_change,
		COALESCE(fi.unit_change,0)   AS import_unit_change,
		COALESCE(fs.pack_change,0)   AS sales_pack_change,
		COALESCE(fs.unit_change,0)   AS sales_unit_change,
		COALESCE(fr.pack_change,0)   AS return_pack_change,
		COALESCE(fr.unit_change,0)   AS return_unit_change,
		COALESCE(ft.pack_change,0)   AS transfer_pack_change,
		COALESCE(ft.unit_change,0)   AS transfer_unit_change,
		COALESCE(finv.pack_change,0) AS inventory_pack_change,
		COALESCE(finv.unit_change,0) AS inventory_unit_change,
	    b.company_id
	FROM base_stock b
	LEFT JOIN future_imports fi ON fi.product_id = b.product_id
	LEFT JOIN future_sales fs ON fs.product_id = b.product_id
	LEFT JOIN future_returns fr ON fr.product_id = b.product_id
	LEFT JOIN future_transfers ft ON ft.product_id = b.product_id
	LEFT JOIN future_inventory finv ON finv.product_id = b.product_id
	`

	// args
	args = append(args, params.StartDate, params.StoreId)

	// Filters
	if params.Search != "" {
		query += " WHERE b.name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if params.CompanyId != "" {
		if strings.Contains(query, "WHERE") {
			query += " AND b.company_id = ?"
		} else {
			query += " WHERE b.company_id = ?"
		}
		args = append(args, params.CompanyId)
	}

	// Order, Limit, Offset
	query += utils.BuildProductOrderClause(params.Order)

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}
	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}

	// Execute query
	if err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get store products given day: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, total, nil
}

func (s *Services) GetDiscountCardReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.DiscountCardReport, int64, error) {
	var (
		res        []domain.DiscountCardReport
		totalCount int64
		args       []any
	)

	query := `
		SELECT
		    ROW_NUMBER() OVER(ORDER BY s.id) as id,
    		s.id as store_id,
    		s.name AS store_name,
    		c.id as customer_id,
    		c.full_name AS customer_name,
    		COUNT(DISTINCT sa.id) AS check_count,
    		MAX(dc.percent) as percent,
			ROUND(SUM(sa.total_amount + sa.total_discount), 2) AS total_without_discount,
			ROUND(SUM(sa.total_discount), 2) AS total_discount,
			ROUND(SUM(sa.total_amount), 2) AS total_with_discount,
			COUNT(*) OVER() AS total_count
		FROM sales sa
		JOIN customers c ON sa.customer_id = c.id
		JOIN stores s ON sa.store_id = s.id
        LEFT JOIN discount_cards dc on c.id = dc.customer_id
	`

	filter := " WHERE sa.stage IN(9, 11) AND sa.sale_type = 'SALE' "
	group := " GROUP BY s.id, s.name, c.id, c.full_name "
	order := utils.BuildDiscountCardOrderClause(params.Order)

	// Store filter
	if len(params.StoreIds) > 0 {
		filter += " AND s.id IN (?)"
		args = append(args, params.StoreIds)
	}
	if params.CompanyId != "" {
		filter += " AND s.company_id = ? "
		args = append(args, params.CompanyId)
	}

	// Search filter (by customer name)
	if params.Search != "" {
		search := "%" + params.Search + "%"
		filter += " AND (c.full_name ILIKE ?)"
		args = append(args, search)
	}

	// Date filter
	if params.StartDate != "" && params.EndDate != "" {
		filter += " AND sa.completed_at BETWEEN ? AND ?"
		args = append(args, params.StartDate, params.EndDate)
	} else if params.StartDate != "" {
		filter += " AND sa.completed_at >= ?"
		args = append(args, params.StartDate)
	}

	// Final query
	finalQuery := query + filter + group + order + " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	err := s.db.WithContext(ctx).Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get discount card report: %v", err)
		return res, 0, domain.InternalServerError
	}

	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}
