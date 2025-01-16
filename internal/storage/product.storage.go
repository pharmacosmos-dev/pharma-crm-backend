package storage

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
)

func (s *Storage) ListStoreProduct(ctx context.Context, storeID string, search string) ([]*domain.StoreProductResponse, error) {
	var res []*domain.StoreProductResponse

	// Prepare search condition
	searchCondition := ""
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		searchCondition = `
			AND (
				p.name ILIKE ? 
				OR p.barcode LIKE ? 
				OR c.name ILIKE ?
			)
		`
	}

	// Build query with optional search
	query := fmt.Sprintf(`
		SELECT 
			sp.*, 
			p.name, 
			p.barcode, 
			c.name AS category_name,
			DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day 
		FROM store_products sp
		JOIN products p ON p.id = sp.product_id
		JOIN category_products cp ON p.id = cp.product_id
		JOIN categories c ON c.id = cp.category_id
		WHERE sp.store_id = ?
		%s
	`, searchCondition)

	// Execute query with appropriate arguments
	var err error
	if search != "" {
		err = s.db.Raw(query, storeID, search, search, search).Scan(&res).Error
	} else {
		err = s.db.Raw(query, storeID).Scan(&res).Error
	}

	// Handle errors and return response
	if err != nil {
		s.log.Warn("Error on listing store products for store %s with search '%s': %v", storeID, search, err.Error())
		return nil, err
	}
	for i := range res {
		if res[i].UnitPerPack > 0 && res[i].UnitQuantity > 0 && res[i].PackQuantity*res[i].UnitPerPack != res[i].UnitQuantity {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)", res[i].PackQuantity, res[i].UnitQuantity%res[i].UnitPerPack, res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].PackQuantity)
		}
	}

	return res, nil
}

func (s *Storage) SimilarProducts(ctx context.Context, productID string, offset int, limit int) ([]domain.StoreProductResponse, error) {
	var res []domain.StoreProductResponse
	err := s.db.Raw(`
	SELECT 
		sp.*, 
		p.name, 
		p.barcode, 
		c.name AS category_name, 
		DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day
	FROM store_products sp
	JOIN products p ON p.id = sp.product_id
	JOIN category_products cp ON p.id = cp.product_id
	JOIN categories c ON c.id = cp.category_id
	WHERE p.id != ? AND sp.store_id = (SELECT store_id FROM store_products WHERE product_id = ? LIMIT 1)  LIMIT ? OFFSET ?
	`, productID, productID, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing similar products for product %s: %v", productID, err.Error())
		return nil, err
	}

	return res, nil
}
