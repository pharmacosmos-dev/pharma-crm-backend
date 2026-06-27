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
	startDate := params.StartDate.GetTime()
	endDate := params.EndDate.GetTime()
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
	`, datesQuery, params.StartDate.UTC().Format(time.RFC3339), params.EndDate.UTC().Format(time.RFC3339))

	query = query + filter + group + order
	// Queryni bajarish
	err := s.db.Raw(query, args...).Scan(&res).Error
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
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		baseQuery = baseQuery.Where("eb.created_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		baseQuery = baseQuery.Where("eb.created_at <= ?", params.EndDate.UTC())
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
			"ROUND(SUM(eb.quantity + (eb.unit_quantity::numeric/p.unit_per_pack)), 2) AS count",
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
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at <= ?", params.EndDate.UTC())
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
		"(sp.supply_price / p.unit_per_pack) * ci.unit_quantity * CASE WHEN s.sale_type = 'RETURN' THEN -1 ELSE 1 END AS supply_price_sum",
		"((sp.retail_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity AS markup_sum",
		"(sp.retail_price / p.unit_per_pack) * ci.unit_quantity * CASE WHEN s.sale_type = 'RETURN' THEN -1 ELSE 1 END AS retail_price_sum",
				"(sp.vat_price / p.unit_per_pack) * ci.unit_quantity AS vat_sum",

		"ci.id AS cart_item_id",
		"ci.unit_quantity",
		"ci.marking_count",

		"s.sale_number",
		"s.sale_type",
		"ROUND((ci.total_price::numeric / NULLIF(SUM(ci.total_price) OVER (PARTITION BY s.id), 0)) * s.total_discount, 2) * CASE WHEN s.sale_type = 'RETURN' THEN -1 ELSE 1 END AS total_discount",
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
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	var res domain.ProductStatusReport
	if err := qb.Take(&res).Error; err != nil {
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
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.id IN(?)", params.StoreIds)
	} else if params.StoreId != "" {
		qb = qb.Where("s.id = ?", params.StoreId)
	}
	if len(params.CompanyIds) > 0 {
		qb = qb.Where("s.company_id IN(?)", params.CompanyIds)
	}
	if params.Search != "" {
		qb = qb.Where("s.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("sa.completed_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("sa.completed_at <= ?", params.EndDate.UTC())
	}
	// Count distinct (store_id, sale_date) combinations using a subquery
	var totalCount int64
	countQuery := qb.Select("s.id", "sa.completed_at::date AS sale_date").Group("s.id, sale_date")
	if err := s.db.Table("(?) as sub", countQuery).Count(&totalCount).Error; err != nil {
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
			"(sa.completed_at + INTERVAL '5 hours')::date AS sale_date",
			"SUM(sa.cash) AS cash",
			"SUM(sa.uzcard) AS uzcard",
			"SUM(sa.humo) AS humo",
			"SUM(sa.click) AS click",
			"SUM(sa.payme) AS payme",
			"SUM(sa.alif) AS alif",
			"SUM(sa.uzum) AS uzum",
			"SUM(sa.uzum_tez_kor) AS uzum_tez_kor",  
			"SUM(sa.total_amount) AS total_amount",
			"SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sa.total_amount * (-1) ELSE 0 END) AS return_amount",
			"SUM(sa.total_discount) AS discount_amount",
			"SUM(sa.loyalty_card) AS loyalty_card_amount",
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
			"SUM(sa.total_amount) AS total_transaction_sum",
			"COUNT(*) AS total_transaction",
			"SUM(CASE WHEN sa.sale_type = 'RETURN' THEN sa.total_amount ELSE 0 END) AS total_returnals_sum",
			"COUNT(*) FILTER (WHERE sa.sale_type = 'RETURN') AS total_returned_count",
			"SUM(sa.total_discount) AS total_discount_sum",
			"COUNT(*) FILTER (WHERE sa.total_discount > 0) AS total_discount_count",
			"SUM(sa.cash) AS total_cash_sum",
			"COUNT(*) FILTER (WHERE sa.cash != 0) AS total_cash_count",
			"SUM(sa.humo) AS total_humo_sum",
			"COUNT(*) FILTER (WHERE sa.humo != 0) AS total_humo_count",
			"SUM(sa.uzcard) AS total_uzcard_sum",
			"COUNT(*) FILTER (WHERE sa.uzcard != 0) AS total_uzcard_count",
			"SUM(sa.click) AS total_click_sum",
			"COUNT(*) FILTER (WHERE sa.click != 0) AS total_click_count",
			"SUM(sa.payme) AS total_payme_sum",
			"COUNT(*) FILTER (WHERE sa.payme != 0) AS total_payme_count",
			"SUM(sa.alif) AS total_alif_sum",
			"COUNT(*) FILTER (WHERE sa.alif != 0) AS total_alif_count",
			"SUM(sa.uzum) AS total_uzum_sum",
			"COUNT(*) FILTER (WHERE sa.uzum != 0) AS total_uzum_count",
			"SUM(sa.loyalty_card) AS total_loyalty_card_sum",
			"COUNT(*) FILTER (WHERE sa.loyalty_card != 0) AS total_loyalty_card_count",
			"SUM(sa.uzum_tez_kor) AS total_uzum_tez_kor_sum",
			"COUNT(*) FILTER (WHERE sa.uzum_tez_kor != 0) AS total_uzum_tez_kor_count",
		).
		Table("stores s").
		Joins("JOIN sales sa ON sa.store_id = s.id")

	// Filters
	qb = qb.Where("sa.stage IN (?)", constants.FinishedSaleStages)

	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.id IN(?)", params.StoreIds)
	} else if params.StoreId != "" {
		qb = qb.Where("s.id = ?", params.StoreId)
	}
	if len(params.CompanyIds) > 0 {
		qb = qb.Where("s.company_id IN(?)", params.CompanyIds)
	}
	if params.Search != "" {
		qb = qb.Where("s.name ILIKE ?", "%"+params.Search+"%")
	}
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("sa.completed_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("sa.completed_at <= ?", params.EndDate.UTC())
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
	qb := s.db.WithContext(ctx).
		Select(
			"p.id AS id",
			"p.name AS name",
			"p.unit_per_pack AS unit_per_pack",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0)) / p.unit_per_pack AS count",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0)) % p.unit_per_pack AS unit_quantity",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0))::NUMERIC / p.unit_per_pack AS quantity",
			"COALESCE(SUM(ci.total_price) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.total_price) FILTER (WHERE s.stage = 11), 0) AS total_amount",
			"COALESCE(SUM(ci.total_price - ci.discount_amount) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.total_price - ci.discount_amount) FILTER (WHERE s.stage = 11), 0) AS net_amount",
		).
		Table("cart_items ci").
		Joins("JOIN sales s ON s.id = ci.sale_id").
		Joins("JOIN products p ON p.id = ci.product_id").
		Where("s.stage IN(?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	if params.Search != "" {
		qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
	}

	// Store filter
	if len(params.StoreIds) > 0 {
		qb.Where("s.store_id IN (?)", params.StoreIds)
	}

	// Sorting (replaced switch)
	order := topProductOrderClause(params.Order)

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get dashboard top products count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	qb = qb.Group("p.id").Order(order).Limit(params.Limit).Offset(params.Offset)
	var res []domain.TopProducts
	// Execute query
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard top products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func topProductOrderClause(order string) string {
	switch order {
	case "+name":
		return "p.name"
	case "-name":
		return "p.name DESC"
	case "+count":
		return "quantity ASC"
	case "-count":
		return "quantity DESC"
	case "+total_amount":
		return "total_amount"
	case "-total_amount":
		return "total_amount DESC"
	case "+net_amount":
		return "net_amount"
	case "-net_amount":
		return "net_amount DESC"
	default:
		return "total_amount DESC"
	}
}

// get report top seller
func (s *Services) GetTopSellersReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.TopSeller, int64, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"e.id",
			"e.full_name",
			"st.name AS store_name",
			"COUNT(s.id) AS count",
			"SUM(s.total_amount) AS total_amount",
		).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Joins("JOIN employees e ON s.employee_id = e.id").
		Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	// Store filter
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN (?)", params.StoreIds)
	}
	if params.StoreId != "" {
		qb = qb.Where("s.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}

	// BuildTopSellerOrderClause returns "ORDER BY x" for raw SQL usage in dashboard;
	// strip that prefix here since GORM's .Order() adds its own.
	order := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(utils.BuildTopSellerOrderClause(params.Order)), "ORDER BY"))

	var totalCount int64
	if err := qb.Group("e.id, e.full_name, st.name").Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get dashboard top products count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	qb = qb.Group("e.id, e.full_name, st.name").Order(order).Limit(params.Limit).Offset(params.Offset)
	var res []domain.TopSeller
	// Execute query
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard top products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetTopStoresReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.TopStores, int64, error) {
	order := utils.BuildTopStoreOrderClause(params.Order)

	qb := s.db.WithContext(ctx).
		Select(
			"st.id AS id",
			"st.name AS name",
			"COUNT(s.id) AS count",
			"SUM(s.total_amount) AS total_amount",
		).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	if params.Search != "" {
		qb.Where("st.name ILIKE ?", "%"+params.Search+"%")
	}

	// Store filter
	if len(params.StoreIds) > 0 {
		qb.Where("s.store_id IN (?)", params.StoreIds)
	}

	// Company filter
	if len(params.CompanyIds) > 0 {
		qb.Where("st.company_id IN (?)", params.CompanyIds)
	}

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get dashboard top products count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// Limit & Offset
	qb = qb.Group("st.id").Order(order).Limit(params.Limit).Offset(params.Offset)

	// Execute query
	var res []domain.TopStores
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("Failed to get top stores: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get dashboard bonus products
func (s *Services) GetBonusProductsReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.BonusProducts, int64, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"p.id AS id",
			"p.name AS name",
			"p.unit_per_pack AS unit_per_pack",
			"SUM(eb.quantity) + SUM(eb.unit_quantity) / p.unit_per_pack AS count",
			"SUM(eb.unit_quantity) % p.unit_per_pack AS unit_quantity",
			"SUM(eb.bonus_amount) AS bonus_amount",
		).
		Table("employee_bonus eb").
		Joins("JOIN products p ON p.id = eb.product_id")

	if params.Search != "" {
		qb = qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
	}

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at <= ?", params.EndDate.UTC())
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("p.company_id IN (?)", params.CompanyIds)
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Joins("JOIN employees e ON eb.employee_id = e.id").
			Where("e.store_id IN (?)", params.StoreIds)
	}

	qb = qb.Group("p.id")

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get bonus products count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	order := utils.BuildBonusProductOrderClause(params.Order)

	var res []domain.BonusProducts
	err := qb.
		Order(order).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get bonus products: %v", err)
		return nil, 0, domain.InternalServerError
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
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		startTime = params.StartDate.UTC()
	} else {
		return res, domain.BadRequestError
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		endTime = params.EndDate.UTC()
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
	if len(params.CompanyIds) > 0 {
		filter += " AND p.company_id IN (?) "
		args = append(args, params.CompanyIds)
	} else if params.CompanyId != "" {
		filter += " AND p.company_id = ? "
		args = append(args, params.CompanyId)
	}
	filter += " AND eb.created_at BETWEEN ? AND ?"
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
	prevFilter += " AND eb.created_at BETWEEN ? AND ?"
	args = append(args, beforeStart, beforeEnd)

	query += prevJoin + prevFilter + " GROUP BY p.id ) AS prev ON curr.id = prev.id"

	// execute
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get bonus_products report stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetBonusProductsByEmployeeId(ctx context.Context, params *domain.ReportQueryParam) ([]domain.BonusProductsByEmployeeDto, int64, error) {
	qb := s.db.WithContext(ctx).
		Joins("JOIN products p ON eb.product_id = p.id").
		Joins("JOIN sales s ON s.id = eb.sale_id").
		Table("employee_bonus eb")

	qb = qb.Where("eb.employee_id = ?", params.EmployeeId)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at >= ?", params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at <= ?", params.EndDate.UTC())
	}

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get bonus_products total_count; %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.BonusProductsByEmployeeDto
	err := qb.Select(
		"eb.id",
		"eb.sale_id",
		"eb.bonus_amount",
		"eb.unit_quantity",
		"(eb.quantity * p.unit_per_pack) + eb.unit_quantity AS u_quantity",
		"eb.created_at",

		"p.id AS product_id",
		"p.material_code",
		"p.name AS product_name",
		"p.unit_per_pack",

		"s.sale_number",
		"s.sale_type",
	).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("eb.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get bonus_products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	for i := range res {
		if res[i].UQuantity%res[i].UnitPerPack > 0 {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)",
				res[i].UQuantity/res[i].UnitPerPack,
				res[i].UQuantity%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].UQuantity/res[i].UnitPerPack)
		}

	}

	return res, totalCount, nil
}

func (s *Services) GetStoreSummaryReport(ctx context.Context, params *domain.ReportQueryParam) ([]domain.StoreSummary, int64, error) {
	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM stores WHERE is_active = true`
	err := s.db.WithContext(ctx).Raw(countQuery).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get stores summary total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	query := `
	WITH sale_cte AS (
        SELECT
                store_id,
                SUM(total_amount) AS sale_amount,
                SUM(total_discount) AS discount_amount,
				SUM(loyalty_card) AS loyalty_card_amount
        FROM sales
        WHERE stage IN(9, 11) AND (completed_at BETWEEN ? AND ?)
        GROUP BY store_id
	),
	import_cte AS (
			SELECT
					im.store_id,
					COALESCE(SUM(im.received_sum), 0) AS import_amount
			FROM imports im
			WHERE im.status = 'new' AND im.entry_type = 1
			GROUP BY im.store_id
	),
	stock_cte AS (
			SELECT
					sp.store_id,
					ROUND(SUM(sp.unit_quantity * (sp.retail_price / p.unit_per_pack)), 2) AS stock_amount
			FROM store_products sp
				JOIN products p ON sp.product_id = p.id
					WHERE sp.unit_quantity > 0
			GROUP BY sp.store_id
	),
	import_stock_cte AS (
			SELECT
					sp.store_id,
					ROUND(SUM((idet.retail_price / NULLIF(p.unit_per_pack, 0)) * sp.unit_quantity), 2) AS import_stock_amount
			FROM store_products sp
				JOIN products p ON sp.product_id = p.id
				JOIN import_details idet ON idet.id = sp.import_detail_id
			WHERE sp.unit_quantity > 0
			GROUP BY sp.store_id
	)
	SELECT
			st.name AS name,
			COALESCE(s.sale_amount, 0) AS sale_amount,
			COALESCE(s.discount_amount, 0) AS discount_amount,
			COALESCE(s.loyalty_card_amount, 0) AS loyalty_card_amount,
			COALESCE(i.import_amount, 0) AS import_amount,
			COALESCE(k.stock_amount, 0) AS stock_amount,
			COALESCE(isk.import_stock_amount, 0) AS import_stock_amount,
			ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) - COALESCE(s.loyalty_card_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(k.stock_amount, 0), 2) AS total,
			ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) - COALESCE(s.loyalty_card_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(isk.import_stock_amount, 0), 2) AS import_total
	FROM stores st
	LEFT JOIN sale_cte s ON st.id = s.store_id
	LEFT JOIN import_cte i ON st.id = i.store_id
	LEFT JOIN stock_cte k ON st.id = k.store_id
	LEFT JOIN import_stock_cte isk ON st.id = isk.store_id
	WHERE st.is_active = true
	`

	var args []any
	var startDate time.Time
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		startDate = params.StartDate.UTC()
	} else {
		now := time.Now().UTC()
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	endDate := domain.AddDefaultDuration(domain.CustomTime(startDate), params.EndDate).UTC()
	args = append(args, startDate, endDate)
	if params.Search != "" {
		query += " AND st.name LIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.StoreIds) > 0 {
		query += " AND st.id IN(?)"
		args = append(args, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		query += " AND st.company_id IN(?)"
		args = append(args, params.CompanyIds)
	}
	if params.Order != "" {
		order := utils.BuildStoreSummaryOrderClause(params.Order)
		query += order
	}
	query += " LIMIT ? OFFSET ? "
	args = append(args, params.Limit, params.Offset)

	var res []domain.StoreSummary
	// Execute query
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store summary report: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetStoreSummaryReportStats(ctx context.Context, params *domain.ReportQueryParam) (domain.StoreSummaryStats, error) {
	var (
		res    domain.StoreSummaryStats
		args   []any
		filter = ""
	)

	var startDateStats time.Time
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		startDateStats = params.StartDate.UTC()
	} else {
		now := time.Now().UTC()
		startDateStats = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	endDateStats := domain.AddDefaultDuration(domain.CustomTime(startDateStats), params.EndDate).UTC()
	args = append(args, startDateStats, endDateStats)
	if params.Search != "" {
		filter += " AND st.name LIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.StoreIds) > 0 {
		filter += " AND st.id IN(?)"
		args = append(args, params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		filter += " AND st.company_id IN(?)"
		args = append(args, params.CompanyIds)
	}

	query := fmt.Sprintf(`
	WITH sale_cte AS (
		SELECT
			store_id,
			SUM(total_amount) AS sale_amount,
			SUM(total_discount) AS discount_amount,
			SUM(loyalty_card) AS loyalty_card_amount
		FROM sales
		WHERE stage IN(9, 11) AND (completed_at BETWEEN ? AND ?)
		GROUP BY store_id
	),
	import_cte AS (
		SELECT
			im.store_id,
			COALESCE(SUM(im.received_sum), 0) AS import_amount
		FROM imports im
		WHERE im.status = 'new' AND im.entry_type = 1
		GROUP BY im.store_id
	),
	stock_cte AS (
		SELECT
			sp.store_id,
			ROUND(SUM(sp.unit_quantity * (sp.retail_price/p.unit_per_pack)), 2) AS stock_amount
		FROM store_products sp
			JOIN products p ON sp.product_id = p.id
				WHERE sp.unit_quantity > 0
		GROUP BY sp.store_id
	),
	import_stock_cte AS (
		SELECT
			sp.store_id,
			ROUND(SUM((idet.retail_price / NULLIF(p.unit_per_pack, 0)) * sp.unit_quantity), 2) AS import_stock_amount
		FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			JOIN import_details idet ON idet.id = sp.import_detail_id
		WHERE sp.unit_quantity > 0
		GROUP BY sp.store_id
	),
	store_summary AS (
		SELECT
			st.name AS name,
			COALESCE(s.sale_amount, 0) AS sale_amount,
			COALESCE(s.discount_amount, 0) AS discount_amount,
			COALESCE(s.loyalty_card_amount, 0) AS loyalty_card_amount,
			COALESCE(i.import_amount, 0) AS import_amount,
			COALESCE(k.stock_amount, 0) AS stock_amount,
			COALESCE(isk.import_stock_amount, 0) AS import_stock_amount,
			ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) - COALESCE(s.loyalty_card_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(k.stock_amount, 0), 2) AS total,
			ROUND(COALESCE(s.sale_amount, 0) - COALESCE(s.discount_amount, 0) - COALESCE(s.loyalty_card_amount, 0) + COALESCE(i.import_amount, 0) + COALESCE(isk.import_stock_amount, 0), 2) AS import_total
		FROM stores st
		LEFT JOIN sale_cte s ON st.id = s.store_id
		LEFT JOIN import_cte i ON st.id = i.store_id
		LEFT JOIN stock_cte k ON st.id = k.store_id
		LEFT JOIN import_stock_cte isk ON st.id = isk.store_id
		WHERE st.is_active = true
		%s
	)
	SELECT
		SUM(sale_amount) AS total_sale_amount,
		SUM(discount_amount) AS total_discount_amount,
		SUM(loyalty_card_amount) AS total_loyalty_card_amount,
		SUM(import_amount) AS total_import_amount,
		SUM(stock_amount) AS total_stock_amount,
		SUM(import_stock_amount) AS total_import_stock_amount,
		SUM(total) AS total,
		SUM(import_total) AS import_total
	FROM store_summary
	`, filter)

	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store summary stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetStoreProductsGivenDay(ctx context.Context, params *domain.StoreProductGivenDayParams) ([]domain.OstatokForDate, int64, error) {
	args := []any{
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId, params.Date,
		params.StoreId,
	}

	// Main backward stock query
	query := `
	WITH import_data AS (
			SELECT
				p.id AS product_id,
				SUM(imd.scanned_count * p.unit_per_pack) import_quantity
			FROM import_details imd
			JOIN products p ON imd.product_id = p.id
			JOIN imports im ON imd.import_id = im.id
			WHERE im.entry_type = 1
			AND im.status = 'completed'
			AND im.store_id = ?
			AND im.created_at >= ?
			GROUP BY p.id
		),
		sale_data AS (
			SELECT
				p.id AS product_id,
				SUM(ci.unit_quantity) AS sold_quantity
			FROM cart_items ci
			JOIN store_products sp ON ci.store_product_id = sp.id
			JOIN products p ON sp.product_id = p.id
			JOIN sales s ON ci.sale_id = s.id
			WHERE s.stage = 9
			AND s.store_id = ?
			AND s.completed_at >= ?
			GROUP BY p.id
		),
		return_data AS (
			SELECT
				p.id AS product_id,
				SUM(ci.unit_quantity) AS return_quantity
			FROM cart_items ci
			JOIN store_products sp ON ci.store_product_id = sp.id
			JOIN products p ON sp.product_id = p.id
			JOIN sales s ON ci.sale_id = s.id
			WHERE s.stage = 11
			AND s.store_id = ?
			AND s.completed_at >= ?
			GROUP BY p.id
		),
		vozvrat_data AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS vozvrat_quantity
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE tr.entry_type = 2
			AND tr.status IN('sent-to-1c', 'completed')
			AND tr.from_store_id = ?
			AND tr.created_at >= ?
			GROUP BY p.id
		),
		transfer_in AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS transfer_in_quantity
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE tr.entry_type = 1
			AND tr.status = 'completed'
			AND tr.to_store_id = ?
			AND tr.created_at >= ?
			GROUP BY p.id
		),
		transfer_out AS (
			SELECT
				p.id AS product_id,
				SUM(td.accepted_count * p.unit_per_pack) AS transfer_out_quantity
			FROM transfer_details td
			JOIN transfers tr ON td.transfer_id = tr.id
			JOIN products p ON td.product_id = p.id
			WHERE tr.entry_type = 1
			AND tr.status = 'completed'
			AND tr.from_store_id = ?
			AND tr.created_at >= ?
			GROUP BY p.id
		),
		inventory_data AS (
			SELECT
				p.id AS product_id,
				SUM(imd.scanned_count - imd.received_count) AS inventory_quantity
			FROM import_details imd
			JOIN products p ON imd.product_id = p.id
			JOIN imports im ON imd.import_id = im.id
			WHERE im.entry_type = 2
			AND im.status = 'completed'
			AND im.store_id = ?
			AND im.updated_at >= ?
			GROUP BY p.id
			),
		ostatok AS (
			SELECT
				p.id AS product_id,
				SUM(sp.unit_quantity) AS ostatok,
				MAX(sp.expire_date) AS expire_date,
				MAX(sp.supply_price) AS supply_price,
				MIN(sp.supply_price) AS min_supply_price,
				MAX(sp.retail_price) AS retail_price,
				MIN(sp.retail_price) AS min_retail_price
			FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			WHERE sp.store_id = ?
			GROUP BY p.id
		)
	SELECT
		p.id            AS product_id,
		p.name          AS name,
		p.unit_per_pack AS unit_per_pack,
		os.expire_date  AS expire_date,
		os.supply_price AS supply_price,
		os.min_supply_price AS min_supply_price,
		os.retail_price AS retail_price,
		os.min_retail_price AS min_retail_price,
		os.ostatok - COALESCE(im.import_quantity, 0) +
		COALESCE(sd.sold_quantity, 0) -
		COALESCE(rd.return_quantity, 0) +
		COALESCE(vd.vozvrat_quantity, 0) -
		COALESCE(ti.transfer_in_quantity, 0) +
		COALESCE(tro.transfer_out_quantity, 0) +
		COALESCE(ind.inventory_quantity, 0) AS unit_quantity
	FROM products p
	JOIN ostatok os ON os.product_id = p.id
	LEFT JOIN import_data im ON im.product_id = p.id
	LEFT JOIN sale_data sd ON sd.product_id = p.id
	LEFT JOIN return_data rd ON rd.product_id = p.id
	LEFT JOIN vozvrat_data vd ON vd.product_id = p.id
	LEFT JOIN transfer_in ti ON ti.product_id = p.id
	LEFT JOIN transfer_out tro ON tro.product_id = p.id
	LEFT JOIN inventory_data ind ON ind.product_id = p.id
	`
	if params.Search != "" {
		query += " WHERE p.name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}

	query += " LIMIT ? OFFSET ?;"
	args = append(args, params.Limit, params.Offset)

	// Execute query
	var res []domain.OstatokForDate
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get store products given day: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, int64(len(res)), nil
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
	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		filter += " AND sa.completed_at >= ?"
		args = append(args, params.StartDate.UTC())
	}
	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		filter += " AND sa.completed_at <= ?"
		args = append(args, params.EndDate.UTC())
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
