package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"

	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// region Create

func (s *Services) CreateProduct(ctx context.Context, req *domain.ProductRequest) (*domain.Product, error) {
	// begin transaction
	tx := s.db.Begin()
	// Ensure the transaction is rolled back if any error occurs
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
		}
	}()

	var res domain.Product

	req.Id = uuid.New().String()
	req.Photos = utils.StringArray(req.Photos)
	req.Status = constants.ProductStatusActive
	req.MaterialCode = utils.GenerateMaterialCode()

	query := `
	INSERT INTO products (
		id, 
		material_code,
		shelf_id,
		unit_type_id,
		producer_id,
		name,
		barcode,
		photos,
		unit_per_pack,
		description,
		status
	)
	`
	err := tx.WithContext(ctx).
		Raw(query,
			req.Id,
			req.MaterialCode,
			req.ShelfId,
			req.UnitTypeId,
			req.ProducerId,
			req.Name,
			req.Barcode,
			req.Photos,
			req.UnitPerPack,
			req.Description,
			req.Status,
		).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create new product: %v", err)
		return nil, domain.InternalServerError
	}

	// check category length
	if len(req.CategoryIds) > 0 {
		var categoryProduct = make([]domain.CategoryProduct, len(req.CategoryIds))
		for i := range req.CategoryIds {
			categoryProduct[i].ProductId = req.Id
			categoryProduct[i].CategoryId = req.CategoryIds[i]
			categoryProduct[i].IsOpen = true
		}
		// create category products
		err = tx.WithContext(ctx).Create(&categoryProduct).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not create category: %v", err)
			return nil, domain.InternalServerError
		}
	}
	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// create new producer
func (s *Services) CreateProducer(ctx context.Context, tx *gorm.DB, code string) (*domain.Producer, error) {
	var producer domain.Producer
	query := `INSERT INTO producers (code) VALUES (?) RETURNING *`
	err := tx.WithContext(ctx).WithContext(ctx).Raw(query, code).Scan(&producer).Error
	if err != nil {
		s.log.Errorf("could not create new producer: %v", err)
		return nil, domain.InternalServerError
	}
	return &producer, nil
}

// Create
func (s *Services) CreateProductPhotoAlert(req *domain.ProductPhotoAlertCreate) error {
	alert := domain.CreateProductPhotoAlert{
		ProductID: req.ProductID,
		Category:  req.Category,
		Reason:    req.Reason,
		CreatedBy: &req.CreatedBy,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.db.Table("product_photo_alerts").Create(&alert).Error
}

// region Get

func (s *Services) GetProductById(ctx context.Context, productId string, storeId string) (*domain.Product, error) {
	var tmpProduct struct {
		Id           string            `gorm:"id"`
		MaterialCode int               `gorm:"material_code"`
		Name         string            `gorm:"name"`
		Barcode      string            `gorm:"barcode"`
		Photos       utils.StringArray `gorm:"type:text[]"`
		Description  string            `gorm:"description"`
		UnitPerPack  int               `gorm:"unit_per_pack"`
		Status       string            `gorm:"status"`
		IsMarking    bool              `gorm:"is_marking"`
		IsActive     bool              `gorm:"is_active"`
		CreatedAt    *time.Time        `gorm:"created_at"`
		UpdatedAt    *time.Time        `gorm:"updated_at"`

		SupplyPrice     float64 `gorm:"supply_price"`
		RetailPrice     float64 `gorm:"retail_price"`
		RetailUnitPrice float64 `gorm:"retail_unit_price"`
		Quantity        float64 `gorm:"quantity"`
		Vat             float64 `gorm:"vat"`
		Markup          float64 `gorm:"markup"`
		VatPrice        float64 `gorm:"vat_price"`
		MarkupPrice     float64 `gorm:"markup_price"`
		Sum             float64 `gorm:"sum"`
		Manufacturer    string  `gorm:"manufacturer"`
		ExpireDate      string  `gorm:"expire_date"`
		BonusPercent    float64 `gorm:"bonus_percent"`
		BonusAmount     float64 `gorm:"bonus_amount"`
		UnitTypeId      string  `gorm:"unit_type_id"`
		UnitName        string  `gorm:"unit_name"`
		UnitShortName   string  `gorm:"unit_short_name"`
		ShelfId         string  `gorm:"shelf_id"`
		ShelfName       string  `gorm:"shelf_name"`
		ProducerId      string  `gorm:"producer_id"`
		ProducerName    string  `gorm:"producer_name"`
	}

	// dynamic JOIN with subquery to get latest store_product
	rawJoin := `
		LEFT JOIN store_products sp ON sp.id = (
			SELECT id FROM store_products
			WHERE product_id = p.id
	`
	if storeId != "" {
		rawJoin += " AND store_id = ?"
	}

	rawJoin += `
			ORDER BY created_at DESC
			LIMIT 1
		)
	`

	qb := s.db.WithContext(ctx).
		Select(
			"p.id",
			"p.material_code",
			"p.name",
			"p.photos",
			"p.barcode",
			"p.description",
			"p.unit_per_pack",
			"p.status",
			"p.is_marking",
			"p.created_at",
			"p.updated_at",

			"sp.supply_price",
			"sp.retail_price",
			"ROUND(sp.retail_price / p.unit_per_pack, 2) AS retail_unit_price",
			"sp.vat_price",
			"sp.vat",

			"ut.id AS unit_type_id",
			"ut.unit_name AS unit_name",
			"ut.short_name AS unit_short_name",

			"pr.id AS producer_id",
			"pr.name AS producer_name",

			"sh.id AS shelf_id",
			"sh.name AS shelf_name",

			"pb.bonus_amount",
		).Table("products p").
		Joins("LEFT JOIN unit_types ut ON ut.id = p.unit_type_id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Joins("LEFT JOIN product_bonuses pb ON pb.product_id = p.id").
		Joins("LEFT JOIN shelves sh ON p.shelf_id = p.shelf_id").
		Joins(rawJoin, storeId)

	err := qb.Take(&tmpProduct, "p.id = ?", productId).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get product by id: %v", err)
		return nil, domain.InternalServerError
	}

	res := domain.Product{
		Id:           tmpProduct.Id,
		MaterialCode: tmpProduct.MaterialCode,
		Name:         tmpProduct.Name,
		Barcode:      tmpProduct.Barcode,
		Photos:       tmpProduct.Photos,
		Description:  tmpProduct.Description,
		UnitPerPack:  tmpProduct.UnitPerPack,
		Status:       tmpProduct.Status,
		IsActive:     tmpProduct.IsActive,
		IsMarking:    tmpProduct.IsMarking,
		CreatedAt:    tmpProduct.CreatedAt,
		UpdatedAt:    tmpProduct.UpdatedAt,
		BonusAmount:  tmpProduct.BonusAmount,

		SupplyPrice:     tmpProduct.SupplyPrice,
		RetailPrice:     tmpProduct.RetailPrice,
		RetailUnitPrice: tmpProduct.RetailUnitPrice,
		VatPrice:        tmpProduct.VatPrice,
		Vat:             tmpProduct.Vat,

		UnitTypeID: tmpProduct.UnitTypeId,
		UnitName:   tmpProduct.UnitName,
		UnitType: domain.NewNullStruct(domain.UnitType{
			Id:        tmpProduct.UnitTypeId,
			UnitName:  tmpProduct.UnitName,
			ShortName: tmpProduct.UnitShortName,
		}, tmpProduct.UnitTypeId != ""),

		ProducerID: tmpProduct.ProducerId,
		Producer: domain.NewNullStruct(domain.Producer{
			Id:   &tmpProduct.ProducerId,
			Name: tmpProduct.ProducerName,
		}, tmpProduct.ProducerId != ""),
		ShelfID: tmpProduct.ShelfId,
		Shelf: domain.NewNullStruct(domain.Shelf{
			Id:   tmpProduct.ShelfId,
			Name: tmpProduct.ShelfName,
		}, tmpProduct.ShelfId != ""),
	}
	res.Categories = []domain.Category{}
	res.Markings = []string{}

	return &res, nil
}

// get products get list
func (s *Services) GetProducts(ctx context.Context, params *domain.ProductQueryParam, user *domain.EmployeeClaims) ([]domain.ProductData, int64, error) {
	var (
		res        []domain.ProductData
		totalCount int64
	)

	// Pre-aggregate store_products
	storeJoin := `
	LEFT JOIN (
		SELECT 
			product_id,
			SUM(unit_quantity) as total_quantity,
			MIN(expire_date) FILTER (WHERE unit_quantity > 0) as min_expire_date,
			MAX(supply_price) AS supply_price,
			MAX(retail_price) AS retail_price
		FROM store_products`

	if params.StoreId != "" {
		storeJoin += fmt.Sprintf(" WHERE store_id = '%s'", params.StoreId)
	}

	storeJoin += ` GROUP BY product_id
	) sp_agg ON p.id = sp_agg.product_id`

	qb := s.db.WithContext(ctx).
		Table("products p").
		Joins(storeJoin).
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Joins("LEFT JOIN categories c ON p.category_id = c.id").
		Joins("LEFT JOIN product_barcodes pb ON p.id = pb.product_id AND pb.status = ?", constants.GeneralStatusCompleted).
		Group("p.id, pr.id, c.id, sp_agg.total_quantity, sp_agg.min_expire_date, sp_agg.supply_price, sp_agg.retail_price")

	if params.ProducerId != "" {
		qb = qb.Where("p.producer_id = ?", params.ProducerId)
	}
	if params.NoBarcode {
		qb = qb.Where("p.barcode IS NULL OR p.barcode = ''")
	}
	if params.SearchField != "" {
		search := fmt.Sprintf("%%%s%%", params.SearchField)
		if utils.DefineProductSearchQuery(params.SearchField) == "barcode" {
			qb = qb.Where("pb.barcode LIKE ?", search)
		} else {
			qb = qb.Where("p.name ILIKE ?", search)
		}
	}
	now := time.Now().Add(constants.DateTimeTashkent)

	if params.Status != "" {
		switch params.Status {
		case "active", "inactive":
			qb = qb.Where("p.status = ?", params.Status).Having("sp_agg.total_quantity > 0")
		case "low-stock":
			qb = qb.Having("sp_agg.total_quantity/p.unit_per_pack < 3").Having("sp_agg.total_quantity > 0")
		case "zero-stock":
			qb = qb.Having("sp_agg.total_quantity = 0")
		case "expired":
			qb = qb.Where("sp_agg.min_expire_date < ?", now).Having("sp_agg.total_quantity > 0")
		case "imminent":
			qb = qb.Where("sp_agg.min_expire_date BETWEEN ? AND ?", now, now.AddDate(0, 3, 0)).Having("sp_agg.total_quantity > 0")
		}
	}
	if params.CategoryId != "" {
		qb = qb.Where("p.category_id = ?", params.CategoryId)
	}

	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count products: %v", err)
		return nil, 0, domain.InternalServerError
	}

	switch params.Order {
	case "+name":
		qb = qb.Order("p.name")
	case "-name":
		qb = qb.Order("p.name DESC")
	case "+expire_date":
		qb = qb.Order("sp_agg.min_expire_date")
	case "-expire_date":
		qb = qb.Order("sp_agg.min_expire_date DESC")
	default:
		qb = qb.Order("p.updated_at DESC")
	}

	err := qb.Select(
		"p.id",
		"p.material_code",
		"p.name",
		"p.description",
		"p.barcode",
		"p.unit_per_pack",
		"p.photos",
		"p.status",
		"p.is_marking",
		"p.mxik",
		"p.unit_code",
		"p.unit_label",
		"p.created_at",
		"p.updated_at",

		"pr.name AS manufacturer",
		"ARRAY_AGG(pb.barcode) FILTER (WHERE pb.barcode IS NOT NULL) AS barcodes",
		"c.name AS category_name",

		"COALESCE(sp_agg.total_quantity, 0) AS unit_quantity",
		"sp_agg.min_expire_date AS expire_date",
		"DATE_PART('day', sp_agg.min_expire_date::timestamp - NOW()) AS expire_day",
		"sp_agg.supply_price",
		"sp_agg.retail_price",
	).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error

	if err != nil {
		s.log.Errorf("could not get products list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	for i := range res {
		if res[i].UnitQuantity%res[i].UnitPerPack > 0 {
			res[i].Units = fmt.Sprintf("%d (%d/%d)",
				res[i].UnitQuantity/res[i].UnitPerPack,
				res[i].UnitQuantity%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Units = fmt.Sprintf("%d", res[i].UnitQuantity/res[i].UnitPerPack)
		}
	}

	// check len and take empty array
	if len(res) == 0 {
		res = []domain.ProductData{}
	}

	return res, totalCount, nil
}

func (s *Services) GetProductsByStores(ctx context.Context, params *domain.ProductQueryParam) ([]domain.ProductData, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"s.id AS store_id",
			"s.name AS store_name",

			"p.id",
			"p.material_code",
			"p.name",
			"p.barcode",
			"p.unit_per_pack",

			"SUM(sp.unit_quantity) AS unit_quantity",
			"MAX(sp.supply_price) AS supply_price",
			"MAX(sp.retail_price) AS retail_price",
			"MIN(sp.expire_date) AS expire_date",
		).
		Table("stores s").
		Joins("JOIN store_products sp ON s.id = sp.store_id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Group("s.id, s.name, p.id, p.name").
		Order("s.name, p.name")
	var res []domain.ProductData
	err := qb.Limit(params.Limit).Offset(params.Offset).Debug().Find(&res).Error

	if err != nil {
		s.log.Errorf("could not get products list: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetProductStats(ctx context.Context, params *domain.ProductQueryParam) (domain.ProductStats, error) {

	now := time.Now().Add(constants.DateTimeTashkent)
	threeMonthsLater := now.AddDate(0, 3, 0)

	// Birinchi subquery - har bir mahsulot uchun aggregated ma'lumotlar
	subQuery := s.db.WithContext(ctx).
		Table("products p").
		Select(
			"p.id",
			"p.status",
			"p.unit_per_pack",
			"COALESCE(SUM(sp.unit_quantity), 0) AS total_unit_quantity",
			"MIN(sp.expire_date) AS min_expire_date",
			"SUM(sp.unit_quantity * (sp.retail_price/p.unit_per_pack)) AS total_amount",
		).
		Joins("LEFT JOIN store_products sp ON p.id = sp.product_id").
		Group("p.id")

	// Filtrlarni subquery ga qo'shamiz
	if params.StoreId != "" {
		subQuery = subQuery.Where("sp.store_id = ?", params.StoreId)
	}

	if params.CompanyId != "" {
		subQuery = subQuery.Joins("LEFT JOIN stores st ON sp.store_id = st.id").
			Where("st.company_id = ?", params.CompanyId)
	}

	if params.ProducerId != "" {
		subQuery = subQuery.Where("p.producer_id = ?", params.ProducerId)
	}
	if params.NoBarcode {
		subQuery = subQuery.Where("p.barcode IS NULL OR p.barcode = ''")
	}
	if params.SearchField != "" {
		search := fmt.Sprintf("%%%s%%", params.SearchField)
		if utils.DefineProductSearchQuery(params.SearchField) == "barcode" {
			subQuery = subQuery.
				Joins("LEFT JOIN product_barcodes pb ON p.id = pb.product_id AND pb.status = ?", constants.GeneralStatusCompleted).
				Where("pb.barcode LIKE ?", search)
		} else {
			subQuery = subQuery.Where("p.name ILIKE ?", search)
		}
	}

	// Raw SQL yordamida to'g'ridan-to'g'ri query
	var res domain.ProductStats
	query := `
		SELECT 
			COUNT(*) AS total_count,
			ROUND(SUM(total_unit_quantity / unit_per_pack), 2) AS total_quantity,
			ROUND(SUM(total_amount), 2) AS total_stock_amount,
			COUNT(*) FILTER (WHERE (total_unit_quantity / unit_per_pack) < 3 AND total_unit_quantity > 0) AS low_stock_count,
			COUNT(*) FILTER (WHERE total_unit_quantity = 0) AS zero_stock_count,
			COUNT(*) FILTER (WHERE status = ? AND total_unit_quantity > 0) AS active_count,
			COUNT(*) FILTER (WHERE status = ?) AS inactive_count,
			COUNT(*) FILTER (WHERE min_expire_date BETWEEN ? AND ? AND total_unit_quantity > 0) AS imminent_count,
			COUNT(*) FILTER (WHERE min_expire_date < ? AND total_unit_quantity > 0) AS expired_count
		FROM (?) as products_agg
	`

	err := s.db.WithContext(ctx).
		Raw(query,
			constants.GeneralStatusActive,
			constants.GeneralStatusInactive,
			now,
			threeMonthsLater,
			now,
			subQuery,
		).Scan(&res).Error

	if err != nil {
		s.log.Errorf("could not get product stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) GetProductsForSearch(ctx context.Context, params *domain.StoreProductQueryParam) ([]domain.StoreProductResponse, error) {

	// Qidiruv tipini aniqlash
	searchType := utils.DefineProductSearchQuery(params.Search)

	// Agar nom bo'yicha qidiruv bo'lsa, translitdan o'tkazamiz
	searchTerms := []string{params.Search}
	if params.Search != "" && searchType == "name/category" {
		transliterated := utils.Translit(params.Search)
		// Agar translit qilingan qiymat asl qiymatdan farq qilsa, ikkalasini ham qo'shamiz
		if transliterated != params.Search {
			searchTerms = append(searchTerms, transliterated)
		}
	}

	// Base select fields
	selectFields := []string{
		"sp.id",
		"sp.product_id",
		"sp.store_id",
		"sp.unit_quantity/p.unit_per_pack AS pack_quantity",
		"sp.unit_quantity % p.unit_per_pack AS unit_quantity",
		"sp.unit_quantity AS u_quantity",
		"sp.small_quantity",
		"sp.retail_price",
		"sp.expire_date",
		"DATE_PART('day', sp.expire_date::timestamp - NOW()) AS expire_day",
		"sp.created_at",
		"sp.updated_at",

		"p.name",
		"p.barcode",
		"p.unit_per_pack",

		"pr.name AS producer_name",
		"pb.bonus_amount",
	}

	// Similarity score faqat nom bo'yicha qidiruvda qo'shiladi
	if params.Search != "" && searchType == "name/category" {
		// Har ikkala search term uchun ham similarity hisoblaymiz va maksimumini olamiz
		similarityParts := make([]string, len(searchTerms))
		for i, term := range searchTerms {
			similarityParts[i] = fmt.Sprintf("similarity(p.name, '%s')",
				strings.ReplaceAll(term, "'", "''")) // SQL injection oldini olish
		}
		selectFields = append(selectFields,
			fmt.Sprintf("GREATEST(%s) AS similarity_score", strings.Join(similarityParts, ", ")))
	} else {
		selectFields = append(selectFields, "NULL AS similarity_score")
	}

	qb := s.db.WithContext(ctx).
		Select(strings.Join(selectFields, ", ")).
		Table("store_products sp").
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Joins("LEFT JOIN product_bonuses pb ON pb.product_id = p.id").
		Where("sp.store_id = ? AND sp.unit_quantity > 0", params.StoreId)

	if params.Search != "" {
		switch searchType {
		case "barcode":
			qb = qb.Where(`
				EXISTS (
					SELECT 1
					FROM product_barcodes pbr
					WHERE pbr.product_id = p.id
					AND pbr.barcode LIKE ?
				)`, "%"+params.Search+"%").
				Order("sp.expire_date ASC")

		case "marking":
			qb = qb.Joins("LEFT JOIN product_markings pm ON pm.import_detail_id = sp.import_detail_id").
				Where("pm.marking = ?", params.Search).
				Order("sp.expire_date ASC")

		default: // name/category
			conditions := make([]string, 0, len(searchTerms)*2)
			args := make([]interface{}, 0, len(searchTerms)*2)

			for _, term := range searchTerms {
				conditions = append(conditions, "p.name ILIKE ?")
				args = append(args, "%"+term+"%")
			}

			whereClause := strings.Join(conditions, " OR ")
			qb = qb.Where(whereClause, args...).
				Order("similarity_score DESC")
		}
	}

	var res []domain.StoreProductResponse
	err := qb.
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error

	if err != nil {
		s.log.Errorf("could not search store_products: %v", err)
		return nil, domain.InternalServerError
	}

	// quantity format
	for i := range res {
		if res[i].UQuantity%res[i].UnitPerPack > 0 {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)",
				res[i].UQuantity/res[i].UnitPerPack,
				res[i].UQuantity%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].UQuantity/res[i].UnitPerPack)
		}
	}

	return res, nil
}

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

func (s *Services) GetStoreProductByIdAndStoreId(ctx context.Context, tx *gorm.DB, id string, storeId string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct

	err := tx.WithContext(ctx).
		Select(
			"sp.id",
			"sp.product_id",
			"sp.store_id",
			"sp.pack_quantity",
			"sp.unit_quantity",
			"sp.retail_price",
			"sp.supply_price",
			"sp.vat",
			"sp.is_marking",
			"sp.expire_date",
			"p.unit_per_pack",
		).
		Table("store_products sp").
		Joins("JOIN products p ON sp.product_id = p.id").
		Where("sp.store_id = ?", storeId).
		Where("sp.id = ?", id).
		First(&storeProduct).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Error("could not get store_product by id: %v", err)
		return nil, domain.InternalServerError
	}

	return &storeProduct, nil
}

func (s *Services) GetStoreProductById(ctx context.Context, tx *gorm.DB, id string) (*domain.StoreProduct, error) {
	var storeProduct domain.StoreProduct
	err := tx.WithContext(ctx).
		Raw(`
		SELECT 
			sp.id,
			sp.product_id,
			sp.store_id,
			sp.unit_quantity,
			sp.retail_price,
			sp.supply_price,
			sp.expire_date,
			sp.vat_price,
			sp.created_at,
			sp.updated_at,
			pb.bonus_amount AS bonus_amount, 
			p.unit_per_pack AS unit_per_pack
		FROM store_products sp 
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN product_bonuses pb ON pb.product_id = p.id
		WHERE sp.id = ?`, id).
		Scan(&storeProduct).Error

	if err != nil {
		s.log.Errorf("could not get store_product by id error: %v", err)
		return nil, domain.InternalServerError
	}

	return &storeProduct, nil
}

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

func (s *Services) GetProductIKPUByMxik(ctx context.Context, mxik string) (*domain.ProductMeasurement, error) {
	var measurement domain.ProductMeasurement
	err := s.db.First(&measurement, "mxik_code = ?", mxik).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &measurement, nil
}

func (s *Services) GetProducerByCode(ctx context.Context, tx *gorm.DB, code string) (*domain.Producer, error) {
	var producer domain.Producer
	err := tx.WithContext(ctx).Raw(`SELECT id, name, code, created_at, updated_at FROM producers WHERE code = ?`, code).Scan(&producer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			producerData, err := s.CreateProducer(ctx, tx, code)
			if err != nil {
				return nil, err
			}
			producer = *producerData
		}
		s.log.Errorf("could not get producer by code: %v", err)
		return nil, domain.InternalServerError
	}
	return &producer, nil
}

func (s *Services) GetStoreProductsByProductId(ctx context.Context, params *domain.ProductQueryParam, user *domain.EmployeeClaims) ([]domain.StoreProduct, int64, error) {
	var (
		res        []domain.StoreProduct
		totalCount int64
	)

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	// build query
	query := s.db.
		Select(
			"sp.id",
			"sp.store_id",
			"sp.product_id",
			"sp.unit_quantity/p.unit_per_pack AS pack_quantity",
			"sp.unit_quantity%p.unit_per_pack AS unit_quantity",
			"sp.unit_quantity AS u_quantity",
			"sp.supply_price",
			"sp.retail_price",
			"CASE WHEN sp.supply_price = 0 OR sp.supply_price IS NULL THEN 0 "+
				"ELSE ROUND(((sp.retail_price - sp.supply_price)*100)/sp.supply_price, 2) END AS markup",
			"sp.expire_date",
			"sp.is_marking",
			"sp.serial_number",
			"sp.created_at",
			"sp.updated_at",
			"sp.vat",
			"sp.vat_price",

			"p.unit_per_pack",
			"p.barcode",

			"u.short_name",
			"st.name AS store_name",
		).
		Table("store_products sp").
		Joins("JOIN products p ON p.id = sp.product_id").
		Joins("JOIN stores st ON sp.store_id = st.id").
		Joins("LEFT JOIN unit_types u ON u.id = p.unit_type_id").
		Where("sp.product_id = ?", params.ProductId)

	if params.StoreId != "" {
		query = query.Where("sp.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		query = query.Where("st.company_id = ?", params.CompanyId)
	}
	// complete query
	err := query.WithContext(ctx).
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("sp.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get store_products by product_id: %v", err)
		return nil, 0, domain.InternalServerError
	}

	for i := range res {
		res[i].Store = domain.NewNullStruct(domain.Store{
			Id:   res[i].StoreId,
			Name: res[i].StoreName,
		}, res[i].StoreId != "")

		if res[i].UQuantity%res[i].UnitPerPack > 0 {
			res[i].Quantity = fmt.Sprintf("%d (%d/%d)",
				res[i].UQuantity/res[i].UnitPerPack,
				res[i].UQuantity%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Quantity = fmt.Sprintf("%d", res[i].UQuantity/res[i].UnitPerPack)
		}
	}
	return res, totalCount, nil
}

func (s *Services) GetNoorProducts(param *domain.NoorQueryParam) ([]domain.NoorProduct, error) {
	var res []domain.NoorProduct
	query := `
	SELECT
		p.id,
		p.name,
		p.photos,
		p.description,
		p.description_uz,
		p.description_ru,
		p.description_kr,
		p.category_id
	FROM
		products p
	LIMIT ? OFFSET ?;
`
	err := s.db.Raw(query, param.Limit, param.Offset).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get products for noor: %v", err)
		return res, err
	}

	return res, nil
}

func (s *Services) GetNoorStoreProducts(params *domain.NoorQueryParam) ([]domain.NoorStoreProduct, error) {
	if _, err := time.Parse(time.RFC3339, params.UpdatedAt); err != nil {
		s.log.Errorf("could not parse updated_at param: %v", err)
		return nil, &domain.Error{
			Code:    domain.InvalidTimeFormatError.Code,
			Message: "",
		}
	}

	var res []domain.NoorStoreProduct
	query := `
	SELECT
		sp.store_id,
		sp.product_id,
		SUM(sp.unit_quantity/(p.unit_per_pack/p.blister_count)) AS quantity,
		ROUND(MIN(sp.retail_price/p.blister_count), 0) AS price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	WHERE sp.unit_quantity/p.blister_count > 0 AND sp.updated_at >= ?
	GROUP BY sp.product_id, sp.store_id
	LIMIT ? OFFSET ?;
	`
	// execute query
	err := s.db.Raw(query, params.UpdatedAt, params.Limit, params.Offset).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store_products for noor: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

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

func (s *Services) GetProductMovements(ctx context.Context, params *domain.ProductQueryParam, user *domain.EmployeeClaims) ([]domain.ImportProductData, int64, error) {
	var (
		res        []domain.ImportProductData
		totalCount int64
		query      string
		args       []any
	)

	// check if employee is not admin or superadmin
	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
		params.CompanyId = user.CompanyId
	}

	baseQuery := `
WITH var_data AS (
SELECT
	p.id AS product_id,
	p.unit_per_pack
FROM products p
WHERE p.id = ?
),
import_data AS (
    SELECT
        im.id, im.public_id, im.entry_type, im.created_at,
        s.name AS store_name,
        SUM(imd.accepted_count) * vd.unit_per_pack AS quantity,
        SUM(imd.accepted_count * imd.retail_price_vat) AS sum,
        COALESCE(im.name, '') AS name,
        vd.unit_per_pack
    FROM imports im
    JOIN stores s ON im.store_id = s.id
    JOIN import_details imd ON im.id = imd.import_id
    JOIN var_data vd ON imd.product_id = vd.product_id
    WHERE im.entry_type = 1 AND im.status = 'completed'
    %s
    GROUP BY im.id, s.id, vd.unit_per_pack
),
inventory_data AS (
    SELECT
        im.id, im.public_id, im.entry_type, im.created_at,
        s.name AS store_name,
        SUM(imd.scanned_count-imd.received_count) AS quantity,
        ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/vd.unit_per_pack)), 2) AS sum,
        im.name AS name,
        vd.unit_per_pack
    FROM imports im
    JOIN stores s ON im.store_id = s.id
    JOIN import_details imd ON im.id = imd.import_id
    JOIN var_data vd ON imd.product_id = vd.product_id
    WHERE im.entry_type = 2 AND im.status = 'completed'
    %s
    GROUP BY im.id, s.id, vd.unit_per_pack
),
sales_data AS (
    SELECT
        sa.id, sa.sale_number AS public_id,
        CASE WHEN sa.sale_type = 'SALE' THEN 4 ELSE 7 END AS entry_type,
        sa.completed_at AS created_at,
        st.name AS store_name,
        CASE WHEN sa.sale_type = 'SALE' THEN SUM(ci.unit_quantity) * (-1) ELSE SUM(ci.unit_quantity) END AS quantity,
		CASE WHEN sa.sale_type = 'SALE' THEN sa.total_amount * (-1) ELSE sa.total_amount END as sum,
        sa.sale_type AS name,
        vd.unit_per_pack
    FROM sales sa
    JOIN stores st ON st.id = sa.store_id
    JOIN cart_items ci ON ci.sale_id = sa.id
    JOIN store_products sp ON sp.id = ci.store_product_id
    JOIN var_data vd ON sp.product_id = vd.product_id
    WHERE sa.stage IN (9, 11)
    %s
    GROUP BY sa.id, st.id, vd.unit_per_pack
),
vozvrat_data AS (
    SELECT
        tr.id, tr.public_id::int, 5 AS entry_type, tr.created_at,
        s.name AS store_name,
        SUM(td.accepted_count) * vd.unit_per_pack * (-1) AS quantity,
        SUM((td.accepted_count/vd.unit_per_pack) * td.retail_price) * (-1) AS sum,
        tr.name as name,
        vd.unit_per_pack
    FROM transfer_details td
    JOIN transfers tr ON td.transfer_id = tr.id
    JOIN var_data vd ON td.product_id = vd.product_id
    JOIN stores s ON s.id = tr.from_store_id
    WHERE (tr.status = 'completed' OR tr.status = 'sent-to-1c') AND tr.entry_type = 2
    %s
    GROUP BY tr.id, s.id, vd.unit_per_pack
),
transfer_in_data AS (
    SELECT
        tr.id, tr.public_id::int,
        6 AS entry_type,
        tr.created_at,
        fs.name || ' -> ' || ts.name as store_name,
        SUM(td.accepted_count) * vd.unit_per_pack AS quantity,
        SUM(td.accepted_count * td.retail_price) AS sum,
        tr.name as name,
        vd.unit_per_pack
    FROM transfer_details td
    JOIN transfers tr ON td.transfer_id = tr.id
    JOIN var_data vd ON td.product_id = vd.product_id
    JOIN stores fs ON fs.id = tr.from_store_id
    JOIN stores ts ON ts.id = tr.to_store_id
    WHERE (tr.status = 'completed' OR tr.status = 'sent-to-1c') AND tr.entry_type = 1
    %s
    GROUP BY tr.id, fs.id, ts.id, vd.unit_per_pack
 ),
 transfer_out_data AS (
    SELECT
        tr.id, tr.public_id::int,
        6 AS entry_type,
        tr.created_at,
        fs.name || ' -> ' || ts.name as store_name,
        SUM(td.accepted_count) * vd.unit_per_pack * (-1) AS quantity,
        SUM(td.accepted_count * td.retail_price * (-1)) AS sum,
        tr.name as name,
        vd.unit_per_pack
    FROM transfer_details td
    JOIN transfers tr ON td.transfer_id = tr.id
    JOIN var_data vd ON td.product_id = vd.product_id
    JOIN stores fs ON fs.id = tr.from_store_id
    JOIN stores ts ON ts.id = tr.to_store_id
    WHERE (tr.status = 'completed' OR tr.status = 'sent-to-1c') AND tr.entry_type = 1
    %s
    GROUP BY tr.id, fs.id, ts.id, vd.unit_per_pack
 )
SELECT *, COUNT(*) OVER() AS total_count
FROM (
    SELECT * FROM import_data
    UNION ALL
    SELECT * FROM sales_data
    UNION ALL
    SELECT * FROM inventory_data
    UNION ALL
    SELECT * FROM vozvrat_data
    UNION ALL
    SELECT * FROM transfer_in_data
	UNION ALL
    SELECT * FROM transfer_out_data
) all_data
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
	`

	// dynamic query conditions
	if params.StoreId == "" && params.CompanyId == "" {
		query = fmt.Sprintf(baseQuery, "", "", "", "", "", "")
		args = []any{params.ProducerId, params.Limit, params.Offset}

	} else if params.StoreId != "" && params.CompanyId == "" {
		query = fmt.Sprintf(
			baseQuery,
			"AND im.store_id = ?",
			"AND im.store_id = ?",
			"AND sa.store_id = ?",
			"AND tr.from_store_id = ?",
			"AND tr.to_store_id = ?",
			"AND tr.from_store_id = ?",
		)
		args = []any{
			params.ProducerId,
			params.StoreId,
			params.StoreId,
			params.StoreId,
			params.StoreId,
			params.StoreId,
			params.StoreId, // for transfer_data
			params.Limit, params.Offset,
		}

	} else if params.StoreId == "" && params.CompanyId != "" {
		query = fmt.Sprintf(
			baseQuery,
			"AND s.company_id = ?",
			"AND s.company_id = ?",
			"AND st.company_id = ?",
			"AND s.company_id = ?",
			"AND ts.company_id = ?",
			"AND fs.company_id = ?",
		)
		args = []any{
			params.ProducerId,
			params.CompanyId,
			params.CompanyId,
			params.CompanyId,
			params.CompanyId,
			params.CompanyId,
			params.CompanyId, // for transfer_data
			params.Limit, params.Offset,
		}

	} else { // both storeId and companyId
		query = fmt.Sprintf(
			baseQuery,
			"AND im.store_id = ? AND s.company_id = ?",
			"AND im.store_id = ? AND s.company_id = ?",
			"AND sa.store_id = ? AND st.company_id = ?",
			"AND tr.from_store_id = ? AND s.company_id = ?",
			"AND tr.to_store_id = ? AND ts.company_id = ?",
			"AND tr.from_store_id = ? AND fs.company_id = ?",
		)
		args = []any{
			params.ProducerId,
			params.StoreId, params.CompanyId, // import_data
			params.StoreId, params.CompanyId, // inventory_data
			params.StoreId, params.CompanyId, // sales_data
			params.StoreId, params.CompanyId, // vozvrat_data
			params.StoreId, params.CompanyId,
			params.StoreId, params.CompanyId, // transfer_data
			params.Limit, params.Offset,
		}
	}

	// Execute query
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get product_movements: %v", err)
		return res, totalCount, err
	}

	// Get total count
	if len(res) > 0 {
		totalCount = res[0].TotalCount
	}

	for i := range res {
		if int(res[i].Quantity)%res[i].UnitPerPack > 0 {
			res[i].Count = fmt.Sprintf("%d (%d/%d)",
				int(res[i].Quantity)/res[i].UnitPerPack,
				int(res[i].Quantity)%res[i].UnitPerPack,
				res[i].UnitPerPack)
		} else {
			res[i].Count = fmt.Sprintf("%d", int(res[i].Quantity)/res[i].UnitPerPack)
		}
	}

	return res, totalCount, nil
}

func (s *Services) ProductListForArzon(ctx context.Context, storeId string) ([]domain.ProductArzon, error) {
	var res []domain.ProductArzon
	err := s.db.WithContext(ctx).
		Raw(`
	SELECT 
		p.id, p.name, 
		COALESCE(pr.name, '') AS producer_name, 
		MIN(sp.retail_price) AS retail_price
	FROM store_products sp
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	WHERE sp.store_id = ? AND sp.unit_quantity > 0
	GROUP BY p.id, pr.id;
	`, storeId).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get products for arzon_apteka: %v", err)
		return res, domain.InternalServerError
	}
	return res, nil
}

func (s *Services) GetProductIdByCode(ctx context.Context, code int64) (string, error) {
	var id string
	err := s.db.WithContext(ctx).Raw(`SELECT id FROM products WHERE material_code = ?`, code).Scan(&id).Error
	if err != nil {
		s.log.Errorf("could not get product_id by material_code: %v", err)
		return id, domain.InternalServerError
	}
	return id, nil
}

func (s *Services) GetMinMaxProducts(ctx context.Context, params *domain.ProductQueryParam) ([]domain.MinMaxProduct, int64, error) {
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
	if params.StoreId != "" {
		filter += " AND spt.store_id = ? "
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		filter += " AND s.company_id = ?"
		args = append(args, params.CompanyId)
	}

	if params.SearchField != "" {
		filter += " AND p.name ILIKE ? "
		args = append(args, "%"+params.SearchField+"%")
	}
	// collect total query
	totalCountQuery += filter
	err := s.db.Raw(totalCountQuery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get min_max_products total_count: %v", err)
		return res, totalCount, domain.InternalServerError
	}
	// collect query
	query += filter + order + " LIMIT ? OFFSET ?" // add limit, offset for pagination
	args = append(args, params.Limit, params.Offset)
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get min_max_products: %v", err)
		return res, totalCount, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) ListExcludedProducts(param *domain.ProductQueryParam) ([]domain.ExcludedProductResponse, int64, error) {
	var (
		res        []domain.ExcludedProductResponse
		totalCount int64
		args       []any
		countArgs  []any
		filter     = "WHERE 1=1"
	)

	// filter by store if given
	if param.StoreId != "" {
		filter += " AND ep.store_id = ?"
		args = append(args, param.StoreId)
		countArgs = append(countArgs, param.StoreId)
	}
	if param.CompanyId != "" {
		filter += " AND s.company_id = ?"
		args = append(args, param.CompanyId)
		countArgs = append(countArgs, param.CompanyId)
	}

	// filter by product name
	if param.SearchField != "" {
		search := "%" + param.SearchField + "%"
		filter += " AND p.name ILIKE ?"
		args = append(args, search)
		countArgs = append(countArgs, search)
	}

	// set default pagination
	if param.Limit == 0 {
		param.Limit = 10
	}
	if param.Offset < 0 {
		param.Offset = 0
	}

	// main query
	query := `
		SELECT
			ep.id,
			p.id AS product_id,
			p.name AS product_name,
			ep.store_id,
			COALESCE(s.name, 'Global') AS store_name,
			e.full_name AS created_by,
			ep.created_at
		FROM excluded_products ep
		JOIN products p ON p.id = ep.product_id
		LEFT JOIN stores s ON ep.store_id = s.id
		LEFT JOIN employees e ON ep.created_by = e.id
	` + filter + `
		ORDER BY ep.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, param.Limit, param.Offset)

	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("Failed to list excluded products: %v", err)
		return nil, 0, err
	}

	// count query
	countQuery := `
		SELECT COUNT(*)
		FROM excluded_products ep
		JOIN products p ON p.id = ep.product_id
		LEFT JOIN stores s ON ep.store_id = s.id
	` + filter

	if err := s.db.Raw(countQuery, countArgs...).Scan(&totalCount).Error; err != nil {
		s.log.Error("Failed to count excluded products: %v", err)
		return nil, 0, err
	}

	return res, totalCount, nil
}

func (s *Services) GetProductsByImport(ctx context.Context, params *domain.ProductQueryParam) ([]domain.ProductByImport, int64, error) {
	qb := s.db.WithContext(ctx).
		Table("store_products sp").
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("JOIN stores st ON sp.store_id = st.id").
		Joins("LEFT JOIN import_details imd ON sp.import_detail_id = imd.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id")

	// filter by store_id
	if params.StoreId != "" {
		qb = qb.Where("sp.store_id = ?", params.StoreId)
	}

	if params.CompanyId != "" {
		qb = qb.Where("sp.company_id = ?", params.CompanyId)
	}

	if params.SearchField != "" {
		search := fmt.Sprintf("%%%s%%", params.SearchField)
		if utils.DefineProductSearchQuery(params.SearchField) == "barcode" {
			qb = qb.Where("p.barcode LIKE ?", search)
		} else {
			qb = qb.Where("p.name ILIKE ?", search)
		}
	}

	if params.NoBarcode {
		qb = qb.Where("(p.barcode IS NULL OR p.barcode = '')")
	}

	if params.ProducerId != "" {
		qb = qb.Where("p.producer_id = ?", params.ProducerId)
	}

	if params.ImportId != "" {
		qb = qb.Where("imd.import_id = ?", params.ImportId)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get total_count in get_products_by_import: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.ProductByImport
	err := qb.
		Select(
			"p.material_code",
			"p.name",
			"p.barcode",
			"p.unit_per_pack",

			"sp.id",
			"sp.product_id",
			"sp.unit_quantity / p.unit_per_pack AS quantity",
			"sp.unit_quantity % p.unit_per_pack AS unit_quantity",
			"sp.unit_quantity AS u_quantity",
			"sp.is_marking",
			"sp.is_checking",
			"sp.serial_number",
			"sp.expire_date",
			"sp.retail_price",
			"sp.supply_price",
			"sp.mxik",
			"sp.unit_code",
			"sp.unit_label",
			"sp.created_at",
			"sp.updated_at",

			"st.name AS store_name",
			"pr.name AS producer_name",
		).
		Order("sp.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get product by import: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) GetSoldProductsBySaleId(ctx context.Context, saleId string) ([]domain.ProductRes, error) {
	// get products info
	var products []domain.ProductRes
	query := `
	SELECT
		p.id,
		p.name,
		p.barcode,
		p.is_marking,
        p.photos,
		p.mxik AS class_code,
		p.unit_label AS package_name,
        ROUND((sp.vat_price / p.unit_per_pack) * ci.unit_quantity, 2) AS vat,
        sp.id AS store_product_id,
        sp.vat AS vat_percent,
        ci.unit_quantity / p.unit_per_pack AS quantity,
        ci.unit_quantity % p.unit_per_pack as unit_quantity,
		ci.marking_count,
		ci.unit_price AS pack_price,
		ci.total_price,
		ci.discount_amount,
		((ci.discount_price/p.unit_per_pack)*ci.unit_quantity) AS  total_discount,
        ROUND(ci.unit_price / p.unit_per_pack, 2) AS unit_price,
        ROUND((pb.bonus_amount/p.unit_per_pack) * ci.unit_quantity, 2) AS bonus_amount
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN unit_types u ON p.unit_type_id = u.id
	LEFT JOIN product_bonuses pb ON pb.product_id = sp.product_id
	WHERE ci.sale_id = ?
	`
	err := s.db.WithContext(ctx).Raw(query, saleId).Scan(&products).Error
	if err != nil {
		s.log.Errorf("could not get sale products: %v", err)
		return nil, domain.InternalServerError
	}

	return products, nil
}

func (s *Services) ListProductPhotoAlert(ctx context.Context, params *domain.ProductQueryParam) ([]domain.ProductPhotoAlert, int64, error) {
	var (
		alerts     []domain.ProductPhotoAlert
		totalCount int64
	)

	query := s.db.WithContext(ctx).Table("product_photo_alerts").Select("product_photo_alerts.*, p.name, p.photos, p.unit_per_pack, e.full_name as created_by").
		Joins("JOIN products p ON p.id = product_photo_alerts.product_id").
		Joins("JOIN employees e ON e.id = product_photo_alerts.created_by")

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Category != 0 {
		query = query.Where("category = ?", params.Category)
	}
	if params.SearchField != "" {
		query = query.Where("p.name ILIKE ?", "%"+params.SearchField+"%")
	}
	//if param.CompanyID != "" {
	//	// agar products jadvalida company_id bo‘lsa, join qilib filterlash kerak
	//	query = query.Joins("JOIN products p ON p.id = product_photo_alerts.product_id").
	//		Where("p.company_id = ?", param.CompanyID)
	//}

	// count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit).Offset(params.Offset)
	}

	if err := query.Order("created_at DESC").Scan(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, totalCount, nil
}

func (s *Services) GetSingleProductDashboard(ctx context.Context, params *domain.ProductQueryParam) (domain.SingeProductDashoard, error) {
	var (
		res   domain.SingeProductDashoard
		query string
		args  []any
	)

	baseQuery := `
WITH var_data AS (
    SELECT
        p.id AS product_id,
        p.unit_per_pack
    FROM products p
    WHERE p.id = ?
),  
import_data AS (
    SELECT
        (SUM(imd.accepted_count) * vd.unit_per_pack)::INTEGER AS import_count,
        SUM(imd.accepted_count * imd.retail_price_vat) AS import_amount
    FROM imports im
        JOIN stores s ON im.store_id = s.id
        JOIN import_details imd ON im.id = imd.import_id
        JOIN var_data vd ON imd.product_id = vd.product_id
    WHERE im.entry_type = 1 AND im.status = 'completed'
        %s
    GROUP BY vd.unit_per_pack
),
sales_data AS (
    SELECT
        SUM(ci.unit_quantity)::INTEGER * (-1) AS sale_count,
        sum(sa.total_amount * (-1)) AS sale_amount
    FROM sales sa
        JOIN stores st ON st.id = sa.store_id
        JOIN cart_items ci ON ci.sale_id = sa.id
        JOIN store_products sp ON sp.id = ci.store_product_id
        JOIN var_data vd ON sp.product_id = vd.product_id
    WHERE sa.stage IN (9, 11) and sale_type = 'SALE'
        %s
),
return_sales_data AS (
    SELECT
        SUM(ci.unit_quantity)::INTEGER AS return_sale_count,
        sum(sa.total_amount * (-1)) AS return_sale_amount
    FROM sales sa
        JOIN stores st ON st.id = sa.store_id
        JOIN cart_items ci ON ci.sale_id = sa.id
        JOIN store_products sp ON sp.id = ci.store_product_id
        JOIN var_data vd ON sp.product_id = vd.product_id
    WHERE sa.stage IN (9, 11) and sale_type = 'RETURN'
        %s
),
vozvrat_data AS (
    SELECT
        (SUM(td.accepted_count) * vd.unit_per_pack)::INTEGER * (-1) AS return_to_sklad_count,
        ROUND(SUM((td.accepted_count/vd.unit_per_pack) * td.retail_price), 2) * (-1) AS return_to_sklad_amount
    FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN var_data vd ON td.product_id = vd.product_id
        JOIN stores s ON s.id = tr.from_store_id
    WHERE (tr.status = 'completed' OR tr.status = 'sent-to-1c') AND tr.entry_type = 2
        %s
    GROUP BY vd.unit_per_pack
),
transfer_data AS (
    SELECT
        (SUM(td.accepted_count) %s * vd.unit_per_pack)::INTEGER * (-1) AS transfer_out_count,
        SUM(td.accepted_count * td.retail_price * (-1)) %s AS transfer_out_amount,
        (SUM(td.accepted_count) %s * vd.unit_per_pack)::INTEGER AS transfer_in_count,
        SUM(td.accepted_count  * td.retail_price) %s AS transfer_in_amount
    FROM transfer_details td
        JOIN transfers tr ON td.transfer_id = tr.id
        JOIN var_data vd ON td.product_id = vd.product_id
        JOIN stores fs ON fs.id = tr.from_store_id
        JOIN stores ts ON ts.id = tr.to_store_id
    WHERE (tr.status = 'completed' OR tr.status = 'sent-to-1c') AND tr.entry_type = 1
        %s
    GROUP BY vd.unit_per_pack
),
product_quantity as (
    select
        sum(sp.unit_quantity)::INTEGER as unit_quantity
    from
        store_products as sp
    join var_data vd on vd.product_id = sp.product_id
    where 1 = 1 %s
),
imventory_quantity as (
	SELECT
		(SUM(case when imd.scanned_count-imd.received_count > 0 then imd.scanned_count-imd.received_count else 0 end))::INTEGER AS inventory_plus_count,
		(SUM(case when imd.scanned_count-imd.received_count < 0 then imd.scanned_count-imd.received_count else 0 end))::INTEGER AS inventory_minus_count,
		ROUND(SUM(case when imd.scanned_count-imd.received_count > 0 then imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack) else 0 end), 2) AS inventory_plus_amount,
		ROUND(SUM(case when imd.scanned_count-imd.received_count < 0 then imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack) else 0 end), 2) AS inventory_minus_amount
	FROM import_details imd
	JOIN imports im on im.id = imd.import_id
	JOIN var_data p ON imd.product_id = p.product_id
	LEFT JOIN stores s ON s.id = im.store_id
	WHERE im.entry_type = 2 AND im.status = 'completed' %s
)
SELECT
    COALESCE(pq.unit_quantity, 0) AS unit_quantity,
    COALESCE(sd.sale_count, 0) AS sale_count,
    COALESCE(sd.sale_amount, 0) AS sale_amount,
    COALESCE(rsd.return_sale_count, 0) AS return_sale_count,
    COALESCE(rsd.return_sale_amount, 0) AS return_sale_amount,
    COALESCE(id.import_count, 0) AS import_count,
    COALESCE(id.import_amount, 0) AS import_amount,
    COALESCE(vd.return_to_sklad_count, 0) AS return_to_sklad_count,
    COALESCE(vd.return_to_sklad_amount, 0) AS return_to_sklad_amount,
    COALESCE(td.transfer_out_count, 0) AS transfer_out_count,
    COALESCE(td.transfer_out_amount, 0) AS transfer_out_amount,
    COALESCE(td.transfer_in_count, 0) AS transfer_in_count,
    COALESCE(td.transfer_in_amount, 0) AS transfer_in_amount,
	COALESCE(imq.inventory_plus_count, 0) AS inventory_plus_count,
	COALESCE(imq.inventory_minus_count, 0) AS inventory_minus_count,
	COALESCE(imq.inventory_plus_amount, 0) AS inventory_plus_amount,
	COALESCE(imq.inventory_minus_amount, 0) AS inventory_minus_amount	
FROM product_quantity pq
LEFT JOIN import_data id ON true
LEFT JOIN sales_data sd ON true
LEFT JOIN return_sales_data rsd ON true
LEFT JOIN vozvrat_data vd ON true
LEFT JOIN transfer_data td ON true
LEFT JOIN imventory_quantity imq ON true
`

	// dynamic query conditions
	if params.StoreId == "" && params.CompanyId == "" {
		query = fmt.Sprintf(baseQuery, "", "", "", "", "", "", "", "", "", "", "")
		args = []any{params.ProductId}

	} else if params.StoreId != "" && params.CompanyId == "" {
		query = fmt.Sprintf(
			baseQuery,
			"AND im.store_id = ?",
			"AND sa.store_id = ?",
			"AND sa.store_id = ?",
			"AND tr.from_store_id = ?",
			"filter ( where tr.from_store_id = ? )",
			"filter ( where tr.from_store_id = ? )",
			"filter ( where tr.to_store_id = ? )",
			"filter ( where tr.to_store_id = ? )",
			"AND (tr.from_store_id = ? OR tr.to_store_id = ?)",
			"AND store_id = ?",
			"AND s.id = ?",
		)
		args = []any{
			params.ProductId,
			params.StoreId,                 // import_data
			params.StoreId,                 // sales_data
			params.StoreId,                 // return_sales_data
			params.StoreId,                 // vozvrat_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId, params.StoreId, // transfer_data
			params.StoreId, // product_quantity
			params.StoreId, // inventory_quantity
		}

	} else if params.StoreId == "" && params.CompanyId != "" {
		query = fmt.Sprintf(
			baseQuery,
			"AND s.company_id = ?",
			"AND st.company_id = ?",
			"AND st.company_id = ?",
			"AND s.company_id = ?",
			"filter ( where fs.company_id = ?)",
			"filter ( where fs.company_id = ? )",
			"filter ( where ts.company_id = ? )",
			"filter ( where ts.company_id = ? )",
			"AND (fs.company_id = ? OR ts.company_id = ?)",
			"AND company_id = ?",
			"AND s.company_id = ?",
		)
		args = []any{
			params.ProductId,
			params.CompanyId,                   // import_data
			params.CompanyId,                   // sales_data
			params.CompanyId,                   // return_sales_data
			params.CompanyId,                   // vozvrat_data
			params.CompanyId,                   // transfer_data
			params.CompanyId,                   // transfer_data
			params.CompanyId,                   // transfer_data
			params.CompanyId,                   // transfer_data
			params.CompanyId, params.CompanyId, // transfer_data
			params.CompanyId, // product_quantity
			params.CompanyId, // import_quantity
		}

	} else { // both storeId and companyId
		query = fmt.Sprintf(
			baseQuery,
			"AND im.store_id = ? AND s.company_id = ?",
			"AND sa.store_id = ? AND st.company_id = ?",
			"AND sa.store_id = ? AND st.company_id = ?",
			"AND tr.from_store_id = ? AND s.company_id = ?",
			"filter ( where tr.from_store_id = ? )",
			"filter ( where tr.from_store_id = ? )",
			"filter ( where tr.to_store_id = ? )",
			"filter ( where tr.to_store_id = ? )",
			"AND (tr.from_store_id = ? OR tr.to_store_id = ?)",
			"AND store_id = ?",
			"AND s.id = ?", // inventory_quantity
		)
		args = []any{
			params.ProductId,
			params.StoreId, params.CompanyId, // import_data
			params.StoreId, params.CompanyId, // sales_data
			params.StoreId, params.CompanyId, // return_sales_data
			params.StoreId, params.CompanyId, // vozvrat_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId,                 // transfer_data
			params.StoreId, params.StoreId, // transfer_data
			params.StoreId, // product_quantity
			params.StoreId, // inventory_quantity
		}
	}

	// Execute query
	err := s.db.WithContext(ctx).Debug().Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get single product dashboard: %v", err)
		return res, err
	}
	return res, nil
}

// region Update

func (s *Services) UpdateProductIsMarking(req *domain.UpdateIsMarking) error {
	// build query
	query := `UPDATE products SET is_marking = ? WHERE id = ?`
	// complete the update query
	err := s.db.Exec(query, req.IsMarking, req.ProductId).Error
	if err != nil {
		s.log.Errorf("could not update is_marking: %v", err)
		return domain.InternalServerError
	}
	query = `UPDATE store_products SET is_marking = ? WHERE product_id = ?`
	// complete the update query
	err = s.db.Exec(query, req.IsMarking, req.ProductId).Error
	if err != nil {
		s.log.Errorf("could not update is_marking: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) UpdateRetailPrice(ctx context.Context, tx *gorm.DB, id string, newPrice float64) error {
	// update retail price
	err := tx.WithContext(ctx).Exec(`UPDATE store_products SET retail_price = ? WHERE id = ?`, newPrice, id).Error
	if err != nil {
		s.log.Errorf("could not update store_product retail_price: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) UpdateProductQuantity(ctx context.Context, req *domain.OnecUpdateQuantityRequest) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			s.log.Errorf("Panic recovered in UpdateProductQuantity: %v", r)
		}
	}()

	publicId := strings.TrimPrefix(req.Dok.DocumentNumber, "NP-")

	var transferId string
	err := tx.WithContext(ctx).
		Raw(`SELECT id FROM transfers WHERE public_id = ?`,
			publicId).Scan(&transferId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer Id by public_id: %v", err)
		return domain.InternalServerError
	}
	if transferId == "" {
		_ = tx.Rollback()
		s.log.Error("could not get transfer_id by id")
		return domain.InternalServerError
	}

	for _, item := range req.Товары {
		diff := int(item.AcceptedCount - item.GivenCount)
		err = tx.WithContext(ctx).
			Exec(`
			UPDATE store_products
			SET 
				unit_quantity = unit_quantity + (? * p.unit_per_pack)
			FROM products p
			WHERE store_products.product_id = p.id
			  AND store_products.id = ?`,
				diff, item.StoreProductId,
			).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Error("ERROR on updating product quantity: ", err)
			return domain.InternalServerError
		}

		err = tx.WithContext(ctx).
			Exec(`
			UPDATE transfer_details
			SET onec_count = ?, updated_at = NOW()
			WHERE transfer_id = ? AND store_product_id = ?`,
				item.GivenCount, transferId, item.StoreProductId,
			).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update transfer details: %v", err)
			return domain.InternalServerError
		}
	}

	// Update transfer status
	err = tx.WithContext(ctx).
		Exec(`
		UPDATE transfers
		SET 
			status = ?,
			updated_at = NOW()
		WHERE id = ?`,
			constants.GeneralStatusCompleted,
			transferId,
		).Error

	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update transfer status: %v", err)
		return domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction accept vozvrat from onec: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) UpdatePackaging(ctx context.Context, req *domain.UpdatePackagingRequest) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	err := tx.WithContext(ctx).Exec("UPDATE products SET unit_per_pack = ? WHERE unit_per_pack = 1 AND id = ?;",
		req.UnitPerPack, req.ProductId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update product packaging: %v", err)
		return domain.InternalServerError
	}

	// 3. Recalculate unit_quantity in store_products
	err = tx.WithContext(ctx).
		Exec(`
		UPDATE store_products sp
			SET unit_quantity = unit_quantity * ?
		FROM products p
		  WHERE sp.product_id = p.id
			AND sp.product_id = ?;`,
			req.UnitPerPack, req.ProductId,
		).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not recalc store_products.unit_quantity: %v", err)
		return err
	}

	err = tx.WithContext(ctx).Exec(`
	UPDATE cart_items ci
	SET unit_quantity = ci.unit_quantity * ?
	FROM store_products sp
	WHERE ci.store_product_id = sp.id
	AND sp.product_id = ?
		`, req.UnitPerPack, req.ProductId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not recalc cart_items.unit_quantity with unit_per_pack: %v", err)
		return domain.InternalServerError
	}

	// 4. Commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit update packaging transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) IncrementQuantity(tx *gorm.DB, id string, quantity int) error {
	err := tx.Exec(`UPDATE store_products SET unit_quantity = unit_quantity + ? WHERE id = ?`, quantity, id).Error
	if err != nil {
		s.log.Error("could not update store_product quantity: %v", err)
		return err
	}

	return nil
}

func (s *Services) UpdateProductUnitValues(ctx context.Context, req *domain.UpdateBarcodeRequest, user *domain.EmployeeClaims) error {
	if req.Barcode != "" {
		// update barcode
		err := s.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", req.Id).Update("barcode", req.Barcode).Error
		if err != nil {
			s.log.Errorf("could not update product barcode: %v", err)
			return domain.InternalServerError
		}
		// update barcode store_products
		err = s.db.WithContext(ctx).Model(&domain.StoreProduct{}).Where("product_id = ?", req.Id).Update("barcode", req.Barcode).Error
		if err != nil {
			s.log.Errorf("could not update store_products barcode: %v", err)
			return domain.InternalServerError
		}
		// insert into product_barcodes
		err = s.db.WithContext(ctx).Exec(`
			INSERT INTO product_barcodes (
				product_id, 
				barcode, 
				old_barcode, 
				status, 
				created_by
				)
			SELECT 
				p.id, 
				?, 
				p.barcode, 
				'completed', 
				?
			FROM products p
			WHERE p.id = ?
		`, req.Barcode, user.UserId, req.Id).Error
		if err != nil {
			s.log.Errorf("could not create product_barcodes: %v", err)
			return domain.InternalServerError
		}
	} else if req.Mxik != "" {
		// update mxik
		err := s.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", req.Id).Update("mxik", req.Mxik).Error
		if err != nil {
			s.log.Errorf("could not update products mxik: %v", err)
			return domain.InternalServerError
		}
		// store mxik
		err = s.db.WithContext(ctx).Model(&domain.StoreProduct{}).Where("product_id = ?", req.Id).Update("mxik", req.Mxik).Error
		if err != nil {
			s.log.Errorf("could not update store_products mxik: %v", err)
			return domain.InternalServerError
		}
	} else if req.UnitCode != "" {
		err := s.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", req.Id).Update("unit_code", req.UnitCode).Error
		if err != nil {
			s.log.Errorf("could not update products unit_code: %v", err)
			return domain.InternalServerError
		}
		err = s.db.WithContext(ctx).Model(&domain.StoreProduct{}).Where("product_id = ?", req.Id).Update("unit_code", req.UnitCode).Error
		if err != nil {
			s.log.Errorf("could not update store_products unit_code: %v", err)
			return domain.InternalServerError
		}
	} else if req.UnitLabel != "" {
		err := s.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", req.Id).Update("unit_label", req.UnitLabel).Error
		if err != nil {
			s.log.Errorf("could not update products unit_label: %v", err)
			return domain.InternalServerError
		}
		err = s.db.WithContext(ctx).Model(&domain.StoreProduct{}).Where("product_id = ?", req.Id).Update("unit_label", req.UnitLabel).Error
		if err != nil {
			s.log.Errorf("could not update store_products unit_label: %v", err)
			return domain.InternalServerError
		}
	}

	return nil
}

// region Delete

func (s *Services) DeleteProductPhotoAlert(id string) error {
	res := s.db.Table("product_photo_alerts").Where("id = ?", id).Delete(nil)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}
