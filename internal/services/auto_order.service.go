package services

import (
	"fmt"
	"strings"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

// get auto order get list
func (s *Services) ListAutoOrder(param *domain.AutoOrderParam) ([]domain.AutoOrder, int64, error) {
	var (
		autoOrders []domain.AutoOrder
		totalCount int64
	)

	// 1. get employee info
	var employee domain.Employee
	err := s.db.First(&employee, "id = ?", param.UserId).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	if !helper.IsAdmin(employee, s.cfg) {
		if employee.StoreId != "" {
			param.StoreID = employee.StoreId
		}
		param.CompanyID = employee.CompanyId
	}

	// 2. Build WHERE conditions dynamically
	var whereClauses []string
	params := map[string]interface{}{
		"limit":  param.Limit,
		"offset": param.Offset,
	}

	if param.StoreID != "" {
		whereClauses = append(whereClauses, "store_id = @store_id")
		params["store_id"] = param.StoreID
	}
	if param.CompanyID != "" {
		whereClauses = append(whereClauses, "s.company_id = @company_id")
		params["company_id"] = param.CompanyID
	}
	if param.Status != "" {
		whereClauses = append(whereClauses, "status = @status")
		params["status"] = param.Status
	}
	if param.Search != "" {
		whereClauses = append(whereClauses, "(CAST(public_id AS TEXT) ILIKE @search OR EXISTS (SELECT 1 FROM stores s WHERE s.id = auto_orders.store_id AND s.name ILIKE @search))")
		params["search"] = "%" + param.Search + "%"
	}
	if param.StartDate != "" && param.EndDate != "" {
		whereClauses = append(whereClauses, "created_at::date BETWEEN @start_date AND @end_date")
		params["start_date"] = param.StartDate
		params["end_date"] = param.EndDate
	}

	// 3. Join WHERE clauses
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 4. Main query with CTE
	query := fmt.Sprintf(`
		WITH latest_orders AS (
			SELECT ao.*
			FROM auto_orders ao
			JOIN stores s ON ao.store_id = s.id
			%s
			ORDER BY ao.created_at DESC
			LIMIT @limit OFFSET @offset
		)
		SELECT
			lo.*,
			SUM(aod.order_count) AS adjusted_order_quantity,
			SUM(aod.response_order_count) AS response_order_quantity
		FROM latest_orders lo
		JOIN stores s ON lo.store_id = s.id
		LEFT JOIN auto_order_details aod ON lo.id = aod.auto_order_id
		GROUP BY lo.id, lo.public_id, lo.store_id, lo.status, lo.auto_order_date, lo.completed_date, lo.created_by, lo.updated_by, lo.created_at, lo.updated_at
		ORDER BY lo.created_at DESC
	`, whereSQL)

	// 5. Run query
	err = s.db.Raw(query, params).Scan(&autoOrders).Error
	if err != nil {
		s.log.Warn("Failed to get auto orders: %v", err)
		return nil, 0, err
	}

	var stores []domain.Store
	err = s.db.Find(&stores).Error
	if err != nil {
		s.log.Warn("could not get store list: %v", err)
		return nil, 0, err
	}
	storesMap := make(map[string]*domain.Store, len(stores))

	for _, v := range stores {
		storesMap[v.Id] = &v
	}

	for i, k := range autoOrders {
		autoOrders[i].Store = storesMap[k.StoreId]
	}

	// faqat filterlar kerak
	countParams := map[string]any{}
	for k, v := range params {
		if k != "limit" && k != "offset" {
			countParams[k] = v
		}
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) as count
		FROM auto_orders ao
		JOIN stores s ON ao.store_id = s.id
		%s
	`, whereSQL)

	if whereSQL != "" {
		err = s.db.Raw(countQuery, countParams).Scan(&totalCount).Error
	} else {
		err = s.db.Raw(countQuery).Scan(&totalCount).Error
	}

	if err != nil {
		s.log.Warn("Failed to count auto orders: %v", err)
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

	-- Current stock per product name
	stock_data AS (
		SELECT
			p.name,
			ROUND(SUM(sp.unit_quantity::numeric / NULLIF(p.unit_per_pack, 0)), 4) AS current_stock
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON v.store_id = sp.store_id
		WHERE sp.store_id = v.store_id
		GROUP BY p.name
	),

	-- Sales count per product name in the last N days
	sales_data AS (
		SELECT
			p.name,
			ROUND(SUM(ci.quantity::numeric + (ci.unit_quantity::numeric / NULLIF(p.unit_per_pack, 0))), 4) AS sale_count
		FROM store_products sp
		JOIN cart_items ci ON sp.id = ci.store_product_id
		JOIN sales sl ON sl.id = ci.sale_id
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON sl.store_id = v.store_id
		WHERE (sl.completed_at + interval '5 hours')::date >= (CURRENT_DATE - v.sale_period * INTERVAL '1 day')
		AND sl.status = 'completed'
		AND sl.store_id = v.store_id
		GROUP BY p.name
	),

	-- Get all products with or without thresholds
	products_with_thresholds AS (
		SELECT
			p.name,
			p.id AS product_id,
			p.material_code,
			COALESCE(MAX(spt.kvant), 1) AS kvant,
			COALESCE(MAX(spt.min_quantity), 1) AS min_stock,
			COALESCE(MAX(spt.max_quantity), 1000) AS max_stock,
			COALESCE(NULLIF(p.unit_per_pack, 0), 1) AS unit_per_pack
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN store_product_thresholds spt
			ON p.id = spt.product_id AND spt.store_id = sp.store_id
		WHERE sp.store_id = (SELECT store_id FROM vars LIMIT 1)
		GROUP BY p.name, p.id, p.material_code, p.unit_per_pack
	),

	-- Count of new imports per product name
	imports AS (
		SELECT
			p.name,
			SUM(imd.scanned_count) AS new_imports_count
		FROM import_details imd
		JOIN imports im ON im.id = imd.import_id
		JOIN products p ON imd.product_id = p.id
		JOIN vars v ON im.store_id = v.store_id
		WHERE im.status = 'new' AND im.entry_type = 1 AND im.store_id = v.store_id
		GROUP BY p.name
	),

	-- Merge all main data per product name
	all_data AS (
		SELECT
			v.auto_order_id,
			v.store_id,
			t.name,
			t.product_id,
			t.material_code,
			COALESCE(sd.current_stock, 0) AS current_stock,
			COALESCE(sa.sale_count, 0) AS sale_count,
			t.kvant,
			t.min_stock,
			t.max_stock,
			t.unit_per_pack,
			v.import_day,
			v.sale_period
		FROM products_with_thresholds t
		JOIN vars v ON TRUE
		LEFT JOIN stock_data sd ON t.name = sd.name
		LEFT JOIN sales_data sa ON t.name = sa.name
	),

	-- Final calculation logic
	final_calc AS (
		SELECT
			*,
			ROUND(sale_count / sale_period, 4) AS daily_sale_count,
			ROUND((sale_count / sale_period) * import_day, 4) AS delivery_day_consumption,
			current_stock - ROUND((sale_count / sale_period) * import_day, 4) AS stock_on_delivery_date,
			ROUND((sale_count / sale_period) * 3, 4) AS reserve_quantity,
			current_stock - ROUND((sale_count / sale_period) * import_day, 4)
				+ ROUND((sale_count / sale_period) * 3, 4) AS future_stock
		FROM all_data
	),

	-- Order amount calculation
	order_calc AS (
		SELECT
			fc.*,
			GREATEST(min_stock - stock_on_delivery_date, 0) AS required_stock,
			(CASE
				WHEN sale_count = 0 THEN 0
				ELSE LEAST(GREATEST(min_stock - stock_on_delivery_date, 0), max_stock)
			END) AS raw_order,

			(CASE
				WHEN sale_count = 0 THEN 0
				ELSE COALESCE(ROUND(ROUND(
					LEAST(GREATEST(min_stock - stock_on_delivery_date, 0), max_stock) / 1
				) * 1), 0)
			END) - COALESCE(im.new_imports_count, 0) AS order_count
		FROM final_calc fc
		LEFT JOIN imports im ON im.name = fc.name
	)

	-- Final output
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
		GREATEST(order_count, 0) AS order_count
	FROM order_calc
	WHERE order_count > 0
	ORDER BY name;
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
