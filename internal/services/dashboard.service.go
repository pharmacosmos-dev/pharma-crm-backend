package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

// get dashboard count and amount data
func (s *Services) DashboardTotalCountStats(param *domain.DashboardQueryParam) (*domain.TotalCountStats, error) {
	// declarations
	var (
		res   domain.TotalCountStats
		stock struct {
			StockTotalAmount   float64 `gorm:"stock_total_amount" json:"stock_total_amount"`
			ExpiringSoonCount  int64   `gorm:"expiring_soon_count" json:"expiring_soon_count"`
			TotalProductCount  int64   `gorm:"total_product_count" json:"total_product_count"`
			ExpiringSoonAmount float64 `gorm:"expiring_soon_amount" json:"expiring_soon_amount"`
		}
		totalSale struct {
			TotalSaleCount  float64 `gorm:"total_sale_count" json:"total_sale_count"`
			TotalSaleAmount float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
		}
	)
	// queries
	var (
		// get total sale count and amount
		querys = s.db.Table("sales").Select("COUNT(*) AS total_sale_count, SUM(total_amount) AS total_sale_amount").Where("status = 'completed' AND sale_type = 'SALE'")
		// get total product count and amount
		queryp = s.db.Table("store_products").Select(`
			COALESCE(SUM(pack_quantity), 0) AS total_product_count,
			COALESCE(SUM(pack_quantity*retail_price), 0) AS stock_total_amount,
			COALESCE(SUM(CASE WHEN expire_date <= NOW() + INTERVAL '10 days' THEN pack_quantity ELSE 0 END), 0) AS expiring_soon_count,
			COALESCE(SUM(CASE WHEN expire_date <= NOW() + INTERVAL '10 days' THEN pack_quantity*retail_price ELSE 0 END), 0) AS expiring_soon_amount
		`).Where("expire_date::date >= current_date")
		// get total net income
		queryc = s.db.Table("cart_items ci").
			Select(`COALESCE(SUM((ci.unit_price - sp.supply_price) * ci.quantity), 0) AS total_net_income`).
			Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
			Joins("JOIN sales s ON ci.sale_id = s.id").Where("s.status = 'completed'")
	)
	// filter by several store ids
	if len(param.StoreIds) > 0 {
		querys = querys.Where("store_id IN (?)", param.StoreIds)
		queryp = queryp.Where("store_id IN (?)", param.StoreIds)
		queryc = queryc.Where("s.store_id IN (?)", param.StoreIds)
	}

	// if start date is not empty
	if param.StartDate != "" && param.EndDate == "" {
		querys = querys.Where("completed_at::date = ?", param.StartDate)
		queryp = queryp.Where("expire_date::date >= ?", param.StartDate)
		queryc = queryc.Where("s.completed_at::date = ?", param.StartDate)
	}

	// if end date is not empty
	if param.StartDate != "" && param.EndDate != "" {
		querys = querys.Where("completed_at::date >= ? AND completed_at::date <= ?", param.StartDate, param.EndDate)
		queryp = queryp.Where("expire_date::date >= ? AND expire_date::date <= ?", param.StartDate, param.EndDate)
		queryc = queryc.Where("s.completed_at::date >= ? AND s.completed_at::date <= ?", param.StartDate, param.EndDate)
	}

	// get total sale count and amount

	err := querys.Scan(&totalSale).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	// get total product count

	err = queryp.Scan(&stock).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	var totalNetIncome float64
	// get total net income
	err = queryc.Scan(&totalNetIncome).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	res.TotalSaleCount = totalSale.TotalSaleCount
	res.TotalSaleAmount = totalSale.TotalSaleAmount
	res.TotalProductCount = stock.TotalProductCount
	res.StockTotalAmount = stock.StockTotalAmount
	res.ExpiringSoonCount = stock.ExpiringSoonCount
	res.ExpiringSoonAmount = stock.ExpiringSoonAmount
	res.TotalNetIncome = totalNetIncome

	// get store count by checking store_id is not emply
	if param.StoreId != "" {
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
func (s *Services) DashboardChartStats(param *domain.DashboardQueryParam) ([]domain.ChartResponse, error) {
	var res []domain.ChartResponse

	// queries
	var (
		args  []any
		query = `
		SELECT COUNT(*) as count, SUM(total_amount) as total_amount, 
		%s AS created_at, %s AS id
		FROM sales`
		filter     = " WHERE status = 'completed'"
		group      string
		timeColumn string
	)

	// intervalType ga qarab vaqtni formatlash
	switch param.Type {
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
	if param.StoreId != "" {
		filter += " AND store_id IN (?)"
		args = append(args, param.StoreId)
	}
	// check store_ids
	if len(param.StoreIds) > 0 {
		filter += " AND store_id IN (?)"
		args = append(args, param.StoreIds)
	}
	// filter by only start_date if end_date is empty
	if param.StartDate != "" && param.EndDate == "" {
		filter += " AND completed_at::date = ?"
		args = append(args, param.StartDate)
	}
	// filter by start_date and end_date if both are not empty
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND completed_at::date >= ? AND completed_at::date <= ?"
		args = append(args, param.StartDate, param.EndDate)
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
		filter = " WHERE ci.status = 'sold'"
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
	var (
		res []domain.BonusProducts
	)
	// query
	var (
		args  []any
		query = `
		SELECT
			p.id,
			p.name,
			CAST(SUM(ci.quantity) AS TEXT) || ',' || CAST(SUM(ci.unit_quantity) AS TEXT) AS count,
			COALESCE(SUM(pb.bonus_amount), 0) AS bonus_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		JOIN products p ON sp.product_id = p.id
		JOIN product_bonuses pb ON sp.product_id = pb.product_id`
		filter = " WHERE ci.status = 'sold'"
		group  = " GROUP BY p.id, p.name"
		order  = " ORDER BY bonus_amount DESC"
	)
	if param.StoreId != "" {
		filter += " AND sp.store_id = ?"
		args = append(args, param.StoreId)
	}
	// check store_ids
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
			SUM(ci.quantity) AS count,
			COALESCE(SUM(s.total_amount), 0)  AS total_amount
		FROM sales s
		JOIN employees e ON s.employee_id = e.id
		LEFT JOIN cart_items ci ON ci.sale_id = s.id`
		filter = "	WHERE s.status = 'completed'"
		group  = " GROUP BY e.id, e.full_name"
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
	if param.StartDate != "" && param.EndDate == "" {
		filter += " AND s.completed_at::date = ?"
		args = append(args, param.StartDate)
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += " AND s.completed_at::date >= ? AND s.completed_at::date <= ?"
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
