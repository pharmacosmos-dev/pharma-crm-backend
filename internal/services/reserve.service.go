package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// Rezerv hujjatlari: reserves (hujjat) + reserve_details (qatorlar), imports/import_details kabi.
// Status oqimi: new -> checking -> done. done bo'lgan hujjat va uning qatorlari o'zgarmaydi.

// reserveSelectSQL — hujjat qatorini do'kon va xodim nomlari bilan o'qish uchun.
const reserveSelectSQL = `
	SELECT
		r.id,
		r.dok_number,
		r.store_id,
		COALESCE(s.name, '')            AS store_name,
		r.status,
		r.total_quantity,
		r.total_product_count,
		COALESCE(r.comment, '')         AS comment,
		COALESCE(r.created_by::text, '') AS created_by,
		COALESCE(ce.full_name, '')       AS created_by_name,
		COALESCE(r.updated_by::text, '') AS updated_by,
		COALESCE(ue.full_name, '')       AS updated_by_name,
		r.completed_at,
		r.created_at,
		r.updated_at
	FROM reserves r
	LEFT JOIN stores s     ON s.id = r.store_id
	LEFT JOIN employees ce ON ce.id = r.created_by
	LEFT JOIN employees ue ON ue.id = r.updated_by
`

// reserveDetailSelectSQL — qatorlar; available_quantity hujjat do'konidagi qoldiq (birlikda).
const reserveDetailSelectSQL = `
	SELECT
		d.id,
		d.reserve_id,
		d.product_id,
		COALESCE(d.reserved_product_id::text, '') AS reserved_product_id,
		d.material_code,
		d.product_name,
		COALESCE(p.barcode, '')          AS barcode,
		COALESCE(p.unit_per_pack, 1)     AS unit_per_pack,
		d.quantity,
		d.checked_quantity,
		COALESCE(st.available_quantity, 0) AS available_quantity,
		COALESCE(d.created_by::text, '')  AS created_by,
		COALESCE(d.updated_by::text, '')  AS updated_by,
		COALESCE(ue.full_name, '')        AS updated_by_name,
		d.created_at,
		d.updated_at
	FROM reserve_details d
	JOIN reserves r        ON r.id = d.reserve_id
	LEFT JOIN products p   ON p.id = d.product_id
	LEFT JOIN employees ue ON ue.id = d.updated_by
	LEFT JOIN LATERAL (
		SELECT COALESCE(SUM(sp.unit_quantity), 0) AS available_quantity
		FROM store_products sp
		WHERE sp.product_id = d.product_id AND sp.store_id = r.store_id
	) st ON TRUE
`

// region Create

// CreateReserve — hujjat va (berilgan bo'lsa) qatorlarini bitta tranzaksiyada yaratadi.
func (s *Services) CreateReserve(
	ctx context.Context, req *domain.ReserveRequest, userId string,
) (*domain.Reserve, error) {
	if req.StoreId == "" {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.store_id_required")
	}

	var reserveId string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// dok_number bo'sh bo'lsa bazadagi sequence default'i ishlaydi.
		query := `
			INSERT INTO reserves (dok_number, store_id, comment, created_by, updated_by)
			VALUES (COALESCE(NULLIF(?, ''), 'RZ-' || nextval('reserves_dok_number_seq')), ?, NULLIF(?, ''), NULLIF(?, '')::uuid, NULLIF(?, '')::uuid)
			RETURNING id`

		if err := tx.Raw(query,
			strings.TrimSpace(req.DokNumber), req.StoreId, strings.TrimSpace(req.Comment), userId, userId,
		).Row().Scan(&reserveId); err != nil {
			if isUniqueViolation(err) {
				return domain.AlreadyExistsError
			}
			s.log.Errorf("reserve: could not create: %v", err)
			return domain.InternalServerError
		}

		if err := s.upsertReserveDetails(ctx, tx, reserveId, req.Items, userId); err != nil {
			return err
		}

		return s.refreshReserveTotals(ctx, tx, reserveId, userId)
	})
	if err != nil {
		return nil, err
	}

	return s.GetReserveById(ctx, reserveId)
}

// upsertReserveDetails — qatorlarni bitta statement bilan yozadi: mahsulot nomi va
// material_code products'dan snapshot qilinadi, 1C ro'yxatidagi qator ham bog'lanadi.
// Bitta requestda bir mahsulot ikki marta kelsa oxirgisi qoladi (ON CONFLICT bitta
// qatorga ikki marta tegolmaydi).
func (s *Services) upsertReserveDetails(
	ctx context.Context, tx *gorm.DB, reserveId string, items []domain.ReserveItemRequest, userId string,
) error {
	if len(items) == 0 {
		return nil
	}

	productIds := make([]string, 0, len(items))
	quantities := make([]float64, 0, len(items))
	position := make(map[string]int, len(items))

	for i, item := range items {
		productId := strings.TrimSpace(item.ProductId)
		if productId == "" {
			return domain.NewError(http.StatusBadRequest, fmt.Sprintf("reserve.empty_product_id: position=%d", i+1))
		}
		if item.Quantity < 0 {
			return domain.NewError(http.StatusBadRequest, fmt.Sprintf("reserve.invalid_quantity: position=%d", i+1))
		}

		if idx, ok := position[productId]; ok {
			quantities[idx] = item.Quantity
			continue
		}
		position[productId] = len(productIds)
		productIds = append(productIds, productId)
		quantities = append(quantities, item.Quantity)
	}

	query := `
		INSERT INTO reserve_details (
			reserve_id, product_id, reserved_product_id, material_code, product_name, quantity, created_by, updated_by
		)
		SELECT
			?::uuid,
			p.id,
			rp.id,
			COALESCE(p.material_code::text, ''),
			COALESCE(p.name, ''),
			t.quantity,
			NULLIF(?, '')::uuid,
			NULLIF(?, '')::uuid
		FROM unnest(?::uuid[], ?::numeric[]) AS t(product_id, quantity)
		JOIN products p ON p.id = t.product_id AND p.deleted_at IS NULL
		LEFT JOIN reserved_products rp ON rp.material_code = p.material_code::text
		ON CONFLICT (reserve_id, product_id) DO UPDATE SET
			quantity   = EXCLUDED.quantity,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING 1`

	var written []int
	result := tx.WithContext(ctx).Raw(query,
		reserveId, userId, userId, pq.Array(productIds), pq.Array(quantities),
	).Scan(&written)
	if result.Error != nil {
		s.log.Errorf("reserve: could not upsert %d details: %v", len(productIds), result.Error)
		return domain.InternalServerError
	}
	// JOIN products topilmagan qatorni tashlab ketadi — bunday holatda hujjat yarim qolmasligi
	// uchun butun tranzaksiya bekor qilinadi.
	if len(written) != len(productIds) {
		return domain.NewError(http.StatusBadRequest, "reserve.product_not_found")
	}

	return nil
}

// refreshReserveTotals — jami miqdor va mahsulot soni har doim qatorlardan hisoblanadi.
func (s *Services) refreshReserveTotals(ctx context.Context, tx *gorm.DB, reserveId, userId string) error {
	query := `
		UPDATE reserves SET
			total_quantity      = COALESCE((SELECT SUM(quantity) FROM reserve_details WHERE reserve_id = reserves.id), 0),
			total_product_count = (SELECT COUNT(*) FROM reserve_details WHERE reserve_id = reserves.id),
			updated_by          = COALESCE(NULLIF(?, '')::uuid, updated_by),
			updated_at          = NOW()
		WHERE id = ?`

	if err := tx.WithContext(ctx).Exec(query, userId, reserveId).Error; err != nil {
		s.log.Errorf("reserve: could not refresh totals of %s: %v", reserveId, err)
		return domain.InternalServerError
	}

	return nil
}

// region Get

func (s *Services) GetReserves(
	ctx context.Context, params *domain.ReserveQueryParams,
) ([]domain.Reserve, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}

	if params.StoreId != "" {
		where = append(where, "r.store_id = ?")
		args = append(args, params.StoreId)
	}
	if params.Status != "" {
		where = append(where, "r.status = ?")
		args = append(args, params.Status)
	}
	if params.Search != "" {
		where = append(where, "(r.dok_number ILIKE ? OR r.comment ILIKE ?)")
		args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
	}
	// Sana filtri Toshkent kunida: created_at timestamptz, UTC'da kesilsa kechqurungi
	// hujjatlar keyingi kunga tushib qolardi.
	if params.StartDate != "" {
		where = append(where, "(r.created_at AT TIME ZONE 'Asia/Tashkent')::date >= ?::date")
		args = append(args, params.StartDate)
	}
	if params.EndDate != "" {
		where = append(where, "(r.created_at AT TIME ZONE 'Asia/Tashkent')::date <= ?::date")
		args = append(args, params.EndDate)
	}
	condition := strings.Join(where, " AND ")

	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM reserves r WHERE ` + condition
	if err := s.db.WithContext(ctx).Raw(countQuery, args...).Row().Scan(&totalCount); err != nil {
		s.log.Errorf("reserve: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	listArgs := append(append([]any{}, args...), params.Limit, params.Offset)
	listQuery := reserveSelectSQL + ` WHERE ` + condition + ` ORDER BY r.created_at DESC LIMIT ? OFFSET ?`

	var reserves []domain.Reserve
	if err := s.db.WithContext(ctx).Raw(listQuery, listArgs...).Scan(&reserves).Error; err != nil {
		s.log.Errorf("reserve: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return reserves, totalCount, nil
}

// GetReserveById — hujjat va uning barcha qatorlari.
func (s *Services) GetReserveById(ctx context.Context, id string) (*domain.Reserve, error) {
	var reserve domain.Reserve
	result := s.db.WithContext(ctx).Raw(reserveSelectSQL+` WHERE r.id = ?`, id).Scan(&reserve)
	if result.Error != nil {
		s.log.Errorf("reserve: could not get %s: %v", id, result.Error)
		return nil, domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return nil, domain.NotFoundError
	}

	details, _, err := s.GetReserveDetails(ctx, &domain.ReserveDetailQueryParams{ReserveId: id})
	if err != nil {
		return nil, err
	}
	reserve.Details = details

	return &reserve, nil
}

// GetReserveDetails — hujjat qatorlari. Limit berilmasa hammasi qaytadi.
func (s *Services) GetReserveDetails(
	ctx context.Context, params *domain.ReserveDetailQueryParams,
) ([]domain.ReserveDetail, int64, error) {
	where := []string{"d.reserve_id = ?"}
	args := []any{params.ReserveId}

	if params.Search != "" {
		where = append(where, "(d.product_name ILIKE ? OR d.material_code ILIKE ?)")
		args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
	}
	condition := strings.Join(where, " AND ")

	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM reserve_details d WHERE ` + condition
	if err := s.db.WithContext(ctx).Raw(countQuery, args...).Row().Scan(&totalCount); err != nil {
		s.log.Errorf("reserve detail: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	listQuery := reserveDetailSelectSQL + ` WHERE ` + condition + ` ORDER BY d.created_at ASC, d.id ASC`
	if params.Limit > 0 {
		listQuery += ` LIMIT ? OFFSET ?`
		args = append(args, params.Limit, params.Offset)
	}

	var details []domain.ReserveDetail
	if err := s.db.WithContext(ctx).Raw(listQuery, args...).Scan(&details).Error; err != nil {
		s.log.Errorf("reserve detail: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return details, totalCount, nil
}

// region Update

// UpdateReserveStatus — new -> checking -> done. done bo'lgan hujjat qayta o'zgarmaydi.
func (s *Services) UpdateReserveStatus(
	ctx context.Context, id, status, userId string,
) (*domain.Reserve, error) {
	if !isValidReserveStatus(status) {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.invalid_status")
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.lockReserveStatus(ctx, tx, id)
		if err != nil {
			return err
		}
		if current == constants.GeneralStatusDone {
			return domain.AlreadyCompletedError
		}

		// done bo'lganda completed_at yoziladi, orqaga qaytarilsa tozalanadi.
		query := `
			UPDATE reserves SET
				status       = ?,
				completed_at = CASE WHEN ? = 'done' THEN NOW() ELSE NULL END,
				updated_by   = COALESCE(NULLIF(?, '')::uuid, updated_by),
				updated_at   = NOW()
			WHERE id = ?`

		if err := tx.WithContext(ctx).Exec(query, status, status, userId, id).Error; err != nil {
			s.log.Errorf("reserve: could not update status of %s: %v", id, err)
			return domain.InternalServerError
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetReserveById(ctx, id)
}

// UpsertReserveDetail — hujjatga mahsulot qo'shadi yoki miqdorini yangilaydi.
func (s *Services) UpsertReserveDetail(
	ctx context.Context, req *domain.ReserveDetailRequest, userId string,
) (*domain.ReserveDetail, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.checkReserveEditable(ctx, tx, req.ReserveId); err != nil {
			return err
		}

		items := []domain.ReserveItemRequest{{ProductId: req.ProductId, Quantity: req.Quantity}}
		if err := s.upsertReserveDetails(ctx, tx, req.ReserveId, items, userId); err != nil {
			return err
		}

		return s.refreshReserveTotals(ctx, tx, req.ReserveId, userId)
	})
	if err != nil {
		return nil, err
	}

	return s.getReserveDetailByProduct(ctx, req.ReserveId, req.ProductId)
}

// UpdateReserveDetail — faqat berilgan maydonlarni yangilaydi (quantity, checked_quantity).
func (s *Services) UpdateReserveDetail(
	ctx context.Context, id string, req *domain.ReserveDetailUpdateRequest, userId string,
) (*domain.ReserveDetail, error) {
	if req.Quantity == nil && req.CheckedQuantity == nil {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.nothing_to_update")
	}
	if (req.Quantity != nil && *req.Quantity < 0) || (req.CheckedQuantity != nil && *req.CheckedQuantity < 0) {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.invalid_quantity")
	}

	var reserveId string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Raw(`SELECT reserve_id FROM reserve_details WHERE id = ?`, id).
			Row().Scan(&reserveId); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.NotFoundError
			}
			s.log.Errorf("reserve detail: could not get %s: %v", id, err)
			return domain.InternalServerError
		}

		if err := s.checkReserveEditable(ctx, tx, reserveId); err != nil {
			return err
		}

		query := `
			UPDATE reserve_details SET
				quantity         = COALESCE(?::numeric, quantity),
				checked_quantity = COALESCE(?::numeric, checked_quantity),
				updated_by       = COALESCE(NULLIF(?, '')::uuid, updated_by),
				updated_at       = NOW()
			WHERE id = ?`

		if err := tx.WithContext(ctx).Exec(query, req.Quantity, req.CheckedQuantity, userId, id).Error; err != nil {
			s.log.Errorf("reserve detail: could not update %s: %v", id, err)
			return domain.InternalServerError
		}

		return s.refreshReserveTotals(ctx, tx, reserveId, userId)
	})
	if err != nil {
		return nil, err
	}

	var detail domain.ReserveDetail
	if err := s.db.WithContext(ctx).Raw(reserveDetailSelectSQL+` WHERE d.id = ?`, id).Scan(&detail).Error; err != nil {
		s.log.Errorf("reserve detail: could not read %s: %v", id, err)
		return nil, domain.InternalServerError
	}

	return &detail, nil
}

// QuickAddReserveDetail — bitta so'rov bilan qo'shish: frontend faqat do'kon, mahsulot va
// miqdorni yuboradi. Do'konning ochiq hujjati (status != done) topiladi, bo'lmasa yangisi
// ochiladi; mahsulot allaqachon bo'lsa miqdor USTIGA QO'SHILADI (almashtirilmaydi).
//
// Butun amal bitta tranzaksiyada va do'kon bo'yicha advisory lock ostida: ikkita parallel
// so'rov bitta do'konga ikkita hujjat ochib yuborishi mumkin emas.
func (s *Services) QuickAddReserveDetail(
	ctx context.Context, req *domain.ReserveQuickDetailRequest, userId string,
) (*domain.ReserveDetail, error) {
	if req.StoreId == "" {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.store_id_required")
	}
	if req.Quantity <= 0 {
		return nil, domain.NewError(http.StatusBadRequest, "reserve.invalid_quantity")
	}

	var reserveId string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "reserve_quick:"+req.StoreId).Error; err != nil {
			s.log.Errorf("reserve: could not acquire quick add lock: %v", err)
			return domain.InternalServerError
		}

		// Ochiq hujjat: eng oxirgi yaratilgani, done bo'lmagani.
		err := tx.WithContext(ctx).Raw(`
			SELECT id FROM reserves
			WHERE store_id = ? AND status <> ?
			ORDER BY created_at DESC
			LIMIT 1
			FOR UPDATE
		`, req.StoreId, constants.GeneralStatusDone).Row().Scan(&reserveId)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.log.Errorf("reserve: could not find open document of store %s: %v", req.StoreId, err)
			return domain.InternalServerError
		}

		if reserveId == "" {
			createQuery := `
				INSERT INTO reserves (dok_number, store_id, created_by, updated_by)
				VALUES ('RZ-' || nextval('reserves_dok_number_seq'), ?, NULLIF(?, '')::uuid, NULLIF(?, '')::uuid)
				RETURNING id`

			if err := tx.WithContext(ctx).Raw(createQuery, req.StoreId, userId, userId).
				Row().Scan(&reserveId); err != nil {
				s.log.Errorf("reserve: could not create document for store %s: %v", req.StoreId, err)
				return domain.InternalServerError
			}
		}

		addQuery := `
			INSERT INTO reserve_details (
				reserve_id, product_id, reserved_product_id, material_code, product_name, quantity, created_by, updated_by
			)
			SELECT
				?::uuid,
				p.id,
				rp.id,
				COALESCE(p.material_code::text, ''),
				COALESCE(p.name, ''),
				?::numeric,
				NULLIF(?, '')::uuid,
				NULLIF(?, '')::uuid
			FROM products p
			LEFT JOIN reserved_products rp ON rp.material_code = p.material_code::text
			WHERE p.id = ?::uuid AND p.deleted_at IS NULL
			ON CONFLICT (reserve_id, product_id) DO UPDATE SET
				quantity   = reserve_details.quantity + EXCLUDED.quantity,
				updated_by = EXCLUDED.updated_by,
				updated_at = NOW()
			RETURNING id`

		var detailIds []string
		result := tx.WithContext(ctx).Raw(addQuery,
			reserveId, req.Quantity, userId, userId, req.ProductId,
		).Scan(&detailIds)
		if result.Error != nil {
			s.log.Errorf("reserve: could not quick add product %s: %v", req.ProductId, result.Error)
			return domain.InternalServerError
		}
		if len(detailIds) == 0 {
			return domain.NewError(http.StatusBadRequest, "reserve.product_not_found")
		}

		return s.refreshReserveTotals(ctx, tx, reserveId, userId)
	})
	if err != nil {
		return nil, err
	}

	return s.getReserveDetailByProduct(ctx, reserveId, req.ProductId)
}

// region Delete

// DeleteReserve — faqat 'new' holatidagi hujjat o'chiriladi, qatorlar CASCADE bilan ketadi.
func (s *Services) DeleteReserve(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.lockReserveStatus(ctx, tx, id)
		if err != nil {
			return err
		}
		if current != constants.GeneralStatusNew {
			return domain.NewError(http.StatusConflict, "reserve.only_new_can_be_deleted")
		}

		if err := tx.WithContext(ctx).Exec(`DELETE FROM reserves WHERE id = ?`, id).Error; err != nil {
			s.log.Errorf("reserve: could not delete %s: %v", id, err)
			return domain.InternalServerError
		}

		return nil
	})
}

func (s *Services) DeleteReserveDetail(ctx context.Context, id, userId string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reserveId string
		err := tx.WithContext(ctx).
			Raw(`DELETE FROM reserve_details WHERE id = ? RETURNING reserve_id`, id).
			Row().Scan(&reserveId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.NotFoundError
			}
			s.log.Errorf("reserve detail: could not delete %s: %v", id, err)
			return domain.InternalServerError
		}

		return s.refreshReserveTotals(ctx, tx, reserveId, userId)
	})
}

// region Helpers

// lockReserveStatus — hujjatni FOR UPDATE bilan qulflab statusini o'qiydi: parallel
// so'rovlar bir vaqtda statusni o'zgartirib yubormasligi uchun.
func (s *Services) lockReserveStatus(ctx context.Context, tx *gorm.DB, id string) (string, error) {
	var status string
	err := tx.WithContext(ctx).Raw(`SELECT status FROM reserves WHERE id = ? FOR UPDATE`, id).Row().Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.NotFoundError
		}
		s.log.Errorf("reserve: could not lock %s: %v", id, err)
		return "", domain.InternalServerError
	}

	return status, nil
}

// checkReserveEditable — yakunlangan hujjatning qatorlari o'zgartirilmaydi.
func (s *Services) checkReserveEditable(ctx context.Context, tx *gorm.DB, reserveId string) error {
	status, err := s.lockReserveStatus(ctx, tx, reserveId)
	if err != nil {
		return err
	}
	if status == constants.GeneralStatusDone {
		return domain.AlreadyCompletedError
	}

	return nil
}

func (s *Services) getReserveDetailByProduct(ctx context.Context, reserveId, productId string) (*domain.ReserveDetail, error) {
	var detail domain.ReserveDetail
	err := s.db.WithContext(ctx).
		Raw(reserveDetailSelectSQL+` WHERE d.reserve_id = ? AND d.product_id = ?`, reserveId, productId).
		Scan(&detail).Error
	if err != nil {
		s.log.Errorf("reserve detail: could not read (%s, %s): %v", reserveId, productId, err)
		return nil, domain.InternalServerError
	}

	return &detail, nil
}

func isValidReserveStatus(status string) bool {
	switch status {
	case constants.GeneralStatusNew, constants.GeneralStatusChecking, constants.GeneralStatusDone:
		return true
	}

	return false
}
