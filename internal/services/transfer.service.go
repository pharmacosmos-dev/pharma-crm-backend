package services

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"math"
	"time"

	"github.com/spf13/cast"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
)

// Create inventory creates a new inventory
func (s *Services) CreateTransfer(req *domain.TransferRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log) // recover if panic happened
	// insert inventory into inventories table
	err := tx.Raw(`
	INSERT INTO 
		transfers (
			from_store_id, 
			to_store_id, 
			name,  
			created_by
			)
	VALUES (?, ?, ?, ?) 
	RETURNING id`,
		req.FromStoreId,
		req.ToStoreId,
		req.Name,
		req.CreatedBy,
	).Scan(&id).Error
	if err != nil {
		s.log.Warn("ERROR on creating return: %v", err)
		return err
	}
	defer RollbackIfError(tx, &err) // return transaction if error happened

	// if no products provided, get all products from store_products
	// and insert them into inventory_details
	err = tx.Exec(`
		INSERT INTO transfer_details(
			transfer_id, 
			store_product_id, 
			product_id, 
			received_count, 
			supply_price, 
			retail_price, 
			expire_date, 
			serial_number
			) SELECT 
				?, 
				sp.id, 
				sp.product_id, 
				sp.unit_quantity::numeric/p.unit_per_pack, 
				sp.supply_price,
				sp.retail_price,
				sp.expire_date, 
				sp.serial_number
			FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			WHERE sp.store_id = ? AND (sp.pack_quantity > 0 OR sp.unit_quantity > 0);
		`, id, req.FromStoreId).Error
	if err != nil {
		s.log.Error("ERROR on creating inventory details: ", err)
		return err
	}

	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Error("ERROR on committing transaction: ", err)
		return err
	}
	return nil
}

// get return by id
func (s *Services) GetTransferById(transferID string) (*domain.Transfer, error) {
	var res domain.Transfer
	err := s.db.Model(&domain.Transfer{}).
		Preload("FromStore").
		Preload("ToStore").
		Preload("CreatedBy").
		Preload("AcceptedBy").
		Select(`
			transfers.*,
			SUM(td.received_count) AS received_count,
			SUM(td.scanned_count) AS accepted_count,
			SUM(td.scanned_count*td.supply_price) AS supply_price_sum,
			SUM(td.scanned_count*td.retail_price) AS retail_price_sum`).
		Joins("LEFT JOIN transfer_details td ON transfers.id = td.transfer_id").
		Group("transfers.id").
		First(&res, "transfers.id = ?", transferID).Error
	if err != nil {
		s.log.Warn("ERROR on getting return by id: %v", err)
		return nil, err
	}
	return &res, nil
}

// get inventory list
func (s *Services) TransferList(param *domain.ReturnParam) ([]domain.Transfer, int64, error) {
	var res []domain.Transfer
	var totalCount int64
	query := s.db.Model(&domain.Transfer{}).
		Preload("FromStore").
		Preload("ToStore").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Preload("AcceptedBy").
		Select(`
			transfers.*,
			SUM(trd.received_count) AS received_count,
			SUM(trd.accepted_count) AS accepted_count,
			SUM(trd.received_count-trd.scanned_count) AS shortage,
			SUM(CASE WHEN trd.accepted_count > trd.received_count THEN trd.accepted_count - trd.received_count ELSE 0 END) AS surplus,
			SUM(trd.received_count*trd.supply_price) AS received_supply_sum,
			SUM(trd.received_count*trd.retail_price) AS received_retail_sum,
			SUM(trd.accepted_count*trd.supply_price) AS accepted_supply_sum,
			SUM(trd.accepted_count*trd.retail_price) AS accepted_retail_sum
			`).
		Joins("LEFT JOIN transfer_details trd ON transfers.id = trd.transfer_id").
		Where("transfers.entry_type = ?", 1)

	// filter by from store id
	if param.StoreId != "" {
		query = query.Where("transfers.from_store_id = ?  OR transfers.to_store_id = ?", param.StoreId, param.StoreId)
	}

	// filter by search keyword
	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where("transfers.public_id LIKE ? OR transfers.name ILIKE ?", param.Search, param.Search)
	}

	// filter by inventory status
	if param.Status != "" {
		query = query.Where("transfers.status = ?", param.Status)
	}

	if param.StartDate != "" {
		query = query.Where("transfers.created_at >= ?", param.StartDate)
	}

	if param.EndDate != "" {
		query = query.Where("transfers.created_at <= ?", param.EndDate)
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

func (s *Services) TransferStatus(param *domain.ReturnParam) (*domain.TransferStatusSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(trd.received_count), 0) AS received_count,
			COALESCE(SUM(trd.accepted_count), 0) AS accepted_count,
			COALESCE(SUM(trd.received_count * trd.retail_price), 0) AS received_retail_sum,
			COALESCE(SUM(trd.accepted_count * trd.retail_price), 0) AS accepted_retail_sum
		FROM transfers
		LEFT JOIN transfer_details trd ON transfers.id = trd.transfer_id
		WHERE transfers.entry_type = 1
	`

	var args []any

	if param.StoreId != "" {
		query += " AND (transfers.from_store_id = ? OR transfers.to_store_id = ?)"
		args = append(args, param.StoreId, param.StoreId)
	}
	if param.Search != "" {
		search := fmt.Sprintf("%%%s%%", param.Search)
		query += " AND (transfers.public_id ILIKE ? OR transfers.name ILIKE ?)"
		args = append(args, search, search)
	}
	if param.Status != "" {
		query += " AND transfers.status = ?"
		args = append(args, param.Status)
	}

	var res domain.TransferStatusSummary
	err := s.db.Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Error("Failed to get transfer status summary: %v", err)
		return nil, err
	}

	return &res, nil
}

// get inventory detail list
func (s *Services) TransferDetailList(param *domain.ReturnDetailParam) ([]domain.TransferDetail, int64, error) {
	var res []domain.TransferDetail
	var totalCount int64
	query := s.db.
		Model(&domain.TransferDetail{}).
		Select(`
			transfer_details.id,
			transfer_details.product_id, 
			transfer_details.transfer_id,
			transfer_details.received_count,
			transfer_details.accepted_count,
			FLOOR(transfer_details.scanned_count) AS scanned_count,
			ROUND(MOD(transfer_details.scanned_count * p.unit_per_pack, p.unit_per_pack), 0) AS scanned_unit,
			transfer_details.expire_date, 
			transfer_details.serial_number, 
			transfer_details.supply_price, 
			transfer_details.retail_price,
			transfer_details.created_at, 
			transfer_details.updated_at,
			ROUND(transfer_details.received_count*transfer_details.retail_price, 2) AS received_sum,
			ROUND(transfer_details.scanned_count*transfer_details.retail_price, 2) AS scanned_sum,
			p.name, 
			p.material_code, 
			p.unit_per_pack, 
			p.barcode, 
			ut.short_name
			`).
		Joins("JOIN products p ON transfer_details.product_id = p.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("transfer_details.transfer_id = ?", param.TransferId)

	if param.Search != "" {
		query = query.Where("p.name ILIKE ? OR p.barcode LIKE ?", "%"+param.Search+"%", "%"+param.Search+"%")
	}
	// filter with inventory stats
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

// get inventory detail status count
func (s *Services) TransferDetailStatsCount(param *domain.ReturnDetailParam) (domain.TransferDetailStatus, error) {
	var res domain.TransferDetailStatus

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

// confirm inventory
func (s *Services) SendTransfer(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log)

	// update confirm inventory
	query := `UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`
	err := tx.Exec(query, config.SENT, userId, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		return err
	}
	// rollback transaction if error happened
	defer RollbackIfError(tx, &err)

	query2 := `DELETE FROM transfer_details WHERE scanned_count = 0 AND transfer_id = ?;`
	err = tx.Exec(query2, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on deleting scanned 0 inventory details: %v", err)
		return err
	}

	var returnDetails []domain.TransferDetail
	query3 := `
		SELECT td.*, p.unit_per_pack
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

	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on commiting transaction: %v", err)
		return err
	}
	return nil
}

func (s *Services) EditStatusToCheckingTransfer(id string) error {
	return s.db.Model(&domain.Transfer{}).Where("id = ?", id).Update("status", config.CHECKING).Error
}

func (s *Services) BarcodeTransfer(Id string, req domain.BarcodeRequest) error {
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
	err := tx.Where("transfer_id = ? AND product_id = ?", Id, product.Id).
		First(&detail).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("transfer detail not found: %w", err)
	}
	if req.AcceptedCount > 0 {
		if err = tx.Model(&detail).Update("accepted_count", req.AcceptedCount).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update accepted_count: %w", err)
		}
	} else {
		if err = tx.Model(&detail).Update("accepted_count", gorm.Expr("accepted_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to increment accepted_count: %w", err)
		}
	}

	return tx.Commit().Error
}

func (s *Services) SendTransferTo1C(transferID string) error {
	var transfer domain.Transfer
	err := s.db.First(&transfer, "id = ?", transferID).Error
	if err != nil {
		s.log.Warn("ERROR on getting transfer: %v", err)
		return err
	}

	var details []domain.TransferDetail
	err = s.db.Raw(`
	SELECT
		td.id,
		td.transfer_id,
		td.product_id,
		td.store_product_id,
		td.received_count,
		td.scanned_count,
		td.accepted_count,
		td.expire_date,
		td.serial_number,
		td.supply_price AS supply_price_vat,
		td.retail_price AS retail_price_vat,
		td.created_at,
		td.updated_at,
		p.unit_per_pack,
		COALESCE(p.name,'') AS product_name,
		p.material_code,
		p.barcode,
		COALESCE(pr.code, '') AS producer_code,
		COALESCE(idt.retail_price, 0.00) AS retail_price,
		COALESCE(idt.supply_price, 0.00) AS supply_price
	FROM transfer_details td
		JOIN products p ON p.id = td.product_id
		LEFT JOIN producers pr ON pr.id = p.producer_id
		LEFT JOIN store_products sp ON sp.id = td.store_product_id
		LEFT JOIN import_details idt ON idt.id = sp.import_detail_id
	WHERE
		td.transfer_id = ? AND td.scanned_count > 0;
	`, transferID).Scan(&details).Error
	if err != nil {
		s.log.Warn("ERROR on getting transfer_detail list: %v", err)
		return err
	}

	// get store info
	var toStore, fromStore domain.Store
	err = s.db.First(&toStore, "id = ?", transfer.ToStoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting toStore info: %v", err)
		return err
	}

	err = s.db.First(&fromStore, "id = ?", transfer.FromStoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting fromStore info: %v", err)
		return err
	}

	var data1C domain.TransferData1C

	for _, v := range details {
		data1C.Товары = append(data1C.Товары, domain.TransferProduct1C{
			MaterialCode:        v.MaterialCode,
			Name:                v.ProductName,
			Barcode:             v.Barcode,
			Manufacturer:        v.ProducerCode,
			ProductSeriesNumber: v.SerialNumber,
			ExpireDate:          v.ExpireDate,
			Quantity:            v.ReceivedCount,
			RetailPrice:         v.RetailPrice,
			RetailPriceVat:      v.RetailPriceVat,
			SupplyPrice:         v.SupplyPrice,
			SupplyPriceVat:      v.SupplyPriceVat,
			Sum:                 v.ScannedCount * v.RetailPrice,
			SumVat:              v.ScannedCount * v.RetailPriceVat,
		})
	}

	data1C.Dok.DocumentDate = transfer.UpdatedAt.Add(5 * time.Hour).Format(time.RFC3339)
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(transfer.PublicId)

	data1C.Apteka.Name = toStore.Name
	data1C.Apteka.StoreCode = toStore.StoreCode
	data1C.AptekaOtkud.Name = fromStore.Name
	data1C.AptekaOtkud.StoreCode = fromStore.StoreCode
	fmt.Println("--------->", data1C)
	err = s.DoRequest(context.Background(), data1C, "/perekit")
	if err != nil {
		s.log.Warn("ERROR on sending to 1C: %v", err)
		return err
	}

	return nil
}

// confirm inventory
func (s *Services) ConfirmTransfer(transferID string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log) // recover function for checking panic
	// update confirm inventory
	var transfer domain.Transfer
	query := `UPDATE transfers SET status = ?, accepted_by = ?, accepted_at = NOW() WHERE id = ? RETURNING *`
	err := tx.Raw(query, config.COMPLETED, userId, transferID).Scan(&transfer).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		return err
	}
	defer RollbackIfError(tx, &err) // rollback function for return transcation

	// update confirm inventory details
	query1 := `UPDATE transfer_details SET accepted_count = scanned_count, updated_at = NOW() WHERE transfer_id = ?`
	err = tx.Exec(query1, transferID).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory details: %v", err)
		return err
	}
	var res []domain.TransferDetail
	err = tx.Raw(`
	SELECT
		td.id,
		td.transfer_id,
		td.product_id,
		td.store_product_id,
		td.received_count,
		td.scanned_count,
		td.accepted_count,
		td.expire_date,
		td.serial_number,
		td.supply_price AS supply_price_vat,
		td.retail_price AS retail_price_vat,
		td.created_at,
		td.updated_at,
		p.unit_per_pack,
		COALESCE(p.name,'') AS product_name,
		p.material_code,
		p.barcode,
		COALESCE(pr.code, '') AS producer_code,
		COALESCE(idt.retail_price, 0.00) AS retail_price,
		COALESCE(idt.supply_price, 0.00) AS supply_price
	FROM transfer_details td
		JOIN products p ON p.id = td.product_id
		LEFT JOIN producers pr ON pr.id = p.producer_id
		LEFT JOIN store_products sp ON sp.id = td.store_product_id
		LEFT JOIN import_details idt ON idt.id = sp.import_detail_id
		WHERE
			td.transfer_id = ? AND td.scanned_count > 0;
	`, transferID).Scan(&res).Error
	if err != nil {
		s.log.Warn("ERROR on gettig transfer_detail list: %v", err)
		return err
	}

	// insert transfered products to store_product
	query2 := `
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
				serial_number
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var data1C domain.TransferData1C
	for _, v := range res {
		// execute query
		err = tx.Exec(query2,
			v.ProductId,
			transfer.ToStoreId,
			v.ScannedCount,
			math.Round(v.ScannedCount*float64(v.UnitPerPack)),
			v.RetailPriceVat,
			v.SupplyPriceVat,
			12,
			v.ExpireDate,
			(v.RetailPriceVat*12)/112,
			v.SerialNumber).Error
		if err != nil {
			s.log.Warn("ERROR on inserting store product: %v", err)
			return err
		}

		// collect inventory products to send 1C
		data1C.Товары = append(data1C.Товары, domain.TransferProduct1C{
			MaterialCode:        v.MaterialCode,
			Name:                v.ProductName,
			Barcode:             v.Barcode,
			Manufacturer:        v.ProducerCode,
			ProductSeriesNumber: v.SerialNumber,
			ExpireDate:          v.ExpireDate,
			Quantity:            v.ReceivedCount,
			RetailPrice:         v.RetailPrice, // vat bilan oddiysi almashgan
			RetailPriceVat:      v.RetailPriceVat,
			SupplyPrice:         v.SupplyPrice,
			SupplyPriceVat:      v.SupplyPriceVat,
			Sum:                 v.ScannedCount * v.RetailPrice,
			SumVat:              v.ScannedCount * v.RetailPriceVat,
		})
	}
	// get store info
	var toStore domain.Store
	err = tx.First(&toStore, "id = ?", transfer.ToStoreId).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info: %v", err)
		return err
	}

	// get store info
	var fromStore domain.Store
	err = tx.First(&fromStore, "id = ?", transfer.FromStoreId).Error
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

	// get document data and number
	data1C.Dok.DocumentDate = transfer.UpdatedAt.Format(time.RFC3339)
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(transfer.PublicId)

	// get store info
	data1C.Apteka.Name = toStore.Name
	data1C.Apteka.StoreCode = toStore.StoreCode
	data1C.AptekaOtkud.Name = fromStore.Name
	data1C.AptekaOtkud.StoreCode = fromStore.StoreCode

	// send inventory products data to 1C
	err = s.DoRequest(context.Background(), data1C, "/perekit")
	if err != nil {
		s.log.Warn("ERROR on sending inventory: %v", err)
		return err
	}
	return nil
}

// canceled inventory
func (s *Services) CancelTransfer(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer recoverTransaction(tx, s.log)
	// update confirm inventory
	query := `UPDATE transfers SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, config.CANCELED, userId, returnId).Error
	if err != nil {
		s.log.Warn("ERROR on updating inventory %v", err)
		return err
	}
	defer RollbackIfError(tx, &err)
	err = tx.Commit().Error
	if err != nil {
		s.log.Warn("ERROR on commiting transaction %v", err)
		return err
	}

	return nil
}
