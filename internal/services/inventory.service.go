package services

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
)

// region Create

// Create inventory creates a new inventory
func (s *Services) CreateInventory(ctx context.Context, req *domain.InventoryRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// insert inventory into inventories table
	err := tx.WithContext(ctx).
		Raw(`
	INSERT INTO imports (
		store_id, 
		name, 
		inventory_type, 
		created_by, 
		entry_type, 
		import_date
		)
	VALUES (?, ?, ?, ?, ?, ?) 
	RETURNING id`,
			req.StoreId, req.Name, req.Type, req.CreatedBy, 2, time.Now(),
		).Scan(&id).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create inventory: %v", err)
		return domain.InternalServerError
	}
	if req.Type == "FULL" {
		// FULL: insert all store products with unit_quantity as-is (original logic)
		err = tx.WithContext(ctx).Exec(`
			INSERT INTO import_details (
				import_id,
				product_id,
				store_product_id,
				received_count,
				supply_price_vat,
				retail_price_vat,
				expire_date,
				series_number,
				imported_at
			)
			SELECT
				?,
				p.id,
				sp.id,
				COALESCE(sp.unit_quantity, 0),
				COALESCE(sp.supply_price, 0.00),
				COALESCE(sp.retail_price, 0.00),
				sp.expire_date,
				sp.serial_number,
				sp.created_at
			FROM products p
			LEFT JOIN store_products sp ON sp.product_id = p.id AND sp.store_id = ?
			`, id, req.StoreId).Error
	} else {
		// SOME: insert only the given product_ids with received_count corrected by movement diff
		// diff = correct_quantity - unit_quantity (same as GetProductMovementUnits/ExportMovementUnits Q column)
		err = tx.WithContext(ctx).Exec(`
			WITH import_data AS (
				SELECT
					imd.product_id,
					(SUM(imd.accepted_count) * p.unit_per_pack)::INTEGER AS import_count
				FROM imports im
					JOIN import_details imd ON im.id = imd.import_id
					JOIN products p ON p.id = imd.product_id
				WHERE im.store_id = ? AND im.entry_type = 1 AND im.status = 'completed'
				GROUP BY imd.product_id, p.unit_per_pack
			),
			sold AS (
				SELECT
					sp.product_id,
					SUM(ci.unit_quantity) AS sold_quantity
				FROM sales s
					JOIN cart_items ci ON ci.sale_id = s.id
					JOIN store_products sp ON sp.id = ci.store_product_id
				WHERE s.store_id = ? AND s.stage IN (9, 11) AND s.sale_type = 'SALE'
				GROUP BY sp.product_id
			),
			return_sales AS (
				SELECT
					sp.product_id,
					SUM(ci.unit_quantity) AS return_quantity
				FROM sales s
					JOIN cart_items ci ON ci.sale_id = s.id
					JOIN store_products sp ON sp.id = ci.store_product_id
				WHERE s.store_id = ? AND s.stage IN (9, 11) AND s.sale_type = 'RETURN'
				GROUP BY sp.product_id
			),
			transfer_in AS (
				SELECT
					td.product_id,
					(SUM(td.accepted_count) * p.unit_per_pack)::INTEGER AS transfer_in_count
				FROM transfer_details td
					JOIN transfers t ON td.transfer_id = t.id
					JOIN products p ON p.id = td.product_id
				WHERE t.to_store_id = ? AND t.entry_type = 1 AND (t.status = 'completed' OR t.status = 'sent-to-1c')
				GROUP BY td.product_id, p.unit_per_pack
			),
			transfer_out AS (
				SELECT
					td.product_id,
					(SUM(td.accepted_count) * p.unit_per_pack)::INTEGER AS transfer_out_count
				FROM transfer_details td
					JOIN transfers t ON td.transfer_id = t.id
					JOIN products p ON p.id = td.product_id
				WHERE t.from_store_id = ? AND t.entry_type = 1 AND (t.status = 'completed' OR t.status = 'sent-to-1c')
				GROUP BY td.product_id, p.unit_per_pack
			),
			vozvrat AS (
				SELECT
					td.product_id,
					(SUM(td.accepted_count) * p.unit_per_pack)::INTEGER AS vozvrat_count
				FROM transfer_details td
					JOIN transfers t ON td.transfer_id = t.id
					JOIN products p ON p.id = td.product_id
				WHERE t.from_store_id = ? AND t.entry_type = 2 AND (t.status = 'completed' OR t.status = 'sent-to-1c')
				GROUP BY td.product_id, p.unit_per_pack
			),
			product_quantity AS (
				SELECT
					sp.product_id,
					SUM(sp.unit_quantity)::INTEGER AS unit_quantity
				FROM store_products sp
				WHERE sp.store_id = ?
				GROUP BY sp.product_id
			),
			inventory_quantity AS (
				SELECT
					imd.product_id,
					SUM(CASE WHEN imd.scanned_count - imd.received_count > 0 THEN imd.scanned_count - imd.received_count ELSE 0 END)::INTEGER AS inventory_plus_count,
					SUM(CASE WHEN imd.scanned_count - imd.received_count < 0 THEN imd.scanned_count - imd.received_count ELSE 0 END)::INTEGER AS inventory_minus_count
				FROM import_details imd
					JOIN imports im ON im.id = imd.import_id
				WHERE im.store_id = ? AND im.entry_type = 2 AND im.status = 'completed'
				GROUP BY imd.product_id
			),
			movement_diff AS (
				SELECT
					p.id AS product_id,
					COALESCE(im.import_count, 0) + COALESCE(rs.return_quantity, 0) + COALESCE(tin.transfer_in_count, 0) +
					COALESCE(inv.inventory_plus_count, 0) + COALESCE(inv.inventory_minus_count, 0) -
					COALESCE(s.sold_quantity, 0) - COALESCE(tout.transfer_out_count, 0) - COALESCE(v.vozvrat_count, 0) -
					COALESCE(pq.unit_quantity, 0) AS diff
				FROM products p
				JOIN product_quantity pq ON pq.product_id = p.id
				LEFT JOIN import_data im ON im.product_id = p.id
				LEFT JOIN sold s ON s.product_id = p.id
				LEFT JOIN return_sales rs ON rs.product_id = p.id
				LEFT JOIN transfer_in tin ON tin.product_id = p.id
				LEFT JOIN transfer_out tout ON tout.product_id = p.id
				LEFT JOIN vozvrat v ON v.product_id = p.id
				LEFT JOIN inventory_quantity inv ON inv.product_id = p.id
			),
			ranked_sp AS (
				SELECT sp.*,
					ROW_NUMBER() OVER (PARTITION BY sp.product_id ORDER BY sp.created_at) AS rn
				FROM store_products sp
				WHERE sp.store_id = ?
			)
			INSERT INTO import_details (
				import_id,
				product_id,
				store_product_id,
				received_count,
				supply_price_vat,
				retail_price_vat,
				expire_date,
				series_number,
				imported_at
			)
			SELECT
				?,
				p.id,
				rsp.id,
				COALESCE(rsp.unit_quantity, 0) + CASE WHEN rsp.rn = 1 THEN COALESCE(md.diff, 0) ELSE 0 END,
				COALESCE(rsp.supply_price, 0.00),
				COALESCE(rsp.retail_price, 0.00),
				rsp.expire_date,
				rsp.serial_number,
				rsp.created_at
			FROM products p
			LEFT JOIN ranked_sp rsp ON rsp.product_id = p.id
			LEFT JOIN movement_diff md ON md.product_id = p.id
			WHERE p.id = ANY(?)
			`,
			req.StoreId,                  // import_data
			req.StoreId,                  // sold
			req.StoreId,                  // return_sales
			req.StoreId,                  // transfer_in
			req.StoreId,                  // transfer_out
			req.StoreId,                  // vozvrat
			req.StoreId,                  // product_quantity
			req.StoreId,                  // inventory_quantity
			req.StoreId,                  // ranked_sp
			id,                           // import_id
			pq.Array(req.ProductIds),     // product filter
		).Error
	}
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could create inventory details: %v", err)
		return domain.InternalServerError
	}
	// // commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit create inventory transaction: %v", err)
		return domain.InternalServerError
	}

	go s.setNewInventoryAmount(id)

	return nil
}

func (s *Services) setNewInventoryAmount(inventoryId string) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	query := `
	UPDATE imports
	SET
		received_count = COALESCE((
			SELECT SUM(d.received_count / NULLIF(p.unit_per_pack, 0))
			FROM import_details d
			JOIN products p ON p.id = d.product_id
			WHERE d.import_id = ?
		), 0),
		received_sum = COALESCE((
			SELECT SUM((d.received_count / NULLIF(p.unit_per_pack, 0)) * d.retail_price_vat)
			FROM import_details d
			JOIN products p ON p.id = d.product_id
			WHERE d.import_id = ?
		), 0),
		updated_at = NOW()
	WHERE id = ?;
	`
	err := s.db.WithContext(ctx).Exec(query, inventoryId, inventoryId, inventoryId).Error
	if err != nil {
		s.log.Errorf("could not set new inventory amount: %v", err)
	}
}

// region Get
// get inventory by id
func (s *Services) GetInventoryById(ctx context.Context, params *domain.InventoryParam) (*domain.Inventory, error) {
	var tmp struct {
		Id            string     `gorm:"id"`
		PublicId      string     `gorm:"public_id"`
		StoreId       string     `gorm:"store_id"`
		Name          string     `gorm:"name"`
		InventoryType string     `gorm:"inventory_type"`
		Status        string     `gorm:"status"`
		CreatedBy     string     `gorm:"created_by"`
		UpdatedBy     string     `gorm:"updated_by"`
		ReceivedCount float64    `gorm:"received_count"`
		ReceivedSum   float64    `gorm:"received_sum"`
		CreatedAt     *time.Time `gorm:"created_at"`
		UpdatedAt     *time.Time `gorm:"updated_at"`
		StoreName     string     `gorm:"store_name"`
	}

	err := s.db.WithContext(ctx).
		Select(
			"im.id",
			"im.public_id",
			"im.store_id",
			"im.name",
			"im.inventory_type",
			"im.status",
			"im.created_by",
			"im.accepted_by as updated_by",
			"im.created_at",
			"im.updated_at",
			"s.name AS store_name",
		).
		Table("imports im").
		Joins("JOIN stores s ON im.store_id = s.id").
		Where("im.id = ?", params.InventoryId).
		Take(&tmp).Error
	if err != nil {
		s.log.Errorf("could not get inventory by id: %v", err)
		return nil, domain.InternalServerError
	}

	totalQuery := `
	SELECT
		ROUND(SUM(imd.received_count/p.unit_per_pack), 2) AS total_current_count,
		ROUND(SUM(imd.retail_price_vat * (imd.received_count/p.unit_per_pack)), 2) AS total_current_sum,
		ROUND(SUM(imd.scanned_count/p.unit_per_pack), 2) AS total_fact_count,
		ROUND(SUM(imd.retail_price_vat * (imd.scanned_count/p.unit_per_pack)), 2) AS total_fact_sum,
		ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)), 2) AS total_difference_sum
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`
	var (
		totalSumData domain.InventoryDetailSum
		args         = []any{}
		filter       = " WHERE imd.import_id = ? "
	)
	args = append(args, params.InventoryId)
	// filter by search key
	if params.Search != "" {
		switch utils.DefineProductSearchQuery(params.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+params.Search+"%")
		case "name/category":
			filter += " AND (p.name ILIKE ? OR pr.name ILIKE ?) "
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
		}
	}

	// filter with inventory stats
	if params.Type != "" {
		switch params.Type {
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
	err = s.db.WithContext(ctx).Raw(totalQuery, args...).Scan(&totalSumData).Error
	if err != nil {
		s.log.Errorf("could not inventory total_sum_data on inventory details: %v", err)
		return nil, err
	}
	res := domain.Inventory{
		Id:            tmp.Id,
		PublicId:      tmp.PublicId,
		StoreId:       tmp.StoreId,
		Name:          tmp.Name,
		InventoryType: tmp.InventoryType,
		Status:        tmp.Status,
		CreatedById:   tmp.CreatedBy,
		UpdatedById:   tmp.UpdatedBy,
		CreatedAt:     tmp.CreatedAt,
		UpdatedAt:     tmp.UpdatedAt,
		Store: domain.NewNullStruct(domain.InventoryStore{
			Id:   tmp.StoreId,
			Name: tmp.StoreName,
		}, tmp.StoreId != ""),
		CurrentCount:  totalSumData.TotalCurrentCount,
		CurrentSum:    totalSumData.TotalCurrentSum,
		FactCount:     totalSumData.TotalFactCount,
		FactSum:       totalSumData.TotalFactSum,
		DifferenceSum: totalSumData.TotalDifferenceSum,
	}

	return &res, nil
}

// get inventory list
func (s *Services) GetInventories(ctx context.Context, params *domain.InventoryParam) ([]domain.Inventory, int64, error) {
	var tmp []struct {
		Id              string     `gorm:"id"`
		PublicId        string     `gorm:"public_id"`
		StoreId         string     `gorm:"store_id"`
		Name            string     `gorm:"name"`
		InventoryType   string     `gorm:"inventory_type"`
		Status          string     `gorm:"status"`
		CreatedBy       string     `gorm:"created_by"`
		UpdatedBy       string     `gorm:"updated_by"`
		CurrentCount    float64    `gorm:"current_count"`
		CurrentSum      float64    `gorm:"current_sum"`
		FactCount       float64    `gorm:"fact_count"`
		FactSum         float64    `gorm:"fact_sum"`
		DifferenceCount float64    `gorm:"difference_count"`
		DifferenceSum   float64    `gorm:"difference_sum"`
		CreatedAt       *time.Time `gorm:"created_at"`
		UpdatedAt       *time.Time `gorm:"updated_at"`
		StoreName       string     `gorm:"store_name"`
		CreatedByName   string     `gorm:"created_by_name"`
		UpdatedByName   string     `gorm:"updated_by_name"`
	}

	qb := s.db.WithContext(ctx).
		Joins("JOIN stores s ON im.store_id = s.id").
		Joins("LEFT JOIN employees em ON im.created_by = em.id").
		Joins("LEFT JOIN employees em2 ON im.accepted_by = em2.id").
		Table("imports im").
		Where("entry_type = ?", 2)
	// filter by store id
	if params.StoreId != "" {
		qb = qb.Where("im.store_id = ? ", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("stores.company_id = ? ", params.CompanyId)
		qb = qb.Joins(" LEFT JOIN stores ON im.store_id = stores.id")
	}
	// filter by search keyword
	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.public_id::text LIKE ? OR im.name ILIKE ?", params.Search, params.Search)
	}
	// filter by inventory type
	if params.Type != "" {
		qb = qb.Where("im.inventory_type = ?", params.Type)
	}
	// filter by inventory status
	if params.Status != "" {
		qb = qb.Where("im.status = ?", params.Status)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get inventory total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// complete qb
	err := qb.
		Select(
			"im.id",
			"im.public_id",
			"im.store_id",
			"im.name",
			"im.inventory_type",
			"im.status",
			"im.entry_type",
			"im.created_by",
			"im.accepted_by AS updated_by",
			"im.received_count AS current_count",
			"im.received_sum AS current_sum",
			"im.scanned_count AS fact_count",
			"im.scanned_sum AS fact_sum",
			"(im.scanned_count - im.received_count) AS difference_count",
			"(im.scanned_sum - im.received_sum) AS difference_sum",
			"im.created_at",
			"im.updated_at",

			"s.name AS store_name",
			"em.full_name AS created_by_name",
			"em2.full_name AS updated_by_name",
		).
		Order("im.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmp).Error
	if err != nil {
		s.log.Errorf("could not get inventory list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res = []domain.Inventory{}
	for _, item := range tmp {
		res = append(res, domain.Inventory{
			Id:              item.Id,
			PublicId:        item.PublicId,
			StoreId:         item.StoreId,
			Name:            item.Name,
			Status:          item.Status,
			InventoryType:   item.InventoryType,
			CurrentCount:    item.CurrentCount,
			CurrentSum:      item.CurrentSum,
			FactCount:       item.FactCount,
			FactSum:         item.FactSum,
			DifferenceCount: item.DifferenceCount,
			DifferenceSum:   item.DifferenceSum,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
			CreatedById:     item.CreatedBy,
			CreatedBy: domain.NewNullStruct(domain.InventoryEmployee{
				Id:       item.CreatedBy,
				FullName: item.CreatedByName,
			}, item.CreatedBy != ""),
			UpdatedById: item.UpdatedBy,
			UpdatedBy: domain.NewNullStruct(domain.InventoryEmployee{
				Id:       item.UpdatedBy,
				FullName: item.UpdatedByName,
			}, item.UpdatedBy != ""),
			Store: domain.NewNullStruct(domain.InventoryStore{
				Id:   item.StoreId,
				Name: item.StoreName,
			}, item.StoreId != ""),
		})
	}

	return res, totalCount, nil
}

func (s *Services) GetInventoryStats(ctx context.Context, params *domain.InventoryParam) (*domain.InventoryStatusSummary, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"SUM(im.received_count) AS current_count",
			"SUM(im.received_sum) AS current_sum",
			"SUM(im.scanned_count) AS fact_count",
			"SUM(im.scanned_sum) AS fact_sum",
			"SUM(im.scanned_count - im.received_count) AS difference_count",
			"SUM(im.scanned_sum - im.received_sum) AS difference_sum",
		).Table("imports im").
		Joins("JOIN stores st ON im.store_id = st.id")
	qb = qb.Where("im.entry_type = ?", constants.ProductMovementInventory)

	if params.StoreId != "" {
		qb = qb.Where("im.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.Search != "" {
		search := fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.public_id::text LIKE ? OR im.name ILIKE ?", search)
	}
	if params.Type != "" {
		qb = qb.Where("im.inventory_type = ?", params.Type)
	}
	if params.Status != "" {
		qb = qb.Where("im.status = ?", params.Status)
	}

	var res domain.InventoryStatusSummary
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get inventory stats: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get inventory detail list
func (s *Services) InventoryDetailList(ctx context.Context, params *domain.InventoryParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
	var (
		res          []domain.InventoryDetail
		totalSumData domain.InventoryDetailSum
		totalCount   int64
		args         = []any{}
		filter       = " WHERE imd.import_id = ? "
		orderBy      = ""
		group        = " GROUP BY p.id, pr.id, imd.import_id "
	)
	args = append(args, params.InventoryId)
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
		ROUND(SUM(imd.retail_price_vat * ((imd.scanned_count - imd.received_count)/p.unit_per_pack)), 2) AS total_difference_sum,
		ROUND(SUM(imd.scanned_count/p.unit_per_pack)) AS scanned,
        ROUND(SUM((imd.received_count - imd.scanned_count)/p.unit_per_pack)) AS shortage,
        ROUND(SUM(imd.received_count/p.unit_per_pack)) AS "all",
        ROUND(SUM(CASE WHEN imd.scanned_count > imd.received_count THEN (imd.scanned_count - imd.received_count)/p.unit_per_pack ELSE 0 END)) AS surplus,
        ROUND(SUM(imd.accepted_count/p.unit_per_pack)) AS accepted
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	`

	if params.Search != "" {
		switch utils.DefineProductSearchQuery(params.Search) {
		case "barcode":
			filter += ` AND EXISTS (
				SELECT 1
				FROM product_barcodes pb2
				WHERE pb2.product_id = p.id
				  AND pb2.status = 'completed'
				  AND pb2.barcode ILIKE ?
			)`
			args = append(args, "%"+params.Search+"%")
		case "name/category":
			filter += " AND (p.name ILIKE ? OR pr.name ILIKE ?) "
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
		}
	}
	// filter with inventory stats
	if params.Type != "" {
		switch params.Type {
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
	switch params.Order {
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
	err := s.db.WithContext(ctx).Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not getting inventory details total count: %v", err)
		return res, totalSumData, 0, domain.InternalServerError
	}

	// total sum query completed
	totalQuery += filter
	err = s.db.WithContext(ctx).Raw(totalQuery, args...).Scan(&totalSumData).Error
	if err != nil {
		s.log.Errorf("could not get total_sum_data on inventory details: %v", err)
		return res, totalSumData, 0, err
	}
	// complete query
	query += filter + group + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)
	// execute query
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get inventory detail list: %v", err)
		return res, totalSumData, 0, err
	}
	if len(res) == 0 {
		res = []domain.InventoryDetail{}
	}

	return res, totalSumData, totalCount, nil
}

// get inventory detail list
func (s *Services) InventoryDetailedFlow(ctx context.Context, params *domain.InventoryParam) ([]domain.InventoryDetail, domain.InventoryDetailSum, int64, error) {
	var (
		res        []domain.InventoryDetail
		totalData  domain.InventoryDetailSum
		totalCount int64
		args       = []any{}
		filter     = " WHERE import_id = ? AND product_id = ? "
		orderBy    = " ORDER BY imd.imported_at DESC"
	)
	args = append(args, params.InventoryId, params.ProductId)
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

	if params.Search != "" {
		switch utils.DefineProductSearchQuery(params.Search) {
		case "barcode":
			filter += " AND p.barcode LIKE ?"
			args = append(args, "%"+params.Search+"%")
		case "name/category":
			filter += " AND p.name ILIKE ?"
			args = append(args, "%"+params.Search+"%")
		default:
			filter += " AND (p.name ILIKE ? OR p.barcode LIKE ?)"
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%")
		}
	}

	// execute total count query
	tquery += filter
	// get total count
	err := s.db.WithContext(ctx).Raw(tquery, args...).Scan(&totalCount).Error
	if err != nil {
		s.log.Errorf("could not get total count: %v", err)
		return res, totalData, 0, domain.InternalServerError
	}
	// execute total sum query
	totalQuery += filter
	err = s.db.WithContext(ctx).Raw(totalQuery, args...).Scan(&totalData).Error
	if err != nil {
		s.log.Errorf("could not get inventory flow total data: %v", err)
		return res, totalData, totalCount, domain.InternalServerError
	}

	// complete query
	query += filter + orderBy + " LIMIT ? OFFSET ?;"
	args = append(args, params.Limit, params.Offset)
	// execute query
	err = s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get inventory detail list: %v", err)
		return res, totalData, 0, domain.InternalServerError
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

// region Update
// confirm inventory
func (s *Services) ConfirmInventory(ctx context.Context, inventoryId string, userId string) error {
	var err error
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var res domain.Inventory
	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? RETURNING *`
	err = tx.WithContext(ctx).Raw(query, constants.GeneralStatusCompleted, userId, inventoryId).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update to confirm inventory %v", err)
		return domain.InternalServerError
	}

	detailQuery := `
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
	WHERE imd.import_id = ?`

	// SOME: only scanned items (to deduct); FULL: only items that differ
	if res.InventoryType == "SOME" {
		detailQuery += ` AND imd.scanned_count > 0`
	} else {
		detailQuery += ` AND imd.received_count != imd.scanned_count`
	}

	var inventoryDetails []domain.ImportDetail
	err = tx.WithContext(ctx).Raw(detailQuery, inventoryId).Scan(&inventoryDetails).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get inventory_details on confirming: %v", err)
		return domain.InternalServerError
	}

	var dataOnec domain.InventoryData1C

	if res.InventoryType == "SOME" {
		// SOME: skanerlangan miqdorni to'g'ridan-to'g'ri o'rnatish (set scanned_count as actual stock)
		for _, imd := range inventoryDetails {
			err = tx.WithContext(ctx).Exec(`
				UPDATE store_products
				SET
					unit_quantity = ?,
					pack_quantity = FLOOR(?::numeric / ?::integer)
				WHERE id = ?`,
				imd.ScannedCount,
				imd.ScannedCount,
				imd.UnitPerPack,
				imd.StoreProductId,
			).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("could not update store_product quantity on confirm SOME inventory: %v", err)
				return domain.InternalServerError
			}
			dataOnec.Товары = append(dataOnec.Товары, domain.InventoryProduct1C{
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
	} else {
		// FULL: surplus → yangi store_product qo'shish, shortage → mavjudni yangilash
		err = tx.WithContext(ctx).Exec(`
			INSERT INTO store_products(
				product_id, store_id, pack_quantity, unit_quantity,
				retail_price, supply_price, vat, expire_date,
				vat_price, import_detail_id, serial_number
			)
			SELECT
				imd.product_id,
				?,
				FLOOR((imd.scanned_count - imd.received_count) / p.unit_per_pack),
				imd.scanned_count - imd.received_count,
				imd.retail_price_vat,
				imd.supply_price_vat,
				12,
				imd.expire_date,
				(imd.retail_price_vat * 12) / 112,
				imd.id,
				imd.series_number
			FROM import_details imd
			JOIN products p ON imd.product_id = p.id
			WHERE imd.import_id = ? AND imd.scanned_count > imd.received_count`,
			res.StoreId, inventoryId,
		).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not inserting inventory to store_product: %v", err)
			return domain.InternalServerError
		}

		for _, imd := range inventoryDetails {
			if imd.ScannedCount < imd.ReceivedCount {
				err = tx.WithContext(ctx).Exec(`
					UPDATE store_products
					SET
						pack_quantity = ?,
						unit_quantity = ?
					WHERE id = ?`,
					int(imd.ScannedCount/float64(imd.UnitPerPack)),
					imd.ScannedCount,
					imd.StoreProductId,
				).Error
				if err != nil {
					_ = tx.Rollback()
					s.log.Errorf("could not update store_product quantity on confirm inventory: %v", err)
					return domain.InternalServerError
				}
			}
			dataOnec.Товары = append(dataOnec.Товары, domain.InventoryProduct1C{
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
	}

	if len(dataOnec.Товары) < 1 {
		_ = tx.Rollback()
		s.log.Warnf("empty products list in confirm inventory: %v", err)
		return domain.NotEnoughProductError
	}

	// get store info
	var store domain.Store
	err = tx.WithContext(ctx).First(&store, "id = ?", res.StoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get store info: %v", err)
		return domain.InternalServerError
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction on confirming inventory: %v", err)
		return domain.InternalServerError
	}

	// get store info
	dataOnec.Apteka.Name = store.Name
	dataOnec.Apteka.StoreCode = store.StoreCode
	// get document data and number
	dataOnec.Dok.DocumentDate = res.UpdatedAt.Format("2006-01-02T15:04:05")
	dataOnec.Dok.DocumentNumber = "NP-" + cast.ToString(res.PublicId)

	go s.setConfirmInventoryAmount(inventoryId)
	// send inventory products data to 1C
	go s.DoRequestOnec(context.Background(), dataOnec, constants.OnecPathInventar)

	return nil
}

func (s *Services) setConfirmInventoryAmount(inventoryId string) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	query := `
		UPDATE imports
            SET
                scanned_count = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0) / NULLIF(p.unit_per_pack, 0))
                    FROM import_details d
                    JOIN products p ON p.id = d.product_id
                    WHERE d.import_id = ?
                ), 0),
                scanned_sum = COALESCE((
                    SELECT SUM((COALESCE(d.scanned_count, 0) / NULLIF(p.unit_per_pack, 0)) * d.retail_price_vat)
                    FROM import_details d
                    JOIN products p ON p.id = d.product_id
                    WHERE d.import_id = ?
                ), 0),
                updated_at = NOW()
            WHERE id = ?;
	`

	err := s.db.WithContext(ctx).Exec(query, inventoryId, inventoryId, inventoryId).Error
	if err != nil {
		s.log.Errorf("could not set confirm inventory amount: %v", err)
		return
	}
}

func (s *Services) UpdateInventoryFactQuantity(ctx context.Context, request *domain.InventoryAddProduct, inventoryId string) error {

	// update barcode and retail price
	if request.Barcode != "" && request.RetailPrice > 0 {
		err := s.updateInventoryBarcodeAndPrice(ctx, request, inventoryId)
		if err != nil {
			return err
		}
		return nil
	}

	var tmp []struct {
		Id            string  `gorm:"id"`
		ScannedCount  float64 `gorm:"scanned_count"`
		ReceivedCount float64 `gorm:"received_count"`
		UnitPerPack   int     `gorm:"unit_per_pack"`
	}
	// find import_detail row
	err := s.db.WithContext(ctx).Raw(`
	SELECT
		imd.id AS id,
		imd.scanned_count AS scanned_count,
		imd.received_count AS received_count,
		p.unit_per_pack AS unit_per_pack
	FROM import_details imd
	JOIN products p ON p.id = imd.product_id
	WHERE imd.product_id = ? AND imd.import_id = ? ORDER BY imd.imported_at`,
		request.Id, inventoryId,
	).Scan(&tmp).Error
	if err != nil {
		s.log.Errorf("could not find import_detail row: %v", err)
		return domain.InternalServerError
	}

	if len(tmp) == 0 {
		return domain.NotFoundError
	}

	unitPerPack := tmp[0].UnitPerPack
	if unitPerPack <= 0 {
		return domain.InvalidRequestBodyError
	}

	if request.FactQuantity == 0 && request.FactUnit == 0 {
		s.log.Infof("Resetting scanned_count - inventoryId: %s, productId: %s", inventoryId, request.Id)
		err = s.db.WithContext(ctx).Exec("UPDATE import_details SET scanned_count = ? WHERE import_id = ? AND product_id = ?", 0, inventoryId, request.Id).Error
		if err != nil {
			s.log.Errorf("Error on updating scanned_count: %v", err)
			return domain.InternalServerError
		}
		return nil
	}

	// deltas[i] — shu qatorga qancha qo'shilishi kerak (unit hisobida)
	deltas := make([]float64, len(tmp))

	// Pack yo'li: faqat diff_qty (pack darajasi) ga qarab ishlaydi
	if request.FactQuantity > 0 {
		remainingPacks := request.FactQuantity
		for i, row := range tmp {
			if remainingPacks <= 0 {
				break
			}
			recvPacks := float64(int(row.ReceivedCount) / unitPerPack)
			scanPacks := float64(int(row.ScannedCount) / unitPerPack)
			shortagePacks := recvPacks - scanPacks
			if shortagePacks > 0 {
				fill := shortagePacks
				if remainingPacks < fill {
					fill = remainingPacks
				}
				deltas[i] += fill * float64(unitPerPack)
				remainingPacks -= fill
			}
		}
		// qolganini oxirgi qatorga (pack * unitPerPack)
		if remainingPacks > 0 {
			deltas[len(tmp)-1] += remainingPacks * float64(unitPerPack)
		}
	}

	// Unit yo'li: faqat diff_unit (dona darajasi) ga qarab ishlaydi
	if request.FactUnit > 0 {
		remainingUnits := request.FactUnit
		for i, row := range tmp {
			if remainingUnits <= 0 {
				break
			}
			recvUnits := float64(int(row.ReceivedCount) % unitPerPack)
			scanUnits := float64(int(row.ScannedCount) % unitPerPack)
			shortageUnits := recvUnits - scanUnits
			if shortageUnits > 0 {
				fill := shortageUnits
				if remainingUnits < fill {
					fill = remainingUnits
				}
				deltas[i] += fill
				remainingUnits -= fill
			}
		}
		// qolganini oxirgi qatorga
		if remainingUnits > 0 {
			deltas[len(tmp)-1] += remainingUnits
		}
	}

	// Har bir qatorni yangilash
	for i, row := range tmp {
		if deltas[i] == 0 {
			continue
		}
		err := s.db.WithContext(ctx).Exec(`
			UPDATE import_details
			SET scanned_count = scanned_count + ?
			WHERE id = ?
		`, deltas[i], row.Id).Error
		if err != nil {
			s.log.Errorf("could not update scanned_count: %v", err)
			return domain.InternalServerError
		}
	}

	return nil
}

func (s *Services) updateInventoryBarcodeAndPrice(ctx context.Context, request *domain.InventoryAddProduct, inventoryId string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	err := tx.WithContext(ctx).Exec("UPDATE products SET barcode = ? WHERE id = ?", request.Barcode, request.Id).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update product barcode: %v", err)
		return domain.InternalServerError
	}

	err = tx.WithContext(ctx).Exec(`
	INSERT INTO product_barcodes (
				product_id, 
				barcode, 
				status
				)
    		SELECT 
				?, ?, ?
    		WHERE NOT EXISTS (
    		    SELECT 1 FROM product_barcodes 
    		    WHERE product_id = ? AND barcode = ? AND status = ?
    		)
	`,
		request.Id,
		request.Barcode,
		constants.GeneralStatusCompleted,
		request.Id,
		request.Barcode,
		constants.GeneralStatusCompleted,
	).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not insert product barcode: %v", err)
		return domain.InternalServerError
	}

	err = tx.WithContext(ctx).Exec("UPDATE import_details SET retail_price_vat = ? WHERE retail_price_vat = 0 AND product_id = ? AND import_id = ?",
		request.RetailPrice, request.Id, inventoryId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update retail_price_vat on inventory_details: %v", err)
		return domain.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit inventory update barcode transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// canceled inventory
func (s *Services) CancelInventory(inventoryId string, userId string) error {
	// start transaction

	// update confirm inventory
	query := `UPDATE imports SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := s.db.Exec(query, constants.GeneralStatusCanceled, userId, inventoryId).Error
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
	err = s.db.First(&store, "id = ?", inventory.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info: %v", err)
		return err
	}
	// get store info
	data1C.Apteka.Name = store.Name
	data1C.Apteka.StoreCode = store.StoreCode

	// get document data and number
	data1C.Dok.DocumentDate = inventory.UpdatedAt.Format("2006-01-02T15:04:05")
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(inventory.PublicId)

	// send inventory products data to 1C
	err = s.DoRequestOnec(context.Background(), data1C, "/inventar")
	if err != nil {
		s.log.Warn("ERROR on sending inventory: %v", err)
		return err
	}

	return nil
}

// region Delete
func (s *Services) DeleteInventory(ctx context.Context, inventoryId string) error {
	err := s.db.
		WithContext(ctx).
		Delete(&domain.Import{}, "id = ?", inventoryId).
		Where("status = ?", constants.GeneralStatusNew).
		Where("entry_type = ?", 2).Error
	if err != nil {
		s.log.Error("could not delete inventory(%s): %v", inventoryId, err)
		return domain.InternalServerError
	}
	return nil
}
