package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
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
			ROUND(COALESCE(sp.pack_quantity::numeric + (sp.unit_quantity::numeric%p.unit_per_pack)/p.unit_per_pack, 0.00), 4) AS quantity,
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
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on committing transaction: %v", err)
		tx.Rollback()
		return err
	}
	return nil
}

// get inventory by id
func (s *Services) GetInventoryById(inventoryID string) (*domain.Inventory, error) {
	var res domain.Inventory
	err := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			imports.*,
			SUM(imd.accepted_count) AS measurement_count,
			SUM(imd.accepted_count*imd.supply_price_vat) AS supply_price_sum,
			SUM(imd.accepted_count*imd.retail_price_vat) AS retail_price_sum, 
			SUM((received_count - scanned_count)*imd.supply_price_vat) AS shortage_supply_sum,
			SUM((received_count - scanned_count)*imd.retail_price_vat) AS shortage_retail_sum,
			SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*imd.supply_price_vat) AS surplus_supply_sum,
			SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*imd.retail_price_vat) AS surplus_retail_sum
			`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").
		Group("imports.id").
		First(&res, "imports.id = ?", inventoryID).Error
	if err != nil {
		s.log.Warn("ERROR on getting write-off by id: %v", err)
		return nil, err
	}
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
			SUM(imd.received_count) AS measurement_count,
			SUM(imd.received_count-imd.scanned_count) AS shortage,
			SUM(CASE WHEN imd.scanned_count > imd.received_count THEN imd.accepted_count - imd.received_count ELSE 0 END) AS surplus,
			SUM(imd.received_count*imd.retail_price_vat) AS current_sum,
			SUM(imd.scanned_count*imd.retail_price_vat) AS fact_sum`).
		Joins("LEFT JOIN import_details imd ON imports.id = imd.import_id").
		Where("imports.entry_type = ?", 2)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("imports.store_id = ? ", param.StoreId)
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
		Group("imports.id").
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

// get inventory detail list
func (s *Services) InventoryDetailList(param *domain.InventoryDetailParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
	var (
		res          []domain.InventoryDetail
		totalSumData domain.InventoryDetailSum
		totalCount   int64
		args         = []any{}
		filter       = " WHERE imd.import_id = ? "
		orderBy      = ""
		group        = " GROUP BY p.id, imd.import_id "
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
        p.unit_per_pack,
		MAX(imd.retail_price_vat) AS retail_price,
        SUM(imd.received_count) AS current_quantity,
        ROUND((SUM(imd.received_count) - FLOOR(SUM(received_count)))*p.unit_per_pack, 0)  AS current_unit,
        SUM(imd.scanned_count) AS fact_quantity,
        ROUND((SUM(imd.scanned_count) - FLOOR(SUM(scanned_count)))*p.unit_per_pack, 0) AS fact_unit,
        (SUM(imd.scanned_count) - SUM(imd.received_count)) AS difference_quantity,
        ROUND(((SUM(imd.scanned_count) - FLOOR(SUM(imd.scanned_count))) -
        (SUM(imd.received_count) - FLOOR(SUM(imd.received_count)))) * p.unit_per_pack, 0) AS difference_unit,
        SUM(imd.retail_price_vat * imd.received_count) AS current_sum,
        SUM(imd.retail_price_vat * imd.scanned_count) AS fact_sum,
        SUM(imd.retail_price_vat * (imd.scanned_count - imd.received_count)) AS difference_sum
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
		SUM(imd.retail_price_vat * imd.received_count) AS total_current_sum,
		SUM(imd.retail_price_vat * imd.scanned_count) AS total_fact_sum,
		SUM(imd.retail_price_vat * (imd.scanned_count - imd.received_count)) AS total_difference_sum
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
		orderBy = " ORDER BY current_quantity DESC "
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
func (s *Services) InventoryDetailedFlow(param *domain.InventoryDetailParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
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
		imd.received_count AS current_quantity,
		ROUND((imd.received_count - FLOOR(imd.received_count)) * p.unit_per_pack, 0) AS current_unit,
		imd.scanned_count AS fact_quantity,
		ROUND((imd.scanned_count - FLOOR(imd.scanned_count)) * p.unit_per_pack, 0) AS fact_unit,
		imd.scanned_count - imd.received_count AS difference_quantity,
		ROUND((ABS(imd.scanned_count - imd.received_count) - FLOOR(ABS(imd.scanned_count - imd.received_count))) * p.unit_per_pack, 0)*SIGN(imd.scanned_count - imd.received_count) AS difference_unit,
		imd.retail_price_vat * imd.received_count AS current_sum,
		imd.retail_price_vat * imd.scanned_count AS fact_sum,
		imd.retail_price_vat * (imd.scanned_count - imd.received_count) AS difference_sum,
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
        SUM(imd.retail_price_vat * imd.received_count) AS total_current_sum,
        SUM(imd.retail_price_vat * imd.scanned_count) AS total_fact_sum,
        SUM(imd.retail_price_vat * (imd.scanned_count - imd.received_count)) AS total_difference_sum
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
func (s *Services) InventoryDetailStatsCount(param *domain.InventoryDetailParam) (domain.InventoryDetailStatus, error) {
	var res domain.InventoryDetailStatus

	query := `
	SELECT
		SUM(scanned_count) AS scanned,
		SUM(received_count - scanned_count) AS shortage,
		SUM(received_count) AS "all",
		SUM(CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END) AS surplus,
		SUM(accepted_count) AS accepted,
		SUM((received_count - scanned_count)*import_details.supply_price_vat) AS shortage_supply_sum,
		SUM((received_count - scanned_count)*import_details.retail_price_vat) AS shortage_retail_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*import_details.supply_price_vat) AS surplus_supply_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*import_details.retail_price_vat) AS surplus_retail_sum
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
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var res domain.Inventory
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? RETURNING *`
	err := tx.Debug().Raw(query, config.COMPLETED, userId, inventoryId).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	fmt.Println("AFTER update imports status: ")
	// delete correct inventory products if current and fact will be equal (received_count = scanned_count)
	// err = tx.Debug().Exec(`DELETE FROM import_details WHERE import_id = ? AND received_count = scanned_count`, inventoryId).Error
	// if err != nil {
	// 	s.log.Warn("ERROR on deleting inventory_details recieve and scan will be equal: %v", err)
	// 	tx.Rollback()
	// 	return err
	// }
	fmt.Println("---->>> ", "salom")
	// get inventory details list if fact and current quantity will not be equal
	var inventoryDetails []domain.ImportDetail
	query1 := `
	SELECT
		imd.*, p.material_code, p.name AS product_name,
		p.barcode, p.unit_per_pack, pr.code AS producer_code
	FROM import_details imd
		JOIN products p ON imd.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
	WHERE imd.import_id = ? AND imd.received_count != imd.scanned_count
	`
	// execute get import details as inventory details
	err = tx.Debug().Raw(query1, inventoryId).Scan(&inventoryDetails).Error
	if err != nil {
		s.log.Warn("ERROR on getting inventory_details: %v", err)
		tx.Rollback()
		return err
	}
	// add new inventory products to store_products if fact greater then current quantity
	// We only add delta -> (scanned_count - received_count) as a new product
	storeProduct := `
	INSERT INTO store_products(
           product_id, store_id, pack_quantity, unit_quantity,
           retail_price, supply_price,
           vat, expire_date, vat_price,
           import_detail_id, serial_number)
	SELECT
		imd.product_id, ?, FLOOR(imd.scanned_count - imd.received_count), (imd.scanned_count - imd.received_count) * p.unit_per_pack,
		imd.retail_price_vat, imd.supply_price_vat, 12, imd.expire_date,imd.retail_price_vat*12/112,  imd.id, imd.series_number
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	WHERE imd.import_id = ? AND imd.scanned_count > imd.received_count;
	`
	// execute store_product create query
	err = tx.Debug().Exec(storeProduct, res.StoreId, inventoryId).Error
	if err != nil {
		s.log.Warn("ERROR on inserting inventory to store_product: %v", err)
		tx.Rollback()
		return err
	}
	// update store_products quantities if fact quantity greater then current (received_count > scanned_count)
	// collect 1C inventar request data
	var data1C domain.InventoryData1C
	for _, imd := range inventoryDetails {
		if imd.ScannedCount < imd.ReceivedCount {
			err = tx.Debug().Exec(`UPDATE store_products SET pack_quantity = ?, unit_quantity = ? WHERE id = ?`,
				int(imd.ScannedCount), int(imd.ScannedCount*float64(imd.UnitPerPack)), imd.StoreProductId).Error
			if err != nil {
				s.log.Warn("ERROR on updating store_product quantity on confirm inventory: %v", err)
				tx.Rollback()
				return err
			}
		}
		// collect inventory products to send 1C
		data1C.Товары = append(data1C.Товары, domain.InventoryProduct1C{
			MaterilaCode:        imd.MaterialCode,
			Name:                imd.ProductName,
			Barcode:             imd.Barcode,
			Manufacturer:        imd.ProducerCode,
			ProductSeriesNumber: imd.SeriesNumber,
			ExpireDate:          imd.ExpireDate,
			Quantity:            imd.ReceivedCount,
			QuantityInventar:    imd.ScannedCount,
			RetailPrice:         imd.RetailPrice,
			RetailPriceVat:      imd.RetailPriceVat,
			SupplyPrice:         imd.SupplyPrice,
			SupplyPriceVat:      imd.SupplyPriceVat,
			Sum:                 imd.ScannedCount * imd.RetailPrice,
			SumVat:              imd.ScannedCount * imd.RetailPriceVat,
		})

	}

	// get store info
	var store domain.Store
	err = tx.Debug().First(&store, "id = ?", res.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info: %v", err)
		tx.Rollback()
		return err
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		tx.Rollback()
		return err
	}

	// get store info
	data1C.Apteka.Name = store.Name
	data1C.Apteka.StoreCode = store.StoreCode
	// declare current time
	now := time.Now()
	// get document data and number
	data1C.Dok.DocumentDate = now.Format(time.RFC3339)
	data1C.Dok.DocumentNumber = "PH" + cast.ToString(now.Unix())

	t, _ := json.Marshal(&data1C)
	fmt.Println("--->>> ", string(t))

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
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? AND status = ?`
	err := tx.Exec(query, config.CANCELED, userId, inventoryId, config.PENDING).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Warn("ERROR on commiting transaction %v", err)
		tx.Rollback()
		return err
	}

	return nil
}
