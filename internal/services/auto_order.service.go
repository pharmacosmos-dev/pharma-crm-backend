package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

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
			p.name,
			ROUND(SUM(sp.pack_quantity + (sp.unit_quantity::numeric % p.unit_per_pack) / p.unit_per_pack), 4) AS current_stock
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON sp.store_id = v.store_id
		GROUP BY p.name
	),
	sales_data AS (
		SELECT
			p.name,
			ROUND(SUM(ci.quantity + (ci.unit_quantity::numeric / p.unit_per_pack)), 4) AS sale_count
		FROM store_products sp
		JOIN cart_items ci ON sp.id = ci.store_product_id
		JOIN sales sl ON sl.id = ci.sale_id AND sl.status = 'completed'
		JOIN products p ON sp.product_id = p.id
		JOIN vars v ON sl.store_id = v.store_id
		WHERE (sl.completed_at + interval '5 hours')::date >= (CURRENT_DATE - v.sale_period * INTERVAL '1 day')
		GROUP BY p.name
	),
	thresholds AS (
		SELECT
			p.name,
			MAX(spt.kvant) AS kvant,
			MAX(spt.min_quantity) AS min_stock,
			MAX(spt.max_quantity) AS max_stock,
			MAX(p.unit_per_pack) AS unit_per_pack,
			MAX(p.material_code) AS material_code
		FROM store_product_thresholds spt
		JOIN products p ON spt.product_id = p.id
		JOIN vars v ON spt.store_id = v.store_id
		GROUP BY p.name
	),
	excluded_products_union AS (
		SELECT p.name
		FROM excluded_products ep
		JOIN products p ON ep.product_id = p.id
		JOIN vars v ON ep.store_id = v.store_id
		UNION
		SELECT p.name
		FROM excluded_products ep
		JOIN products p ON ep.product_id = p.id
		WHERE ep.store_id IS NULL
	),
	imports AS (
		SELECT p.name, COUNT(*) AS new_imports_count
		FROM imports i
		JOIN import_details ip ON i.id = ip.import_id
		JOIN products p ON ip.product_id = p.id
		JOIN vars v ON i.store_id = v.store_id
		WHERE i.status = 'new'
		GROUP BY p.name
	),
	all_data AS (
		SELECT
			v.auto_order_id,
			v.store_id,
			t.name,
			COALESCE(sd.current_stock, 0) AS current_stock,
			COALESCE(sa.sale_count, 0) AS sale_count,
			COALESCE(t.kvant, 0) AS kvant,
			COALESCE(t.min_stock, 0) AS min_stock,
			COALESCE(t.max_stock, 0) AS max_stock,
			COALESCE(t.unit_per_pack, 1) AS unit_per_pack,
			COALESCE(t.material_code, 0) AS material_code,
			v.import_day,
			v.sale_period
		FROM thresholds t
		JOIN vars v ON TRUE
		LEFT JOIN stock_data sd ON t.name = sd.name
		LEFT JOIN sales_data sa ON t.name = sa.name
	),
	final_calc AS (
		SELECT
			*,
			ROUND(sale_count / sale_period, 4) AS daily_sale_count,
			ROUND(sale_count / sale_period * import_day, 4) AS delivery_day_consumption,
			current_stock - ROUND(sale_count / sale_period * import_day, 4) AS stock_on_delivery_date,
			ROUND(sale_count / sale_period * 3, 4) AS reserve_quantity,
			current_stock - ROUND(sale_count / sale_period * import_day, 4) + ROUND(sale_count / sale_period * 3, 4) AS future_stock
		FROM all_data
	),
	order_calc AS (
		SELECT
			fc.*,
			GREATEST(min_stock - stock_on_delivery_date, 0) AS required_stock,
			(CASE
				WHEN kvant > 0 THEN ROUND(ROUND(GREATEST(min_stock - stock_on_delivery_date, 0) / kvant) * kvant)
				ELSE 0
			END)  AS order_count
		FROM final_calc fc
		LEFT JOIN imports im ON im.name = fc.name
		LEFT JOIN excluded_products_union ex ON ex.name = fc.name
		WHERE ex.name IS NULL
	)
	SELECT *
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
