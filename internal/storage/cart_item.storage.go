package storage

import "github.com/pharma-crm-backend/domain"

func (s *Storage) CartItemList(saleID string, limit, offset int) (*domain.CartItemData, error) {
	var res []domain.CartItemResponse
	err := s.db.Raw(`
	SELECT
		c.*,
		p.name,
		p.barcode,
		p.unit_per_pack,
		sp.expire_date,
		sp.bonus_amount,
		sp.bonus_percent,
		u.unit_name,
		u.short_name
	FROM cart_items c
	JOIN store_products sp ON c.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	WHERE c.status = 'pending' AND c.sale_id = ?
	ORDER BY c.created_at LIMIT ? OFFSET ?
	`, saleID, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}
	var data domain.CartItemData
	err = s.db.Raw(`
	SELECT 
		SUM(total_price) AS sum,
		SUM(quantity) AS item_count, 
		SUM(discount_amount*quantity) AS discount_amount, 
		COUNT(*) AS count
	FROM cart_items
	WHERE sale_id = ?
	`, saleID).Scan(&data).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}
	data.TotalAmount = data.Sum - data.DiscountAmount
	data.Data = res
	return &data, nil
}
