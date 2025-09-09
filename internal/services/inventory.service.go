package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
)

// Create inventory creates a new inventory
func (s *Services) CreateInventory(req *domain.InventoryRequest) error {
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
	INSERT INTO imports (store_id, name, inventory_type, created_by, entry_type, import_date)
	VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		req.StoreId, req.Name, req.Type, req.CreatedBy, 2, time.Now(),
	).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory: %v", err)
		tx.Rollback()
		return err
	}
	// insert all products (including those not in store_products)
	err = tx.Exec(`
		INSERT INTO import_details (
			import_id, product_id, store_product_id,  received_count, supply_price_vat, retail_price_vat, expire_date, series_number, imported_at
				)
		SELECT
			?,
			p.id,
			sp.id,
			COALESCE(sp.unit_quantity, 0),
			COALESCE(sp.supply_price, 0.00) AS supply_price,
			COALESCE(sp.retail_price, 0.00) AS retail_price,
			sp.expire_date,
			sp.serial_number,
			sp.created_at
		FROM
			products p
		LEFT JOIN
			store_products sp ON sp.product_id = p.id and sp.store_id = ?
		`, id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on creating inventory detail: %v", err)
		tx.Rollback()
		return err
	}
	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on committing transaction: %v", err)
		tx.Rollback()
		return err
	}
	return nil
}

// get inventory by id
func (s *Services) GetInventoryById(param *domain.InventoryParam) (*domain.Inventory, error) {
	var (
		res          domain.Inventory
		totalSumData domain.InventoryDetailSum
		args         = []any{}
		filter       = " WHERE imd.import_id = ? "
	)
	args = append(args, param.InventoryId)

	err := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			id, public_id,
			store_id, name,
			inventory_type,
			status, created_by,
			accepted_by as updated_by,
			created_at, updated_at
		`).First(&res, "id = ?", param.InventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}

	totalQuery := `
	SELECT
		SUM(imd.retail_price_vat * (imd.received_count/p.unit_per_pack)) AS total_current_sum,
		SUM(imd.retail_price_vat * (imd.scanned_count/p.unit_per_pack)) AS total_fact_sum,
		SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)) AS total_difference_sum
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`
	// filter by search key
	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+param.Search+"%")
		case "name/category":
			filter += " AND (p.name ILIKE ? OR pr.name ILIKE ?) "
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		}
	}

	// filter with inventory stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			filter += " AND imd.received_count > imd.scanned_count "
		case "scanned":
			filter += " AND imd.scanned_count > 0 "
		case "surplus":
			filter += " AND imd.scanned_count > imd.received_count "
		case "zero_price":
			filter += " AND imd.retail_price_vat = 0 AND imd.scanned_count > 0 "
		}
	}

	// total sum query completed
	totalQuery += filter
	err = s.db.Raw(totalQuery, args...).Scan(&totalSumData).Error
	if err != nil {
		s.log.Warn("ERROR on getting total_sum_data on inventory details: %v", err)
		return &res, err
	}
	res.CurrentSum = totalSumData.TotalCurrentSum
	res.FactSum = totalSumData.TotalFactSum
	res.DifferenceSum = totalSumData.TotalDifferenceSum

	return &res, nil
}

// get inventory list
func (s *Services) InventoryList(param *domain.InventoryParam) ([]domain.Inventory, int64, error) {
	var res []domain.Inventory
	var totalCount int64
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			0 AS current_count,
			0 AS fact_count,
			0 AS difference_count,
			0 AS current_sum,
			0 AS fact_sum,
			0 AS difference_sum
		`).
		Where("entry_type = ?", 2)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("imports.store_id = ? ", param.StoreId)
	}
	if param.CompanyId != "" {
		query = query.Where("stores.company_id = ? ", param.CompanyId)
		query = query.Joins(" LEFT JOIN stores ON imports.store_id = stores.id")
	}
	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("CAST(imports.public_id AS TEXT) LIKE ? OR imports.name ILIKE ?", param.Search, param.Search)
	}
	// filter by inventory type
	if param.Type != "" {
		query = query.Where("imports.inventory_type = ?", param.Type)
	}
	// filter by inventory status
	if param.Status != "" {
		query = query.Where("imports.status = ?", param.Status)
	}
	// complete query
	err := query.
		Order("imports.created_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory list: %v", err)
		return res, 0, err
	}
	if len(res) == 0 {
		res = []domain.Inventory{}
	}

	return res, totalCount, nil
}

func (s *Services) InventoryStatus(param *domain.InventoryParam) (*domain.InventoryStatusSummary, error) {
	query := `
	SELECT
		ROUND(COALESCE(SUM((imd.received_count::numeric/p.unit_per_pack) * imd.retail_price_vat), 0), 2) AS current_sum,
		ROUND(COALESCE(SUM((imd.scanned_count::numeric/p.unit_per_pack) * imd.retail_price_vat), 0), 2) AS fact_sum,
		ROUND(COALESCE(SUM(((imd.scanned_count-imd.received_count)::numeric/p.unit_per_pack)  * imd.retail_price_vat), 0), 2) AS difference_sum,
		ROUND(COALESCE(SUM(imd.received_count::numeric/p.unit_per_pack), 0)) AS current_count,
		ROUND(COALESCE(SUM(imd.scanned_count::numeric/p.unit_per_pack), 0)) AS fact_count,
		ROUND(COALESCE(SUM((imd.scanned_count - imd.received_count)::numeric/p.unit_per_pack), 0)) AS difference_count
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	JOIN imports im ON im.id = imd.import_id
	LEFT JOIN stores ON im.store_id = stores.id
	WHERE im.entry_type = 2;
	`

	var args []any

	if param.StoreId != "" {
		query += " AND im.store_id = ?"
		args = append(args, param.StoreId)
	}
	if param.CompanyId != "" {
		query += " AND stores.company_id = ?"
		args = append(args, param.CompanyId)
	}
	if param.Search != "" {
		search := fmt.Sprintf("%%%s%%", param.Search)
		query += " AND (CAST(im.public_id AS TEXT) LIKE ? OR im.name ILIKE ?)"
		args = append(args, search, search)
	}
	if param.Type != "" {
		query += " AND im.inventory_type = ?"
		args = append(args, param.Type)
	}
	if param.Status != "" {
		query += " AND im.status = ?"
		args = append(args, param.Status)
	}

	var result domain.InventoryStatusSummary
	if err := s.db.Raw(query, args...).Scan(&result).Error; err != nil {
		s.log.Error("Failed to get inventory status stats: %v", err)
		return nil, err
	}

	return &result, nil
}

// get inventory detail list
func (s *Services) InventoryDetailList(param *domain.InventoryParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
	var (
		res          []domain.InventoryDetail
		totalSumData domain.InventoryDetailSum
		totalCount   int64
		args         = []any{}
		filter       = " WHERE imd.import_id = ? "
		orderBy      = ""
		group        = " GROUP BY p.id, pr.id, imd.import_id "
	)
	args = append(args, param.InventoryId)
	//
	query := `
	SELECT
		p.id AS id,
		imd.import_id AS inventory_id,
		p.id AS product_id,
		p.material_code,
		p.barcode,
		p.name,
		COALESCE(pr.name, '') AS producer_name,
		p.unit_per_pack,
		MAX(imd.retail_price_vat) AS retail_price,
		MAX(imd.expire_date) AS expire_date,
		ROUND(SUM(imd.received_count/p.unit_per_pack), 4) AS current_quantity,
		ROUND(SUM(imd.received_count%p.unit_per_pack), 4)  AS current_unit,
		ROUND(SUM(imd.scanned_count/p.unit_per_pack), 4) AS fact_quantity,
		ROUND(SUM(imd.scanned_count%p.unit_per_pack), 4) AS fact_unit,
		ROUND(SUM((imd.scanned_count-imd.received_count)/p.unit_per_pack), 4) AS difference_quantity,
		ROUND(SUM((imd.scanned_count-imd.received_count)%p.unit_per_pack), 4)   AS difference_unit,
		ROUND(SUM(imd.retail_price_vat * (imd.received_count/p.unit_per_pack)), 2) AS current_sum,
		ROUND(SUM(imd.retail_price_vat * (imd.scanned_count/p.unit_per_pack)), 2) AS fact_sum,
		ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)), 2) AS difference_sum
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	`
	tquery := `
	SELECT
		COUNT(DISTINCT p.id) AS total_count
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	totalQuery := `
	SELECT
		ROUND(SUM(imd.retail_price_vat * (imd.received_count/p.unit_per_pack)), 2) AS total_current_sum,
		ROUND(SUM(imd.retail_price_vat * (imd.scanned_count/p.unit_per_pack)), 2) AS total_fact_sum,
		ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)), 2) AS total_difference_sum
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+param.Search+"%")
		case "name/category":
			filter += " AND (p.name ILIKE ? OR pr.name ILIKE ?) "
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		}
	}
	// filter with inventory stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			filter += " AND imd.received_count > imd.scanned_count "
		case "scanned":
			filter += " AND imd.scanned_count > 0 "
		case "surplus":
			filter += " AND imd.scanned_count > imd.received_count "
		case "zero_price":
			filter += " AND imd.retail_price_vat = 0 AND imd.scanned_count > 0 "
		case "checking":
			filter += " AND imd.expire_date IS NOT NULL "
		}
	}

	// order by
	switch param.Order {
	case "+name":
		orderBy = " ORDER BY p.name ASC "
	case "-name":
		orderBy = " ORDER BY p.name DESC "
	case "+current_sum":
		orderBy = " ORDER BY current_sum ASC "
	case "-current_sum":
		orderBy = " ORDER BY current_sum DESC "
	case "+fact_sum":
		orderBy = " ORDER BY fact_sum ASC "
	case "-fact_sum":
		orderBy = " ORDER BY fact_sum DESC "
	case "+difference_sum":
		orderBy = " ORDER BY difference_sum ASC "
	case "-difference_sum":
		orderBy = " ORDER BY difference_sum DESC "
	default:
		orderBy = " ORDER BY p.name ASC "
	}
	// execute total count query
	tquery += filter
	// get total count
	err := s.db.Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting total count: %v", err)
		return res, totalSumData, 0, err
	}

	// total sum query completed
	totalQuery += filter
	err = s.db.Raw(totalQuery, args...).Scan(&totalSumData).Error
	if err != nil {
		s.log.Warn("ERROR on getting total_sum_data on inventory details: %v", err)
		return res, totalSumData, 0, err
	}
	// complete query
	query += filter + group + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, param.Limit, param.Offset)
	// execute query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory detail list: %v", err)
		return res, totalSumData, 0, err
	}
	if len(res) == 0 {
		res = []domain.InventoryDetail{}
	}

	return res, totalSumData, totalCount, nil
}

// get inventory detail list
func (s *Services) InventoryDetailedFlow(param *domain.InventoryParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
	var (
		res        []domain.InventoryDetail
		totalData  domain.InventoryDetailSum
		totalCount int64
		args       = []any{}
		filter     = " WHERE import_id = ? AND product_id = ? "
		orderBy    = " ORDER BY imd.imported_at DESC"
	)
	args = append(args, param.InventoryId, param.ProductId)
	//
	query := `
	SELECT
		imd.id,
		imd.import_id AS inventory_id,
		p.id AS product_id,
		p.material_code, p.name,
		p.barcode,
		p.unit_per_pack,
		ROUND(imd.received_count/p.unit_per_pack, 4) AS current_quantity,
		ROUND(imd.received_count%p.unit_per_pack, 0) AS current_unit,
		ROUND(imd.scanned_count/p.unit_per_pack, 4) AS fact_quantity,
		ROUND(imd.scanned_count%p.unit_per_pack, 0) AS fact_unit,
		ROUND((imd.scanned_count - imd.received_count)/p.unit_per_pack, 4) AS difference_quantity,
		ROUND((imd.scanned_count - imd.received_count)%p.unit_per_pack) AS difference_unit,
		ROUND(imd.retail_price_vat * (imd.received_count/p.unit_per_pack), 2) AS current_sum,
		ROUND(imd.retail_price_vat *(imd.scanned_count/p.unit_per_pack), 2) AS fact_sum,
		ROUND(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack), 2) AS difference_sum,
		imd.retail_price_vat AS retail_price,
		imd.expire_date
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
	`
	tquery := `
	SELECT
		COUNT(*) AS total_count
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
	`

	totalQuery := `
	SELECT
        ROUND(SUM(imd.retail_price_vat * (imd.received_count/p.unit_per_pack)), 2) AS total_current_sum,
        ROUND(SUM(imd.retail_price_vat * (imd.scanned_count/p.unit_per_pack)), 2) AS total_fact_sum,
        ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)), 2) AS total_difference_sum
	FROM import_details imd
        JOIN products p ON imd.product_id = p.id
	`

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+param.Search+"%")
		case "name/category":
			filter += " AND p.name ILIKE ?"
			args = append(args, "%"+param.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+param.Search+"%", "%"+param.Search+"%")
		}
	}

	// execute total count query
	tquery += filter
	// get total count
	err := s.db.Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Warn("ERROR on getting total count: %v", err)
		return res, totalData, 0, err
	}
	// execute total sum query
	totalQuery += filter
	err = s.db.Raw(totalQuery, args...).Scan(&totalData).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory flow total data: %v", err)
		return res, totalData, totalCount, err
	}

	// complete query
	query += filter + orderBy + " LIMIT ? OFFSET ?;"
	args = append(args, param.Limit, param.Offset)
	// execute query
	err = s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory detail list: %v", err)
		return res, totalData, 0, err
	}

	return res, totalData, totalCount, nil
}

// get inventory detail status count
func (s *Services) InventoryDetailStatsCount(param *domain.InventoryParam) (domain.InventoryDetailStatus, error) {
	var res domain.InventoryDetailStatus

	query := `
	SELECT
		ROUND(SUM(scanned_count/p.unit_per_pack)) AS scanned,
		ROUND(SUM((received_count - scanned_count)/p.unit_per_pack)) AS shortage,
		ROUND(SUM(received_count/p.unit_per_pack)) AS "all",
		ROUND(SUM(CASE WHEN scanned_count > received_count THEN (scanned_count - received_count)/p.unit_per_pack ELSE 0 END)) AS surplus,
		ROUND(SUM(accepted_count/p.unit_per_pack)) AS accepted,
		ROUND(SUM(((received_count - scanned_count)/p.unit_per_pack)*import_details.supply_price_vat), 2) AS shortage_supply_sum,
		ROUND(SUM(((received_count - scanned_count)/p.unit_per_pack)*import_details.retail_price_vat), 2) AS shortage_retail_sum,
		ROUND(SUM((CASE WHEN scanned_count > received_count THEN (scanned_count - received_count)/p.unit_per_pack ELSE 0 END)*import_details.supply_price_vat), 2) AS surplus_supply_sum,
		ROUND(SUM((CASE WHEN scanned_count > received_count THEN (scanned_count - received_count)/p.unit_per_pack ELSE 0 END)*import_details.retail_price_vat), 2) AS surplus_retail_sum
	FROM import_details
	JOIN products p ON import_details.product_id = p.id
	WHERE import_id = ?;
	`
	err := s.db.Raw(query, param.InventoryId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}

	return res, nil
}

// confirm inventory
func (s *Services) ConfirmInventory(inventoryId string, userId string) error {
	var err error
	// start transaction
	tx := s.db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var res domain.Inventory
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? RETURNING *`
	err = tx.Raw(query, config.COMPLETED, userId, inventoryId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		return err
	}

	// get inventory details list if fact and current quantity will not be equal
	var inventoryDetails []domain.ImportDetail
	query1 := `
	SELECT
		imd.*, 
		p.material_code, 
		p.name AS product_name,
		p.barcode, 
		p.unit_per_pack, 
		pr.code AS producer_code
	FROM 
		import_details imd
	JOIN
		products p ON imd.product_id = p.id
	LEFT JOIN
		producers pr ON p.producer_id = pr.id
	WHERE
		imd.import_id = ? AND imd.received_count != imd.scanned_count
	`
	// execute get import details as inventory details
	err = tx.Raw(query1, inventoryId).Scan(&inventoryDetails).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory_details: %v", err)
		return err
	}
	// add new inventory products to store_products if fact greater then current quantity
	// We only add delta -> (scanned_count - received_count) as a new product
	storeProduct := `
	INSERT INTO store_products(
            product_id,
            store_id,
            pack_quantity,
            unit_quantity,
            retail_price,
            supply_price,
            vat,
            expire_date,
            vat_price,
            import_detail_id,
            serial_number
            )
	SELECT
		imd.product_id,
		?,
		FLOOR((imd.scanned_count - imd.received_count)/p.unit_per_pack),
		imd.scanned_count - imd.received_count,
		imd.retail_price_vat,
		imd.supply_price_vat,
		12,
		imd.expire_date,
		(imd.retail_price_vat*12)/112,
		imd.id,
		imd.series_number
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	WHERE imd.import_id = ? AND imd.scanned_count > imd.received_count;
	`
	// execute store_product create query
	err = tx.Exec(storeProduct, res.StoreId, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on inserting inventory to store_product: %v", err)
		return err
	}
	// update store_products quantities if fact quantity greater then current (received_count > scanned_count)
	// collect 1C inventar request data
	var data1C domain.InventoryData1C
	for _, imd := range inventoryDetails {
		if imd.ScannedCount < imd.ReceivedCount {
			err = tx.Exec(`
			UPDATE 
				store_products 
			SET 
				pack_quantity = ?, 
				unit_quantity = ? 
			WHERE id = ?;`,
				int(imd.ScannedCount/float64(imd.UnitPerPack)),
				imd.ScannedCount,
				imd.StoreProductId).Error
			if err != nil {
				s.log.Warn("ERROR on updating store_product quantity on confirm inventory: %v", err)
				return err
			}
		}
		// collect inventory products to send 1C
		data1C.Товары = append(data1C.Товары, domain.InventoryProduct1C{
			MaterialCode:        imd.MaterialCode,
			Name:                imd.ProductName,
			Barcode:             imd.Barcode,
			Manufacturer:        imd.ProducerCode,
			ProductSeriesNumber: imd.SeriesNumber,
			ExpireDate:          imd.ExpireDate,
			Quantity:            utils.RoundTo((imd.ReceivedCount / float64(imd.UnitPerPack)), 4),
			QuantityInventar:    utils.RoundTo((imd.ScannedCount / float64(imd.UnitPerPack)), 4),
			RetailPrice:         imd.RetailPrice,
			RetailPriceVat:      imd.RetailPriceVat,
			SupplyPrice:         imd.SupplyPrice,
			SupplyPriceVat:      imd.SupplyPriceVat,
			Sum:                 utils.RoundTo((imd.ScannedCount/float64(imd.UnitPerPack))*imd.RetailPrice, 2),
			SumVat:              utils.RoundTo((imd.ScannedCount/float64(imd.UnitPerPack))*imd.RetailPriceVat, 2),
		})

	}

	if len(data1C.Товары) < 1 {
		s.log.Warn("empty products list in confirm inventory: %v", err)
		return errors.New("not enough products in the inventory")
	}

	// get store info
	var store domain.Store
	err = tx.First(&store, "id = ?", res.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info: %v", err)
		return err
	}

	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		return err
	}

	// get store info
	data1C.Apteka.Name = store.Name
	data1C.Apteka.StoreCode = store.StoreCode
	// get document data and number
	data1C.Dok.DocumentDate = res.UpdatedAt.Format("2006-01-02T15:04:05")
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(res.PublicId)

	// send inventory products data to 1C
	err = s.DoRequest(context.Background(), data1C, "/inventar")
	if err != nil {
		s.log.Warn("ERROR on sending inventory: %v", err)
		return err
	}

	return nil
}

// canceled inventory
func (s *Services) CancelInventory(inventoryId string, userId string) error {
	// start transaction

	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := s.db.Exec(query, config.CANCELED, userId, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		return err
	}

	return nil
}

// send inventory details to 1C
func (s *Services) SendInventory1C(inventoryID string) error {
	var inventory domain.Import
	err := s.db.First(&inventory, "id = ?", inventoryID).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	// get inventory details list if fact and current quantity will not be equal
	var inventoryDetails []domain.ImportDetail
	query1 := `
	SELECT
		imd.*, 
		p.material_code, 
		p.name AS product_name,
		p.barcode, 
		p.unit_per_pack, 
		pr.code AS producer_code
	FROM 
		import_details imd
	JOIN 
		products p ON imd.product_id = p.id
	LEFT JOIN 
		producers pr ON p.producer_id = pr.id
	WHERE 
		imd.import_id = ? AND imd.received_count != imd.scanned_count
	`
	// execute get import details as inventory details
	err = s.db.Raw(query1, inventoryID).Scan(&inventoryDetails).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory_details: %v", err)
		return err
	}
	// collect 1C inventar request data
	var data1C domain.InventoryData1C
	for _, imd := range inventoryDetails {
		// collect inventory products to send 1C
		data1C.Товары = append(data1C.Товары, domain.InventoryProduct1C{
			MaterialCode:        imd.MaterialCode,
			Name:                imd.ProductName,
			Barcode:             imd.Barcode,
			Manufacturer:        imd.ProducerCode,
			ProductSeriesNumber: imd.SeriesNumber,
			ExpireDate:          imd.ExpireDate,
			Quantity:            utils.RoundTo((imd.ReceivedCount / float64(imd.UnitPerPack)), 4),
			QuantityInventar:    utils.RoundTo((imd.ScannedCount / float64(imd.UnitPerPack)), 4),
			RetailPrice:         imd.RetailPrice,
			RetailPriceVat:      imd.RetailPriceVat,
			SupplyPrice:         imd.SupplyPrice,
			SupplyPriceVat:      imd.SupplyPriceVat,
			Sum:                 utils.RoundTo((imd.ScannedCount/float64(imd.UnitPerPack))*imd.RetailPrice, 2),
			SumVat:              utils.RoundTo((imd.ScannedCount/float64(imd.UnitPerPack))*imd.RetailPriceVat, 2),
		})

	}

	// get store info
	var store domain.Store
	err = s.db.First(&store, "id = ?", inventory.StoreID).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info: %v", err)
		return err
	}
	// get store info
	data1C.Apteka.Name = store.Name
	data1C.Apteka.StoreCode = store.StoreCode

	// get document data and number
	data1C.Dok.DocumentDate = inventory.UpdatedAt.Format("2006-01-02T15:04:05")
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(inventory.PublicID)

	// send inventory products data to 1C
	err = s.DoRequest(context.Background(), data1C, "/inventar")
	if err != nil {
		s.log.Warn("ERROR on sending inventory: %v", err)
		return err
	}

	return nil
}

func (s *Services) DeleteInventory(ctx context.Context, inventoryId string) error {
	err := s.db.
		WithContext(ctx).
		Delete(&domain.Import{}, "id = ?", inventoryId).
		Where("status = ?", constants.NEW).
		Where("entry_type = ?", 2).Error
	if err != nil {
		s.log.Error("could not delete inventory(%s): %v", inventoryId, err)
		return errors.New(constants.InternalServerError)
	}
	return nil
}
