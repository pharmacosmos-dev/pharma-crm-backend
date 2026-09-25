package services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// region Post Reserved

func (s *Services) ImportReservedProducts(
	ctx context.Context, req *domain.ReservedProductImportRequest,
) (*domain.ReservedProductImportResult, error) {
	const (
		maxItems = 30000 //100000
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

	go func() {
		statsCtx, cancel := context.WithTimeout(context.Background(), reservedProductSalesStatsTimeout)
		defer cancel()

		if err := s.RefreshReservedProductSalesStats(statsCtx); err != nil {
			s.log.Errorf("reserved products: could not refresh sales stats after import: %v", err)
		}
	}()

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

const (
	reservedProductSalesWindowDays = 15
	reservedProductSalesStatsTimeout = 5 * time.Minute
)

func (s *Services) RefreshReservedProductSalesStats(ctx context.Context) error {
	query := `
	WITH sold AS (
		SELECT
			p.material_code::text AS material_code,
			COALESCE(SUM(ci.unit_quantity) FILTER (
				WHERE s.completed_at >= (now() AT TIME ZONE 'UTC') - make_interval(days => ?)
			), 0) AS last_window,
			COALESCE(SUM(ci.unit_quantity) FILTER (
				WHERE s.completed_at < (now() AT TIME ZONE 'UTC') - make_interval(days => ?)
			), 0) AS prev_window
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		-- cart_items.product_id eski yozuvlarda bo'sh bo'lishi mumkin, store_product orqali topiladi
		LEFT JOIN store_products sp ON sp.id = ci.store_product_id
		JOIN products p ON p.id = COALESCE(ci.product_id, sp.product_id)
		WHERE s.stage = ?
			AND s.sale_type = ?
			AND s.completed_at >= (now() AT TIME ZONE 'UTC') - make_interval(days => ?)
			AND p.material_code IS NOT NULL
		GROUP BY p.material_code
	),
	calc AS (
		SELECT
			rp.id,
			COALESCE(sold.last_window, 0) AS last_window,
			COALESCE(sold.prev_window, 0) AS prev_window
		FROM reserved_products rp
		LEFT JOIN sold ON sold.material_code = rp.material_code
	)
	UPDATE reserved_products rp SET
		sold_quantity_15d      = c.last_window,
		sold_quantity_prev_15d = c.prev_window,
		sold_change_percent    = CASE
			WHEN c.prev_window > 0
			THEN ROUND(((c.last_window - c.prev_window)::numeric / c.prev_window) * 100, 2)
			ELSE NULL
		END,
		sold_calculated_at     = NOW()
	FROM calc c
	WHERE c.id = rp.id`

	result := s.db.WithContext(ctx).Exec(query,
		reservedProductSalesWindowDays,
		reservedProductSalesWindowDays,
		constants.SaleStageFinished,
		constants.SaleTypeSale,
		reservedProductSalesWindowDays*2,
	)
	if result.Error != nil {
		s.log.Errorf("reserved products: could not calculate sales stats: %v", result.Error)
		return domain.InternalServerError
	}

	s.log.Infof("reserved products sales stats refreshed: %d rows", result.RowsAffected)

	return nil
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

	// product_id va unit_per_pack ro'yxatning o'zidan kelishi kerak: rezerv hujjati
	// (reserve_details.product_id) shu id bilan yig'iladi. storeID berilsa qoldiq ham qo'shiladi.
	if len(products) > 0 {
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
			// Ochiq hujjat (status != done) bo'lsa, unga kiritilgan miqdorlar ham qo'shiladi.
			// Hujjat done bo'lsa yoki umuman bo'lmasa — reserved_quantity 0 bo'lib qoladi.
			openReserveId, err := s.getOpenReserveId(ctx, storeID)
			if err != nil {
				return nil, 0, err
			}

			stocks, err := s.getReservedProductStockMap(ctx, storeID, openReserveId, materialCodes)
			if err != nil {
				return nil, 0, err
			}

			for i := range products {
				stock, ok := stocks[strings.TrimSpace(products[i].MaterialCode)]
				if !ok {
					continue
				}
				products[i].ProductId = stock.ProductId
				products[i].UnitPerPack = stock.UnitPerPack
				products[i].AvailableQuantity = stock.AvailableQuantity
				products[i].ReservedQuantity = stock.ReservedQuantity
			}
		}
	}

	return products, totalCount, nil
}

// reservedProductStock — material_code bo'yicha topilgan CRM mahsuloti, do'kondagi qoldiq
// va ochiq rezerv hujjatiga kiritilgan miqdor.
type reservedProductStock struct {
	ProductId         string
	UnitPerPack       int
	AvailableQuantity float64
	ReservedQuantity  float64
}

// getReservedProductStockMap — reserved_products.material_code (text) va products.material_code
// (int) taqqoslanadi. Taqqoslash int'da: text'ga CAST qilinsa products'dagi unique indeks
// ishlamay, har so'rovda butun jadval skanerlanardi. Raqam bo'lmagan kod products'da baribir
// uchramaydi — u tashlab ketiladi.
func (s *Services) getReservedProductStockMap(
	ctx context.Context, storeID, openReserveId string, materialCodes []string,
) (map[string]reservedProductStock, error) {
	codes := make([]int64, 0, len(materialCodes))
	for _, code := range materialCodes {
		if parsed, err := strconv.ParseInt(strings.TrimSpace(code), 10, 64); err == nil {
			codes = append(codes, parsed)
		}
	}
	if len(codes) == 0 {
		return map[string]reservedProductStock{}, nil
	}

	var rows []struct {
		MaterialCode      string  `gorm:"column:material_code"`
		ProductId         string  `gorm:"column:product_id"`
		UnitPerPack       int     `gorm:"column:unit_per_pack"`
		AvailableQuantity float64 `gorm:"column:available_quantity"`
		ReservedQuantity  float64 `gorm:"column:reserved_quantity"`
	}

	// storeID yoki openReserveId bo'sh bo'lsa mos LEFT JOIN hech qanday qatorga tushmaydi:
	// qoldiq/kiritilgan miqdor 0 bo'ladi, lekin product_id va unit_per_pack baribir qaytadi.
	// reserved_quantity'da MAX: (reserve_id, product_id) unique, ya'ni ko'pi bilan bitta qator —
	// store_products bo'yicha ko'paygan satrlarda o'sha qiymat takrorlanadi, SUM buni qo'shib yuborardi.
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			p.material_code::text              AS material_code,
			p.id::text                         AS product_id,
			COALESCE(p.unit_per_pack, 1)       AS unit_per_pack,
			COALESCE(SUM(sp.unit_quantity), 0) AS available_quantity,
			COALESCE(MAX(rd.quantity), 0)      AS reserved_quantity
		FROM products p
		LEFT JOIN store_products sp  ON sp.product_id = p.id AND sp.store_id = NULLIF(?, '')::uuid
		LEFT JOIN reserve_details rd ON rd.product_id = p.id AND rd.reserve_id = NULLIF(?, '')::uuid
		WHERE p.material_code = ANY(?::int[])
			AND p.deleted_at IS NULL
		GROUP BY p.id, p.material_code, p.unit_per_pack
	`, storeID, openReserveId, pq.Array(codes)).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("reserved products: could not load stock by material_code for store_id=%s: %v", storeID, err)
		return nil, domain.InternalServerError
	}

	result := make(map[string]reservedProductStock, len(rows))
	for _, row := range rows {
		result[strings.TrimSpace(row.MaterialCode)] = reservedProductStock{
			ProductId:         row.ProductId,
			UnitPerPack:       row.UnitPerPack,
			AvailableQuantity: row.AvailableQuantity,
			ReservedQuantity:  row.ReservedQuantity,
		}
	}

	return result, nil
}
