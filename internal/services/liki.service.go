package services

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// Liki integratsiyasi biznes logikasi. Noor funksiyalaridan nusxa olingan, lekin
// ulardan mustaqil: Noor tomonidagi o'zgarish /liki/* API'lariga ta'sir qilmaydi.

// region Products

func (s *Services) GetLikiProducts(params *domain.NoorQueryParam) ([]domain.NoorProduct, error) {
	var (
		res    []domain.NoorProduct
		filter = ` WHERE p.requires_prescription = false AND EXISTS (
			SELECT 1 FROM online_products_price opp
			WHERE opp.product_id = p.id AND opp.type = ?
		) `
		args = []any{constants.ServiceTypeUzum}
	)

	// producer/country/category — LEFT JOIN: products'dagi FK'lar nullable,
	// INNER JOIN bo'lsa ular to'ldirilmagan mahsulotlar ro'yxatdan tushib qolardi.
	query := `
	SELECT
		p.id,
		p.name,
		p.photos,
		p.unit_code,
		p.description,
		p.description_uz,
		p.description_ru,
		p.description_kr,
		p.unit_per_pack,
		p.requires_prescription,
		p.unit_label,

		COALESCE(p.category_id::text, '')  AS category_id,
		COALESCE(cat.name, '')             AS category_name,
		COALESCE(p.producer_id::text, '')  AS producer_id,
		COALESCE(prod.name, '')            AS producer_name,
		COALESCE(p.country_id::text, '')   AS country_id,
		COALESCE(cnt.name, '')             AS country_name,
		COALESCE(p.unit_type_id::text, '') AS unit_type_id,
		COALESCE(ut.unit_name, '')         AS unit_name,

		sp.expire_date
	FROM
		products p
		LEFT JOIN categories cat ON cat.id = p.category_id
		LEFT JOIN producers prod ON prod.id = p.producer_id
		LEFT JOIN countries cnt  ON cnt.id = p.country_id
		LEFT JOIN unit_types ut ON ut.id = p.unit_type_id

		LEFT JOIN LATERAL (
			-- sof sana ("2027-03-15"), timestamp emas; NULL bo'lsa javobda expire_date: null
			SELECT TO_CHAR(sp.expire_date, 'YYYY-MM-DD') AS expire_date
			FROM store_products sp
			WHERE sp.product_id = p.id
			  AND sp.unit_quantity > 0
			ORDER BY sp.created_at DESC
			LIMIT 1
		) sp ON TRUE
	` + filter + `
	ORDER BY p.created_at, p.id LIMIT ? OFFSET ?;`

	args = append(args, params.Limit, params.Offset)

	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get products for liki: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetLikiStoreProducts(params *domain.NoorQueryParam) ([]domain.NoorStoreProduct, error) {
	var (
		res    []domain.NoorStoreProduct
		filter = " WHERE p.requires_prescription = false AND sp.unit_quantity > 0"
		args   = []any{constants.ServiceTypeUzum}
	)

	if params.ShopId != "" {
		filter += " AND sp.store_id = ? "
		args = append(args, params.ShopId)
	}

	args = append(args, params.Limit, params.Offset)

	query := `
	SELECT
		sp.store_id,
		sp.product_id,
		SUM(sp.unit_quantity/(p.unit_per_pack/p.blister_count)) AS quantity,
		ROUND(opp.retail_price)::int AS price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	JOIN LATERAL (
		SELECT retail_price
		FROM online_products_price
		WHERE product_id = sp.product_id AND type = ?
		ORDER BY created_at DESC
		LIMIT 1
	) opp ON true
	` + filter + `
	GROUP BY sp.product_id, sp.store_id, opp.retail_price
	HAVING SUM(sp.unit_quantity/(p.unit_per_pack/p.blister_count)) > 0
	LIMIT ? OFFSET ?;
	`

	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get store_products for liki: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

// GetLikiStoreProductQuantities — do'kondagi berilgan mahsulotlar qoldig'i (qadoqda),
// har bir mahsulotdan eng katta partiya. Narxi yo'q mahsulot javobga tushmaydi.
func (s *Services) GetLikiStoreProductQuantities(storeId string, productIds []string) ([]domain.NoorStoreProductQuantity, error) {
	var res []domain.NoorStoreProductQuantity

	query := `
	SELECT
		ids.product_id::text AS product_id,
		COALESCE(sp.unit_quantity / NULLIF(p.unit_per_pack, 0), 0)::int AS quantity,
		ROUND(opp.retail_price)::int AS price
	FROM unnest(string_to_array(?, ',')::uuid[]) WITH ORDINALITY AS ids(product_id, ord)
	JOIN LATERAL (
		SELECT retail_price
		FROM online_products_price
		WHERE product_id = ids.product_id AND type = ?
		ORDER BY created_at DESC
		LIMIT 1
	) opp ON TRUE
	LEFT JOIN LATERAL (
		SELECT unit_quantity
		FROM store_products
		WHERE product_id = ids.product_id
		AND store_id = ?
		AND unit_quantity > 0
		ORDER BY unit_quantity DESC
		LIMIT 1
	) sp ON TRUE
	LEFT JOIN products p ON p.id = ids.product_id
	ORDER BY ids.ord;`

	err := s.db.Raw(query, strings.Join(productIds, ","), constants.ServiceTypeUzum, storeId).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store_product quantities for liki: %v", err)
		return nil, domain.InternalServerError
	}

	// store_id so'rovdan keladi, SQL uni qaytarmaydi.
	for i := range res {
		res[i].StoreId = storeId
	}

	return res, nil
}

// region Categories / Stores

// GetLikiCategories — kategoriya daraxti tekis ro'yxat ko'rinishida, ota kategoriya bolalaridan oldin.
func (s *Services) GetLikiCategories() ([]domain.NoorCategory, error) {
	var res []domain.NoorCategory

	query := `
	WITH RECURSIVE category_hierarchy AS (
		SELECT
			id,
			name AS name_ru,
			name_uz,
			name_en,
			name_kr,
			category_id AS parent_id,
			photo,
			ARRAY[id] AS path
		FROM categories
		WHERE category_id IS NULL

		UNION ALL

		SELECT
			c.id,
			c.name AS name_ru,
			c.name_uz,
			c.name_en,
			c.name_kr,
			c.category_id AS parent_id,
			c.photo,
			ch.path || c.id
		FROM categories c
		INNER JOIN category_hierarchy ch ON c.category_id = ch.id
	)
	SELECT
		id,
		name_ru,
		name_uz,
		name_en,
		name_kr,
		parent_id,
		photo
	FROM category_hierarchy
	ORDER BY path;
	`
	if err := s.db.Raw(query).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get categories for liki: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetLikiStores() ([]domain.NoorStore, error) {
	var res []domain.NoorStore

	query := `
	SELECT
		id,
		name,
		phone,
		address,
		location,
		work_hours,
		is_fullday,
		is_active
	FROM stores
		WHERE is_active = true
	ORDER BY store_code;
	`
	if err := s.db.Raw(query).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get stores for liki: %v", err)
		return nil, domain.InternalServerError
	}

	// "lat,long" satrini Point ga aylantirish; noto'g'ri qiymatlar o'tkazib yuboriladi
	for i := range res {
		loc := strings.TrimSpace(res[i].Location)
		if loc == "" || loc == "null" {
			continue
		}

		parts := strings.Split(loc, ",")
		if len(parts) != 2 {
			s.log.Warn("invalid location format: %q", loc)
			continue
		}

		lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			s.log.Warn("ERROR on parsing latitude (%q): %v", parts[0], err)
			continue
		}

		long, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			s.log.Warn("ERROR on parsing longitude (%q): %v", parts[1], err)
			continue
		}

		res[i].Location1 = domain.Point{
			Lat:  lat,
			Long: long,
		}
	}

	return res, nil
}

// region Orders

func (s *Services) CreateLikiSale(ctx context.Context, req *domain.OnlineOrderRequest) (int, error) {
	// idempotency: bir xil external_id qayta kelsa birinchi buyurtma qaytariladi
	if saleNumber, found, err := s.getLikiSaleNumberByExternalId(ctx, req.ExternalId); err != nil {
		return 0, err
	} else if found {
		return saleNumber, nil
	}

	saleId := uuid.New().String()

	// mahsulot qoldig'ini tekshirish va cart_items yig'ish
	cartItems, err := s.GetOrCheckOnlineCartItems(ctx, req.ShopId, req.Products, saleId)
	if err != nil {
		return 0, err
	}

	// kassir markirovkani skaner qilishi uchun is_marking va barcode
	if err = s.fillLikiCartItemMarkings(ctx, cartItems); err != nil {
		return 0, err
	}

	customer, err := s.GetOrCreateCustomerByPhone(ctx, &req.ClientInfo)
	if err != nil {
		return 0, err
	}

	res, err := s.CreateOnlineSale(ctx, &domain.OnlineSaleCreate{
		Id:            saleId,
		StoreId:       req.ShopId,
		CustomerId:    customer.Id,
		VendorOrderId: req.ExternalId,
		ServiceType:   likiSaleServiceType,
		ClientComment: req.ClientComment,
		Items:         cartItems,
	})
	if err != nil {
		return 0, err
	}

	go s.NotifyOnlineOrder(req.ShopId, res.SaleNumber)

	return res.SaleNumber, nil
}

// likiSaleServiceType — Liki buyurtmasi sales.service_type da shu qiymat bilan yoziladi.
// Hozircha alohida liki to'lov turi va sales ustuni yo'q, shuning uchun Noor bilan bir xil.
const likiSaleServiceType = constants.ServiceTypeNoor

// buyurtmada skaner qilingan barcode yo'q, shuning uchun mahsulot barcode'i olinadi
// (avval markirovkalisi); FetchCartItems is_marking ni cart_items.barcode orqali o'qiydi
func (s *Services) fillLikiCartItemMarkings(ctx context.Context, items []domain.CartItemOnlineRequest) error {
	if len(items) == 0 {
		return nil
	}

	spIds := make([]string, 0, len(items))
	for i := range items {
		spIds = append(spIds, items[i].StoreProductId)
	}

	var rows []struct {
		Id        string  `gorm:"column:id"`
		IsMarking bool    `gorm:"column:is_marking"`
		Barcode   *string `gorm:"column:barcode"`
	}
	err := s.db.WithContext(ctx).Raw(`
	SELECT
		sp.id,
		COALESCE(pb.is_marking, sp.is_marking, false) AS is_marking,
		COALESCE(pb.barcode, NULLIF(sp.barcode, ''), NULLIF(p.barcode, '')) AS barcode
	FROM store_products sp
	JOIN products p ON p.id = sp.product_id
	LEFT JOIN LATERAL (
		SELECT barcode, is_marking
		FROM product_barcodes
		WHERE product_id = p.id
		AND barcode IS NOT NULL AND barcode <> ''
		ORDER BY is_marking DESC, created_at DESC
		LIMIT 1
	) pb ON true
	WHERE sp.id IN (?)`, spIds).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("could not get liki cart_item markings: %v", err)
		return domain.InternalServerError
	}

	bySp := make(map[string]int, len(rows))
	for i := range rows {
		bySp[rows[i].Id] = i
	}
	for i := range items {
		if j, ok := bySp[items[i].StoreProductId]; ok {
			items[i].IsMarking = rows[j].IsMarking
			items[i].Barcode = rows[j].Barcode
		}
	}

	return nil
}

// external_id bo'yicha buyurtmaning sale_number'i (sales.vendor_order_id da saqlanadi)
func (s *Services) getLikiSaleNumberByExternalId(ctx context.Context, externalId string) (int, bool, error) {
	if externalId == "" {
		return 0, false, nil
	}

	var saleNumbers []int
	err := s.db.WithContext(ctx).
		Table("sales").
		Where("vendor_order_id = ? AND service_type = ?", externalId, likiSaleServiceType).
		Order("created_at").
		Limit(1).
		Pluck("sale_number", &saleNumbers).Error
	if err != nil {
		s.log.Errorf("could not get liki sale by external_id %s: %v", externalId, err)
		return 0, false, domain.InternalServerError
	}
	if len(saleNumbers) == 0 {
		return 0, false, nil
	}

	return saleNumbers[0], true, nil
}

// CancelLikiOrder — Liki mijozi buyurtmani bekor qiladi
func (s *Services) CancelLikiOrder(ctx context.Context, saleNumber int64) error {
	var storeId string
	err := s.db.WithContext(ctx).
		Raw("UPDATE sales SET online_status = ? WHERE sale_number = ? RETURNING store_id",
			constants.SaleOnlineStageCanceled, saleNumber).Scan(&storeId).Error
	if err != nil {
		s.log.Errorf("could not update sale for canceling liki order: %v", err)
		return domain.InternalServerError
	}

	s.NotifyOnlineOrderCancel(storeId, int(saleNumber))

	return nil
}

