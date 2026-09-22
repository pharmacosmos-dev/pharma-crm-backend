package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// region Post Reserved

func (s *Services) ImportReservedProducts(
	ctx context.Context, req *domain.ReservedProductImportRequest,
) (*domain.ReservedProductImportResult, error) {
	const (
		
		maxItems = 100000
		lockName = "reserved_products_import"
	)

	if len(req.Products) == 0 {
		return nil, domain.NewError(http.StatusBadRequest, "reserved_products.empty_list")
	}
	if len(req.Products) > maxItems {
		return nil, domain.NewError(http.StatusBadRequest,
			fmt.Sprintf("reserved_products.too_many_items: max=%d, got=%d", maxItems, len(req.Products)))
	}

	var (
		indexes          = make([]int64, 0, len(req.Products))
		codes            = make([]string, 0, len(req.Products))
		names            = make([]string, 0, len(req.Products))
		seen             = make(map[string]struct{}, len(req.Products))
		skippedDuplicate int
	)

	for i, item := range req.Products {
		if item.Index < 1 {
			return nil, domain.NewError(http.StatusBadRequest,
				fmt.Sprintf("reserved_products.invalid_index: position=%d, index=%d", i+1, item.Index))
		}

		code := strings.TrimSpace(item.MaterialCode.String())
		if code == "" {
			return nil, domain.NewError(http.StatusBadRequest,
				fmt.Sprintf("reserved_products.empty_material_code: position=%d, index=%d", i+1, item.Index))
		}

		if _, ok := seen[code]; ok {
			skippedDuplicate++
			continue
		}
		seen[code] = struct{}{}

		indexes = append(indexes, int64(item.Index))
		codes = append(codes, code)
		names = append(names, strings.TrimSpace(item.ProductName))
	}

	var counts struct {
		Inserted    int
		Updated     int
		Deactivated int
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Parallel import kelsa kutib turmaymiz — darhol 409 qaytariladi, 1C keyin
		// qayta uradi. Aks holda ikkita ro'yxat bir-birining mahsulotlarini deaktiv qilardi.
		var locked bool
		if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(hashtext(?))`, lockName).Row().Scan(&locked); err != nil {
			s.log.Errorf("reserved products: could not acquire import lock: %v", err)
			return domain.InternalServerError
		}
		if !locked {
			return domain.ConflictError
		}
	
		const importSQL = `
			WITH incoming AS (
				SELECT * FROM unnest(?::int[], ?::text[], ?::text[]) AS t(sort_index, material_code, name)
			),
			existing AS (
				SELECT rp.material_code
				FROM reserved_products rp
				JOIN incoming i ON i.material_code = rp.material_code
			),
			upserted AS (
				INSERT INTO reserved_products (sort_index, material_code, name, is_active)
				SELECT i.sort_index, i.material_code, i.name, TRUE FROM incoming i
				ON CONFLICT (material_code) DO UPDATE SET
					sort_index = EXCLUDED.sort_index,
					name       = EXCLUDED.name,
					is_active  = TRUE,
					updated_at = NOW()
				RETURNING 1
			),
			deactivated AS (
				UPDATE reserved_products rp
				SET is_active = FALSE, updated_at = NOW()
				WHERE rp.is_active
				  AND NOT EXISTS (SELECT 1 FROM incoming i WHERE i.material_code = rp.material_code)
				RETURNING 1
			)
			SELECT
				(SELECT COUNT(*) FROM upserted) - (SELECT COUNT(*) FROM existing) AS inserted,
				(SELECT COUNT(*) FROM existing)                                   AS updated,
				(SELECT COUNT(*) FROM deactivated)                                AS deactivated
		`

		if err := tx.Raw(importSQL, pq.Array(indexes), pq.Array(codes), pq.Array(names)).
			Scan(&counts).Error; err != nil {
			s.log.Errorf("reserved products: could not import %d items: %v", len(codes), err)
			return domain.InternalServerError
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := &domain.ReservedProductImportResult{
		TotalReceived:    len(req.Products),
		Inserted:         counts.Inserted,
		Updated:          counts.Updated,
		Deactivated:      counts.Deactivated,
		SkippedDuplicate: skippedDuplicate,
	}
	s.log.Infof("reserved products import: %+v", *result)

	return result, nil
}

// region Get Reserved

func (s *Services) GetReservedProducts(
	ctx context.Context, params *domain.ReservedProductQueryParams, storeID string,
) ([]domain.ReservedProduct, int64, error) {

	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&domain.ReservedProduct{})
		if params.IsActive != nil {
			q = q.Where("is_active = ?", *params.IsActive)
		}
		if params.Search != "" {
			search := fmt.Sprintf("%%%s%%", params.Search)
			q = q.Where("name ILIKE ? OR material_code ILIKE ?", search, search)
		}
		return q
	}

	var totalCount int64
	if err := newQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("reserved products: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var products []domain.ReservedProduct
	err := newQuery().
		Order("sort_index ASC, material_code ASC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&products).Error
	if err != nil {
		s.log.Errorf("reserved products: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if storeID != "" && len(products) > 0 {
		materialCodes := make([]string, 0, len(products))
		seen := make(map[string]struct{}, len(products))
		for _, product := range products {
			code := strings.TrimSpace(product.MaterialCode)
			if code == "" {
				continue
			}
			if _, exists := seen[code]; exists {
				continue
			}
			seen[code] = struct{}{}
			materialCodes = append(materialCodes, code)
		}

		if len(materialCodes) > 0 {
			stocks, err := s.getReservedProductStockMap(ctx, storeID, materialCodes)
			if err != nil {
				return nil, 0, err
			}

			for i := range products {
				products[i].AvailableQuantity = stocks[strings.TrimSpace(products[i].MaterialCode)]
			}
		}
	}

	return products, totalCount, nil
}

func (s *Services) getReservedProductStockMap(
	ctx context.Context, storeID string, materialCodes []string,
) (map[string]float64, error) {
	if storeID == "" || len(materialCodes) == 0 {
		return map[string]float64{}, nil
	}

	var rows []struct {
		MaterialCode      string  `gorm:"column:material_code"`
		AvailableQuantity float64 `gorm:"column:available_quantity"`
	}

	err := s.db.WithContext(ctx).Raw(`
		SELECT CAST(p.material_code AS TEXT) AS material_code,
			COALESCE(SUM(sp.unit_quantity), 0) AS available_quantity
		FROM products p
		LEFT JOIN store_products sp ON sp.product_id = p.id AND sp.store_id = ?
		WHERE CAST(p.material_code AS TEXT) = ANY(?)
		GROUP BY CAST(p.material_code AS TEXT)
	`, storeID, pq.Array(materialCodes)).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("reserved products: could not load stock by material_code for store_id=%s: %v", storeID, err)
		return nil, domain.InternalServerError
	}

	result := make(map[string]float64, len(rows))
	for _, row := range rows {
		result[strings.TrimSpace(row.MaterialCode)] = row.AvailableQuantity
	}

	return result, nil
}
