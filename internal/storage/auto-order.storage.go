package storage

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Storage) ListAutoOrder(ctx context.Context, limit, offset int, storeID string) ([]domain.AutoOrder, int64, error) {
	var (
		autoOrders     []domain.AutoOrder
		totalCount     int64
		storeCondition string
	)
	if storeID != "" {
		storeCondition = fmt.Sprintf("WHERE st.store_id = '%s'", storeID)
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
				THEN (w.weekly_quantity-st.current_stock)*1.1
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
	`, storeCondition)
	err := s.db.Raw(query, limit, offset).Scan(&autoOrders).Error
	if err != nil {
		return nil, 0, err
	}
	// Extract total_count from the first row
	if len(autoOrders) > 0 {
		totalCount = autoOrders[0].TotalCount
	}
	return autoOrders, totalCount, nil
}
