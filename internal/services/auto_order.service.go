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
	err := s.db.Debug().Raw(query, limit, offset).Scan(&autoOrders).Error
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
	WITH vars AS (
		SELECT
			?::uuid AS store_id,
			?::uuid AS auto_order_id,
			2 AS import_day,
			?::int AS sale_period
	),
	stock_data AS (
		SELECT
			sp.product_id,
			ROUND(SUM(sp.pack_quantity + (sp.unit_quantity::numeric % p.unit_per_pack) / p.unit_per_pack), 4) AS current_stock
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON sp.store_id = v.store_id
		GROUP BY sp.product_id, p.unit_per_pack
	),
	sales_data AS (
		SELECT
			sp.product_id,
			ROUND(SUM(ci.quantity + (ci.unit_quantity::numeric / p.unit_per_pack)), 4) AS sale_count
		FROM store_products sp
		JOIN cart_items ci ON sp.id = ci.store_product_id
		JOIN sales sl ON sl.id = ci.sale_id AND sl.status = 'completed'
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON sl.store_id = v.store_id
		WHERE (sl.completed_at + interval '5 hours')::date >= (CURRENT_DATE - INTERVAL '15 days')
		GROUP BY sp.product_id, p.unit_per_pack
	),
	thresholds AS (
		SELECT
			product_id,
			COALESCE(MAX(kvant), 0) AS kvant,
			COALESCE(MAX(min_quantity), 0) AS min_stock,
			COALESCE(MAX(max_quantity), 0) AS max_stock
		FROM store_product_thresholds
		JOIN vars v ON store_product_thresholds.store_id = v.store_id
		GROUP BY product_id
	),
	product_base AS (
		SELECT DISTINCT sp.product_id
		FROM store_products sp
		JOIN vars v ON sp.store_id = v.store_id
	),
 	imports AS (
         SELECT
             i.store_id,
             ip.product_id,
             COUNT(*) AS new_imports_count
         FROM imports i
                  JOIN import_details ip ON i.id = ip.import_id
                  JOIN vars v ON i.store_id = v.store_id
         WHERE i.status = 'new'
         GROUP BY i.store_id, ip.product_id
	),
     excluded AS (
         SELECT ep.product_id
         FROM excluded_products ep
                  JOIN vars v ON ep.store_id = v.store_id
         UNION
         SELECT product_id
         FROM excluded_products
         WHERE store_id IS NULL
     ),

	all_data AS (
		SELECT
			v.auto_order_id,
			v.store_id,
			p.id AS product_id,
			p.material_code,
			p.name,
			p.unit_per_pack,
			COALESCE(sd.current_stock, 0) AS current_stock,
			COALESCE(sa.sale_count, 0) AS sale_count,
			COALESCE(t.kvant, 0) AS kvant,
			COALESCE(t.min_stock, 0) AS min_stock,
			COALESCE(t.max_stock, 0) AS max_stock,
			v.import_day,
			v.sale_period
		FROM product_base pb
		JOIN products p ON pb.product_id = p.id
		JOIN vars v ON TRUE
		LEFT JOIN stock_data sd ON p.id = sd.product_id
		LEFT JOIN sales_data sa ON p.id = sa.product_id
		LEFT JOIN thresholds t ON p.id = t.product_id
	),
	final_calc AS (
		SELECT
			*,
			ROUND(sale_count / sale_period, 4) AS daily_sale_count,
			ROUND((sale_count / sale_period) * import_day, 4) AS delivery_day_consumption,
			current_stock - ROUND((sale_count / sale_period) * import_day, 4) AS stock_on_delivery_date,
			ROUND((sale_count / sale_period) * 3, 4) AS reserve_quantity,
			current_stock - ROUND((sale_count / sale_period) * import_day, 4) + ROUND((sale_count / sale_period) * 3, 4) AS future_stock,
			current_stock - ROUND((sale_count / sale_period) * import_day, 4) AS future_stock_with_reserve
		FROM all_data
	),
	order_calc AS (
		SELECT
             fc.*,
			GREATEST(min_stock - stock_on_delivery_date, 0) AS required_stock,
			CASE
				WHEN kvant > 0 THEN
					ROUND(ROUND(GREATEST(min_stock - stock_on_delivery_date, 0) / kvant) * kvant)
				ELSE 0
                 END
                 - COALESCE(im.new_imports_count, 0) AS order_count,
             ex.product_id AS excluded_product_id
         FROM final_calc fc
                  LEFT JOIN imports im ON im.store_id = fc.store_id AND im.product_id = fc.product_id
                  LEFT JOIN excluded ex ON ex.product_id = fc.product_id
	)
	SELECT
    	fc.auto_order_id,
    	fc.product_id,
    	fc.material_code,
    	fc.name,
    	fc.current_stock,
    	fc.min_stock,
    	fc.max_stock,
    	fc.kvant,
    	fc.sale_count,
    	fc.daily_sale_count,
    	fc.import_day,
    	fc.sale_period,
    	fc.stock_on_delivery_date,
    	fc.reserve_quantity,
    	fc.future_stock,
    	fc.future_stock_with_reserve,
		order_count
	FROM order_calc fc
	WHERE order_count > 0
  	AND fc.excluded_product_id IS NULL
	ORDER BY fc.name;
	`
	err := s.db.Raw(query, storeID, autoOrderID, day).Scan(&res).Error

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
		Order("auto_order_details.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting auto order details: %v", err)
		return res, totalCount, err
	}

	return res, totalCount, nil
}
