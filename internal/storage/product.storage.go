package storage

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Storage) ListStoreProduct(ctx context.Context, storeID string, search string, limit, offset int) ([]*domain.StoreProductResponse, error) {
	var (
		res []*domain.StoreProductResponse
		err error
	)

	query := s.db.Model(&domain.StoreProduct{}).
		Table("store_products sp").
		Select(`
			sp.*, 
			p.name, 
			p.barcode, 
			p.unit_per_pack,
			c.name AS category_name,
			DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day,
			u.unit_name,
			u.short_name`).
		Joins("JOIN products p ON p.id = sp.product_id").
		Joins("LEFT JOIN category_products cp ON p.id = cp.product_id").
		Joins("LEFT JOIN categories c ON c.id = cp.category_id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Where("sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0)", storeID)

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("p.name ILIKE ? OR p.barcode LIKE ? OR c.name ILIKE ?", search, search, search)
	}
	err = query.Limit(limit).Offset(offset).Order("sp.pack_quantity").Find(&res).Error

	if err != nil {
		s.log.Warn("Error on listing store products for store %s with search '%s': %v", storeID, search, err.Error())
		return nil, err
	}
	for i := range res {
		if res[i].UnitQuantity > 0 && res[i].PackQuantity*res[i].UnitPerPack != res[i].UnitQuantity {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)", res[i].PackQuantity, res[i].UnitQuantity, res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].PackQuantity)
		}
	}

	return res, nil
}

func (s *Storage) SimilarProducts(ctx context.Context, productID string, offset int, limit int) ([]domain.StoreProductResponse, error) {
	var res []domain.StoreProductResponse
	err := s.db.WithContext(ctx).Debug().
		Table("products p").
		Select(`
			p.name, p.barcode, p.unit_per_pack, sp.*, 
			u.unit_name, u.short_name, 
			DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day`).
		Joins("JOIN category_products cp ON p.id = cp.product_id").
		Joins("JOIN store_products sp ON sp.product_id = p.id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Where(`cp.category_id = (
		SELECT category_id
		FROM category_products
		WHERE product_id = ?
		)`, productID).
		Where(`sp.store_id = (
		SELECT store_id
		FROM store_products
		WHERE product_id = ?
		LIMIT 1
		)`, productID).
		Where("p.id <> ?", productID).
		Limit(limit).Offset(offset).
		Debug().
		Find(&res).Error
	if err != nil {
		s.log.Warn("Error on listing similar products for product %s: %v", productID, err.Error())
		return nil, err
	}

	for i := range res {
		if res[i].PackQuantity*res[i].UnitPerPack != res[i].UnitQuantity {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)", res[i].PackQuantity, res[i].UnitQuantity, res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].PackQuantity)
		}
	}

	return res, nil
}

func (s *Storage) GetStoreProductByBarcode(ctx context.Context, barcode string) (domain.StoreProductResponse, error) {
	var res domain.StoreProductResponse
	err := s.db.Raw(`
	SELECT
		sp.*,
		p.name,
		p.barcode,
		c.name AS category_name,
		DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day,
		u.unit_name AS unit_name,
		u.short_name
	FROM store_products sp
	JOIN products p ON p.id = sp.product_id
	LEFT JOIN category_products cp ON p.id = cp.product_id
	LEFT JOIN categories c ON c.id = cp.category_id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	WHERE p.barcode = ? ORDER BY sp.expire_date
	`, barcode).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing products for product %s: %v", barcode, err.Error())
		return domain.StoreProductResponse{}, err
	}

	return res, nil
}
