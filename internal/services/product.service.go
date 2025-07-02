package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"

	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// get store products get list
func (s *Services) ProductSearch(param *domain.StoreProductQueryParam) ([]*domain.StoreProductResponse, error) {
	var (
		res        []*domain.StoreProductResponse
		err        error
		args       = []any{}
		filter     = " WHERE sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0) "
		pagination = " LIMIT ? OFFSET ? "
		order      = " ORDER BY similarity_score DESC NULLS LAST, sp.expire_date "
	)

	searchInput := strings.TrimSpace(param.Search)
	translated := utils.Translit(searchInput)

	var similarityExpr string
	if searchInput != "" && utils.DefineProductSearchQuery(searchInput) == "name/category" {
		similarityExpr = "similarity(p.name, ?) AS similarity_score "
		args = append(args, searchInput) // <== similarity uchun argument
	} else {
		similarityExpr = "NULL AS similarity_score "
	}

	query := fmt.Sprintf(`
	SELECT
		sp.*, p.name, pr.name AS producer_name, pb.bonus_amount, p.barcode, p.unit_per_pack,
		DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day,
		u.unit_name, u.short_name,
		%s
	FROM store_products sp
		JOIN products p ON p.id = sp.product_id
		LEFT JOIN unit_types u ON p.unit_type_id = u.id
		LEFT JOIN producers pr ON pr.id = p.producer_id
		LEFT JOIN product_bonuses pb ON pb.product_id = p.id
	`, similarityExpr)

	args = append(args, param.StoreID)

	if searchInput != "" {
		switch utils.DefineProductSearchQuery(searchInput) {
		case "barcode":
			filter += " AND p.barcode = ? "
			args = append(args, searchInput)
			order = " ORDER BY sp.expire_date "

		case "marking":
			query += " LEFT JOIN product_markings pm ON pm.import_detail_id = sp.import_detail_id "
			filter += " AND pm.marking = ? "
			args = append(args, searchInput)
			order = " ORDER BY sp.expire_date "

		default:
			filter += `
				AND (
					p.name ILIKE ? OR 
					p.name ILIKE ?
				)
			`
			args = append(args, "%"+searchInput+"%", "%"+translated+"%")
		}
	}

	query = query + filter + order + pagination
	args = append(args, param.Limit, param.Offset)

	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("Error on listing store products for store %s with search '%s': %v", param.StoreID, param.Search, err.Error())
		return nil, err
	}

	// quantity format
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
	err := s.db.WithContext(ctx).
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
func (s *Services) GetStoreProductByIdOrBarcode(id string, marking string, storeId string) (*domain.StoreProduct, int, error) {
	var (
		storeProduct domain.StoreProduct
		filter       = " WHERE 1=1 "
		join         = ""
		args         = []any{}
	)

	query := `
	SELECT 
		sp.*, 
		p.unit_per_pack, 
		p.barcode
	FROM store_products sp
		JOIN products p ON sp.product_id = p.id
	`
	filter += " AND sp.store_id = ? "
	args = append(args, storeId)

	if id != "" {
		filter += " AND sp.id = ? "
		args = append(args, id)
	} else if marking != "" && utils.DefineProductSearchQuery(marking) == "marking" {
		filter += " AND pm.marking = ? "
		join = " LEFT JOIN product_markings pm ON pm.product_id = p.id AND pm.import_detail_id = sp.import_detail_id "
		args = append(args, marking)
	} else {
		return nil, 404, errors.New("product not found")
	}
	// collect query
	query = query + join + filter
	// complete query
	err := s.db.Raw(query, args...).Scan(&storeProduct).Error
	if err != nil {
		s.log.Warn("ERROR on getting store_product: %v", err)
		return &storeProduct, 500, err
	}

	if storeProduct.Id != "" {
		if utils.DefineProductSearchQuery(marking) == "marking" {
			isValid := utils.CheckBarcodeWithMarking(storeProduct.Barcode, marking) // <- check barcode with marking
			if !isValid {
				return nil, 422, errors.New("marking and barcode mismatch")
			}
		}
	}

	return &storeProduct, 200, nil
}

// get store products by product id
func (s *Services) GetStoreProductByID(id string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct
	err := s.db.Raw(`
		SELECT 
			sp.*, 
			pb.bonus_amount AS bonus_amount, 
			p.unit_per_pack AS unit_per_pack
		FROM 
			store_products sp 
		JOIN 
			products p ON sp.product_id = p.id
		LEFT JOIN 
			product_bonuses pb ON pb.product_id = p.id
		WHERE 
			sp.id = ?`, id).
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
		res           []domain.ProductData
		totalCount    int64
		args          []any
		filter        = "WHERE 1=1 "
		order         = " ORDER BY p.created_at DESC "
		group         = " GROUP BY p.id, pr.id, u.id "
		expireDayPart = ""
	)

	switch param.Order {
	case "+name":
		order = " ORDER BY p.name ASC "
	case "-name":
		order = " ORDER BY p.name DESC "
	case "+expire_date":
		order = " ORDER BY MIN(sp.expire_date) "
	case "-expire_date":
		order = " ORDER BY MIN(sp.expire_date) DESC "
	default:
		order = " ORDER BY p.created_at DESC "
	}

	// filter with store_id
	if param.StoreID != "" {
		filter += " AND sp.store_id IN (?) "
		expireDayPart = " DATE_PART('day', MIN(sp.expire_date)::timestamp - NOW()) AS expire_day, MIN(sp.expire_date) AS expire_date, "
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
			order = " ORDER BY MIN(sp.expire_date) "
			args = append(args, now.Format("2006-01-02"), now.AddDate(0, 6, 0).Format("2006-01-02"))
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

	query := fmt.Sprintf(`
	SELECT
		p.id, p.name, p.photos, p.barcode, p.material_code, 
		p.unit_per_pack, p.is_marking, p.mxik, p.unit_code, 
		p.unit_label, p.created_at, p.updated_at,
		pr.name AS manufacturer, u.unit_name, u.short_name,
		SUM(sp.pack_quantity) AS quantity,
		%s
		SUM(sp.unit_quantity)%sp.unit_per_pack AS unit_quantity,
		COUNT(1) OVER() AS total_count
	FROM store_products sp
	RIGHT JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	`, expireDayPart, "%")

	// collect query
	query += filter + group + order + " LIMIT ? OFFSET ?"
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

// test get product list
func (s *Services) ListProductExport(param *domain.ProductQueryParam) ([]domain.ProductData, error) {
	var (
		res    []domain.ProductData
		args   []any
		filter = "WHERE 1=1 "
		order  = " ORDER BY p.created_at DESC "
		group  = " GROUP BY p.id, pr.id, u.id "
	)
	// filter with store_id
	if param.StoreID != "" {
		filter += " AND sp.store_id IN (?) "
		order = " ORDER BY sp.expire_date "
		group += " , sp.expire_date "
		args = append(args, param.StoreID)
	}
	// filter with producer id
	if param.ProducerID != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerID)
	}

	query := fmt.Sprintf(`
	SELECT
		st.name AS store_name,
		p.id,  p.material_code, p.name,  p.barcode,
		p.unit_per_pack, p.is_marking, p.mxik, p.created_at, p.updated_at,
		pr.name AS manufacturer, u.unit_name, u.short_name,
		sp.pack_quantity AS quantity,
		sp.unit_quantity%sp.unit_per_pack AS unit_quantity,
		sp.supply_price AS supply_price,
		sp.retail_price AS retail_price,
		sp.expire_date,
		sp.serial_number,
		sp.vat AS vat,
		sp.vat_price AS vat_price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	JOIN stores st ON sp.store_id = st.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	`, "%")

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
			order = " ORDER BY sp.expire_date "
			args = append(args, now.Format("2006-01-02"), now.AddDate(0, 6, 0).Format("2006-01-02"))
		}
	}
	// filter with search
	if param.SearchField != "" {
		filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?) "
		args = append(args, "%"+param.SearchField+"%", "%"+param.SearchField+"%")
	}
	// filter with barcode
	if param.NoBarcode {
		filter += " AND (p.barcode IS NULL OR p.barcode = '') "
	}
	// collect query
	query += filter + order + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	// complete query
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product list: %v", err)
		return res, err
	}
	// check len and take empty array
	if len(res) == 0 {
		res = []domain.ProductData{}
	}

	return res, nil
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
		ROUND(SUM(sp.pack_quantity * retail_price) + SUM((retail_price / p.unit_per_pack) * (sp.unit_quantity % p.unit_per_pack)), 2) AS total_stock_amount,
		SUM(CASE WHEN sp.pack_quantity < 10 AND sp.pack_quantity > 0 THEN sp.pack_quantity ELSE 0 END) AS low_stock_quantity,
		SUM(CASE WHEN sp.pack_quantity = 0 THEN 1 ELSE 0 END) AS zero_stock_count,
		SUM(sp.pack_quantity) FILTER (WHERE sp.expire_date::date BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '3 month')) AS imminent_count,
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
	err := s.db.WithContext(ctx).Raw(query, code).Scan(&producer).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &producer, nil
}

// get noor products list
func (s *Services) GetNoorProducts(param *domain.NoorQueryParam) ([]domain.NoorProduct, error) {
	var res []domain.NoorProduct
	query := `
	SELECT
		p.id,
		p.name,
		p.photos,
		p.description,
		ARRAY_AGG(cp.category_id) FILTER (WHERE cp.category_id IS NOT NULL) AS categories
	FROM
		products p
	LEFT JOIN
		category_products cp ON cp.product_id = p.id
	GROUP BY
		p.id
	LIMIT ? OFFSET ?;
`
	err := s.db.Raw(query, param.Limit, param.Offset).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on listing noor products: %v", err)
		return res, err
	}

	return res, nil
}

// get noor store_products for auto fill
func (s *Services) GetNoorStoreProducts(param *domain.NoorQueryParam) ([]domain.NoorStoreProduct, error) {
	var res []domain.NoorStoreProduct

	query := `
	SELECT
		sp.store_id,
		sp.product_id,
		SUM(sp.unit_quantity/p.blister_count) AS quantity,
		ROUND(MIN(sp.retail_price), 0) AS price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	WHERE sp.unit_quantity/p.blister_count > 0 AND sp.updated_at >= ?
	GROUP BY sp.product_id, sp.store_id
	LIMIT ? OFFSET ?;
	`
	// execute query
	err := s.db.Raw(query, param.UpdatedAt, param.Limit, param.Offset).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting noor store_products: %v", err)
		return res, err
	}

	return res, nil
}

// store list for noor service
func (s *Services) GetNoorStores() ([]domain.NoorStore, error) {
	var (
		res []domain.NoorStore
		err error
	)
	query := `
	SELECT DISTINCT s.*
	FROM stores s
	INNER JOIN store_products sp ON s.id = sp.store_id;
	`
	// execute get store list query
	err = s.db.Raw(query).Scan(&res).Error
	if err != nil {
		s.log.Error("ERROR on listing external products: %v", err.Error())
		return nil, err
	}

	// get lat and long to point struct
	for i := range res {
		if res[i].Location != "" {
			res[i].Location1.Lat, err = strconv.ParseFloat(strings.Split(res[i].Location, ",")[0], 64)
			if err != nil {
				s.log.Warn("ERROR on parsing latitude: %v", err.Error())
			}
			res[i].Location1.Long, err = strconv.ParseFloat(strings.Split(res[i].Location, ",")[1], 64)
			if err != nil {
				s.log.Warn("ERROR on parsing longitude: %v", err.Error())
			}
		}
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
func (s *Services) GetProductMovements(productId, storeId string, limit, offset int) ([]domain.ImportProductData, int64, error) {
	var (
		res        []domain.ImportProductData
		totalCount int64
	)

	// Base query without store_id filter
	query := `
	SELECT *, COUNT(*) OVER() AS total_count FROM (
			SELECT
					im.id, im.public_id,
					im.entry_type, im.created_at,
					s.name AS store_name,
					SUM(imd.accepted_count) AS count,
					SUM(imd.accepted_count * imd.retail_price_vat) AS sum,
					im.name AS name
			FROM imports im
			JOIN stores s ON im.store_id = s.id
			LEFT JOIN import_details imd ON im.id = imd.import_id
			LEFT JOIN products p ON imd.product_id = p.id
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
					ROUND(SUM(ci.quantity)::numeric + SUM(ci.unit_quantity::numeric/p.unit_per_pack), 4) AS count,
					sa.total_amount AS sum,
					sa.sale_type AS name
			FROM sales sa
			JOIN stores st ON st.id = sa.store_id
			JOIN cart_items ci ON ci.sale_id = sa.id
			JOIN store_products sp ON sp.id = ci.store_product_id
			JOIN products p ON sp.product_id = p.id
			WHERE sp.product_id = ?
			AND sa.status = 'completed'
			GROUP BY sa.id, st.id
	) AS all_data
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?;
	`

	// Parameters for the query
	params := []any{productId, productId, limit, offset}

	// Modify query to include store_id filter if provided
	if storeId != "" {
		query = `
		SELECT *, COUNT(*) OVER() AS total_count FROM (
				SELECT
						im.id, im.public_id,
						im.entry_type, im.created_at,
						s.name AS store_name,
						SUM(imd.accepted_count) AS count,
						SUM(imd.accepted_count * imd.retail_price_vat) AS sum,
						im.name AS name
				FROM imports im
				JOIN stores s ON im.store_id = s.id
				LEFT JOIN import_details imd ON im.id = imd.import_id
				LEFT JOIN products p ON imd.product_id = p.id
				WHERE im.store_id = ? AND imd.product_id = ?
				AND im.status = 'completed'
				GROUP BY im.id, s.id

				UNION ALL

				SELECT
						sa.id AS id,
						sa.sale_number AS public_id,
						4 AS entry_type,
						sa.completed_at AS created_at,
						st.name AS store_name,
						ROUND(SUM(ci.quantity)::numeric + SUM(ci.unit_quantity::numeric/p.unit_per_pack), 4) AS count,
						sa.total_amount AS sum,
						sa.sale_type AS name
				FROM sales sa
				JOIN stores st ON st.id = sa.store_id
				JOIN cart_items ci ON ci.sale_id = sa.id
				JOIN store_products sp ON sp.id = ci.store_product_id
				JOIN products p ON sp.product_id = p.id
				WHERE sa.store_id = ? AND sp.product_id = ?
				AND sa.status = 'completed'
				GROUP BY sa.id, st.id
		) AS all_data
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?;
		`
		params = []any{storeId, productId, storeId, productId, limit, offset}
	}

	// Execute query
	err := s.db.Debug().Raw(query, params...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting product movements: %v", err)
		return res, totalCount, err
	}

	// Get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	return res, totalCount, nil
}

// get produt list for arzon apteka
func (s *Services) ProductListForArzon(storeId string) ([]domain.ProductArzon, error) {
	var res []domain.ProductArzon
	err := s.db.Raw(`
	SELECT 
		p.id, p.name, 
		COALESCE(pr.name, '') AS producer_name, 
		MIN(sp.retail_price) AS retail_price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	WHERE sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0)
	GROUP BY p.id, pr.id;
	`, storeId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}
	return res, nil
}

// get product id by material_code
func (s *Services) GetProductIDByCode(code int) (string, error) {
	var id string
	err := s.db.Raw(`SELECT id FROM products WHERE material_code = ?`, code).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on getting product_id by material_code: %v", err)
		return id, err
	}
	return id, nil
}

// update store_product retail price to new
func (s *Services) UpdateRetailPrice(id string, newPrice float64) error {
	// update retail price
	err := s.db.Exec(`UPDATE store_products SET retail_price = ? WHERE id = ?`, newPrice, id).Error
	if err != nil {
		s.log.Warn("ERROR on updating store_product retail_price: %v", err)
		return err
	}
	return nil
}

// get product list with import and ikpu
func (s *Services) GetProductListByImport(param *domain.ProductQueryParam) ([]domain.ProductByIkpu, int64, error) {
	var (
		filter     = " WHERE (sp.pack_quantity > 0 or sp.unit_quantity > 0) "
		order      = " ORDER BY sp.created_at DESC "
		args       = []any{}
		res        []domain.ProductByIkpu
		totalCount int64
	)

	query := `
	SELECT
		sp.id,
		sp.product_id,
		p.material_code,
		st.name AS store_name,
		COALESCE(im.document_number, '') AS import_number,
		p.name,
		COALESCE(sp.barcode, p.barcode) AS barcode,
		COALESCE(pr.name, '') as producer_name,
		sp.pack_quantity as quantity,
		(sp.unit_quantity % p.unit_per_pack) AS unit_quantity,
		p.unit_per_pack,
		sp.is_marking,
		sp.is_checking,
		sp.serial_number,
		sp.expire_date,
		sp.retail_price,
		sp.supply_price,
		sp.mxik, 
		sp.unit_code, 
		sp.unit_label,
		sp.created_at,
		sp.updated_at
	FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		JOIN stores st ON sp.store_id = st.id
	LEFT JOIN import_details imd ON sp.import_detail_id = imd.id
	LEFT JOIN imports im ON imd.import_id = im.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`
	totalQuery := `
	SELECT
		COUNT(*) as total_count
	FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		JOIN stores st ON sp.store_id = st.id
	LEFT JOIN import_details imd ON sp.import_detail_id = imd.id
	LEFT JOIN imports im ON imd.import_id = im.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	// filter by store_id
	if param.StoreID != "" {
		filter += " AND sp.store_id = ? "
		args = append(args, param.StoreID)
	}
	// filter by search keyword
	if param.SearchField != "" {
		filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?) "
		args = append(args, "%"+param.SearchField+"%", "%"+param.SearchField+"%")
	}
	// filter with barcode
	if param.NoBarcode {
		filter += " AND (p.barcode IS NULL OR p.barcode = '') "
	}

	if param.ProducerID != "" {
		filter += " AND p.producer_id = ? "
		args = append(args, param.ProducerID)
	}

	if param.ImportId != "" {
		filter += " AND imd.import_id = ? "
		args = append(args, param.ImportId)
	}
	// collect total count query with filters
	totalQuery += filter
	err := s.db.Raw(totalQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting products by ikpu total_count: %v", err)
		return res, totalCount, err
	}
	// collect get list query
	query += filter + order + "LIMIT ? OFFSET ? "
	args = append(args, param.Limit, param.Offset)
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting products by ikpu: %v", err)
		return res, totalCount, err
	}

	return res, totalCount, nil
}

// get min, max products
func (s *Services) GetMinMaxProducts(param *domain.ProductQueryParam) ([]domain.MinMaxProduct, int64, error) {
	var (
		res        []domain.MinMaxProduct
		totalCount int64
		filter     = " WHERE 1 = 1 "
		order      = " ORDER BY spt.created_at DESC "
		args       = []any{}
	)
	// query for getting product list with kvant, min and max quantity
	query := `
	SELECT
		spt.id,
		spt.store_id,
		spt.product_id,
		s.name AS store_name,
		p.material_code,
		p.name,
		spt.kvant,
		spt.min_quantity,
		spt.max_quantity,
		spt.is_active,
		spt.created_at,
		spt.updated_at
	FROM store_product_thresholds spt
	JOIN products p ON spt.product_id = p.id
	JOIN stores s ON spt.store_id = s.id
	`
	// query for getting total_count
	totalCountQuery := `
	SELECT
		COUNT(*) AS total_count
	FROM store_product_thresholds spt
	JOIN products p ON spt.product_id = p.id
	JOIN stores s ON spt.store_id = s.id
	`
	if param.StoreID != "" {
		filter += " AND spt.store_id = ? "
		args = append(args, param.StoreID)
	}

	if param.SearchField != "" {
		filter += " AND p.name ILIKE ? "
		args = append(args, "%"+param.SearchField+"%")
	}
	// collect total query
	totalCountQuery += filter
	err := s.db.Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting total_count: %v", err)
		return res, totalCount, err
	}
	// collect query
	query += filter + order + " LIMIT ? OFFSET ?" // add limit, offset for pagination
	args = append(args, param.Limit, param.Offset)
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting min_max_products: %v", err)
		return res, totalCount, err
	}

	return res, totalCount, nil
}
