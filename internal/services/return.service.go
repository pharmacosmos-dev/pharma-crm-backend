package services

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math"
	"strings"
	"time"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
)

// Create return creates a new return
func (s *Services) CreateReturn(req *domain.ReturnRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log)
	// insert return into inventories table
	err := tx.Raw(`
	INSERT INTO transfers (
		from_store_id, 
		name,  
		created_by, 
		entry_type)
	VALUES (?, ?, ?, ?) RETURNING id`,
		req.StoreId,
		req.Name,
		req.CreatedBy,
		2,
	).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on creating return: %v", err)
		tx.Rollback()
		return err
	}

	// if no products provided, get all products from store_products
	// and insert them into return_details
	err = tx.Exec(
		`INSERT INTO transfer_details(
			transfer_id,
			store_product_id,
			product_id,
			received_count,
			supply_price,
			retail_price,
			expire_date,
			serial_number)
		SELECT  
			?, 
			sp.id,
			sp.product_id, 
			sp.unit_quantity::numeric/p.unit_per_pack, 
			sp.supply_price, 
			sp.retail_price, 
			sp.expire_date, 
			sp.serial_number
		FROM store_products sp
		JOIN
			products p ON sp.product_id = p.id
		WHERE 
			sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0);`,
		id, req.StoreId).Error
	if err != nil {
		s.log.Warn("ERROR on creating return details: %v", err)
		tx.Rollback()
		return err
	}

	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on committing transaction: %v", err)
		return err
	}
	return nil
}

// get return by id
func (s *Services) GetReturnById(returnId string) (*domain.Return, error) {
	var res domain.Return
	err := s.db.Model(&domain.Transfer{}).
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Select(`
			transfers.*,
			SUM(td.accepted_count) AS return_count,
			SUM(td.received_count) AS received_count,
			SUM(td.scanned_count) AS scanned_count,
			SUM(td.received_count*td.retail_price) AS received_retail_sum,
			SUM(td.accepted_count*td.retail_price) AS accepted_retail_sum`).
		Joins("LEFT JOIN transfer_details td ON transfers.id = td.transfer_id").
		Group("transfers.id").
		First(&res, "transfers.id = ?", returnId).Error
	if err != nil {
		s.log.Warn("ERROR on getting return by id: %v", err)
		return nil, err
	}
	err = s.db.First(&res.Store, "id = ?", res.FromStoreId).Error
	if err != nil {
		s.log.Error(err)
		return &res, err
	}

	return &res, nil
}

// update product quantity
func (s *Services) UpdateReturnDetailQuantity(id string, request *domain.ReturnAddProduct) error {

	// get unit per pack
	var returnDetail struct {
		UnitPerPack   float64 `gorm:"unit_per_pack"`
		ReceivedCount float64 `gorm:"received_count"`
		ScannedCount  float64 `gorm:"scanned_count"`
	}
	err := s.db.Raw(`
	SELECT
		td.received_count,
		td.scanned_count,
		p.unit_per_pack
	FROM transfer_details td
	JOIN products p ON td.product_id = p.id
	WHERE td.id = ?;
	`, request.Id).Scan(&returnDetail).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	// update scanned count with pack quantity
	if request.ScannedPack != nil {
		if float64(*request.ScannedPack) > returnDetail.ReceivedCount {
			return errors.New("scanned count could not be greater current count")
		}
		updateField := "scanned_count"
		if request.Status == "sent" {
			updateField = "accepted_count"
		}
		// add scanned count by transfer detail id
		err = s.db.Exec(fmt.Sprintf(`
		UPDATE 
			transfer_details
		SET
			%s = ?, updated_at = NOW()
		WHERE
			id = ? AND transfer_id = ?;`, updateField),
			request.ScannedPack, request.Id, id).Error
		if err != nil {
			s.log.Error(err)
			return err
		}
		return nil
	}

	// update scanned count with unit quantity
	if request.ScannedUnit != nil {
		quantity := float64(int(returnDetail.ScannedCount)) + float64(*request.ScannedUnit)/returnDetail.UnitPerPack
		if quantity > returnDetail.ReceivedCount {
			return errors.New("scanned count could not be greater current count")
		}
		updateField := "scanned_count"
		if request.Status == "sent" {
			updateField = "accepted_count"
		}
		// add scanned count by transfer detail id
		err = s.db.Exec(fmt.Sprintf(`
		UPDATE 
			transfer_details
		SET 
			%s = ?, updated_at = NOW()
		WHERE 
			id = ? AND transfer_id = ?;`, updateField),
			quantity, request.Id, id).Error
		if err != nil {
			s.log.Error(err)
			return err
		}
		return nil
	}

	return nil

}

// get return list
func (s *Services) ReturnList(param *domain.ReturnParam) ([]domain.Return, int64, error) {
	var res []domain.Return
	var totalCount int64
	query := s.db.Model(&domain.Transfer{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Preload("AcceptedBy").
		Select(`
			transfers.*, 
			SUM(trd.scanned_count) AS return_count,
			SUM(trd.received_count-trd.scanned_count) AS shortage,
			SUM(CASE WHEN trd.accepted_count > trd.received_count THEN trd.accepted_count - trd.received_count ELSE 0 END) AS surplus,
			SUM(trd.scanned_count*trd.supply_price) AS received_supply_sum,
			SUM(trd.scanned_count*trd.retail_price) AS received_retail_sum,
			SUM(trd.accepted_count*trd.supply_price) AS accepted_supply_sum,
			SUM(trd.accepted_count*trd.retail_price) AS accepted_retail_sum
			`).
		Joins("LEFT JOIN transfer_details trd ON transfers.id = trd.transfer_id").
		Where("transfers.entry_type = ?", 2)
	// filter by store id
	if param.StoreId != "" {
		query = query.Where("transfers.from_store_id = ? ", param.StoreId)
	}

	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("transfers.public_id LIKE ? OR transfers.name ILIKE ?", param.Search, param.Search)
	}

	// filter by return status
	if param.Status != "" {
		query = query.Where("transfers.status = ?", param.Status)
	}
	// complete query
	err := query.
		Group("transfers.id").
		Order("transfers.created_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}
	return res, totalCount, nil
}

// get
func (s *Services) ReturnStatus(param *domain.ReturnParam) (*domain.ReturnStatusSummary, error) {
	query := `
		SELECT
			SUM(trd.scanned_count) AS return_count,
			COALESCE(SUM(trd.scanned_count*trd.retail_price), 0) AS received_retail_sum,
			COALESCE(SUM(trd.accepted_count * trd.retail_price), 0) AS accepted_retail_sum
		FROM transfers
		LEFT JOIN transfer_details trd ON transfers.id = trd.transfer_id
		WHERE transfers.entry_type = 2
	`

	var args []any
	var filters []string

	if param.StoreId != "" {
		filters = append(filters, "transfers.from_store_id = ?")
		args = append(args, param.StoreId)
	}
	if param.Search != "" {
		search := "%" + param.Search + "%"
		filters = append(filters, "(transfers.public_id ILIKE ? OR transfers.name ILIKE ?)")
		args = append(args, search, search)
	}
	if param.Status != "" {
		filters = append(filters, "transfers.status = ?")
		args = append(args, param.Status)
	}

	if len(filters) > 0 {
		query += " AND " + strings.Join(filters, " AND ")
	}

	var res domain.ReturnStatusSummary
	if err := s.db.Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Error("Failed to get return status summary: %v", err)
		return nil, err
	}

	return &res, nil
}

// get  detail list
func (s *Services) ReturnDetailList(param *domain.ReturnDetailParam) ([]domain.ReturnDetail, int64, error) {
	var res []domain.ReturnDetail
	var totalCount int64
	query := s.db.
		Model(&domain.TransferDetail{}).
		Select(`
			transfer_details.id, 
			transfer_details.transfer_id as return_id, 
			transfer_details.product_id, 
			transfer_details.store_product_id,
			transfer_details.serial_number, 
			transfer_details.expire_date,
			transfer_details.supply_price, 
			transfer_details.retail_price,
			transfer_details.received_count,
			transfer_details.created_at, 
			transfer_details.updated_at,
			FLOOR(transfer_details.scanned_count) AS scanned_count,
			ROUND(MOD(transfer_details.scanned_count * p.unit_per_pack, p.unit_per_pack), 0) AS scanned_unit,
			FLOOR(transfer_details.accepted_count) AS accepted_count,
			ROUND(MOD(transfer_details.accepted_count * p.unit_per_pack, p.unit_per_pack), 0) AS accepted_unit,
			ROUND(transfer_details.received_count*transfer_details.retail_price, 2) AS received_sum,
			ROUND(transfer_details.scanned_count*transfer_details.retail_price, 2) AS scanned_sum,
    		p.name, p.material_code, p.unit_per_pack, p.barcode, ut.short_name`).
		Joins("JOIN products p ON transfer_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("transfer_details.transfer_id = ?", param.ReturnId)

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
	// filter with return stats
	if param.Type != "" {
		switch param.Type {
		case "shortage":
			query = query.Where("transfer_details.received_count > transfer_details.scanned_count")
		case "scanned":
			query = query.Where("transfer_details.scanned_count > 0")
		case "surplus":
			query = query.Where("transfer_details.scanned_count > transfer_details.received_count")

		}
	}

	err := query.
		Order("transfer_details.updated_at DESC").
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, 0, err
	}

	return res, totalCount, nil
}

// get return detail status count
func (s *Services) ReturnDetailStatsCount(param *domain.ReturnDetailParam) (domain.ReturnDetailStatus, error) {
	var res domain.ReturnDetailStatus

	query := `
	SELECT
		SUM(scanned_count) AS scanned,
		SUM(received_count - scanned_count) AS shortage,
		SUM(received_count) AS "all",
		SUM(CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END) AS surplus,
		SUM(accepted_count) AS accepted,
		SUM((received_count - scanned_count)*transfer_details.supply_price) AS shortage_supply_sum,
		SUM((received_count - scanned_count)*transfer_details.retail_price) AS shortage_retail_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*transfer_details.supply_price) AS surplus_supply_sum,
		SUM((CASE WHEN scanned_count > received_count THEN scanned_count - received_count ELSE 0 END)*transfer_details.retail_price) AS surplus_retail_sum
	FROM transfer_details
	JOIN products p ON transfer_details.product_id = p.id
	WHERE transfer_id = ?;
	`
	err := s.db.Raw(query, param.ReturnId).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return res, err
	}

	return res, nil
}

// send return
func (s *Services) SendReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log)
	// update confirm return
	query := `UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`
	err := tx.Exec(query, config.SENT, userId, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating return %v", err)
		return err
	}

	query2 := `DELETE FROM transfer_details WHERE scanned_count = 0 AND transfer_id = ?;`
	err = tx.Exec(query2, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on deleting scanned 0 return details: %v", err)
		return err
	}
	var returnDetails []domain.ReturnDetail
	query3 := `
		SELECT 
			td.*, 
			p.unit_per_pack
        FROM transfer_details td
		JOIN products p ON td.product_id = p.id
		WHERE td.transfer_id = ? and td.scanned_count > 0;
	`
	err = tx.Raw(query3, returnId).Scan(&returnDetails).Error
	if err != nil {
		s.log.Warn("ERROR on getting return details: %v", err)
		return err
	}

	for _, detail := range returnDetails {
		// update store product quantities
		// if scanned count is 0, skip the update
		err = tx.Exec(`UPDATE store_products SET pack_quantity = GREATEST(?, 0), unit_quantity = GREATEST(unit_quantity - ?, 0), updated_at = NOW() WHERE id = ?`,
			int(detail.ReceivedCount-detail.ScannedCount), math.Round(detail.ScannedCount*float64(detail.UnitPerPack)), detail.StoreProductId).Error
		if err != nil {
			s.log.Warn("ERROR on updating store product pack quantity: %v", err)
			return err
		}
	}
	defer RollbackIfError(tx, &err)
	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		return err
	}
	return nil
}

// confirm return
func (s *Services) SendReturn1C(returnId string) error {
	var (
		returnData domain.ReturnData1C
		store      domain.Store
		returnInfo domain.Transfer
	)
	err := s.db.Model(&domain.Transfer{}).First(&returnInfo, "id = ?", returnId).Error
	if err != nil {
		s.log.Warn("ERROR on getting return data: %v", err)
		return err
	}

	// get store data
	err = s.db.First(&store, "id = ?", returnInfo.FromStoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store data: %v", err)
		return err
	}

	returnData.Dok.DocumentNumber = "NP-" + cast.ToString(returnInfo.PublicId)
	returnData.Dok.DocumentDate = returnInfo.UpdatedAt.Add(time.Hour * 5).Format("2006-01-02T15:04:05")
	returnData.Apteka.Name = store.Name
	returnData.Apteka.StoreCode = store.StoreCode
	// get return data
	query := `
	SELECT
		td.id, 
		td.transfer_id, 
		p.material_code, 
		p.name, 
		p.barcode,
		COALESCE(pr.code, '') as manufacturer, 
		td.serial_number AS product_series_number,
		td.expire_date, 
		td.accepted_count as quantity,
		td.supply_price AS supply_price_vat, 
		td.retail_price AS retail_price_vat,
		(td.retail_price*td.accepted_count) AS sum_vat
	FROM transfer_details td
		JOIN transfers tr ON td.transfer_id = tr.id
		JOIN products p ON td.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
		WHERE td.transfer_id = ? AND tr.status = 'completed' AND tr.from_store_id = ?;
	`

	err = s.db.Raw(query, returnId, returnInfo.FromStoreId).Scan(&returnData.Товары).Error
	if err != nil {
		s.log.Error(err)
		return err
	}

	if len(returnData.Товары) < 1 {
		s.log.Warn("No products found for return %s", returnId)
		return nil
	}

	// send return to 1C
	err = s.DoRequest(context.Background(), returnData, "/vozvrat")
	if err != nil {
		s.log.Warn("ERROR on sending return to 1C: %v", err)
		return err
	}
	return nil
}

func (s *Services) EditStatusToCheckingReturn(Id string) error {
	return s.db.Exec("UPDATE transfers SET status = ? WHERE id = ?", config.SENT, Id).Error
}

func (s *Services) BarcodeReturn(Id string, req domain.BarcodeRequest) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var product domain.Product
	if err := tx.Where("barcode = ?", req.Barcode).First(&product).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("product not found: %w", err)
	}

	var detail domain.TransferDetail
	if err := tx.Where("transfer_id = ? AND product_id = ?", Id, product.Id).
		First(&detail).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("transfer detail not found: %w", err)
	}
	if req.AcceptedCount > 0 {
		if err := tx.Model(&detail).Update("accepted_count", req.AcceptedCount).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update accepted_count: %w", err)
		}
	} else {
		if err := tx.Model(&detail).Update("accepted_count", gorm.Expr("accepted_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to increment accepted_count: %w", err)
		}
	}

	return tx.Commit().Error
}

// confirm return
func (s *Services) ConfirmReturn(returnId, storeId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log) // check recover
	var returnInfo domain.Return
	// update confirm return
	query := `UPDATE transfers SET status = ?, accepted_by = ?, accepted_at = NOW() WHERE id = ? RETURNING *`
	err := tx.Raw(query, config.COMPLETED, userId, returnId).Scan(&returnInfo).Error
	if err != nil {
		s.log.Warn("ERROR on updating return %v", err)
		return err
	}
	defer RollbackIfError(tx, &err) // rollback transcation

	var (
		returnData domain.ReturnData1C
		store      domain.Store
	)

	// get return data
	query2 := `
	SELECT
		td.id, 
		td.transfer_id, 
		p.material_code, 
		p.name, 
		p.barcode,
		p.unit_per_pack,
		COALESCE(pr.code, '') as manufacturer, 
		td.serial_number AS product_series_number,
		td.expire_date,
		td.accepted_count as quantity,
		td.supply_price AS supply_price_vat,
		td.retail_price AS retail_price_vat,
		(td.retail_price*td.accepted_count) AS sum_vat,
		td.scanned_count,
		td.accepted_count,
		td.store_product_id
	FROM transfer_details td
		JOIN transfers tr ON td.transfer_id = tr.id
		JOIN products p ON td.product_id = p.id
		LEFT JOIN producers pr ON p.producer_id = pr.id
		WHERE td.transfer_id = ? AND tr.status = 'completed';
	`
	// get return data
	err = tx.Raw(query2, returnId).Scan(&returnData.Товары).Error
	if err != nil {
		s.log.Error(err)
		return err
	}

	if len(returnData.Товары) < 1 {
		s.log.Warn("No products found for return %s", returnId)
		return nil
	}

	for i := range returnData.Товары {
		err = tx.Exec(`
		UPDATE store_products 
		SET 
			pack_quantity = pack_quantity + ?,
			unit_quantity = unit_quantity + ?
		WHERE id = ?;`,
			int(returnData.Товары[i].ScannedCount-returnData.Товары[i].AcceptedCount),
			(returnData.Товары[i].ScannedCount-returnData.Товары[i].AcceptedCount)*float64(returnData.Товары[i].UnitPerPack),
			returnData.Товары[i].StoreProductId).Error
		if err != nil {
			s.log.Error("ERROR on updating store_product on return confirm: %v", err)
			return err
		}
	}

	// get store data
	err = tx.First(&store, "id = ?", storeId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store data: %v", err)
		return err
	}

	returnData.Dok.DocumentNumber = "NP-" + cast.ToString(returnInfo.PublicId)
	returnData.Dok.DocumentDate = returnInfo.UpdatedAt.Format(time.DateTime)
	returnData.Apteka.Name = store.Name
	returnData.Apteka.StoreCode = store.StoreCode

	// send return to 1C
	err = s.DoRequest(context.Background(), returnData, "/vozvrat")
	if err != nil {
		s.log.Warn("ERROR on sending return to 1C: %v", err)
		return err
	}

	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		return err
	}

	return nil
}

// canceled inventory
func (s *Services) CancelReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// update confirm inventory
	query := `UPDATE transfers SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.CANCELED, userId, returnId).Error
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
