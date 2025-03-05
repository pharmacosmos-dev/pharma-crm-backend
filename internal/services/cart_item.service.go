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
	err := s.db.Debug().Raw(`
	SELECT
		ci.*,
		p.name,
		p.barcode,
		p.unit_per_pack,
		p.description,
		sp.expire_date,
		((sp.retail_price*sp.bonus_percent)/100) as bonus_amount,
		sp.bonus_percent,
		sp.vat as vatPercent,
		ci.discount_price*sp.vat/(100+sp.vat) as vat,
		sp.pack_quantity AS quantity_in_stock,
		sp.unit_quantity AS unit_quantity_in_stock,
		u.unit_name,
		u.short_name,
		sh.name as shelf,
		pr.name AS producer_name,
		p.mxik AS class_code,
		(select unit_code from product_measurements where mxik_code = p.mxik) AS package_code,
		STRING_AGG(c.name, ', ') as category_name
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN shelves sh ON p.shelf_id = sh.id
	LEFT JOIN producers pr ON pr.id = p.producer_id
	LEFT JOIN category_products cp ON p.id = cp.product_id
	LEFT JOIN categories c ON c.id = cp.category_id
	WHERE ci.status = 'pending' AND ci.sale_id = ?
	GROUP BY ci.id, ci.created_at, p.id, sp.id, u.id, sh.id, pr.id
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
		COUNT(*) AS count
	FROM cart_items
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
func (s *Storage) CreateCartItem(req *domain.CartItemRequest, percent, price float64) error {
	err := s.db.Exec(`
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
		req.DiscountType, percent, price, req.DiscountAmount).Error
	if err != nil {
		return err
	}

	return nil
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
