package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/pharma-crm-backend/domain"
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

func (s *Services) UpdateAverateStoreTargetSales() error {
	err := s.db.Exec( `
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

			COALESCE(sales.sales_amount, 0) AS sales_amount,

			COALESCE(cash_boxes.cash_box_count, 0) AS cash_box_count,

			COALESCE(employees.employee_count, 0) AS employee_count,

			COALESCE(attendance.is_open, false) AS is_open
		`).

		Joins(`
			LEFT JOIN (
				SELECT
					store_id,
					SUM(total_amount) AS sales_amount
				FROM sales
				WHERE created_at >= CURRENT_DATE
				  AND created_at < CURRENT_DATE + INTERVAL '1 day'
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
				WHERE deleted_at IS NULL
				  AND is_active = true
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
					latest.store_id,
					TRUE AS is_open
				FROM (
					SELECT DISTINCT ON (employee_id)
						employee_id,
						store_id,
						event_type,
						event_at
					FROM attendance_logs
					WHERE store_id IS NOT NULL
					  AND (event_at AT TIME ZONE 'Asia/Tashkent')::date =
					      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
					ORDER BY
						employee_id,
						event_at DESC,
						id DESC
				) latest
				WHERE latest.event_type = 'check-in'
				GROUP BY latest.store_id
			) attendance
				ON attendance.store_id = stores.id
		`)

	if params.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", params.Search)

		qb = qb.Where(
			"stores.name ILIKE ? OR stores.detailed_name ILIKE ?",
			searchPattern,
			searchPattern,
		)
	}

	if params.IsFranchise != nil {
		qb = qb.Where(`
			stores.company_id IN (
				SELECT id
				FROM companies
				WHERE is_franchise = ?
			)
		`, *params.IsFranchise)
	}

	if params.IsPharma != nil {
		qb = qb.Where(`
			stores.company_id IN (
				SELECT id
				FROM companies
				WHERE is_pharma = ?
			)
		`, *params.IsPharma)
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

func (s *Services) GetStoreByIdMapInfo(ctx context.Context, storeId string,) (*domain.StoreMapInfo, error)	{
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
					SELECT DISTINCT ON (employee_id)
						employee_id,
						store_id,
						event_type,
						event_at
					FROM attendance_logs
					WHERE store_id IS NOT NULL
					  AND (event_at AT TIME ZONE 'Asia/Tashkent')::date =
					      (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
					ORDER BY
						employee_id,
						event_at DESC,
						id DESC
				) latest
				WHERE latest.event_type = 'check-in'
				GROUP BY latest.store_id
			) attendance
				ON attendance.store_id = stores.id
		`).
		Where("stores.id = ?", storeId)

	var storeMapInfo domain.StoreMapInfo

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

	return &storeMapInfo, nil
}
