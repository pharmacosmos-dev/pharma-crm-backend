package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

// get auto order for generate new auto order
func (s *Services) ListAutoOrderForGenerate(ctx context.Context, limit, offset int, storeID, search string) ([]domain.AutoOrder, int64, error) {
	var (
		autoOrders      []domain.AutoOrder
		totalCount      int64
		storeCondition  string
		searchCondition string
	)
	// Add store filter if storeID is provided
	if storeID != "" {
		storeCondition = fmt.Sprintf("WHERE st.store_id = '%s'", storeID)
	}

	// Add search filter if search term is provided
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		searchCondition = fmt.Sprintf(`%s AND  p.name ILIKE '%s'`, storeCondition, search)
	} else {
		searchCondition = storeCondition // Use only store filter if no search term
	}
	query := fmt.Sprintf(`
	WITH weekly_sales AS (
		SELECT
			sp.store_id,
			sp.product_id,
			SUM(ci.quantity) AS weekly_quantity
		FROM
			sales s
		JOIN
			cart_items ci ON s.id = ci.sale_id
		JOIN
			store_products sp ON ci.store_product_id = sp.id
		WHERE
			s.status = 'completed'
			AND s.created_at >= NOW() - INTERVAL '1 week'
		GROUP BY
			sp.store_id, sp.product_id
	),
	monthly_sales AS (
		SELECT
			sp.store_id,
			sp.product_id,
			SUM(ci.quantity) AS monthly_quantity
		FROM
			sales s
		JOIN
			cart_items ci ON s.id = ci.sale_id
		JOIN
			store_products sp ON ci.store_product_id = sp.id
		WHERE
			s.status = 'completed'
			AND s.created_at >= NOW() - INTERVAL '1 month'
		GROUP BY
			sp.store_id, sp.product_id
	),
		stock AS (
			SELECT
				sp.store_id,
				sp.product_id,
				sp.pack_quantity AS current_stock
			FROM
				store_products sp
		)
		SELECT
			gen_random_uuid() AS id,
			st.store_id,
			s.name AS store_name,
			st.product_id,
			p.name AS product_name,
			st.current_stock,
			m.monthly_quantity,
			w.weekly_quantity,
			(w.weekly_quantity-st.current_stock)*1.1 AS order_growth,
			((w.weekly_quantity-st.current_stock)*1.1)*1 AS order_lead_time,
			-- Suggested Order: Based on safety stock (e.g., weekly sales x lead time)
			CASE
				WHEN (w.weekly_quantity-st.current_stock)*1.1>0 
					THEN ROUND((w.weekly_quantity-st.current_stock)*1.1)
				ELSE 0
			END AS suggested_order,
			COUNT(*) OVER() AS total_count
		FROM
			stock st
		INNER JOIN
				stores s ON st.store_id = s.id
		INNER JOIN
				products p on st.product_id = p.id
		LEFT JOIN
			weekly_sales w ON st.store_id = w.store_id AND st.product_id = w.product_id
		LEFT JOIN
			monthly_sales m ON st.store_id = m.store_id AND st.product_id = m.product_id 
		%s
		ORDER BY suggested_order DESC  LIMIT ? OFFSET ?;
	`, searchCondition)
	err := s.db.Raw(query, limit, offset).Scan(&autoOrders).Error
	if err != nil {
		return nil, 0, err
	}
	// Extract total_count from the first row

	return autoOrders, totalCount, nil
}

// get auto order get list
func (s *Services) ListAutoOrder(param *domain.AutoOrderParam) ([]domain.AutoOrder, int64, error) {
	var (
		autoOrders []domain.AutoOrder
		err        error
		totalCount int64
	)

	// get employee info
	var employee domain.Employee
	err = s.db.First(&employee, "id = ?", param.UserId).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, s.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
	}
	// build query
	query := s.db.
		Model(&domain.AutoOrder{}).
		Preload("Store").
		Select(`auto_orders.*, 
		SUM(aod.order_count) AS adjusted_order_quantity,
		SUM(aod.response_order_count) AS response_order_quantity`).
		Joins("JOIN stores s ON auto_orders.store_id = s.id").
		Joins("LEFT JOIN auto_order_details aod ON auto_orders.id = aod.auto_order_id")

	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("CAST(auto_orders.public_id AS TEXT) LIKE ? OR s.name ILIKE ?", param.Search, param.Search)
	}

	if param.StoreID != "" {
		query = query.Where("auto_orders.store_id = ?", param.StoreID)
	}

	if param.Status != "" {
		query = query.Where("auto_orders.status = ?", param.Status)
	}

	if param.StartDate != "" && param.EndDate != "" {
		query = query.Where("auto_orders.created_at::date BETWEEN ? AND ?", param.StartDate, param.EndDate)
	}

	err = query.
		Group("auto_orders.id").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("auto_orders.created_at DESC").
		Debug().
		Find(&autoOrders).Error
	if err != nil {
		s.log.Warn("Failed to get auto orders: %v", err)
		return nil, 0, err
	}
	return autoOrders, totalCount, nil
}

// generate auto order with store_id and day
func (s *Services) GenerateAutoOrderDetail(autoOrderID string, storeID string, day float64) ([]*domain.AutoOrderDetailRequest, error) {
	var res []*domain.AutoOrderDetailRequest
	query := `
	WITH sales_data AS (
		SELECT
			? AS auto_order_id,
			sp.product_id,
			p.material_code,
			p.name,
			ROUND(SUM(sp.pack_quantity::numeric + (sp.unit_quantity::numeric % p.unit_per_pack)/p.unit_per_pack), 4) AS current_stock,
			ROUND(SUM(ci.quantity::numeric + ci.unit_quantity::numeric / p.unit_per_pack), 4) AS sale_count,
			MAX(spt.kvant) AS kvant,
			MAX(spt.min_quantity) AS min_stock,
			MAX(spt.max_quantity) AS max_stock,
			p.unit_per_pack
		FROM sales sl
		INNER JOIN stores s ON sl.store_id = s.id
		INNER JOIN cart_items ci ON sl.id = ci.sale_id
		INNER JOIN store_products sp ON ci.store_product_id = sp.id
		INNER JOIN products p ON sp.product_id = p.id
		INNER JOIN store_product_thresholds spt ON s.id = spt.store_id AND p.id = spt.product_id
		WHERE sl.status = 'completed'
		AND sl.store_id = ?
		AND (sl.completed_at + interval '5 hours')::date >= (CURRENT_DATE - INTERVAL '15 days')
		GROUP BY sp.product_id, p.material_code, p.name, p.unit_per_pack
	),
	calc_logic AS (
		SELECT
			*,
			2 AS import_day,
			15 AS sale_period,
			ROUND(sale_count / 15.0, 4) AS daily_sale_count,
			ROUND(sale_count / 15.0 * 2, 4) AS delivery_day_consumption,
			current_stock - ROUND(sale_count / 15.0 * 2, 4) AS stock_on_delivery_date,
			ROUND(sale_count / 15.0 * 3, 4) AS reserve_quantity,
			(current_stock - ROUND(sale_count / 15.0 * 2, 4)) + ROUND(sale_count / 15.0 * 3, 4) AS future_stock,
			((current_stock - ROUND(sale_count / 15.0 * 2, 4)) + ROUND(sale_count / 15.0 * 3, 4)) - ROUND(sale_count / 15.0 * 3, 4) AS future_stock_with_reserve
		FROM sales_data
	),
	final_calc AS (
		SELECT
			*,
			CASE
				WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve)
				ELSE 0
			END AS w_abs,

			CASE
				WHEN (min_stock - stock_on_delivery_date) <= CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
					THEN CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
				WHEN (min_stock - stock_on_delivery_date) > CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
					THEN (min_stock - stock_on_delivery_date)
			END AS x,

			CASE
				WHEN max_stock > 0 THEN max_stock
				ELSE 1e+99
			END AS z,

			LEAST(
				CASE
					WHEN (min_stock - stock_on_delivery_date) <= CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
						THEN CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
					WHEN (min_stock - stock_on_delivery_date) > CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
						THEN (min_stock - stock_on_delivery_date)
				END,
				CASE
					WHEN max_stock > 0 THEN max_stock
					ELSE 1e+99
				END
			) AS k,

			CASE
				WHEN kvant > 0 THEN ROUND(
					ROUND(
						LEAST(
							CASE
								WHEN (min_stock - stock_on_delivery_date) <= CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
									THEN CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
								WHEN (min_stock - stock_on_delivery_date) > CASE WHEN future_stock_with_reserve < 0 THEN ABS(future_stock_with_reserve) ELSE 0 END
									THEN (min_stock - stock_on_delivery_date)
							END,
							CASE
								WHEN max_stock > 0 THEN max_stock
								ELSE 1e+99
							END
						) / kvant
					) * kvant
				)
				ELSE 0
			END AS j
		FROM calc_logic
	)
	SELECT
	 	auto_order_id,
        product_id,
		material_code,
		name,
		current_stock,
		min_stock,
		max_stock,
		kvant,
		sale_count,
		daily_sale_count,
		import_day,
		sale_period,
		stock_on_delivery_date,
		reserve_quantity,
		future_stock,
		future_stock_with_reserve,
		j AS order_count
	FROM final_calc
	ORDER BY name;
	`
	err := s.db.Debug().Raw(query, autoOrderID, storeID).Scan(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return res, nil
}

// list auto order details
func (s *Services) AutoOrderDetailList(param *domain.AutoOrderParam) ([]domain.AutoOrderDetail, int64, error) {
	var (
		totalCount int64
		res        []domain.AutoOrderDetail
	)
	query := s.db.
		Model(&domain.AutoOrderDetail{}).
		Select("auto_order_details.*, p.material_code, p.name as product_name, u.short_name AS unit_name").
		Preload("AutoOrder").
		Joins("JOIN products p ON p.id = auto_order_details.product_id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id")

	// filter by auto_order_id
	if param.AutoOrderId != "" {
		query = query.Where("auto_order_id = ?", param.AutoOrderId)
	}
	// filter by searching product name
	if param.Search != "" {
		query = query.Where("p.name ILIKE ?", "%"+param.Search+"%")
	}
	// execute query
	err := query.
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting auto order details: %v", err)
		return res, totalCount, err
	}

	return res, totalCount, nil
}
