package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// get cart item list by sale id with limit, offset
func (s *Services) CartItemList(saleID string, limit, offset int) (*domain.CartItemData, error) {
	var res []domain.CartItemResponse
	err := s.db.Raw(`
	WITH ci_amount AS (
		SELECT
			ci.id AS ci_id,
			COALESCE((ci.unit_price*ci.discount_value)/100, 0.00) AS d_amount
		FROM cart_items ci
		JOIN store_products sp ON ci.store_product_id = sp.id
		WHERE ci.sale_id = ?
	)
	SELECT
		ci.id,
		ci.employee_id,
		ci.sale_id,
		ci.store_product_id,
		ci.quantity,
		ci.unit_quantity,
		ci.discount_type,
		ci.discount_value,
		ci.unit_price,
		ci.total_price,
		ci.discount_price,
		ci.created_at,
		ci.updated_at,
		ci.marking_count,
		p.name,
		COALESCE(sp.barcode, p.barcode) AS barcode,
		p.unit_per_pack,
		sp.is_marking,
		sp.is_checking,
		sp.expire_date,
		pb.bonus_amount,
		sp.vat AS vat_percent,

		ROUND(((ci.unit_price - ci_amount.d_amount) * 12) / 112, 2) AS vat_price,
		ROUND((((ci.unit_price - ci_amount.d_amount) * 12) / 112) / p.unit_per_pack, 2) AS unit_vat_price,

		ROUND(sp.vat_price * ci.quantity + (sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2) AS vat,
		ROUND(ci.unit_price / p.unit_per_pack, 2) AS unit_quantity_price,

		sp.pack_quantity AS quantity_stock,
		(sp.unit_quantity % p.unit_per_pack) AS unit_quantity_stock,

		u.unit_name,
		u.short_name,
		sh.name AS shelf,
		COALESCE(sp.mxik, p.mxik) AS class_code,
		COALESCE(sp.unit_code, p.unit_code) AS package_code,
		COALESCE(sp.unit_label, p.unit_label) AS package_name,

		ci_amount.d_amount AS discount_amount,
		ROUND(ci_amount.d_amount / p.unit_per_pack, 2) AS discount_unit_amount,
		ROUND(ci.unit_quantity::numeric / p.unit_per_pack, 2) AS unit_amount

	FROM cart_items ci
	JOIN ci_amount ON ci.id = ci_amount.ci_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN shelves sh ON p.shelf_id = sh.id
	LEFT JOIN product_bonuses pb ON p.id = pb.product_id
	WHERE ci.sale_id = ?
	ORDER BY ci.created_at DESC LIMIT ? OFFSET ?;
	`, saleID, saleID, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing cart items for sale %s: %v", saleID, err.Error())
		return nil, err
	}

	for i := range res {
		unitPrice := res[i].VatPrice / float64(res[i].UnitPerPack)
		res[i].UnitVatPrice = unitPrice // ← no rounding
	}

	var data domain.CartItemData
	err = s.db.Raw(`
	SELECT
		SUM(total_price) AS sum,
		SUM(quantity) AS item_count,
		SUM(ci.discount_amount) AS discount_amount,
		MAX(dc.percent) AS card_percent,
		ROUND(SUM(sp.vat_price * quantity + (sp.vat_price / p.unit_per_pack) * ci.unit_quantity), 2) AS vat_sum,
		COUNT(*) AS count
	FROM cart_items ci
	JOIN store_products sp ON sp.id = ci.store_product_id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN sale_customer_discounts cd ON cd.sale_id = ci.sale_id
	LEFT JOIN discount_cards dc ON cd.customer_id = dc.customer_id
	WHERE  ci.sale_id = ?
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
func (s *Services) CreateCartItem(req *domain.CartItemRequest) (*domain.CartItem, error) {
	var res domain.CartItem
	err := s.db.Raw(`
		INSERT INTO cart_items(
			id, 
			store_product_id,
			sale_id, 
			employee_id,
			quantity,
			unit_quantity,
			unit_price,
			total_price,
			status,
			discount_type,
			discount_value
			)
			VALUES (
			?,?,?,?,?,?,?,?,?,?  ,COALESCE((SELECT COALESCE(discount_percent, 0) 
     FROM sale_customer_discounts 
     WHERE sale_id = ?
     LIMIT 1),0)
)
RETURNING *`,
		uuid.New().String(),
		req.StoreProductID,
		req.SaleId,
		req.EmployeeID,
		req.Quantity,
		req.UnitQuantity,
		req.UnitPrice,
		req.TotalPrice,
		config.PENDING_CART_ITEM,
		req.DiscountType,
		req.SaleId).Scan(&res).Error
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// update cart item by field
func (s *Services) UpdateCartItemField(field string, value string, idField, idValue string) (*domain.CartItem, error) {
	var res domain.CartItem
	err := s.db.Raw(`UPDATE cart_items SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// get cart item list by sale id
func (s *Services) ListCartItemsBySaleID(saleID string) ([]domain.CartItem, error) {
	var res []domain.CartItem
	err := s.db.Raw(`SELECT * FROM cart_items WHERE sale_id = ?`, saleID).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

// get cart items to payme go items
func (s *Services) GetPaymeGoItems(saleID string) ([]domain.PaymeGoItem, error) {
	var res []domain.PaymeGoItem
	query := `

	`
	err := s.db.Raw(query, saleID).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on getting cart_items: ", err)
		return nil, err
	}

	return res, nil
}

// get cart items total amount
func (s *Services) GetCartItemsTotalAmount(saleID string) (float64, error) {
	var res domain.CartItemData
	err := s.db.Raw(`
	SELECT
		SUM(total_price) AS sum,
		SUM(discount_amount*quantity) AS discount_amount
	FROM cart_items ci
	WHERE sale_id = ?`, saleID).Scan(&res).Error
	if err != nil {
		return 0, err
	}
	res.TotalAmount = res.Sum - res.DiscountAmount
	return res.TotalAmount, nil
}

// add marking count to cart items
func (s *Services) AddMarkingCount(ctx context.Context, tx *gorm.DB, req []domain.MarkingData) error {
	if len(req) == 0 {
		return nil
	}
	var err error
	defer RollbackIfError(tx, &err)
	// Build VALUES part: ('uuid1', 5), ('uuid2', 10), ...
	var valueStrings []string
	for _, r := range req {
		valueStrings = append(valueStrings, fmt.Sprintf("('%s', %d)", r.Id, r.MarkingCount))
	}

	query := fmt.Sprintf(`
		UPDATE 
			cart_items AS c
		SET 
			marking_count = v.marking_count
		FROM (
			VALUES %s
		) AS v(id, marking_count)
		WHERE c.id = v.id::uuid;
	`, strings.Join(valueStrings, ","))

	// Execute raw SQL
	err = tx.WithContext(ctx).Exec(query).Error
	if err != nil {
		s.log.Warn("ERROR on bulk updating cart_item marking_count: %v", err)
		return err
	}

	return nil
}

// check order product quantity and return collect cart_item
func (s *Services) GetOrCheckOnlineCartItems(req []domain.OnlineCartItemRequest, saleID string) ([]domain.CartItemOnlineRequest, error) {
	// store product get query
	query := `
	SELECT
		sp.id,
		sp.product_id,
		sp.retail_price,
		sp.unit_quantity,
		sp.pack_quantity,
		sp.unit_quantity/(p.unit_per_pack/p.blister_count) AS quantity,
		p.name AS product_name,
		p.unit_per_pack,
		p.blister_count
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	WHERE sp.unit_quantity/(p.unit_per_pack/p.blister_count) >= ? AND
		sp.product_id = ? AND
		sp.expire_date::date > CURRENT_DATE;
	`
	var (
		temp      = domain.StoreProductOnline{}      // store product temp structure
		cartItems = []domain.CartItemOnlineRequest{} // cart item request structure
	)
	for i := range req {
		err := s.db.Raw(query, req[i].Quantity, req[i].ProductId).Scan(&temp).Error
		if err != nil {
			s.log.Warn("ERROR getting store_product: %v", err)
			return cartItems, errors.New("store_product.not.get")
		}
		if temp.Quantity < req[i].Quantity { // checking quantity enough or not enough
			s.log.Warn("Noor Not enough product")
			return cartItems, fmt.Errorf("not.enough.product: %s", temp.ProductName)
		}
		// quantity calculate:  req.quantity = order_quantity -> based on blister_count
		// example: unit_per_pack = 50, blister_count = 5, count_per_blister = unit_per_pack/blister_count = 10
		// order_quantity = 2 - > blister_count * count_per_blister = 2 * 10 = 20
		// cart_item.quantity = (order_quantity * (unit_per_pack/blister_count))/unit_per_pack = (2 * (50/5))/50 = 0.4 = 0
		// cart_item.unit_quantity = order_quantity * (unit_per_pack/blister_count) = 2 * (50/5) = 20
		quantity := (req[i].Quantity * (temp.UnitPerPack / temp.BlisterCount)) / temp.UnitPerPack
		cartItems = append(cartItems, domain.CartItemOnlineRequest{
			SaleId:         saleID,
			StoreProductID: temp.ID,
			Quantity:       quantity,
			UnitQuantity:   req[i].Quantity * (temp.UnitPerPack / temp.BlisterCount),
			UnitPrice:      temp.RetailPrice,
			TotalPrice:     (temp.RetailPrice / float64(temp.UnitPerPack)) * float64(req[i].Quantity*(temp.UnitPerPack/temp.BlisterCount)),
		})
	}

	return cartItems, nil
}

func (s *Services) GetCartItems(ctx context.Context, tx *gorm.DB, saleID string) ([]*domain.CartItemForDMED, error) {
	var (
		err error
		res []*domain.CartItemForDMED
	)
	defer RollbackIfError(tx, &err)
	err = tx.WithContext(ctx).Raw(`
		SELECT
    		ci.id,
    		ci.quantity,
    		ci.unit_quantity  as unit_quantity,
    		ci.unit_price / p.unit_per_pack as unit_price,
    		p.unit_per_pack,
    		p.barcode,
   			sp.serial_number
		FROM cart_items ci
         		LEFT JOIN store_products sp ON ci.store_product_id = sp.id
         		LEFT JOIN products p ON sp.product_id = p.id
		WHERE ci.sale_id = ?`, saleID).Scan(&res).Error
	if err != nil {
		s.log.Error("", err)
		return nil, err
	}
	return res, nil
}
