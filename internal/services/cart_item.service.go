package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// region Create

func (s *Services) CreateCartItem(ctx context.Context, user *domain.EmployeeClaims, req *domain.CartItemRequest) (*domain.CartItem, error) {
	// get sale info by id
	sale, err := s.GetSaleById(ctx, req.SaleId)
	if err != nil {
		return nil, err
	}

	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return nil, domain.SaleIsClosedError
	}

	req.EmployeeId = user.UserId
	storeProduct, err := s.GetStoreProductByIdAndStoreId(ctx, req.StoreProductId, sale.StoreId)
	if err != nil {
		return nil, err
	}

	cart, err := s.GetCartItemBySaleIdAndSpId(ctx, req.SaleId, req.StoreProductId)
	if err != nil {
		res, err := s.createNewCartItem(ctx, req, storeProduct)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	cart, err = s.updateExistsCartItemQuantity(ctx, cart, storeProduct)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *Services) createNewCartItem(ctx context.Context, req *domain.CartItemRequest, storeProduct *domain.StoreProduct) (*domain.CartItem, error) {
	// check remaining quantity for add cart item
	totalPrice := storeProduct.RetailPrice
	if storeProduct.UnitQuantity >= storeProduct.UnitPerPack {
		req.UnitQuantity = storeProduct.UnitPerPack
	} else if storeProduct.UnitQuantity > 0 {
		req.UnitQuantity = 1
		totalPrice = storeProduct.RetailPrice / float64(storeProduct.UnitPerPack)
	} else {
		return nil, domain.NotEnoughProductError
	}

	var res domain.CartItem
	err := s.db.WithContext(ctx).Raw(`
		INSERT INTO cart_items(
			store_product_id,
			sale_id,
			employee_id,
			unit_quantity,
			unit_price,
			total_price,
			status,
			is_marking,
			discount_type,
			discount_value
			)
			VALUES (
			?,?,?,?,?,?,?,?,? ,COALESCE((SELECT COALESCE(discount_percent, 0)
		FROM sale_customer_discounts
			WHERE sale_id = ?
			LIMIT 1),0)
		)
		RETURNING *`,
		req.StoreProductId,
		req.SaleId,
		req.EmployeeId,
		req.UnitQuantity,
		storeProduct.RetailPrice,
		totalPrice,
		constants.GeneralStatusPending,
		storeProduct.IsMarking,
		req.DiscountType,
		req.SaleId,
	).Scan(&res).Error
	if err != nil {
		s.log.Error("could not create cart_item: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// region Get

func (s *Services) FetchCartItems(ctx context.Context, saleId string, limit, offset int) (*domain.CartItemData, error) {
	var res []domain.CartItemResponse
	err := s.db.WithContext(ctx).Raw(`
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
		ci.unit_quantity / p.unit_per_pack AS quantity,
		ci.unit_quantity % p.unit_per_pack AS unit_quantity,
		ci.discount_type,
		ci.discount_value,
		ci.unit_price,
		ci.total_price,
		ci.discount_price,
		ci.marking_count,
		ci.markings,
		ci.created_at,
		ci.updated_at,
		p.name,
		p.id as product_id,
		COALESCE(sp.barcode, p.barcode) AS barcode,
		p.unit_per_pack,
		sp.is_marking,
		sp.is_checking,
		sp.expire_date,
		pb.bonus_amount,
		sp.vat AS vat_percent,

		ROUND(((ci.unit_price - ci_amount.d_amount) * 12) / 112, 2) AS vat_price,
		ROUND((((ci.unit_price - ci_amount.d_amount) * 12) / 112) / p.unit_per_pack, 4) AS unit_vat_price,

		ROUND((sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2) AS vat,
		ROUND(ci.unit_price / p.unit_per_pack, 2) AS unit_quantity_price,

		sp.unit_quantity / p.unit_per_pack AS quantity_stock,
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
		ORDER BY ci.created_at DESC 
		LIMIT ? OFFSET ?;
	`, saleId, saleId, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Errorf("cound not get cart_items by sale(%s) error: %v", saleId, err.Error())
		return nil, domain.InternalServerError
	}

	var data domain.CartItemData
	err = s.db.WithContext(ctx).Raw(`
	SELECT
		SUM(total_price) AS sum,
		SUM(ci.unit_quantity/p.unit_per_pack) AS item_count,
		SUM(ci.discount_amount) AS discount_amount,
		MAX(dc.percent) AS card_percent,
		ROUND(SUM((sp.vat_price / p.unit_per_pack) * ci.unit_quantity), 2) AS vat_sum,
		SUM(total_price) - SUM(ci.discount_amount) as total_amount,
		COUNT(*) AS count
	FROM cart_items ci
	JOIN store_products sp ON sp.id = ci.store_product_id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN sale_customer_discounts cd ON cd.sale_id = ci.sale_id
	LEFT JOIN discount_cards dc ON cd.customer_id = dc.customer_id
	WHERE  ci.sale_id = ?;`, saleId).
		Scan(&data).Error
	if err != nil {
		s.log.Errorf("could not get cart_item sum amounts: %v", err)
		return nil, err
	}
	if res == nil {
		res = []domain.CartItemResponse{}
	}

	data.Data = res

	return &data, nil
}

// Get cartItem by saleId and storeProductId
func (s *Services) GetCartItemBySaleIdAndSpId(ctx context.Context, saleId, spId string) (*domain.CartItem, error) {
	var cartItem domain.CartItem
	err := s.db.WithContext(ctx).Where("sale_id = ? AND store_product_id = ?", saleId, spId).First(&cartItem).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Error("could not get cart_item by id: %v", err)
		return nil, domain.InternalServerError
	}

	return &cartItem, nil
}

// get cart items total amount
func (s *Services) GetCartItemsTotalAmount(ctx context.Context, saleID string) (*domain.CartItemData, error) {
	var res domain.CartItemData
	err := s.db.WithContext(ctx).
		Select(
			"SUM(total_price) AS sum",
			"SUM(ci.unit_quantity/p.unit_per_pack) AS item_count",
			"SUM(ci.discount_amount) AS discount_amount",
			"MAX(dc.percent) AS card_percent",
			"ROUND(SUM((sp.vat_price / p.unit_per_pack) * ci.unit_quantity), 2) AS vat_sum",
			"SUM(total_price) - SUM(ci.discount_amount) as total_amount",
		).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get cart_items total_price: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
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

func (s *Services) GetCartItems(ctx context.Context, saleID string) ([]*domain.CartItemForDMED, error) {
	var (
		err error
		res []*domain.CartItemForDMED
	)
	query := `
	SELECT
		ci.id,
		ci.store_product_id,
		ci.unit_quantity / p.unit_per_pack AS quantity,
		ci.unit_quantity % p.unit_per_pack  AS unit_quantity,
		ci.unit_price / p.unit_per_pack as unit_price,
		p.unit_per_pack,
		p.barcode,
		sp.serial_number
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	WHERE ci.sale_id = ?
	`
	err = s.db.WithContext(ctx).Raw(query, saleID).Scan(&res).Error
	if err != nil {
		s.log.Error("could not get cart_items for dmed: %v", err)
		return nil, domain.InternalServerError
	}
	return res, nil
}

func (s *Services) GetCartItemById(ctx context.Context, id string) (*domain.CartItem, error) {
	var cartItem domain.CartItem
	err := s.db.First(&cartItem, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get cart_item by id error: %v", err)
		return nil, domain.InternalServerError
	}
	return &cartItem, nil
}

func (s *Services) getCartItemWithProducts(ctx context.Context, saleId string) ([]domain.CartItemResponse, error) {
	var cartItems []domain.CartItemResponse
	err := s.db.
		WithContext(ctx).
		Model(&domain.CartItem{}).
		Select(
			"ci.id",
			"ci.sale_id",
			"ci.store_product_id",
			"ci.unit_quantity % p.unit_per_pack AS unit_quantity",
			"ci.unit_quantity / p.unit_per_pack AS quantity",
			"ci.unit_price",
			"ci.total_price",
			"ci.discount_type",
			"ci.discount_value",
			"ci.discount_amount",
			"ci.discount_price",
			"ci.created_at",
			"ci.updated_at",
			"p.name",
			"p.barcode",
		).
		Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Where("sale_id = ?", saleId).
		Find(&cartItems).Error
	if err != nil {
		s.log.Errorf("could not get cart_items by sale(%s) error: %v", saleId, err)
		return nil, domain.InternalServerError
	}

	return cartItems, nil
}

func (s *Services) getCartItemsByIds(ctx context.Context, ids []string) ([]domain.CartItem, error) {
	var cartItems []domain.CartItem
	err := s.db.
		WithContext(ctx).
		Where("id IN (?)", ids).
		Find(&cartItems).Error
	if err != nil {
		s.log.Errorf("could not get cart_items by ids error: %v", err)
		return nil, domain.InternalServerError
	}

	return cartItems, nil
}

func (s *Services) getCartItemsForFinal(ctx context.Context, saleId string) ([]domain.CartItem, error) {
	var res []domain.CartItem
	err := s.db.WithContext(ctx).Where("sale_id = ?", saleId).Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get cart_items for final: %v", err)
		return res, domain.InternalServerError
	}
	return res, nil
}

// region Update

func (s *Services) UpdateCartItemField(field string, value string, idField, idValue string) (*domain.CartItem, error) {
	var res domain.CartItem
	err := s.db.Raw(`UPDATE cart_items SET `+field+` = ? WHERE `+idField+` = ? RETURNING *`, value, idValue).Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Services) updateExistsCartItemQuantity(ctx context.Context, req *domain.CartItem, storeProduct *domain.StoreProduct) (*domain.CartItem, error) {
	quantity := 0
	totalPrice := 0.00
	// check remaining quantity for add cart item
	if storeProduct.UnitQuantity >= req.UnitQuantity+storeProduct.UnitPerPack {
		quantity += storeProduct.UnitPerPack
		totalPrice += req.UnitPrice
	} else if storeProduct.UnitQuantity > 0 {
		quantity += 1
		totalPrice += ((req.UnitPrice * 100) / float64(storeProduct.UnitPerPack)) / 100
	} else {
		return nil, domain.NotEnoughProductError
	}

	res, err := s.IncrementCartItemQuantity(ctx, s.db, req.Id, quantity, totalPrice)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Services) UpdateCartItemDiscount(ctx context.Context, saleId string, req *domain.CartItemDiscountRequest) error {
	// Get sale
	sale, err := s.GetSaleById(ctx, saleId)
	if err != nil {
		return err
	}

	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return domain.SaleIsClosedError
	}

	// Get cart item total data
	cartItemTotal, err := s.GetCartItemsTotalAmount(ctx, saleId)
	if err != nil {
		return err
	}

	// validate discount type
	if req.DiscountType != constants.DiscountTypePercent && req.DiscountType != constants.PaymentTypeCash {
		return domain.InvalidRequestBodyError
	}

	// validate sum with discount value
	if req.DiscountType == constants.PaymentTypeCash && cartItemTotal.TotalAmount < req.DiscountValue {
		return domain.InvalidRequestBodyError
	}

	cartItems, err := s.getCartItemWithProducts(ctx, saleId)
	if err != nil {
		return err
	}

	// check discount type with percent or cash
	var discountPercent float64
	for i := range cartItems {
		if req.DiscountValue == 0 {
			cartItems[i].DiscountAmount = 0
			discountPercent = 0
		} else if req.DiscountType == constants.DiscountTypePercent && req.DiscountValue <= 100 {
			cartItems[i].DiscountAmount = cartItems[i].UnitPrice * req.DiscountValue / 100
			discountPercent = req.DiscountValue
		} else if req.DiscountType == constants.PaymentTypeCash {
			// a = 1100 b = 1200  d = 900         / a, b items, d - discount sum
			// x = (d / (a + b)) * a = (900 / (1100 + 1200)) * 1100 = 430.47
			// y = (d / (a + b)) * b = (900 / (1100 + 1200)) * 1200 = 469.56
			// percent1 = (1 - (430.47/1100)) * 100 = 60.87 \___  60.87% discount
			// percent2 = (1 - (469.56/1200)) * 100 = 60.87 /
			discountPrice := (req.DiscountValue / cartItemTotal.Sum) * cartItems[i].UnitPrice
			discountPercent = 1 - (discountPrice/cartItems[i].UnitPrice)*100
			cartItems[i].DiscountAmount = cartItems[i].UnitPrice - discountPrice
		} else {
			return domain.InvalidRequestBodyError
		}
		err = s.db.Exec(`
		UPDATE cart_items
		SET
			discount_type = ?,
			discount_value = ?,
			discount_price = (CASE WHEN ? = 0 THEN 0 ELSE unit_price - ? END)
		WHERE id = ?`,
			req.DiscountType,
			discountPercent,
			req.DiscountValue,
			cartItems[i].DiscountAmount,
			cartItems[i].ID).Error
		if err != nil {
			s.log.Errorf("could not update cart_items discount: %v", err)
			return domain.InternalServerError
		}
	}

	return nil
}

func (s *Services) UpdateCartItemQuantity(ctx context.Context, req *domain.CartItemUpdateUnit) (map[string]any, error) {
	// get cart_item before update
	cartItem, err := s.GetCartItemById(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	sale, err := s.GetSaleById(ctx, cartItem.SaleId)
	if err != nil {
		return nil, err
	}

	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return nil, domain.SaleIsClosedError
	}

	// get store_product by id
	storeProduct, err := s.GetStoreProductById(ctx, req.StoreProductId)
	if err != nil {
		return nil, err
	}

	// total remaining unit_quantity -> cart + store_product
	ostatok := storeProduct.UnitQuantity

	// total unit_quantity requested
	reqUnitQuantity := req.UnitQuantity + (req.Quantity * storeProduct.UnitPerPack)

	// validate quantity enough or no
	if ostatok < reqUnitQuantity {
		return nil, domain.NotEnoughProductError
	}

	updateQuantity := reqUnitQuantity - cartItem.UnitQuantity

	// compare old and new quantities
	isIncrease := updateQuantity > 0
	quantityDiff := req.Quantity - (cartItem.UnitQuantity / storeProduct.UnitPerPack)
	unitQuantityDiff := req.UnitQuantity - (cartItem.UnitQuantity % storeProduct.UnitPerPack)

	// calculate cart_item total_price
	totalPrice := (((storeProduct.RetailPrice * 100) / float64(storeProduct.UnitPerPack)) * float64(updateQuantity)) / 100

	_, err = s.IncrementCartItemQuantityBySpId(ctx, s.db, req.StoreProductId, updateQuantity, totalPrice)
	if err != nil {
		return nil, err
	}

	// updated response
	response := map[string]any{
		"id":                 req.Id,
		"store_product_id":   req.StoreProductId,
		"increase":           isIncrease,
		"quantity":           req.Quantity,
		"unit_quantity":      req.UnitQuantity,
		"unit_per_pack":      storeProduct.UnitPerPack,
		"quantity_diff":      quantityDiff,
		"unit_quantity_diff": unitQuantityDiff,
	}

	return response, nil

}

func (s *Services) UpdateCartItemMarkings(ctx context.Context, id string, req *domain.AppendMarkingRequest) error {
	// get cart item
	cartItem, err := s.GetCartItemById(ctx, id)
	if err != nil {
		return err
	}

	// get sale
	sale, err := s.GetSaleById(ctx, cartItem.SaleId)
	if err != nil {
		return err
	}

	// check if sale is completed
	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return domain.SaleIsClosedError
	}

	// check duplicate
	if utils.In(req.Marking, cartItem.Markings...) {
		return domain.AlreadyExistsError
	}

	// append new marking
	err = s.db.Model(&cartItem).Update("markings", gorm.Expr("array_append(markings, ?)", req.Marking)).Error
	if err != nil {
		s.log.Errorf("could not update cart_item markings: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) IncrementCartItemQuantity(ctx context.Context, tx *gorm.DB, id string, quantity int, totalPrice float64) (*domain.CartItem, error) {
	var res domain.CartItem
	query := `UPDATE cart_items SET unit_quantity = unit_quantity + ?, total_price = total_price + ? WHERE id = ? RETURNING *`
	err := tx.WithContext(ctx).Raw(query, quantity, totalPrice, id).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not increment cart_item quantity: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

func (s *Services) IncrementCartItemQuantityBySpId(ctx context.Context, tx *gorm.DB, spId string, quantity int, totalPrice float64) (*domain.CartItem, error) {
	var res domain.CartItem
	query := `UPDATE cart_items SET unit_quantity = unit_quantity + ?, total_price = total_price + ? WHERE store_product_id = ? RETURNING *`
	err := tx.WithContext(ctx).Raw(query, quantity, totalPrice, spId).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not increment cart_item quantity: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

// add marking count to cart items
func (s *Services) updateCartItemsMarkingCount(ctx context.Context, tx *gorm.DB, req []domain.MarkingData) error {
	if len(req) == 0 {
		return nil
	}

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
	err := tx.WithContext(ctx).Exec(query).Error
	if err != nil {
		s.log.Error("could not update cart_item marking_count: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) updateCartItemDiscountValue(ctx context.Context, tx *gorm.DB, percent int, saleId string) error {
	err := tx.WithContext(ctx).Exec(`UPDATE cart_items SET discount_type = ?, discount_value = ? WHERE sale_id = ?;
	`, constants.DiscountTypePercent, percent, saleId).Error
	if err != nil {
		s.log.Errorf("could not update cart_item discount_value and type: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// region Delete

func (s *Services) DeleteCartItem(ctx context.Context, id string) error {
	cartItem, err := s.GetCartItemById(ctx, id)
	if err != nil {
		return err
	}

	sale, err := s.GetSaleById(ctx, cartItem.SaleId)
	if err != nil {
		return err
	}

	// check sale stage
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return domain.SaleIsClosedError
	}

	// delete cart_item
	err = s.deleteCartItemByIds(ctx, s.db, []string{id})
	if err != nil {
		return err
	}

	return nil
}

func (s *Services) DeleteCartItemMarkings(ctx context.Context, id string, req *domain.AppendMarkingRequest) error {
	// get cart item
	cartItem, err := s.GetCartItemById(ctx, id)
	if err != nil {
		return err
	}

	// get sale
	sale, err := s.GetSaleById(ctx, cartItem.SaleId)
	if err != nil {
		return err
	}

	// check if sale is completed
	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return domain.SaleIsClosedError
	}

	// remove marking
	err = s.db.Model(&cartItem).Update("markings", gorm.Expr("array_remove(markings, ?)", req.Marking)).Error
	if err != nil {
		s.log.Errorf("could not remove markings: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) deleteCartItemByIds(ctx context.Context, tx *gorm.DB, id []string) error {
	err := tx.WithContext(ctx).Delete(&domain.CartItem{}, "id IN (?)", id).Error
	if err != nil {
		s.log.Errorf("could not delete cart_item: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) DeleteCartItems(ctx context.Context, ids []string) error {
	// getting cart item
	cartItems, err := s.getCartItemsByIds(ctx, ids)
	if err != nil {
		return err
	}
	sale, err := s.GetSaleById(ctx, cartItems[0].SaleId)
	if err != nil {
		return err
	}

	// check sale status
	if !utils.In(sale.Stage, constants.PendingSaleStages...) {
		return domain.SaleIsClosedError
	}

	err = s.deleteCartItemByIds(ctx, s.db, ids)
	if err != nil {
		return err
	}

	return nil
}
