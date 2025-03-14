package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// get cart item list by sale id with limit, offset
func (s *Storage) CartItemList(saleID string, limit, offset int) (*domain.CartItemData, error) {
	var res []domain.CartItemResponse
	err := s.db.Raw(`
	SELECT
		ci.*,
		p.name,
		p.barcode,
		p.unit_per_pack,
		sp.expire_date,
		pb.bonus_amount as bonus_amount,
		sp.vat AS vat_percent,
		sp.vat_price as vat_price,
		ROUND(sp.vat_price * ci.quantity +
		CASE
			WHEN p.unit_per_pack > 0 THEN (sp.vat_price / p.unit_per_pack) * ci.unit_quantity
			ELSE 0
		END, 2) AS vat,
		sp.pack_quantity AS quantity_in_stock,
		sp.unit_quantity AS unit_quantity_in_stock,
		u.unit_name,
		u.short_name,
		sh.name as shelf,
		p.mxik AS class_code,
		pm.unit_code AS package_code,
		pm.unit_name AS package_name
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN shelves sh ON p.shelf_id = sh.id
	LEFT JOIN product_measurements pm ON pm.mxik_code = p.mxik
	LEFT JOIN product_bonuses pb ON p.id = pb.product_id
	WHERE ci.status = 'pending' AND ci.sale_id = ?
	GROUP BY ci.id, ci.created_at, p.id, sp.id, u.id, sh.id, pm.id, pb.id
	ORDER BY ci.created_at DESC LIMIT ? OFFSET ?
	`, saleID, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}
	for i := range res {
		if res[i].UnitPerPack > 0 && res[i].UnitQuantityInStock != res[i].UnitPerPack*res[i].QuantityInStock {
			res[i].CurrentStock = fmt.Sprintf("%d (%d/%d)", res[i].QuantityInStock, res[i].UnitQuantityInStock%res[i].UnitPerPack, res[i].UnitPerPack)
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
		SUM(ROUND(sp.vat_price * quantity +
		CASE
			WHEN p.unit_per_pack > 0 THEN (sp.vat_price / p.unit_per_pack) * ci.unit_quantity
			ELSE 0
		END, 2)) AS vat_sum,
		COUNT(*) AS count
	FROM cart_items ci
	JOIN store_products sp ON sp.id = ci.store_product_id
	JOIN products p ON sp.product_id = p.id
	WHERE sale_id = ?
	`, saleID).Scan(&data).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}
	if res == nil {
		res = []domain.CartItemResponse{}
	}
	data.TotalAmount = data.Sum - data.DiscountAmount
	data.Data = res
	return &data, nil
}

// create cart item
func (s *Storage) CreateCartItem(req *domain.CartItemRequest, percent, price float64) (*domain.CartItem, error) {
	var res domain.CartItem
	err := s.db.Raw(`
		INSERT INTO cart_items(
			id, store_product_id,
			sale_id, employee_id,
			quantity, unit_quantity, unit_price,
			total_price, status,
			discount_type, discount_value,
			discount_price, discount_amount
			)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), req.StoreProductID, req.SaleId,
		req.EmployeeID, req.Quantity, req.UnitQuantity,
		req.UnitPrice, req.TotalPrice, config.PENDING_CART_ITEM,
		req.DiscountType, percent, price, req.DiscountAmount).Scan(&res).Error
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// update cart item by field
func (s *Storage) UpdateCartItemField(field string, value string, idField, idValue string) (*domain.CartItem, error) {
	var res domain.CartItem
	err := s.db.Raw(`UPDATE cart_items SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// get cart item list by sale id
func (s *Storage) ListCartItemsBySaleID(saleID string) ([]domain.CartItem, error) {
	var res []domain.CartItem
	err := s.db.Raw(`SELECT * FROM cart_items WHERE sale_id = ?`, saleID).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

// create cart item for online sale
func (s *Storage) CreateOnlineCartItem(tx *gorm.DB, req *domain.SaleOnlineItem, saleID string) error {
	// check if product and store exist
	var storProductId string
	err := s.db.Raw(`SELECT id FROM store_products WHERE store_id = ? AND product_id = ?`, req.StoreId, req.ProductId).Scan(&storProductId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("product or store not found")
		}
		return err
	}
	// create new cart item
	err = tx.Exec(`
		INSERT INTO cart_items(
			store_product_id, sale_id, 
			quantity, unit_price,
			total_price, status)
		VALUES(?, ?, ?, ?, ?, ?)`,
		storProductId, saleID, req.Quantity, req.UnitPrice, req.TotalPrice, config.SOLD_CART_ITEM,
	).Error
	if err != nil {
		return err
	}

	return nil
}
