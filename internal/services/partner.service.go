package services

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// SeedOAuthClients ensures the default OAuth clients exist in the database
// This should be called during application startup
func (s *Services) CreatePartnerOAuthClient(ctx context.Context, req *domain.OAuthClient) error {

	// Check if Partner client already exists
	var count int64
	err := s.db.WithContext(ctx).
		Model(&domain.OAuthClient{}).
		Where("client_id = ?", req.ClientId).
		Count(&count).Error

	if err != nil {
		s.log.Errorf("error checking OAuth client existence: %v", err)
		return err
	}

	if count > 0 {
		s.log.Info("Partner OAuth client already exists, skipping seed")
		return nil
	}

	// Hash the client secret
	hashedSecret, err := utils.HashClientSecret(s.cfg.Secret.UzumClientSecret)
	if err != nil {
		s.log.Errorf("failed to hash client secret: %v", err)
		return err
	}

	// Create the Partner client
	uzumClient := domain.OAuthClient{
		ClientId:      req.ClientId,
		ClientSecret:  hashedSecret,
		ClientName:    req.ClientName,
		AllowedScopes: req.AllowedScopes,
		IsActive:      true,
	}

	err = s.db.WithContext(ctx).Create(&uzumClient).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			s.log.Info("Partner OAuth client already exists (duplicate key), skipping")
			return nil
		}
		s.log.Errorf("failed to create OAuth client: %v", err)
		return err
	}

	s.log.Infof("Successfully created OAuth client for Partner (client_id: %s)", uzumClient.ClientId)
	return nil
}

// region Partner catalog API
// Partner API faqat retseptsiz va Uzum online narxi bor mahsulotlarni qaytaradi.

// GetPartnerStores returns active stores, ordered by store_code.
func (s *Services) GetPartnerStores() ([]domain.NoorStore, error) {
	var res []domain.NoorStore

	query := `
	SELECT
		id,
		name,
		phone,
		address,
		location,
		inn,
		work_hours,
		is_fullday,
		is_active
	FROM stores
	WHERE is_active = true
	ORDER BY store_code;`

	err := s.db.Raw(query).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get stores for partner: %v", err)
		return nil, domain.InternalServerError
	}

	// stores.location "lat,long" matn ko'rinishida saqlanadi
	for i := range res {
		loc := strings.TrimSpace(res[i].Location)
		if loc == "" || loc == "null" {
			continue
		}

		parts := strings.Split(loc, ",")
		if len(parts) != 2 {
			s.log.Warnf("invalid store location format: %q", loc)
			continue
		}

		lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			s.log.Warnf("could not parse store latitude (%q): %v", parts[0], err)
			continue
		}

		long, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			s.log.Warnf("could not parse store longitude (%q): %v", parts[1], err)
			continue
		}

		res[i].Location1 = domain.Point{Lat: lat, Long: long}
	}

	return res, nil
}

// GetPartnerCategories returns the category tree flattened, parents before children.
func (s *Services) GetPartnerCategories() ([]domain.NoorCategory, error) {
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
	ORDER BY path;`

	err := s.db.Raw(query).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get categories for partner: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// GetPartnerProducts returns products that have an Uzum online price.
func (s *Services) GetPartnerProducts(params *domain.NoorQueryParam) ([]domain.NoorProduct, error) {
	var res []domain.NoorProduct

	// producer/country/category LEFT JOIN: products'dagi FK'lar nullable,
	// INNER JOIN bo'lsa ular to'ldirilmagan mahsulotlar tushib qoladi.
	// COALESCE — NULL'ni string maydonga scan qilib bo'lmaydi.
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
	FROM products p
		LEFT JOIN categories cat ON cat.id = p.category_id
		LEFT JOIN producers prod ON prod.id = p.producer_id
		LEFT JOIN countries cnt  ON cnt.id = p.country_id
		LEFT JOIN unit_types ut  ON ut.id = p.unit_type_id
		LEFT JOIN LATERAL (
			-- sof sana ("2027-03-15"), timestamp emas
			SELECT TO_CHAR(sp.expire_date, 'YYYY-MM-DD') AS expire_date
			FROM store_products sp
			WHERE sp.product_id = p.id
			  AND sp.unit_quantity > 0
			ORDER BY sp.created_at DESC
			LIMIT 1
		) sp ON TRUE
	WHERE EXISTS (
		SELECT 1 FROM online_products_price opp
		WHERE opp.product_id = p.id AND opp.type = ?
	)
	ORDER BY p.created_at, p.id
	LIMIT ? OFFSET ?;`

	err := s.db.Raw(query, constants.ServiceTypeUzum, params.Limit, params.Offset).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get products for partner: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// GetPartnerStoreProducts returns stock quantity (in packs) and Uzum price per store.
func (s *Services) GetPartnerStoreProducts(params *domain.NoorQueryParam) ([]domain.NoorStoreProduct, error) {
	var (
		res    []domain.NoorStoreProduct
		filter = " WHERE sp.unit_quantity > 0 "
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
	) opp ON TRUE
	` + filter + `
	GROUP BY sp.product_id, sp.store_id, opp.retail_price
	HAVING SUM(sp.unit_quantity/(p.unit_per_pack/p.blister_count)) > 0
	LIMIT ? OFFSET ?;`

	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get store_products for partner: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// endregion
