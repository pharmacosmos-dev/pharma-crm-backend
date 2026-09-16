package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

// Inventory'gacha bo'lgan harakat turlari (lines.type qiymatlari)
const (
	InventoryMovementTypeImport       = "import"
	InventoryMovementTypeSale         = "sale"
	InventoryMovementTypeClientReturn = "client_return"
	InventoryMovementTypeTransferIn   = "transfer_in"
	InventoryMovementTypeTransferOut  = "transfer_out"
	InventoryMovementTypeVozvrat      = "vozvrat"
	InventoryMovementTypeInventory    = "inventory"
)

var inventoryMovementTypes = []string{
	InventoryMovementTypeImport,
	InventoryMovementTypeSale,
	InventoryMovementTypeClientReturn,
	InventoryMovementTypeTransferIn,
	InventoryMovementTypeTransferOut,
	InventoryMovementTypeVozvrat,
	InventoryMovementTypeInventory,
}

// inventoryMovementLinesCTE inventory do'konidagi `page` productlari uchun inventory'gacha bo'lgan
// har bir stock harakatini bitta qator qilib beradi (dona, ishorali: kirim +, chiqim -).
// Undan oldin `cut` (inventory: id, store_id, ts) va `page` (product_id, unit_per_pack) CTE'lari bo'lishi kerak.
//
// Har bir harakat stock HAQIQATDA o'zgargan vaqt bilan kesiladi, hujjat yaratilgan vaqt bilan emas:
//   - import: confirm paytida yaratilgan store_products.created_at (sp.import_detail_id = imd.id).
//     imports'da confirm vaqti yo'q, updated_at esa trigger bilan qayta yoziladi.
//   - sotuv / mijoz qaytarishi: sales.completed_at. Go time.Now() dan yoziladi, konteyner UTC da,
//     ustun esa timestamp (tz'siz) - shuning uchun AT TIME ZONE 'UTC'.
//   - transfer: qabul qiluvchi do'konda confirm paytida yaratilgan store_products.created_at
//     (sp.import_detail_id = td.id). Topilmasa accepted_at - u SQL NOW() bilan Asia/Tashkent
//     seansida timestamp ustunga yoziladi, shuning uchun AT TIME ZONE 'Asia/Tashkent'.
//   - vozvrat: accepted_at (confirm vaqti).
//
// CreateInventory do'konda sent/checking/rejection holatdagi transfer bo'lsa inventory ochmaydi,
// shuning uchun T paytida yo'lda transfer bo'lmaydi va accept vaqti bo'yicha kesish yetarli.
const inventoryMovementLinesCTE = `
lines AS (
	SELECT u.*
	FROM (
		-- import: confirm paytida store_products ga ROUND(accepted_count * unit_per_pack) qo'shiladi
		SELECT
			imd.product_id,
			'import'::text                         AS type,
			im.id                                  AS document_id,
			im.public_id::text                     AS public_id,
			im.status::text                        AS status,
			NULL::uuid                             AS counterparty_store_id,
			COALESCE((
				SELECT MIN(sp.created_at)
				FROM store_products sp
				WHERE sp.store_id = im.store_id
				  AND sp.product_id = imd.product_id
				  AND sp.import_detail_id = imd.id
			), im.created_at)                      AS movement_at,
			ROUND(imd.accepted_count * pg.unit_per_pack) AS quantity
		FROM cut c
		JOIN imports im ON im.store_id = c.store_id
			AND im.entry_type = 1
			AND im.status = 'completed'
			AND im.created_at < c.ts
		JOIN import_details imd ON imd.import_id = im.id
		JOIN page pg ON pg.product_id = imd.product_id
		WHERE imd.accepted_count > 0

		UNION ALL

		-- sotuv (stage 9) va mijoz qaytarishi (stage 11): store_products dan ayiriladi / qaytariladi.
		-- Uzum buyurtmasi yakunlangandan keyin bekor qilinsa stock qaytariladi, stage esa 9 qoladi.
		SELECT
			sp.product_id,
			CASE WHEN s.sale_type = 'SALE' THEN 'sale' ELSE 'client_return' END AS type,
			s.id                                   AS document_id,
			s.sale_number::text                    AS public_id,
			COALESCE(s.status, '')::text           AS status,
			NULL::uuid                             AS counterparty_store_id,
			COALESCE(s.completed_at AT TIME ZONE 'UTC', s.created_at AT TIME ZONE 'Asia/Tashkent') AS movement_at,
			(CASE WHEN s.sale_type = 'SALE' THEN -ci.unit_quantity ELSE ci.unit_quantity END)::numeric AS quantity
		FROM cut c
		JOIN page pg ON TRUE
		JOIN store_products sp ON sp.store_id = c.store_id AND sp.product_id = pg.product_id
		JOIN cart_items ci ON ci.store_product_id = sp.id
		JOIN sales s ON s.id = ci.sale_id
		WHERE s.stage IN (9, 11)
		  AND s.sale_type IN ('SALE', 'RETURN')
		  AND NOT (
			s.sale_type = 'SALE'
			AND COALESCE(s.service_type, '') = 'uzum'
			AND COALESCE(s.online_status, 0) = -1
		  )

		UNION ALL

		-- transfer kirim: confirm paytida qabul qiluvchi do'konga yangi store_products qatori yoziladi
		SELECT
			sp.product_id,
			'transfer_in'::text                    AS type,
			t.id                                   AS document_id,
			COALESCE(t.public_id, '')::text        AS public_id,
			t.status::text                         AS status,
			t.from_store_id                        AS counterparty_store_id,
			sp.created_at                          AS movement_at,
			ROUND(COALESCE(td.accepted_count, 0) * pg.unit_per_pack) AS quantity
		FROM cut c
		JOIN page pg ON TRUE
		JOIN store_products sp ON sp.store_id = c.store_id AND sp.product_id = pg.product_id
		JOIN transfer_details td ON td.id = sp.import_detail_id
		JOIN transfers t ON t.id = td.transfer_id
			AND t.to_store_id = c.store_id
			AND t.entry_type = 1

		UNION ALL

		-- transfer chiqim (entry_type 1) va vozvrat (entry_type 2) manba partiyadan.
		-- Send'da expected ayiriladi, confirm'da qoldiq qaytariladi: sof ta'sir accepted_count.
		-- Vozvrat rejection holatida rad etilgan qism hali qaytmagan: sof ta'sir scanned_count.
		SELECT
			sp.product_id,
			CASE WHEN t.entry_type = 1 THEN 'transfer_out' ELSE 'vozvrat' END AS type,
			t.id                                   AS document_id,
			COALESCE(t.public_id, '')::text        AS public_id,
			t.status::text                         AS status,
			t.to_store_id                          AS counterparty_store_id,
			COALESCE(
				CASE WHEN t.entry_type = 1 THEN (
					SELECT MIN(dsp.created_at)
					FROM store_products dsp
					WHERE dsp.store_id = t.to_store_id
					  AND dsp.product_id = td.product_id
					  AND dsp.import_detail_id = td.id
				) END,
				t.accepted_at AT TIME ZONE 'Asia/Tashkent',
				t.created_at
			)                                      AS movement_at,
			-(CASE
				WHEN t.entry_type = 2 AND t.status = 'rejection'
					THEN ROUND(COALESCE(td.scanned_count, 0) * pg.unit_per_pack)
				ELSE ROUND(COALESCE(td.accepted_count, 0) * pg.unit_per_pack)
			END)                                   AS quantity
		FROM cut c
		JOIN page pg ON TRUE
		JOIN store_products sp ON sp.store_id = c.store_id AND sp.product_id = pg.product_id
		JOIN transfer_details td ON td.store_product_id = sp.id
		JOIN transfers t ON t.id = td.transfer_id
			AND t.from_store_id = c.store_id
			AND t.created_at < c.ts
		WHERE (t.entry_type = 1 AND t.status IN ('completed', 'sent-to-1c', 'failed_sent_to_1c'))
		   OR (t.entry_type = 2 AND t.status IN ('completed', 'sent-to-1c', 'failed_sent_to_1c', 'rejection'))

		UNION ALL

		-- oldingi yakunlangan inventory'lar tuzatishi: scanned_count - received_count (dona).
		-- SOME turida confirm faqat skanerlangan va do'konda mavjud partiyalarni o'zgartiradi.
		SELECT
			imd.product_id,
			'inventory'::text                      AS type,
			im.id                                  AS document_id,
			im.public_id::text                     AS public_id,
			im.status::text                        AS status,
			NULL::uuid                             AS counterparty_store_id,
			im.created_at                          AS movement_at,
			SUM(imd.scanned_count - imd.received_count) AS quantity
		FROM cut c
		JOIN imports im ON im.store_id = c.store_id
			AND im.entry_type = 2
			AND im.status = 'completed'
			AND im.created_at < c.ts
			AND im.id <> c.id
		JOIN import_details imd ON imd.import_id = im.id
		JOIN page pg ON pg.product_id = imd.product_id
		WHERE imd.scanned_count <> imd.received_count
		  AND (
			COALESCE(im.inventory_type, '') <> 'SOME'
			OR (imd.scanned_count > 0 AND imd.store_product_id IS NOT NULL)
		  )
		GROUP BY imd.product_id, im.id
	) u
	WHERE u.movement_at < (SELECT ts FROM cut)
	  AND u.quantity <> 0
)`

type inventoryMovementRow struct {
	ProductId          string     `gorm:"product_id"`
	MaterialCode       int        `gorm:"material_code"`
	Name               string     `gorm:"name"`
	Barcode            string     `gorm:"barcode"`
	UnitPerPack        int        `gorm:"unit_per_pack"`
	ReceivedCount      float64    `gorm:"received_count"`
	ScannedCount       float64    `gorm:"scanned_count"`
	IsScanned          bool       `gorm:"is_scanned"`
	SomeStockAfter     float64    `gorm:"some_stock_after"`
	CalculatedQuantity float64    `gorm:"calculated_quantity"`
	FirstMovementAt    *time.Time `gorm:"first_movement_at"`
	LastMovementAt     *time.Time `gorm:"last_movement_at"`
	TotalCount         int64      `gorm:"total_count"`
	domain.InventoryMovementTotals
}

// getInventoryMovementHeader inventory'ni oladi va foydalanuvchi do'koni/kompaniyasiga tegishliligini tekshiradi.
func (s *Services) getInventoryMovementHeader(ctx context.Context, params *domain.InventoryMovementParam) (*domain.InventoryMovementHeader, error) {
	var header domain.InventoryMovementHeader
	err := s.db.WithContext(ctx).Raw(`
	SELECT
		im.id,
		im.public_id,
		COALESCE(im.name, '')                AS name,
		COALESCE(im.inventory_type, '')      AS inventory_type,
		im.status,
		im.store_id,
		COALESCE(st.name, '')                AS store_name,
		COALESCE(st.company_id::text, '')    AS company_id,
		im.created_at                        AS cutoff_at
	FROM imports im
	JOIN stores st ON st.id = im.store_id
	WHERE im.id = ? AND im.entry_type = 2`, params.InventoryId).Scan(&header).Error
	if err != nil {
		s.log.Errorf("could not get inventory(%s) for movements: %v", params.InventoryId, err)
		return nil, domain.InternalServerError
	}

	if header.Id == "" || header.CutoffAt == nil {
		return nil, domain.NotFoundError
	}

	if params.StoreId != "" && header.StoreId != params.StoreId {
		return nil, domain.ForbiddinError
	}
	if params.CompanyId != "" && header.CompanyId != params.CompanyId {
		return nil, domain.ForbiddinError
	}

	return &header, nil
}

// GetInventoryMovements inventory'dagi har bir product uchun inventory'gacha bo'lgan harakatlar yig'indisini,
// hisoblangan qoldiqni va inventory natijasi bilan farqini qaytaradi.
// params.Limit <= 0 bo'lsa hamma productlar qaytadi (export uchun).
func (s *Services) GetInventoryMovements(ctx context.Context, params *domain.InventoryMovementParam) (*domain.InventoryMovementHeader, []domain.InventoryMovementItem, int64, error) {
	header, err := s.getInventoryMovementHeader(ctx, params)
	if err != nil {
		return nil, nil, 0, err
	}

	isSome := header.InventoryType == "SOME"

	args := map[string]any{
		"inventory_id": header.Id,
		"limit":        params.Limit,
		"offset":       params.Offset,
	}

	productFilter := ""
	if params.ProductId != "" {
		productFilter += " AND inv.product_id = @product_id"
		args["product_id"] = params.ProductId
	} else {
		// FULL inventory katalogdagi barcha productlarni qo'shadi: do'konda hech qachon
		// partiyasi bo'lmagan va sanalmagan productlar ro'yxatni to'ldirib yubormasin
		productFilter += " AND (inv.in_store OR inv.received_count <> 0 OR inv.scanned_count <> 0)"
	}
	if params.Search != "" {
		productFilter += " AND (p.name ILIKE @search OR p.barcode ILIKE @search OR p.material_code::text = @search_exact)"
		args["search"] = "%" + params.Search + "%"
		args["search_exact"] = params.Search
	}

	pageLimit, finalFilter, finalLimit, totalCountExpr := "", "", "", "(SELECT COUNT(*) FROM products_filtered)"
	if params.OnlyDiff {
		// farqni bilish uchun hamma productlarni hisoblash kerak, pagination oxirida
		finalFilter = " WHERE r.scanned_count - r.calculated_quantity <> 0"
		if isSome {
			finalFilter += " AND r.is_scanned"
		}
		totalCountExpr = "COUNT(*) OVER()"
		if params.Limit > 0 {
			finalLimit = " LIMIT @limit OFFSET @offset"
		}
	} else if params.Limit > 0 {
		pageLimit = " ORDER BY name, product_id LIMIT @limit OFFSET @offset"
	}

	query := fmt.Sprintf(`
	WITH cut AS (
		SELECT im.id, im.store_id, im.created_at AS ts
		FROM imports im
		WHERE im.id = @inventory_id
	),
	inv AS (
		SELECT
			imd.product_id,
			SUM(imd.received_count)                               AS received_count,
			SUM(imd.scanned_count)                                AS scanned_count,
			SUM(CASE WHEN imd.scanned_count > 0 AND imd.store_product_id IS NOT NULL
				THEN imd.scanned_count ELSE imd.received_count END) AS some_stock_after,
			BOOL_OR(imd.scanned_count > 0)                        AS is_scanned,
			BOOL_OR(imd.store_product_id IS NOT NULL)             AS in_store
		FROM import_details imd
		WHERE imd.import_id = @inventory_id
		GROUP BY imd.product_id
	),
	products_filtered AS (
		SELECT
			inv.product_id,
			COALESCE(p.name, '')          AS name,
			COALESCE(p.material_code, 0)  AS material_code,
			COALESCE(p.barcode, '')       AS barcode,
			p.unit_per_pack,
			inv.received_count,
			inv.scanned_count,
			inv.some_stock_after,
			inv.is_scanned
		FROM inv
		JOIN products p ON p.id = inv.product_id
		WHERE TRUE %s
	),
	page AS (
		SELECT * FROM products_filtered%s
	),
	%s,
	agg AS (
		SELECT
			l.product_id,
			SUM(l.quantity)  FILTER (WHERE l.type = 'import')        AS import_quantity,
			SUM(-l.quantity) FILTER (WHERE l.type = 'sale')          AS sold_quantity,
			SUM(l.quantity)  FILTER (WHERE l.type = 'client_return') AS returned_quantity,
			SUM(l.quantity)  FILTER (WHERE l.type = 'transfer_in')   AS transfer_in_quantity,
			SUM(-l.quantity) FILTER (WHERE l.type = 'transfer_out')  AS transfer_out_quantity,
			SUM(-l.quantity) FILTER (WHERE l.type = 'vozvrat')       AS vozvrat_quantity,
			SUM(GREATEST(l.quantity, 0)) FILTER (WHERE l.type = 'inventory') AS inventory_plus_count,
			SUM(LEAST(l.quantity, 0))    FILTER (WHERE l.type = 'inventory') AS inventory_minus_count,
			SUM(l.quantity)                                          AS calculated_quantity,
			MIN(l.movement_at)                                       AS first_movement_at,
			MAX(l.movement_at)                                       AS last_movement_at
		FROM lines l
		GROUP BY l.product_id
	),
	result AS (
		SELECT
			pg.product_id,
			pg.material_code,
			pg.name,
			pg.barcode,
			pg.unit_per_pack,
			pg.received_count,
			pg.scanned_count,
			pg.is_scanned,
			pg.some_stock_after,
			COALESCE(a.import_quantity, 0)       AS import_quantity,
			COALESCE(a.sold_quantity, 0)         AS sold_quantity,
			COALESCE(a.returned_quantity, 0)     AS returned_quantity,
			COALESCE(a.transfer_in_quantity, 0)  AS transfer_in_quantity,
			COALESCE(a.transfer_out_quantity, 0) AS transfer_out_quantity,
			COALESCE(a.vozvrat_quantity, 0)      AS vozvrat_quantity,
			COALESCE(a.inventory_plus_count, 0)  AS inventory_plus_count,
			COALESCE(a.inventory_minus_count, 0) AS inventory_minus_count,
			COALESCE(a.calculated_quantity, 0)   AS calculated_quantity,
			a.first_movement_at,
			a.last_movement_at
		FROM page pg
		LEFT JOIN agg a ON a.product_id = pg.product_id
	)
	SELECT r.*, %s AS total_count
	FROM result r%s
	ORDER BY r.name, r.product_id%s`,
		productFilter, pageLimit, inventoryMovementLinesCTE, totalCountExpr, finalFilter, finalLimit)

	var rows []inventoryMovementRow
	if err = s.db.WithContext(ctx).Raw(query, args).Scan(&rows).Error; err != nil {
		s.log.Errorf("could not get inventory(%s) movements: %v", header.Id, err)
		return nil, nil, 0, domain.InternalServerError
	}

	var totalCount int64
	items := make([]domain.InventoryMovementItem, 0, len(rows))
	for _, row := range rows {
		totalCount = row.TotalCount
		items = append(items, buildInventoryMovementItem(header, isSome, row))
	}

	return header, items, totalCount, nil
}

func buildInventoryMovementItem(header *domain.InventoryMovementHeader, isSome bool, row inventoryMovementRow) domain.InventoryMovementItem {
	received := utils.RoundTo(row.ReceivedCount, 4)
	scanned := utils.RoundTo(row.ScannedCount, 4)
	calculated := utils.RoundTo(row.CalculatedQuantity, 4)

	item := domain.InventoryMovementItem{
		ProductId:    row.ProductId,
		MaterialCode: row.MaterialCode,
		Name:         row.Name,
		Barcode:      row.Barcode,
		UnitPerPack:  row.UnitPerPack,
		Inventory: domain.InventoryMovementCounts{
			ReceivedCount: received,
			ScannedCount:  scanned,
			// SOME (qisman) inventory'da skanerlanmagan product umuman sanalmagan
			IsCounted: !isSome || row.IsScanned,
		},
		Movements:          row.InventoryMovementTotals,
		CalculatedQuantity: calculated,
		UntrackedQuantity:  utils.RoundTo(received-calculated, 4),
		FirstMovementAt:    row.FirstMovementAt,
		LastMovementAt:     row.LastMovementAt,
	}

	if item.Inventory.IsCounted {
		inventoryDiff := utils.RoundTo(scanned-received, 4)
		diff := utils.RoundTo(scanned-calculated, 4)
		item.Inventory.Difference = &inventoryDiff
		item.Difference = &diff
	}

	if header.Status == constants.GeneralStatusCompleted {
		// FULL confirm har bir partiyani scanned_count ga tenglaydi; SOME faqat skanerlangan partiyalarni
		after := scanned
		if isSome {
			after = utils.RoundTo(row.SomeStockAfter, 4)
		}
		item.StockAfterInventory = &after
	}

	return item
}

// GetInventoryMovementHistory bitta product bo'yicha inventory'gacha bo'lgan harakatlarni hujjatma-hujjat,
// har biridan keyingi hisoblangan qoldiq (balance) bilan qaytaradi. Birinchi qaytgan qiymat - productning
// GetInventoryMovements dagi yig'indisi (balance oxirida shu calculated_quantity ga teng bo'ladi).
func (s *Services) GetInventoryMovementHistory(ctx context.Context, params *domain.InventoryMovementParam) (*domain.InventoryMovementHeader, *domain.InventoryMovementItem, []domain.InventoryMovementHistoryItem, int64, error) {
	if params.Type != "" && !utils.In(params.Type, inventoryMovementTypes...) {
		return nil, nil, nil, 0, domain.InvalidQueryError
	}

	summaryParams := domain.InventoryMovementParam{
		InventoryId: params.InventoryId,
		ProductId:   params.ProductId,
		StoreId:     params.StoreId,
		CompanyId:   params.CompanyId,
	}
	header, summary, _, err := s.GetInventoryMovements(ctx, &summaryParams)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if len(summary) == 0 {
		return nil, nil, nil, 0, domain.NotFoundError
	}

	args := map[string]any{
		"inventory_id": header.Id,
		"product_id":   params.ProductId,
		"limit":        params.Limit,
		"offset":       params.Offset,
	}

	typeFilter := ""
	if params.Type != "" {
		typeFilter = " WHERE b.type = @type"
		args["type"] = params.Type
	}

	orderBy := "b.movement_at, b.document_id"
	if params.Order == "desc" {
		orderBy = "b.movement_at DESC, b.document_id DESC"
	}

	query := fmt.Sprintf(`
	WITH cut AS (
		SELECT im.id, im.store_id, im.created_at AS ts
		FROM imports im
		WHERE im.id = @inventory_id
	),
	page AS (
		SELECT p.id AS product_id, p.unit_per_pack
		FROM products p
		WHERE p.id = @product_id
	),
	%s,
	events AS (
		SELECT
			l.type,
			l.document_id,
			MAX(l.public_id)        AS public_id,
			MAX(l.status)           AS status,
			l.counterparty_store_id,
			MIN(l.movement_at)      AS movement_at,
			SUM(l.quantity)         AS quantity
		FROM lines l
		GROUP BY l.type, l.document_id, l.counterparty_store_id
	),
	balanced AS (
		SELECT
			e.*,
			SUM(e.quantity) OVER (
				ORDER BY e.movement_at, e.document_id
				ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
			) AS balance
		FROM events e
	)
	SELECT
		b.type,
		b.document_id::text     AS document_id,
		b.public_id,
		b.status,
		COALESCE(st.name, '')   AS counterparty,
		b.movement_at,
		b.quantity,
		b.balance,
		COUNT(*) OVER()         AS total_count
	FROM balanced b
	LEFT JOIN stores st ON st.id = b.counterparty_store_id%s
	ORDER BY %s
	LIMIT @limit OFFSET @offset`,
		inventoryMovementLinesCTE, typeFilter, orderBy)

	var res []domain.InventoryMovementHistoryItem
	if err = s.db.WithContext(ctx).Raw(query, args).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get inventory(%s) movement history for product(%s): %v", header.Id, params.ProductId, err)
		return nil, nil, nil, 0, domain.InternalServerError
	}

	var totalCount int64
	for i := range res {
		totalCount = res[i].TotalCount
		res[i].Quantity = utils.RoundTo(res[i].Quantity, 4)
		res[i].Balance = utils.RoundTo(res[i].Balance, 4)
	}
	if res == nil {
		res = []domain.InventoryMovementHistoryItem{}
	}

	return header, &summary[0], res, totalCount, nil
}
