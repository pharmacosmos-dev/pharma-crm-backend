package storage

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

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
		sp.pack_quantity AS quantity_in_stock,
		sp.unit_quantity AS unit_quantity_in_stock,
		u.unit_name,
		u.short_name,
		sh.name as shelf
	FROM cart_items c
	JOIN store_products sp ON c.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN shelves sh ON p.shelf_id = sh.id
	WHERE c.status = 'pending' AND c.sale_id = ?
	ORDER BY c.created_at LIMIT ? OFFSET ?
	`, saleID, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}
	for i := range res {
		if res[i].UnitQuantityInStock != res[i].UnitPerPack*res[i].QuantityInStock {
			res[i].CurrentStock = fmt.Sprintf("%d (%d/%d)", res[i].QuantityInStock, res[i].UnitQuantityInStock, res[i].UnitPerPack)
		} else {
			res[i].CurrentStock = fmt.Sprintf("%d", res[i].QuantityInStock)
		}
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
