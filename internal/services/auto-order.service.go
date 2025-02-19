package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pharma-crm-backend/domain"
)

func (s *Storage) ListAutoOrder(ctx context.Context, limit, offset int, storeID, search string) ([]domain.AutoOrder, int64, error) {
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

// generate auto order with store_id and day
func (s *Storage) GenerateAutoOrderDetail(ctx context.Context, storeID string, day int) ([]*domain.AutoOrderDetailRequest, error) {
	var res []*domain.AutoOrderDetailRequest
	query := `
	WITH sales_data AS (
		SELECT
			sp.store_id,
			sp.product_id,
			SUM(CASE WHEN s.created_at >= NOW() - INTERVAL '` + strconv.Itoa(day) + ` day' THEN ci.quantity ELSE 0 END) AS day_sale_stock,
			SUM(CASE WHEN s.created_at >= NOW() - INTERVAL '1 month' THEN ci.quantity ELSE 0 END) AS month_sale_stock
		FROM
			sales s
		JOIN
			cart_items ci ON s.id = ci.sale_id
		JOIN
			store_products sp ON ci.store_product_id = sp.id
		WHERE
			ci.status = 'sold' AND s.status = 'completed'
			AND sp.store_id = ?
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
		WHERE
			sp.store_id = ?
	)
	SELECT
		st.store_id,
		s.name AS store_name,
		st.product_id,
		p.name AS product_name,
		st.current_stock,
		COALESCE(sd.month_sale_stock, 0) AS month_sale_stock,
		COALESCE(sd.day_sale_stock, 0) AS day_sale_stock,
		(COALESCE(sd.day_sale_stock, 0) - st.current_stock) * 1.1 AS order_growth,
		((COALESCE(sd.day_sale_stock, 0) - st.current_stock) * 1.1) * 1 AS order_lead_time,
		CASE
			WHEN (COALESCE(sd.day_sale_stock, 0) - st.current_stock) * 1.1 > 0
				THEN ROUND((COALESCE(sd.day_sale_stock, 0) - st.current_stock) * 1.1)
			ELSE 0
		END AS suggested_order_quantity
	FROM
		stock st
	JOIN
		stores s ON st.store_id = s.id
	JOIN
		products p ON st.product_id = p.id
	LEFT JOIN
		sales_data sd ON st.store_id = sd.store_id AND st.product_id = sd.product_id
	LEFT JOIN auto_order_details ON auto_order_details.product_id = st.product_id
	LEFT JOIN auto_orders ON auto_order_details.auto_order_id = auto_orders.id
	WHERE
		st.store_id = ? AND
		(auto_orders.status != 'new' OR auto_orders.status != 'pending' 
		OR auto_orders.status IS NULL);
	`
	err := s.db.Raw(query, storeID, storeID, storeID).Scan(&res).Error

	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	return res, nil
}
