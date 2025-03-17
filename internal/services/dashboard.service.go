package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

// get dashboard count and amount data
func (s *Services) DashboardTotalCountStats(storeId, startDate, endDate string) (*domain.TotalCountStats, error) {
	// declarations
	var (
		res       domain.TotalCountStats
		totalSale struct {
			TotalSaleCount  float64 `gorm:"total_sale_count" json:"total_sale_count"`
			TotalSaleAmount float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
		}
		productCount int64
	)
	// queries
	var (
		args    []interface{}
		querys  = `SELECT COUNT(*) AS total_sale_count, SUM(total_amount) AS total_sale_amount FROM sales`
		queryp  = `SELECT COALESCE(SUM(pack_quantity), 0) AS total_product_count FROM store_products`
		filters = " WHERE status = 'completed'"
		filterp = " WHERE expire_date::date >= current_date "
	)
	// if store id is not empty
	if storeId != "" {
		filters += " AND store_id = ?"
		filterp += " AND store_id = ?"
		args = append(args, storeId)
	}

	// if start date is not empty
	if startDate != "" && endDate == "" {
		filters += " AND completed_at >= ?"
		filterp += " AND created_at >= ?"
		args = append(args, startDate)
	}

	// if end date is not empty
	if startDate != "" && endDate != "" {
		filters += " AND completed_at >= ? AND completed_at <= ?"
		filterp += " AND created_at >= ? AND created_at <= ?"
		args = append(args, startDate, endDate)
	}

	// get total sale count and amount
	var q = querys + filters
	err := s.db.Raw(q, args...).Scan(&totalSale).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	// get total product count
	var qp = queryp + filterp
	err = s.db.Debug().Raw(qp, args...).Scan(&productCount).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	res.TotalSaleCount = totalSale.TotalSaleCount
	res.TotalSaleAmount = totalSale.TotalSaleAmount
	res.TotalProductCount = productCount

	// get store count by checking store_id is not emply
	if storeId != "" {
		res.TotalStoreCount = 1
	} else {
		err = s.db.Model(&domain.Store{}).Count(&res.TotalStoreCount).Error
		if err != nil {
			s.log.Error(err)
			return nil, err
		}
	}

	return &res, nil
}

// get dashboard chart stats data list
func (s *Services) DashboardChartStats(storeId, employeeId string, startDate, endDate string, intervalType string) ([]domain.ChartResponse, error) {
	var res []domain.ChartResponse

	// queries
	var (
		args  []interface{}
		query = `
		SELECT COUNT(*) as count, SUM(total_amount) as total_amount, 
		%s AS created_at, %s AS id
		FROM sales`
		filter     = " WHERE status = 'completed'"
		group      string
		timeColumn string
	)

	// intervalType ga qarab vaqtni formatlash
	switch intervalType {
	case "HALF_HOURLY":
		timeColumn = `
		DATE_TRUNC('hour', completed_at) + 
		INTERVAL '30 minutes' * FLOOR(EXTRACT(minute FROM completed_at) / 30)`
		group = " GROUP BY DATE_TRUNC('hour', completed_at), FLOOR(EXTRACT(minute FROM completed_at) / 30)"
	case "HOURLY":
		timeColumn = "DATE_TRUNC('hour', completed_at)" // Soatlik
		group = " GROUP BY DATE_TRUNC('hour', completed_at)"
	case "DAILY":
		timeColumn = "completed_at::date" // Kunlik
		group = " GROUP BY completed_at::date"
	case "WEEKLY":
		timeColumn = "DATE_TRUNC('week', completed_at)" // Haftalik
		group = " GROUP BY DATE_TRUNC('week', completed_at)"
	case "MONTHLY":
		timeColumn = "DATE_TRUNC('month', completed_at)" // Oylik
		group = " GROUP BY DATE_TRUNC('month', completed_at)"
	case "YEARLY":
		timeColumn = "DATE_TRUNC('year', completed_at)" // Yillik
		group = " GROUP BY DATE_TRUNC('year', completed_at)"
	default:
		timeColumn = "DATE_TRUNC('hour', completed_at)" // Default Soatlik
		group = " GROUP BY DATE_TRUNC('hour', completed_at)"
	}

	// filter by store_id and employee_id if store_id is not empty
	if storeId != "" {
		filter += " AND store_id = ? AND employee_id = ?"
		args = append(args, storeId, employeeId)
	}
	// filter by only start_date if end_date is empty
	if startDate != "" && endDate == "" {
		filter += " AND completed_at >= ?"
		args = append(args, startDate)
	}
	// filter by start_date and end_date if both are not empty
	if startDate != "" && endDate != "" {
		filter += " AND completed_at >= ? AND completed_at <= ?"
		args = append(args, startDate, endDate)
	}

	// final query
	var q = fmt.Sprintf(query, timeColumn, timeColumn) + filter + group
	err := s.db.Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return res, nil
}

// get dashboard top stores
func (s *Services) DashboardTopStores(storeId, employeeId, startDate, endDate string) ([]domain.TopStores, error) {
	// declaration
	var (
		res []domain.TopStores
	)
	// query
	var (
		args   []interface{}
		query  = `SELECT stores.id, stores.name, COUNT(*) AS count, SUM(sales.total_amount) AS total_amount FROM sales INNER JOIN stores ON sales.store_id = stores.id`
		filter = " WHERE sales.status = 'completed'"
		group  = " GROUP BY stores.id"
		order  = " ORDER BY total_amount DESC"
	)
	if storeId != "" {
		filter += " AND sales.id = ? AND sales.employee_id = ?"
		args = append(args, storeId, employeeId)
	}
	if startDate != "" && endDate == "" {
		filter += " AND sales.completed_at >= ?"
		args = append(args, startDate)
	}
	if startDate != "" && endDate != "" {
		filter += " AND sales.completed_at >= ? AND sales.completed_at <= ?"
		args = append(args, startDate, endDate)
	}

	var q = query + filter + group + order
	err := s.db.Debug().Raw(q, args...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return res, nil
}
