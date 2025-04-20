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
		e.id AS employee_id,
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
