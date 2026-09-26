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

	// Barcha do'konlar bo'yicha savdo dinamikasi shu yerda, kuniga bir marta hisoblanadi —
	// GET keyin uni tayyor o'qiydi. Fonda: 30 kunlik savdo ustidan ketadigan og'ir so'rov
	// 1C ning HTTP javobini kutib turmasligi kerak (timeout bo'lsa 1C importni qayta yuborardi).
	// So'rov konteksti javob qaytishi bilan tugaydi, shuning uchun yangi kontekst olinadi.
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
	openReserveId, err := s.getOpenReserveId(ctx, storeID)
	if err != nil {
		return nil, 0, err
	}

	unionSQL, unionArgs := reservedProductListSQL(params, openReserveId)

	var totalCount int64
	if err := s.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM (`+unionSQL+`) t`, unionArgs...).
		Row().Scan(&totalCount); err != nil {
		s.log.Errorf("reserved products: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	listArgs := append(append([]any{}, unionArgs...), params.Limit, params.Offset)
	listSQL := `SELECT * FROM (` + unionSQL + `) t ORDER BY t.sort_index ASC, t.material_code ASC LIMIT ? OFFSET ?`

	var products []domain.ReservedProduct
	if err := s.db.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&products).Error; err != nil {
		s.log.Errorf("reserved products: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}
	if len(products) == 0 {
		return products, totalCount, nil
	}

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
	if len(materialCodes) == 0 {
		return products, totalCount, nil
	}

	stocks, err := s.getReservedProductStockMap(ctx, storeID, openReserveId, materialCodes)
	if err != nil {
		return nil, 0, err
	}

	// Savdo dinamikasi faqat do'kon so'ralganda — sahifadagi mahsulotlar uchun bitta so'rov.
	sales := map[string]reservedProductSales{}
	if storeID != "" {
		sales, err = s.getReservedProductSalesMap(ctx, storeID, materialCodes)
		if err != nil {
			return nil, 0, err
		}
	}

	for i := range products {
		code := strings.TrimSpace(products[i].MaterialCode)

		if stock, ok := stocks[code]; ok {
			products[i].ProductId = stock.ProductId
			products[i].UnitPerPack = stock.UnitPerPack
			products[i].AvailableQuantity = stock.AvailableQuantity
			products[i].ReservedQuantity = stock.ReservedQuantity
			products[i].ReserveDetailId = stock.ReserveDetailId
		}

		// store_id berilmagan bo'lsa jadvaldagi umumiy raqamlar qoladi; berilgan bo'lsa
		// o'sha do'kon raqamlari bilan almashtiriladi (do'konda savdo bo'lmasa — 0 va null).
		if storeID != "" {
			sale := sales[code]
			products[i].SoldQuantity15d = sale.LastWindow
			products[i].SoldQuantityPrev15d = sale.PrevWindow
			products[i].SoldChangePercent = sale.ChangePercent
		}
	}

	return products, totalCount, nil
}

// reservedProductListSQL — ro'yxatning UNION so'rovi va parametrlari.
// Ikkinchi qism (qo'lda kiritilganlar) faqat ochiq hujjat bo'lganda qo'shiladi;
// is_active=false so'ralganda ham qo'shilmaydi — ular "ro'yxatdan chiqqan" emas.
func reservedProductListSQL(params *domain.ReservedProductQueryParams, openReserveId string) (string, []any) {
	where := []string{"1 = 1"}
	args := []any{}

	if params.IsActive != nil {
		where = append(where, "rp.is_active = ?")
		args = append(args, *params.IsActive)
	}
	if params.Search != "" {
		where = append(where, "(rp.name ILIKE ? OR rp.material_code ILIKE ?)")
		args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
	}

	// sold_* — importda hisoblangan, barcha do'konlar bo'yicha umumiy raqamlar.
	// store_id berilsa ular keyin o'sha do'kon raqamlari bilan almashtiriladi.
	query := `
		SELECT
			rp.id::text                 AS id,
			rp.sort_index               AS sort_index,
			rp.name                     AS name,
			rp.material_code            AS material_code,
			rp.is_active                AS is_active,
			'1c'                        AS source,
			rp.sold_quantity_15d        AS sold_quantity_15d,
			rp.sold_quantity_prev_15d   AS sold_quantity_prev_15d,
			rp.sold_change_percent      AS sold_change_percent,
			rp.created_at               AS created_at,
			rp.updated_at               AS updated_at
		FROM reserved_products rp
		WHERE ` + strings.Join(where, " AND ")

	if openReserveId == "" || (params.IsActive != nil && !*params.IsActive) {
		return query, args
	}

	manualWhere := []string{
		"rd.reserve_id = ?::uuid",
		"NOT EXISTS (SELECT 1 FROM reserved_products rp WHERE rp.material_code = rd.material_code)",
	}
	args = append(args, openReserveId)

	if params.Search != "" {
		manualWhere = append(manualWhere, "(rd.product_name ILIKE ? OR rd.material_code ILIKE ?)")
		args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
	}

	// Qo'lda kiritilganlar 1C ro'yxatida yo'q, ya'ni umumiy sold_* raqamlari ham yo'q:
	// store_id berilsa ular do'kon bo'yicha hisoblab to'ldiriladi.
	query += `
		UNION ALL
		SELECT
			''                          AS id,
			0                           AS sort_index,
			rd.product_name             AS name,
			rd.material_code            AS material_code,
			TRUE                        AS is_active,
			'manual'                    AS source,
			0::bigint                   AS sold_quantity_15d,
			0::bigint                   AS sold_quantity_prev_15d,
			NULL::numeric               AS sold_change_percent,
			rd.created_at               AS created_at,
			rd.updated_at               AS updated_at
		FROM reserve_details rd
		WHERE ` + strings.Join(manualWhere, " AND ")

	return query, args
}

// reservedProductIntCodes — reserved_products.material_code (text) products.material_code
// (int) bilan solishtiriladi. Taqqoslash int'da: text'ga CAST qilinsa products'dagi unique
// indeks ishlamay, har so'rovda butun jadval skanerlanardi. Raqam bo'lmagan kod products'da
// baribir uchramaydi — u tashlab ketiladi.
func reservedProductIntCodes(materialCodes []string) []int64 {
	codes := make([]int64, 0, len(materialCodes))
	for _, code := range materialCodes {
		if parsed, err := strconv.ParseInt(strings.TrimSpace(code), 10, 64); err == nil {
			codes = append(codes, parsed)
		}
	}

	return codes
}

// reservedProductStock — material_code bo'yicha topilgan CRM mahsuloti, do'kondagi qoldiq
// va ochiq rezerv hujjatidagi qator.
type reservedProductStock struct {
	ProductId         string
	UnitPerPack       int
	AvailableQuantity float64
	ReservedQuantity  float64
	ReserveDetailId   string
}

func (s *Services) getReservedProductStockMap(
	ctx context.Context, storeID, openReserveId string, materialCodes []string,
) (map[string]reservedProductStock, error) {
	codes := reservedProductIntCodes(materialCodes)
	if len(codes) == 0 {
		return map[string]reservedProductStock{}, nil
	}

	var rows []struct {
		MaterialCode      string  `gorm:"column:material_code"`
		ProductId         string  `gorm:"column:product_id"`
		UnitPerPack       int     `gorm:"column:unit_per_pack"`
		AvailableQuantity float64 `gorm:"column:available_quantity"`
		ReservedQuantity  float64 `gorm:"column:reserved_quantity"`
		ReserveDetailId   string  `gorm:"column:reserve_detail_id"`
	}

	// storeID yoki openReserveId bo'sh bo'lsa mos LEFT JOIN hech qanday qatorga tushmaydi:
	// qoldiq/kiritilgan miqdor 0 bo'ladi, lekin product_id va unit_per_pack baribir qaytadi.
	// reserve_details bo'yicha MAX: (reserve_id, product_id) unique, ya'ni ko'pi bilan bitta
	// qator — store_products bo'yicha ko'paygan satrlarda o'sha qiymat takrorlanadi, SUM
	// buni qo'shib yuborardi.
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			p.material_code::text              AS material_code,
			p.id::text                         AS product_id,
			COALESCE(p.unit_per_pack, 1)       AS unit_per_pack,
			COALESCE(SUM(sp.unit_quantity), 0) AS available_quantity,
			COALESCE(MAX(rd.quantity), 0)      AS reserved_quantity,
			COALESCE(MAX(rd.id::text), '')     AS reserve_detail_id
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
			ReserveDetailId:   row.ReserveDetailId,
		}
	}

	return result, nil
}

// reservedProductSales — do'kon kesimidagi savdo dinamikasi.
type reservedProductSales struct {
	LastWindow    int64
	PrevWindow    int64
	ChangePercent *float64
}

// getReservedProductSalesMap — sahifadagi mahsulotlar uchun do'kondagi oxirgi 15 kunlik va
// undan oldingi 15 kunlik sotuv (cart_items.unit_quantity yig'indisi). Faqat yakunlangan
// sotuvlar (stage 9, sale_type SALE); vozvratlar ayirilmaydi.
//
// So'rov savdolardan boshlanadi — sales(completed_at, stage, store_id) indeksi bitta
// do'konning 30 kunlik cheklarini beradi, cart_items esa sale_id indeksi bilan ulanadi.
// Shuning uchun og'irlikni sahifadagi mahsulot soni emas, do'konning savdo hajmi belgilaydi.
func (s *Services) getReservedProductSalesMap(
	ctx context.Context, storeID string, materialCodes []string,
) (map[string]reservedProductSales, error) {
	codes := reservedProductIntCodes(materialCodes)
	if len(codes) == 0 {
		return map[string]reservedProductSales{}, nil
	}

	var rows []struct {
		MaterialCode  string   `gorm:"column:material_code"`
		LastWindow    int64    `gorm:"column:last_window"`
		PrevWindow    int64    `gorm:"column:prev_window"`
		ChangePercent *float64 `gorm:"column:change_percent"`
	}

	err := s.db.WithContext(ctx).Raw(`
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
			WHERE s.store_id = ?::uuid
				AND s.stage = ?
				AND s.sale_type = ?
				AND s.completed_at >= (now() AT TIME ZONE 'UTC') - make_interval(days => ?)
				AND p.material_code = ANY(?::int[])
			GROUP BY p.material_code
		)
		SELECT
			material_code,
			last_window,
			prev_window,
			CASE
				WHEN prev_window > 0
				THEN ROUND(((last_window - prev_window)::numeric / prev_window) * 100, 2)
			END AS change_percent
		FROM sold
	`,
		reservedProductSalesWindowDays,
		reservedProductSalesWindowDays,
		storeID,
		constants.SaleStageFinished,
		constants.SaleTypeSale,
		reservedProductSalesWindowDays*2,
		pq.Array(codes),
	).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("reserved products: could not load sales stats for store_id=%s: %v", storeID, err)
		return nil, domain.InternalServerError
	}

	result := make(map[string]reservedProductSales, len(rows))
	for _, row := range rows {
		result[strings.TrimSpace(row.MaterialCode)] = reservedProductSales{
			LastWindow:    row.LastWindow,
			PrevWindow:    row.PrevWindow,
			ChangePercent: row.ChangePercent,
		}
	}

	return result, nil
}
