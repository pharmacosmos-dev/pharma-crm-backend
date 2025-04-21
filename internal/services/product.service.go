package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// get store products get list
func (s *Services) ListStoreProduct(param *domain.StoreProductQueryParam) ([]*domain.StoreProductResponse, error) {
	var (
		res []*domain.StoreProductResponse
		err error
	)

	// build query
	query := s.db.Model(&domain.StoreProduct{}).
		Table("store_products sp").
		Select(`
			DISTINCT ON (sp.product_id)
			sp.*, pb.bonus_amount AS bonus_amount, p.name, p.barcode, p.unit_per_pack,
			DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day,
			u.unit_name, u.short_name`).
		Joins("JOIN products p ON p.id = sp.product_id").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Joins("LEFT JOIN product_bonuses pb ON pb.product_id = sp.product_id").
		Where("sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0)", param.StoreID)
	// define search keyword type
	switch utils.DefineProductSearchQuery(param.Search) {
	case "barcode":
		param.Limit = 1
		query = query.Where("p.barcode = ?", param.Search)
	case "marking":
		param.Limit = 1
		query = query.
			Joins("LEFT JOIN product_markings pm ON pm.product_id = sp.product_id").
			Where("pm.marking = ?", param.Search)
	default:
		// Transliterate search keyword Latin to Cyrillic OR Cyrillic to Latin
		translatedWord := utils.Translit(param.Search)
		// define search key
		query = query.
			Joins("LEFT JOIN category_products cp ON p.id = cp.product_id").
			Joins("LEFT JOIN categories c ON c.id = cp.category_id").
			Where("p.name ILIKE ? OR c.name ILIKE ? OR p.name ILIKE ?", "%"+param.Search+"%", "%"+param.Search+"%", "%"+translatedWord+"%").
			Limit(param.Limit).Offset(param.Offset)
	}
	// complete query
	err = query.
		Limit(param.Limit).
		Offset(param.Offset).
		Order("sp.product_id, sp.expire_date").
		Find(&res).Error

	if err != nil {
		s.log.Warn("Error on listing store products for store %s with search '%s': %v", param.StoreID, param.Search, err.Error())
		return nil, err
	}

	// format quantity
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
func (s *Services) SimilarProducts(ctx context.Context, productID string, offset int, limit int) ([]domain.StoreProductResponse, error) {
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
func (s *Services) GetStoreProductByBarcode(ctx context.Context, barcode string) (domain.StoreProductResponse, error) {
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
func (s *Services) GetStoreProductByIdOrBarcode(id string, barcode string, storeId string) (*domain.StoreProduct, error) {
	query := s.db.
		Table("store_products sp").
		Select(`sp.*, pb.bonus_amount AS bonus_amount, p.unit_per_pack, p.barcode`).
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("LEFT JOIN product_bonuses pb ON pb.product_id = p.id").
		Where("sp.store_id = ?", storeId)
	var storeProduct domain.StoreProduct
	if id != "" {
		query = query.Where("sp.id = ?", id).Limit(1)
	} else if barcode != "" {
		// define search key
		switch utils.DefineProductSearchQuery(barcode) {
		case "marking": // if received field that is barcode will be marking it checks with marking code
			query = query.
				Joins("LEFT JOIN product_markings pm ON pm.product_id = p.id").
				Where("pm.marking = ?", barcode).Limit(1)
			// if utils.ExtractNumbers(barcode) != "" {
			// 	query = query.Or("p.barcode = ?", utils.ExtractNumbers(barcode))
			// }
		default: // if received field not be marking it checks default with barcode
			query = query.Where("p.barcode = ?", barcode).Limit(1)
		}
	} else {
		return nil, errors.New("id or barcode is required")
	}

	err := query.Debug().First(&storeProduct).Error
	if err != nil {
		return nil, err
	}

	if storeProduct.Id != "" {
		if utils.DefineProductSearchQuery(barcode) == "marking" {
			isValid := utils.CheckBarcodeWithMarking(storeProduct.Barcode, barcode) // <- bu sizning tayyor tekshiruvchi funksiyangiz
			if !isValid {
				return nil, errors.New("marking and barcode mismatch") // yoki custom xatolik
			}
		}
	}

	return &storeProduct, nil
}

func (s *Services) GetStoreProductByID(id string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct
	err := s.db.Raw(`
		SELECT sp.*, pb.bonus_amount AS bonus_amount, 
		p.unit_per_pack AS unit_per_pack
		FROM store_products sp 
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN product_bonuses pb ON pb.product_id = p.id
		WHERE sp.id = ?`, id).
		Scan(&storeProduct).Error
	if err != nil {
		return nil, err
	}

	return &storeProduct, nil
}

// Change store product stock based on situation (increase or decrease)
func (s *Services) ChangeStoreProductStock(tx *gorm.DB, id string, quantity, unitQuantity int, isIncrease bool) error {
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
func (s *Services) ListProduct(param *domain.ProductQueryParam) ([]domain.ProductData, int64, error) {
	var (
		res        []domain.ProductData
		totalCount int64
		args       []any
		filter     = "WHERE 1=1 "
		group      = " GROUP BY p.id, pr.id, u.id"
	)
	query := fmt.Sprintf(`
	SELECT
		p.id, p.name, p.photos, p.barcode, p.material_code, 
		p.unit_per_pack, p.is_marking, p.mxik, p.created_at, p.updated_at,
		pr.name AS manufacturer, u.unit_name, u.short_name,
		SUM(sp.pack_quantity) AS quantity,
		SUM(CASE WHEN p.unit_per_pack > 0 THEN sp.unit_quantity%sp.unit_per_pack ELSE 0 END) AS unit_quantity,
		COUNT(1) OVER() AS total_count
	FROM store_products sp
	RIGHT JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	`, "%")

	// filter with store_id
	if param.StoreID != "" {
		filter += " AND sp.store_id IN (?) "
		args = append(args, param.StoreID)
	}
	// filter with producer id
	if param.ProducerID != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerID)
	}

	// filter with statuses
	if param.Status != "" {
		switch param.Status {
		case "active", "inactive":
			filter += " AND p.status = ? "
			args = append(args, param.Status)
		case "low-stock":
			filter += " AND (sp.pack_quantity <= 10 AND sp.pack_quantity > 0) "
		case "zero-stock":
			filter += " AND (sp.pack_quantity = 0 AND sp.unit_quantity = 0) "
		case "expired":
			filter += " AND sp.expire_date::date < ?"
			args = append(args, time.Now().Format("2006-01-02"))
		case "imminent":
			filter += " AND (sp.expire_date::date BETWEEN ? AND ?) "
			now := time.Now()
			args = append(args, now.Format("2006-01-02"), now.Add(time.Hour*240).Format("2006-01-02"))
		}
	}
	// filter with search
	if param.SearchField != "" {
		search := "%" + param.SearchField + "%"
		filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?) "
		args = append(args, search, search)
	}
	// filter with barcode
	if param.NoBarcode {
		filter += " AND (p.barcode IS NULL OR p.barcode = '') "
	}
	// collect query
	query += filter + group + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	// complete query
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product list: %v", err)
		return res, totalCount, err
	}
	// check len and take empty array
	if len(res) == 0 {
		res = []domain.ProductData{}
	}
	// get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// Get Products list stats
func (s *Services) ListProductStats(param *domain.ProductQueryParam) (domain.ProductStats, error) {
	var (
		res    domain.ProductStats
		args   []any
		filter = "WHERE 1=1 "
	)
	query := `
	SELECT
		SUM(sp.pack_quantity) AS total_quantity,
		SUM(CASE WHEN p.status = 'active' THEN sp.pack_quantity ELSE 0 END) AS active_count,
		SUM(CASE WHEN p.status = 'inactive' THEN sp.pack_quantity ELSE 0 END) AS inactive_count,
		SUM(CASE WHEN sp.pack_quantity < 10 AND sp.pack_quantity > 0 THEN sp.pack_quantity ELSE 0 END) AS low_stock_quantity,
		SUM(CASE WHEN sp.pack_quantity = 0 THEN 1 ELSE 0 END) AS zero_stock_count,
		SUM(sp.pack_quantity) FILTER (WHERE sp.expire_date::date BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '10 days')) AS imminent_count,
		SUM(sp.pack_quantity) FILTER (WHERE sp.expire_date::date < CURRENT_DATE) AS expired_count,
		COUNT(DISTINCT p.id) AS total_count
	FROM store_products sp
	RIGHT JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	`

	// filter with store_ids
	if param.StoreID != "" {
		filter += " AND sp.store_id IN (?) "
		args = append(args, param.StoreID)
	}
	// filter with producer_id
	if param.ProducerID != "" {
		filter += " AND p.producer_id IN (?) "
		args = append(args, param.ProducerID)
	}
	// filter with search
	if param.SearchField != "" {
		search := "%" + param.SearchField + "%"
		filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?) "
		args = append(args, search, search)
	}
	// check barcode is null or emplty string
	if param.NoBarcode {
		filter += " AND (p.barcode IS NULL OR p.barcode = '') "
	}
	// collect query
	query = query + filter
	// complete query
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product stats: %v", err)
		return res, err
	}

	return res, nil
}

// test get product list
func (s *Services) ListProductExport(param *domain.ProductQueryParam) ([]domain.Product, error) {
	res := []domain.Product{}
	query := s.builder.
		Select(`p.id, p.material_code, p.name, p.photos, p.barcode,
				p.unit_per_pack, p.mxik, p.is_marking, p.created_at, p.updated_at,
				pr.name AS manufacturer, ut.unit_name, ut.short_name`,
		).From("products p").
		Join("LEFT JOIN producers pr ON pr.id = p.producer_id").
		Join("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id")

	countQuery := s.builder.Select("COUNT(*)").From("products p").
		Join("LEFT JOIN producers pr ON pr.id = p.producer_id").
		Join("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id")

	if param.ProducerID != "" {
		query = query.Where("p.producer_id = ?", param.ProducerID)
		countQuery = countQuery.Where("p.producer_id = ?", param.ProducerID)
	}
	if param.NoBarcode {
		query = query.Where("(p.barcode IS NULL OR p.barcode = '')")
		countQuery = countQuery.Where("(p.barcode IS NULL OR p.barcode = '')")
	}
	if param.SearchField != "" {
		search := "%" + param.SearchField + "%"
		query = query.Where("(p.name ILIKE ? OR p.barcode LIKE ?)", search, search)
		countQuery = countQuery.Where("(p.name ILIKE ? OR p.barcode LIKE ?)", search, search)
	}
	sql, args, err := query.Limit(param.Limit).Offset(param.Offset).Build()
	if err != nil {
		s.log.Warn("ERROR on building query: %v", err)
		return res, err
	}
	err = s.db.Debug().Raw(sql, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting products: %v", err)
		return res, err
	}

	countSql, args, err := countQuery.Count().Build()
	if err != nil {
		s.log.Warn("ERROR on building count query: %v", err)
		return res, err
	}
	var totalCount int64
	err = s.db.Debug().Raw(countSql, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting products: %v", err)
		return res, err
	}
	fmt.Println("COUNT: ", &totalCount)
	return res, nil
}

// get product ikpu by mxik
func (s *Services) GetProductIKPUByMxik(ctx context.Context, mxik string) (*domain.ProductMeasurement, error) {
	var measurement domain.ProductMeasurement
	err := s.db.First(&measurement, "mxik_code = ?", mxik).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &measurement, nil
}

// get producer info by code
func (s *Services) GetProducerByCode(ctx context.Context, code string) (*domain.Producer, error) {
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
func (s *Services) CreateProducer(ctx context.Context, code string) (*domain.Producer, error) {
	var producer domain.Producer
	query := `INSERT INTO producers (code) VALUES (?) RETURNING *`
	err := s.db.Debug().WithContext(ctx).Raw(query, code).Scan(&producer).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &producer, nil
}

// get external products list
func (s *Services) GetExternalProducts(limit, offset int, search string) ([]domain.ProductExternal, error) {
	var (
		res []domain.ProductExternal
	)
	query := s.db.
		Table("products p").
		Select("p.id, p.name, p.barcode, p.photos, p.description, u.short_name AS unit_name, sp.price").
		Joins("LEFT JOIN unit_types u ON p.unit_type_id = u.id").
		Joins("JOIN (SELECT product_id, MIN(retail_price) AS price FROM store_products GROUP BY product_id) sp ON p.id = sp.product_id").
		Preload("Stores", func(db *gorm.DB) *gorm.DB {
			return db.Table("store_products sp").
				Select("sp.product_id, s.id, s.name, s.phone, s.address, s.location, s.work_hours, sp.pack_quantity as quantity, sp.unit_quantity, sp.expire_date").
				Joins("JOIN stores s ON s.id = sp.store_id")
		}).
		Limit(limit).Offset(offset)

	err := query.Find(&res).Error
	if err != nil {
		s.log.Error("ERROR on listing external products: %v", err)
		return nil, err
	}
	for i := range res {
		err = s.db.Raw(`SELECT category_id FROM category_products WHERE product_id = ?`, res[i].Id).Scan(&res[i].Categories).Error
		if err != nil {
			s.log.Error("ERROR on listing category ids: %v", err)
			return nil, err
		}
	}

	return res, nil
}

func (s *Services) GetExternalStoresByProductId(productId string) ([]domain.StoreExternal, error) {
	var (
		res []domain.StoreExternal
		err error
	)

	query := `
		SELECT s.id, s.name, s.address, s.phone, s.location, s.work_hours, sp.pack_quantity AS quantity, sp.unit_quantity, sp.expire_date
		FROM store_products sp
		JOIN stores s ON s.id = sp.store_id
		WHERE (sp.pack_quantity > 0 OR sp.unit_quantity > 0) AND sp.expire_date > NOW() AND sp.product_id = ?
		ORDER BY sp.expire_date
	`
	err = s.db.Raw(query, productId).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on listing external products: %v", err.Error())
		return nil, err
	}
	return res, nil
}

// update product is_marking field
func (s *Services) UpdateProductIsMarking(req *domain.UpdateIsMarking) error {
	// build query
	query := `UPDATE products SET is_marking = ? WHERE id = ?`
	// complete the update query
	err := s.db.Exec(query, req.IsMarking, req.ProductId).Error
	if err != nil {
		s.log.Warn("ERROR on updating is_marking: %v", err.Error())
		return err
	}
	return nil
}

// get product movements(Import, Inventory, Write-Off, Sale)
func (s *Services) GetProductMovements(productId string, limit, offset int) ([]domain.ImportProductData, int64, error) {
	var (
		res        []domain.ImportProductData
		totalCount int64
	)
	// build count query
	countQuery := `
	SELECT COUNT(*) FROM (
		SELECT im.id
		FROM imports im
		JOIN stores s ON im.store_id = s.id
		LEFT JOIN import_details imd ON im.id = imd.import_id
		WHERE imd.product_id = ?
		AND im.status = 'completed'
		GROUP BY im.id, s.id

		UNION ALL

		SELECT sa.id
		FROM sales sa
		JOIN stores st ON st.id = sa.store_id
		JOIN cart_items ci ON ci.sale_id = sa.id
		JOIN store_products sp ON sp.id = ci.store_product_id
		WHERE sp.product_id = ?
		AND sa.status = 'completed'
		GROUP BY sa.id, st.id
	) AS total_data;
	`

	// build query
	query := `
	SELECT * FROM (
		SELECT
			im.id, im.public_id, im.entry_type, im.created_at,
			s.name AS store_name,
			SUM(imd.accepted_count) AS count,
			SUM(imd.accepted_count * imd.retail_price_vat) AS sum
		FROM imports im
		JOIN stores s ON im.store_id = s.id
		LEFT JOIN import_details imd ON im.id = imd.import_id
		WHERE imd.product_id = ?
		AND im.status = 'completed'
		GROUP BY im.id, s.id

		UNION ALL

		SELECT
			sa.id AS id,
			sa.sale_number AS public_id,
			4 AS entry_type,
			sa.completed_at AS created_at,
			st.name AS store_name,
			SUM(ci.quantity + (ci.unit_quantity / 10.0)) AS count,
			sa.total_amount AS sum
		FROM sales sa
		JOIN stores st ON st.id = sa.store_id
		JOIN cart_items ci ON ci.sale_id = sa.id
		JOIN store_products sp ON sp.id = ci.store_product_id
		WHERE sp.product_id = ?
		AND sa.status = 'completed'
		GROUP BY sa.id, st.id
	) AS all_data
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?;
	`

	// complete count query
	err := s.db.Raw(countQuery, productId, productId).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on counting product movements: %v", err)
		return res, totalCount, err
	}

	// complete query
	err = s.db.Raw(query, productId, productId, limit, offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product movements: %v", err)
		return res, totalCount, err
	}
	return res, totalCount, nil
}
