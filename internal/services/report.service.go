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
