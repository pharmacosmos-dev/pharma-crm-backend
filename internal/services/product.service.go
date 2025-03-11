package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// get store products get list
func (s *Storage) ListStoreProduct(param *domain.StoreProductQueryParam, storeId string) ([]*domain.StoreProductResponse, error) {
	var (
		res []*domain.StoreProductResponse
		err error
	)
	// build query
	query := s.db.Model(&domain.StoreProduct{}).
		Table("store_products sp").
		Select(`
			sp.*, 
			((sp.retail_price/100)*sp.bonus_percent) AS bonus_amount,
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
		Joins("LEFT JOIN import_details im ON im.product_id = sp.product_id").
		Where("sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0)", storeId)

	if param.Search != "" {
		marking := param.Search
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where(`p.name ILIKE ? OR p.barcode LIKE ? OR c.name ILIKE ? OR ? = ANY(im.marking)`, param.Search, param.Search, param.Search, marking)
	}
	err = query.
		Limit(param.Limit).
		Offset(param.Offset).
		Order("sp.pack_quantity").
		Debug().
		Group("sp.id, p.id, c.id, u.id").
		Find(&res).Error

	if err != nil {
		s.log.Warn("Error on listing store products for store %s with search '%s': %v", storeId, param.Search, err.Error())
		return nil, err
	}
	for i := range res {
		if res[i].UnitPerPack > 0 && res[i].UnitQuantity != res[i].PackQuantity*res[i].UnitPerPack {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)", res[i].PackQuantity, res[i].UnitQuantity%res[i].UnitPerPack, res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].PackQuantity)
		}
	}

	return res, nil
}

// get similar products list
func (s *Storage) SimilarProducts(ctx context.Context, productID string, offset int, limit int) ([]domain.StoreProductResponse, error) {
	var res []domain.StoreProductResponse
	err := s.db.WithContext(ctx).Debug().
		Table("products p").
		Select(`
			p.name, p.barcode, p.unit_per_pack, sp.*, 
			((sp.bonus_percent*sp.retail_price)/100) as bonus_amount,
			u.unit_name, u.short_name,
			DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day`).
		Joins("JOIN category_products cp ON p.id = cp.product_id").
		Joins("JOIN store_products sp ON sp.product_id = p.id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Where(`cp.category_id = (
		SELECT category_id
		FROM category_products
		WHERE product_id = ?
		LIMIT 1
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
		if res[i].UnitPerPack > 0 && res[i].UnitQuantity != res[i].PackQuantity*res[i].UnitPerPack {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)", res[i].PackQuantity, res[i].UnitQuantity%res[i].UnitPerPack, res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].PackQuantity)
		}
	}

	return res, nil
}

// get store product info by barcode
func (s *Storage) GetStoreProductByBarcode(ctx context.Context, barcode string) (domain.StoreProductResponse, error) {
	var res domain.StoreProductResponse
	err := s.db.Raw(`
	SELECT
		sp.*,
		((sp.retail_price/100)*sp.bonus_percent) AS bonus_amount,
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

// get store info by product id
func (s *Storage) GetStoreProductByIdOrBarcode(id string, barcode string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct
	if id != "" {
		err := s.db.Raw(`SELECT sp.*, ((sp.retail_price/100)*sp.bonus_percent) AS bonus_amount, p.unit_per_pack FROM store_products sp JOIN products p ON sp.product_id = p.id WHERE sp.id = ?`, id).
			Scan(&storeProduct).Error
		if err != nil {
			return nil, err
		}
	} else if barcode != "" {
		err := s.db.Raw(`
		SELECT sp.*, ((sp.retail_price/100)*sp.bonus_percent) AS bonus_amount, p.unit_per_pack 
		FROM store_products sp 
		JOIN products p ON sp.product_id = p.id 
		LEFT JOIN import_details im ON im.product_id = sp.product_id
		WHERE p.barcode = ? OR ? = ANY(im.marking)`, barcode, barcode).
			Scan(&storeProduct).Error
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("id or barcode is required")
	}

	return &storeProduct, nil
}

func (s *Storage) GetStoreProductByID(id string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct
	err := s.db.Raw(`SELECT sp.*, ((sp.retail_price/100)*sp.bonus_percent) AS bonus_amount, p.unit_per_pack FROM store_products sp JOIN products p ON sp.product_id = p.id WHERE sp.id = ?`, id).
		Scan(&storeProduct).Error
	if err != nil {
		return nil, err
	}

	return &storeProduct, nil
}

// Change store product stock based on situation (increase or decrease)
func (s *Storage) ChangeStoreProductStock(tx *gorm.DB, id string, quantity, unitQuantity int, isIncrease bool) error {
	var operation = "-"
	if isIncrease {
		operation = "+"
	}
	err := tx.Exec(`UPDATE store_products SET pack_quantity = pack_quantity `+operation+` ?, unit_quantity = unit_quantity `+operation+` ? WHERE id = ?`,
		quantity, unitQuantity, id).Error
	if err != nil {
		return err
	}
	return nil
}

// get products get list
func (s *Storage) ListProduct(param *domain.ProductQueryParam) ([]*domain.Product, int64, error) {
	// get query param values
	var (
		res        []*domain.Product
		totalCount int64
	)

	// Build the base query
	baseQuery := s.db.Model(&domain.Product{}).
		Table("products p").
		Joins("LEFT JOIN store_products sp ON sp.product_id = p.id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Joins("LEFT JOIN producers pr ON pr.id = p.producer_id").
		Joins("LEFT JOIN category_products cp ON cp.product_id = p.id").
		Joins("LEFT JOIN categories c ON c.id = cp.category_id")

	// Apply filters
	if param.StoreID != "" {
		baseQuery = baseQuery.Where("sp.store_id = ?", param.StoreID)
	}
	// filter products with status
	if param.Status != "" {
		switch param.Status {
		case "active":
			baseQuery = baseQuery.Where("p.status = ?", "active")
		case "inactive":
			baseQuery = baseQuery.Where("p.status = ?", "inactive")
		case "low-stock":
			baseQuery = baseQuery.Where("sp.small_quantity = sp.pack_quantity")
		case "zero-stock":
			baseQuery = baseQuery.Where("sp.pack_quantity = ? AND sp.unit_quantity = ?", 0, 0)
		case "expired":
			baseQuery = baseQuery.Where("sp.expire_date::date < ?", time.Now().Format("2006-01-02"))
		case "imminent":
			baseQuery = baseQuery.Where("sp.expire_date BETWEEN ? AND ?", time.Now(), time.Now().AddDate(0, 0, 10))
		}
	} else {
		baseQuery = baseQuery.Where("p.status = ?", "active")
	}
	// search filter for product name, barcode, category name
	if param.SearchField != "" {
		param.SearchField = fmt.Sprintf("%%%s%%", param.SearchField)
		baseQuery = baseQuery.Where("p.name ILIKE ? OR p.barcode LIKE ? OR COALESCE(c.name, '') ILIKE ?",
			param.SearchField, param.SearchField, param.SearchField)
	}
	// filter with supply price greater than or equal to
	if param.SupplyPriceFrom > 0 {
		baseQuery = baseQuery.Where("sp.supply_price >= ?", param.SupplyPriceFrom)
	}
	// filter with supply price less than or equal to
	if param.SupplyPriceTo > 0 {
		baseQuery = baseQuery.Where("sp.supply_price <= ?", param.SupplyPriceTo)
	}
	// filter with retail price greater than or equal to
	if param.RetailPriceFrom > 0 {
		baseQuery = baseQuery.Where("sp.retail_price >= ?", param.RetailPriceFrom)
	}
	// filter with retail price less than or equal to
	if param.RetailPriceTo > 0 {
		baseQuery = baseQuery.Where("sp.retail_price <= ?", param.RetailPriceTo)
	}
	// filter products with producer id
	if param.ProducerID != "" {
		baseQuery = baseQuery.Where("p.producer_id = ?", param.ProducerID)
	}
	// filter with no barcodes
	if param.NoBarcode {
		baseQuery = baseQuery.Where("p.barcode IS NULL OR p.barcode = ''")
	}

	// Count total records using a subquery
	countQuery := baseQuery.Session(&gorm.Session{}).
		Select("COUNT(DISTINCT p.id)").
		Table("products p")
	// Execute the count query
	err := countQuery.Count(&totalCount).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}

	// Execute main query with all fields
	err = baseQuery.
		Preload("Categories").
		Select(`
		p.id, p.name, p.barcode, p.status, p.description,
		p.photos, pr.name as manufacturer, p.material_code,
		AVG(sp.supply_price) AS supply_price,
		AVG(sp.vat) AS vat,
		AVG(sp.markup) AS markup,
		AVG(sp.retail_price) AS retail_price,
		(AVG(sp.supply_price) * AVG(sp.vat) / 100) AS vat_price,
		(AVG(sp.supply_price) * AVG(sp.markup) / 100) AS markup_price,
		SUM(sp.pack_quantity) AS quantity,
		(SUM(sp.pack_quantity) * AVG(sp.retail_price)) AS sum,
		AVG(sp.bonus_percent) AS bonus_percent,
		AVG((sp.bonus_percent*sp.retail_price)/100) AS bonus_amount,
		u.short_name AS unit_name,
		STRING_AGG(c.name, ', ') as category_name,
		p.created_at`).
		Group(`p.id, u.id, pr.id`).
		Order("p.created_at DESC").
		Limit(param.Limit).
		Offset(param.Offset).
		Debug().
		Find(&res).Error

	if err != nil {
		s.log.Error(err)

		return nil, 0, err
	}
	return res, totalCount, nil
}

// get product ikpu by mxik
func (s *Storage) GetProductIKPUByMxik(ctx context.Context, mxik string) (*domain.ProductMeasurement, error) {
	var measurement domain.ProductMeasurement
	err := s.db.First(&measurement, "mxik_code = ?", mxik).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &measurement, nil
}

// get producer info by code
func (s *Storage) GetProducerByCode(ctx context.Context, code string) (*domain.Producer, error) {
	var producer domain.Producer
	err := s.db.Raw(`SELECT id, name, code, created_at, updated_at FROM producers WHERE code = ?`, code).Scan(&producer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			producerData, err := s.CreateProducer(ctx, code)
			if err != nil {
				s.log.Error(err)
				return nil, err
			}
			producer = *producerData
		}
		s.log.Error(err)
		return nil, err
	}
	return &producer, nil
}

// create new producer
func (s *Storage) CreateProducer(ctx context.Context, code string) (*domain.Producer, error) {
	var producer domain.Producer
	query := `INSERT INTO producers (code) VALUES (?) RETURNING *`
	err := s.db.Debug().WithContext(ctx).Raw(query, code).Scan(&producer).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &producer, nil
}
