package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

// create store on importing products to branch
func (s *Services) CreateStoreOnImport(ctx context.Context, tx *gorm.DB, req *domain.StoreRequest) (domain.Store, error) {
	var res domain.Store
	query := `INSERT INTO stores(name, detailed_name, store_code, company_id) VALUES(?, ?, ?, ?) RETURNING *`
	err := tx.WithContext(ctx).Raw(query, req.Name, req.Name, req.StoreCode, req.CompanyId).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not create new store on importing: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

// get store info by import id
func (s *Services) GetStoreByImportId(ctx context.Context, tx *gorm.DB, importId string) (*domain.Store, error) {
	var store domain.Store
	err := tx.WithContext(ctx).Raw(`SELECT stores.* FROM imports JOIN stores ON stores.id = imports.store_id WHERE imports.id = ?`, importId).Scan(&store).Error
	if err != nil {
		s.log.Errorf("could not get store by import id: %v", err)
		return nil, domain.InternalServerError
	}

	return &store, nil
}

// get store info by field and value
func (s *Services) GetStoreByField(field string, value string) (*domain.Store, error) {
	var store domain.Store
	query := fmt.Sprintf("SELECT * FROM stores WHERE %s = ?", field)
	err := s.db.Raw(query, value).Scan(&store).Error
	// check if store is found
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.NotFoundError
	}
	// handle error
	if err != nil {
		s.log.Errorf("could not get store info by field: %v", err)
		return nil, domain.InternalServerError
	}
	return &store, nil
}

func (s *Services) GetStores(ctx context.Context, params *domain.StoreQueryParams) ([]domain.StoreDto, int64, []string, error) {
	qb := s.db.WithContext(ctx).
		Model(&domain.Store{}).
		Joins(`LEFT JOIN store_targets st
			ON st.store_id = stores.id
			AND st.year  = EXTRACT(YEAR  FROM NOW())
			AND st.month = EXTRACT(MONTH FROM NOW())`).
		Select(
			"stores.id",
			"store_code",
			"name",
			"detailed_name",
			"stores.company_id",
			"phone",
			"contact",
			"inn",
			"employee_count",
			"cash_box_count",
			"address",
			"location",
			"terminal_id",
			"ST_AsText(coordinates) AS coordinates",
			"work_hours",
			"is_fullday",
			"COALESCE(st.amount, 0) AS target_amount",
			"average_target_sales",
			"stores.created_at",
			"stores.updated_at",
		)

	if params.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("name ILIKE ? OR detailed_name ILIKE ?", searchPattern, searchPattern)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("stores.company_id IN (?)", params.CompanyIds)
	} else if params.CompanyId != "" {
		qb = qb.Where("stores.company_id = ?", params.CompanyId)
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Where("stores.id IN (?)", params.StoreIds)
	} else if params.StoreId != "" {
		qb = qb.Where("stores.id = ?", params.StoreId)
	}

	if params.IsFranchise != nil {
		qb = qb.Where("stores.company_id IN (SELECT id FROM companies WHERE is_franchise = ?)", *params.IsFranchise)
	}

	if params.IsOnlineOrder != nil {
		qb = qb.Where("stores.is_online_order = ?", *params.IsOnlineOrder)
	}

	totalCount := int64(0)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count stores: %v", err)
		return nil, 0, []string{}, domain.InternalServerError
	}

	if params.Limit > 0 {
		qb = qb.Limit(params.Limit)
	}

	if params.Offset > 0 {
		qb = qb.Offset(params.Offset)
	}
	var stores []domain.StoreDto
	err := qb.Order("created_at DESC").Find(&stores).Error
	if err != nil {
		s.log.Errorf("could not get stores: %v", err)
		return nil, 0, []string{}, domain.InternalServerError
	}

	var ids []string
	err = s.db.WithContext(ctx).Table("stores").Select("id").Find(&ids).Error
	if err != nil {
		s.log.Errorf("could not get store ids: %v", err)
		return nil, 0, []string{}, domain.InternalServerError
	}

	return stores, totalCount, ids, nil
}

// storeEmployeeCountQuery builds the shared base query behind the employee
// count list and its statistics, so both always cover the same set of stores.

func (s *Services) storeEmployeeCountQuery(ctx context.Context, params *domain.StoreEmployeeCountQueryParams) *gorm.DB {
	qb := s.db.WithContext(ctx).
		Model(&domain.Store{}).
		// Xodim doirasi oylik hisoboti (payrollManagementFilterSQL) bilan AYNAN
		// bir xil bo'lishi shart: ikkala ekran bir xil son ko'rsatishi kerak.
		// Shu sababli status va rol filtri bu yerda ham qo'llanadi — savdo bilan
		// bog'liq bo'lmagan xodimlar (menejer, buxgalter) sanoqqa kirmaydi.
		Joins(`
			LEFT JOIN (
				SELECT
					e.store_id,
					COUNT(*) AS employee_count
				FROM employees e
				WHERE e.deleted_at IS NULL
				  AND e.is_active = TRUE
				  AND e.store_id IS NOT NULL
				  AND e.status = ?
				  AND EXISTS (
					  SELECT 1
					  FROM employee_roles er
					  JOIN roles r ON r.id = er.role_id
					  WHERE er.employee_id = e.id
					    AND r.name IN (?, ?)
				  )
				GROUP BY e.store_id
			) actual
				ON actual.store_id = stores.id
		`, constants.GeneralStatusActive, constants.RoleNameCashier, constants.RoleNameZavStore).
		Joins(`
			LEFT JOIN (
				SELECT
					store_id,
					COUNT(*) AS cash_box_count
				FROM cash_boxes
				WHERE deleted_at IS NULL
				  AND is_active = TRUE
				  AND store_id IS NOT NULL
				GROUP BY store_id
			) cash_box
				ON cash_box.store_id = stores.id
		`).
		Where("stores.deleted_at IS NULL").
		Where("stores.is_active = TRUE")

	if params.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("stores.name ILIKE ? OR stores.detailed_name ILIKE ?", searchPattern, searchPattern)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("stores.company_id IN (?)", params.CompanyIds)
	} else if params.CompanyId != "" {
		qb = qb.Where("stores.company_id = ?", params.CompanyId)
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Where("stores.id IN (?)", params.StoreIds)
	} else if params.StoreId != "" {
		qb = qb.Where("stores.id = ?", params.StoreId)
	}

	// Franshiza do'konlari bu hisobotga UMUMAN kirmaydi — shart ixtiyoriy emas.
	// Kompaniyasi biriktirilmagan do'kon ham chiqmaydi: uning franshiza ekanini
	// aniqlab bo'lmaydi (NULL IN (...) hech qachon rost bo'lmaydi).
	//
	// DIQQAT: shu sababli ?is_franchise=true parametri endi doim bo'sh natija
	// beradi — u bu ikki endpointda ma'nosini yo'qotdi.
	qb = qb.Where("stores.company_id IN (SELECT id FROM companies WHERE is_franchise = false)")

	return qb
}

// GetStoreEmployeeCounts returns the planned headcount (stores.employee_count)
// of every store next to the number of employees actually assigned to it.
func (s *Services) GetStoreEmployeeCounts(ctx context.Context, params *domain.StoreEmployeeCountQueryParams) ([]domain.StoreEmployeeCount, int64, error) {
	qb := s.storeEmployeeCountQuery(ctx, params).
		Select(`
			stores.id,
			stores.store_code,
			stores.name,
			stores.company_id,
			COALESCE(stores.employee_count, 0) AS employee_count,
			COALESCE(cash_box.cash_box_count, 0) AS cash_box_count,
			COALESCE(actual.employee_count, 0) AS actual_employee_count,
			COALESCE(actual.employee_count, 0) - COALESCE(stores.employee_count, 0) AS difference
		`)

	totalCount := int64(0)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count store employee counts: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if params.Limit > 0 {
		qb = qb.Limit(params.Limit)
	}

	if params.Offset > 0 {
		qb = qb.Offset(params.Offset)
	}

	res := []domain.StoreEmployeeCount{}
	if err := qb.Order("stores.store_code ASC").Find(&res).Error; err != nil {
		s.log.Errorf("could not get store employee counts: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// GetStoreEmployeeCountStat aggregates the employee count list into a single
// row. It reuses storeEmployeeCountQuery, so the totals always match the list
// under the same filters — pagination aside.
func (s *Services) GetStoreEmployeeCountStat(ctx context.Context, params *domain.StoreEmployeeCountQueryParams) (*domain.StoreEmployeeCountStat, error) {
	var stat domain.StoreEmployeeCountStat

	err := s.storeEmployeeCountQuery(ctx, params).
		Select(`
			COUNT(*) AS total_stores,
			COALESCE(SUM(stores.employee_count), 0) AS total_plan_employees,
			COALESCE(SUM(actual.employee_count), 0) AS actual_employee_count,
			COALESCE(SUM(actual.employee_count), 0) - COALESCE(SUM(stores.employee_count), 0) AS total_diff
		`).
		Scan(&stat).Error
	if err != nil {
		s.log.Errorf("could not get store employee count stat: %v", err)
		return nil, domain.InternalServerError
	}

	return &stat, nil
}

func (s *Services) UpdateAverateStoreTargetSales() error {
	err := s.db.Exec(`
		UPDATE stores
		SET average_target_sales = sub.avg_sales
		FROM (
			SELECT store_id,
				SUM(monthly_total) / COUNT(*) AS avg_sales
			FROM (
				SELECT store_id,
					DATE_TRUNC('month', created_at) AS month,
					SUM(total_amount) AS monthly_total
				FROM sales
				GROUP BY store_id, DATE_TRUNC('month', created_at)
			) monthly_per_store
			GROUP BY store_id
		) sub
		WHERE stores.id = sub.store_id;
	`).Error

	if err != nil {
		s.log.Errorf("could not update average target sales for stores: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) GetAllStoreMapInfo(ctx context.Context, params *domain.StoreMapInfoQueryParams) ([]domain.StoreMapInfo, error) {
	qb := s.db.WithContext(ctx).
		Model(&domain.Store{}).
		Select(`
			stores.id,
			stores.name,
			stores.address,
			stores.store_code,
			stores.inn,
			stores.work_hours,
			stores.phone,
			stores.is_online_order,
			stores.created_at,
			stores.updated_at,

			ST_AsText(stores.coordinates) AS coordinates,

			COALESCE(attendance.is_open, false) AS is_open
		`).
		Joins(`
			LEFT JOIN (
				SELECT
					latest.store_id,
					TRUE AS is_open
				FROM (
					SELECT DISTINCT ON (al.employee_id)
						al.employee_id, al.store_id, al.event_type
					FROM attendance_logs al
					JOIN employees e ON e.id = al.employee_id
						AND e.deleted_at IS NULL
						AND e.is_active = TRUE
						AND e.status = ?
						AND EXISTS (
							SELECT 1
							FROM employee_roles er
							JOIN roles r ON r.id = er.role_id
							WHERE er.employee_id = e.id
							  AND r.name IN (?, ?)
						)
					WHERE al.store_id IS NOT NULL
					  AND (al.event_at AT TIME ZONE 'Asia/Tashkent')::date =
					      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
					ORDER BY al.employee_id, al.event_at DESC, al.id DESC
				) latest
					WHERE latest.event_type = 'check-in'
					GROUP BY latest.store_id
			) attendance
				ON attendance.store_id = stores.id
		`, constants.GeneralStatusActive, constants.RoleNameCashier, constants.RoleNameZavStore).
		Where("stores.deleted_at IS NULL").
		Where("stores.is_active = TRUE").
		Where("stores.coordinates IS NOT NULL")

	if params.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", params.Search)

		qb = qb.Where(
			"stores.name ILIKE ? OR stores.detailed_name ILIKE ?",
			searchPattern,
			searchPattern,
		)
	}

	// is_pharma va is_franchise — guruh tanlash filtri, bir-birini inkor qilmaydi:
	//
	//	is_pharma=true                  → faqat franchise bo'lmagan aptekalar
	//	is_franchise=true               → faqat franchise do'konlar
	//	ikkalasi ham true               → ikkala guruh, ya'ni hammasi
	//	hech biri berilmagan            → hammasi
	//
	// Shuning uchun ular AND bilan emas, tanlangan guruhlar to'plami sifatida
	// birlashtiriladi: ikkala guruh tanlansa filtr umuman qo'yilmaydi (kompaniyasi
	// biriktirilmagan do'konlar ham tushib qolmasligi uchun).
	franchiseGroups := map[bool]bool{}
	if params.IsFranchise != nil {
		franchiseGroups[*params.IsFranchise] = true
	}
	if params.IsPharma != nil {
		franchiseGroups[!*params.IsPharma] = true
	}

	if len(franchiseGroups) == 1 {
		for isFranchise := range franchiseGroups {
			qb = qb.Where(`
				stores.company_id IN (
					SELECT id
					FROM companies
					WHERE is_franchise = ?
				)
			`, isFranchise)
		}
	}

	if params.IsOnline != nil {
		qb = qb.Where(
			"stores.is_online_order = ?",
			*params.IsOnline,
		)
	}

	var stores []domain.StoreMapInfo

	err := qb.
		Order("stores.created_at DESC").
		Find(&stores).Error

	if err != nil {
		s.log.Errorf(
			"could not get all store map info: %v",
			err,
		)
		return nil, domain.InternalServerError
	}

	return stores, nil
}

func (s *Services) GetStoreByIdMapInfo(ctx context.Context, storeId string) (*domain.StoreMapInfoDetail, error) {
	qb := s.db.WithContext(ctx).
		Model(&domain.Store{}).
		Select(`
			stores.id,
			stores.name,
			stores.address,
			stores.store_code,
			stores.inn,
			stores.work_hours,
			stores.phone,
			stores.is_online_order,
			stores.created_at,
			stores.updated_at,

			ST_AsText(stores.coordinates) AS coordinates,

			COALESCE(sales.sales_amount, 0) AS sales_amount,
			COALESCE(sales.sales_count, 0) AS sales_count,
			COALESCE(sales.sales_aggregate_sum, 0) AS sales_aggregate_sum,
			COALESCE(sales.average_sales_amount, 0) AS average_sales_amount,

			COALESCE(cash_boxes.cash_box_count, 0) AS cash_box_count,

			COALESCE(employees.employee_count, 0) AS employee_count,

			COALESCE(attendance.is_open, false) AS is_open,
			attendance.opened_at,
			attendance.closed_at
		`).
		Joins(`
			LEFT JOIN (
				SELECT
					store_id,
					SUM(total_amount) AS sales_amount,
					COUNT(*) AS sales_count,
					SUM(total_amount) AS sales_aggregate_sum,
					SUM(total_amount) / NULLIF(COUNT(*), 0) AS average_sales_amount
				FROM sales
				WHERE created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
				  AND created_at < (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date + INTERVAL '1 day'
				  AND store_id IS NOT NULL
				  AND is_active = true
				GROUP BY store_id
			) sales
				ON sales.store_id = stores.id
		`).
		Joins(`
			LEFT JOIN (
				SELECT
					store_id,
					COUNT(*) AS cash_box_count
				FROM cash_boxes
				WHERE is_active = true
				GROUP BY store_id
			) cash_boxes
				ON cash_boxes.store_id = stores.id
		`).
		Joins(`
			LEFT JOIN (
				SELECT
					store_id,
					COUNT(*) AS employee_count
				FROM employees
				WHERE deleted_at IS NULL
				  AND is_active = true
				  AND store_id IS NOT NULL
				GROUP BY store_id
			) employees
				ON employees.store_id = stores.id
		`).
		Joins(`
			LEFT JOIN (
				SELECT
					timeline.store_id,
					COALESCE(open_state.is_open, false) AS is_open,
					-- opened_at — bugun birinchi kelgan xodimning check-in vaqti
					timeline.opened_at,
					-- closed_at — eng oxirgi ketgan xodimning check-out vaqti, lekin
					-- faqat do'kon haqiqatan yopilgan bo'lsa. Do'kon hali ochiq bo'lsa
					-- (kimdir ishlayotgan bo'lsa) bu NULL: aks holda erta ketgan bitta
					-- xodimning vaqti do'kon yopilgandek ko'rinardi.
					CASE
						WHEN COALESCE(open_state.is_open, false) THEN NULL
						ELSE timeline.closed_at
					END AS closed_at
				FROM (
					SELECT
						al.store_id,
						MIN(al.event_at) FILTER (WHERE al.event_type = 'check-in') AS opened_at,
						MAX(al.event_at) FILTER (WHERE al.event_type = 'check-out') AS closed_at
					FROM attendance_logs al
					JOIN employees e ON e.id = al.employee_id
						AND e.deleted_at IS NULL
						AND e.is_active = TRUE
						AND e.status = ?
						AND EXISTS (
							SELECT 1
							FROM employee_roles er
							JOIN roles r ON r.id = er.role_id
							WHERE er.employee_id = e.id
							  AND r.name IN (?, ?)
						)
					WHERE al.store_id IS NOT NULL
					  AND (al.event_at AT TIME ZONE 'Asia/Tashkent')::date =
					      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
					GROUP BY al.store_id
				) timeline
				LEFT JOIN (
					SELECT latest.store_id, TRUE AS is_open
					FROM (
						SELECT DISTINCT ON (al.employee_id)
							al.employee_id, al.store_id, al.event_type
						FROM attendance_logs al
						JOIN employees e ON e.id = al.employee_id
							AND e.deleted_at IS NULL
							AND e.is_active = TRUE
							AND e.status = ?
							AND EXISTS (
								SELECT 1
								FROM employee_roles er
								JOIN roles r ON r.id = er.role_id
								WHERE er.employee_id = e.id
								  AND r.name IN (?, ?)
							)
						WHERE al.store_id IS NOT NULL
						  AND (al.event_at AT TIME ZONE 'Asia/Tashkent')::date =
						      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
						ORDER BY al.employee_id, al.event_at DESC, al.id DESC
					) latest
					WHERE latest.event_type = 'check-in'
					GROUP BY latest.store_id
				) open_state ON open_state.store_id = timeline.store_id
			) attendance
				ON attendance.store_id = stores.id
		`,
			constants.GeneralStatusActive, constants.RoleNameCashier, constants.RoleNameZavStore,
			constants.GeneralStatusActive, constants.RoleNameCashier, constants.RoleNameZavStore,
		).
		Where("stores.id = ?", storeId)

	var storeMapInfo domain.StoreMapInfoDetail

	err := qb.
		Order("stores.created_at DESC").
		Find(&storeMapInfo).Error

	if err != nil {
		s.log.Errorf(
			"could not get store map info by id: %v",
			err,
		)
		return nil, domain.InternalServerError
	}

	storeMapInfo.Employees = make([]domain.StoreMapInfoEmployee, 0)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			e.id,
			e.full_name,
			today.check_in_at,
			today.check_out_at,
			last_event.event_type AS last_event_type,
			last_event.event_at   AS last_event_at
		FROM employees e
		-- bugungi eng erta kelish va eng kech ketish vaqti
		LEFT JOIN LATERAL (
			SELECT
				MIN(al.event_at) FILTER (WHERE al.event_type = 'check-in')  AS check_in_at,
				MAX(al.event_at) FILTER (WHERE al.event_type = 'check-out') AS check_out_at
			FROM attendance_logs al
			WHERE al.employee_id = e.id
			  AND (al.event_at AT TIME ZONE 'Asia/Tashkent')::date =
			      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
		) today ON TRUE
		-- bugungi eng oxirgi voqea — xodim hozir ishdami yoki yo'qmi shundan bilinadi.
		-- Tartib is_open hisoblanishidagi bilan bir xil (event_at DESC, id DESC),
		-- shuning uchun xodim holati do'kon holatiga zid bo'lib qolmaydi.
		LEFT JOIN LATERAL (
			SELECT al.event_type, al.event_at
			FROM attendance_logs al
			WHERE al.employee_id = e.id
			  AND (al.event_at AT TIME ZONE 'Asia/Tashkent')::date =
			      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
			ORDER BY al.event_at DESC, al.id DESC
			LIMIT 1
		) last_event ON TRUE
		WHERE e.store_id = ?
		  AND e.deleted_at IS NULL
		  AND e.is_active = true
		ORDER BY e.full_name ASC
	`, storeId).Scan(&storeMapInfo.Employees).Error; err != nil {
		s.log.Errorf("could not get store map employees: %v", err)
		return nil, domain.InternalServerError
	}

	storeMapInfo.PaymentTypes = make([]domain.StoreMapInfoPaymentType, 0)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT payment.type, SUM(payment.amount) AS amount
		FROM sales
		CROSS JOIN LATERAL (
			VALUES
				('cash', COALESCE(sales.cash, 0)),
				('click', COALESCE(sales.click, 0)),
				('humo', COALESCE(sales.humo, 0)),
				('uzcard', COALESCE(sales.uzcard, 0)),
				('payme', COALESCE(sales.payme, 0)),
				('alif', COALESCE(sales.alif, 0)),
				('uzum', COALESCE(sales.uzum, 0)),
				('uzum_tez_kor', COALESCE(sales.uzum_tez_kor, 0)),
				('loyalty_card', COALESCE(sales.loyalty_card, 0))
		) AS payment(type, amount)
		WHERE sales.store_id = ?
		  AND sales.created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
		  AND sales.created_at < (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date + INTERVAL '1 day'
		  AND sales.is_active = true
		  AND payment.amount <> 0
		GROUP BY payment.type
		ORDER BY payment.type
	`, storeId).Scan(&storeMapInfo.PaymentTypes).Error; err != nil {
		s.log.Errorf("could not get store map payment types: %v", err)
		return nil, domain.InternalServerError
	}

	return &storeMapInfo, nil
}
