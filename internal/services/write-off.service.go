package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

// create new write-off
func (s *Services) CreateWriteOff(req *domain.WriteOffRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// insert inventory into inventories table
	err := tx.Raw(`
	INSERT INTO imports (store_id, name, created_by, entry_type, import_date, comment)
	VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		req.StoreId, req.Name, req.CreatedBy, 3, time.Now(), req.Comment,
	).Scan(&id).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("ERROR on creating inventory: %v", err)
		return domain.InternalServerError
	}
	// if no products provided, get all products from store_products
	// and insert them into write-off
	err = tx.Exec(
		`INSERT INTO import_details(import_id, product_id, received_count, supply_price_vat, retail_price_vat, expire_date, series_number
			) SELECT ?, product_id, pack_quantity, supply_price, retail_price, expire_date, serial_number
			FROM store_products
			WHERE store_id = ? AND pack_quantity > 0;`,
		id, req.StoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("ERROR on creating inventory details: %v", err)
		return domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("ERROR on commiting transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// get write-off list
func (s *Services) WriteOffList(ctx context.Context, params *domain.WriteOffParam) ([]domain.WriteOff, int64, error) {
	var res []domain.WriteOff
	query := s.db.WithContext(ctx).
		Select(`
			im.id,
			im.public_id,
			im.store_id,
			im.name,
			im.status,
			im.entry_type,
			im.created_at,
			im.updated_at,
			im.created_by,
			im.updated_by,
			SUM(imd.accepted_count) AS writeoff_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum`).
		Table("imports im").
		Joins("LEFT JOIN import_details imd ON im.id = imd.import_id").
		Where("im.entry_type = ?", 3)
	// filter by store id
	if params.StoreId != "" {
		query = query.Where("im.store_id = ? ", params.StoreId)
	}
	if params.CompanyId != "" {
		query = query.Where("stores.company_id = ?", params.CompanyId).
			Joins("LEFT JOIN stores ON im.store_id = stores.id")
	}
	// filter by search keyword
	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where("CAST(im.public_id AS TEXT) LIKE ? OR im.name ILIKE ?", params.Search, params.Search)
	}

	// filter by inventory status
	if params.Status != "" {
		query = query.Where("im.status = ?", params.Status)
	}
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		s.log.Error("could not get total count of write-offs: %v", err)
		return res, 0, domain.InternalServerError
	}

	// complete query
	err := query.
		Group("im.id").
		Order("im.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get write-off list: %v", err)
		return res, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) WriteOffStatus(ctx context.Context, params *domain.WriteOffParam) (*domain.WriteOffStatusSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(imd.accepted_count), 0) AS scanned_count,
			COALESCE(SUM(imd.accepted_count * imd.retail_price_vat), 0) AS retail_price
		FROM imports
		LEFT JOIN import_details imd ON imports.id = imd.import_id
		LEFT JOIN stores ON imports.store_id = stores.id
		WHERE imports.entry_type = 3
	`

	var args []any
	var filters []string

	if params.StoreId != "" {
		filters = append(filters, "imports.store_id = ?")
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		query += " AND stores.company_id = ?"
		args = append(args, params.CompanyId)
	}
	if params.Search != "" {
		search := "%" + params.Search + "%"
		filters = append(filters, "(CAST(imports.public_id AS TEXT) ILIKE ? OR imports.name ILIKE ?)")
		args = append(args, search, search)
	}
	if params.Status != "" {
		filters = append(filters, "imports.status = ?")
		args = append(args, params.Status)
	}
	if len(filters) > 0 {
		query += " AND " + strings.Join(filters, " AND ")
	}

	var res domain.WriteOffStatusSummary
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Error("Failed to get write-off status summary: %v", err)
		return nil, err
	}

	return &res, nil
}

// get write-off by id
func (s *Services) GetWriteOffById(writeOffId string) (*domain.WriteOff, error) {
	var res domain.WriteOff
	err := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			SUM(imd.accepted_count) AS writeoff_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").Group("imports.id").First(&res, "imports.id = ?", writeOffId).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// write-off confirm
func (s *Services) ConfirmWriteOff(ctx context.Context, writeOffId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.WithContext(ctx).Exec(query, constants.GeneralStatusCompleted, userId, writeOffId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update inventory %v", err)
		return domain.InternalServerError
	}
	// update confirm inventory details
	query1 := `UPDATE import_details SET accepted_count = scanned_count, updated_at = NOW() WHERE import_id = ?`
	err = tx.WithContext(ctx).Exec(query1, writeOffId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update inventory details: %v", err)
		return domain.InternalServerError
	}
	query2 := `DELETE FROM import_details WHERE scanned_count = 0 AND import_id = ?;`
	err = tx.WithContext(ctx).Exec(query2, writeOffId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("cound not delete scanned 0 inventory details: %v", err)
		return domain.InternalServerError
	}
	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// canceled write-off
func (s *Services) CancelWriteOff(writeOffId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, constants.GeneralStatusCanceled, userId, writeOffId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update inventory %v", err)
		return domain.InternalServerError
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// get write-off detail list
func (s *Services) WriteOffDetailList(ctx context.Context, param *domain.WriteOffDetailParam) ([]domain.WriteOffDetail, int64, error) {
	var res []domain.WriteOffDetail
	var totalCount int64
	query := s.db.WithContext(ctx).
		Model(&domain.ImportDetail{}).
		Select(`
		import_details.*,
    	p.name, p.material_code, p.barcode, ut.short_name`).
		Joins("JOIN products p ON import_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("import_details.import_id = ?", param.WriteOffId)

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			query = query.Where("p.barcode = ?", param.Search)
		case "name/category":
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ?", param.Search)
		default:
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ? OR p.barcode LIKE ?", param.Search, param.Search)
		}
	}
	// filter with inventory stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			query = query.Where("import_details.received_count > import_details.scanned_count")
		case "scanned":
			query = query.Where("import_details.scanned_count > 0")
		case "surplus":
			query = query.Where("import_details.scanned_count > import_details.received_count")

		}
	}

	err := query.
		Order("import_details.updated_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get write-off details: %v", err)
		return res, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
