package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
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
func (s *Services) UpdateReturnDetailQuantity(ctx context.Context, req *domain.ReturnAddProduct, userId string, transferType int) error {

	// get unit per pack
	var returnDetail struct {
		ProductId     string  `gorm:"product_id"`
		UnitPerPack   float64 `gorm:"unit_per_pack"`
		ReceivedCount float64 `gorm:"received_count"`
		ScannedCount  float64 `gorm:"scanned_count"`
	}
	err := s.db.Raw(`
	SELECT
		td.received_count,
		td.scanned_count,
		p.id AS product_id,
		p.unit_per_pack
	FROM transfer_details td
	JOIN products p ON td.product_id = p.id
	WHERE td.id = ?;
	`, req.Id).Scan(&returnDetail).Error
	if err != nil {
		s.log.Errorf("could not get transfer detail(%s): %v", req.Id, err)
		return domain.InternalServerError
	}

	scannedPack := 0
	if req.ScannedPack != nil {
		scannedPack = *req.ScannedPack
	}

	// transfer log
	transferLog := domain.TransferLog{
		TransferId:       req.TransferId,
		UserId:           userId,
		TransferDetailId: req.Id,
		ProductId:        returnDetail.ProductId,
		TransferType:     transferType,
		Quantity:         scannedPack,
	}

	// update scanned count with pack quantity
	if req.ScannedPack != nil {
		if float64(*req.ScannedPack) > returnDetail.ReceivedCount {
			return errors.New("expected_count could not be greater current count")
		}
		updateField := "expected_count"

		switch req.Status {
		case "checking":
			updateField = "accepted_count"
			transferLog.Stage = constants.TransferLogStageChecking
		case "get":
			updateField = "scanned_count"
			transferLog.Stage = constants.TransferLogStageSent
		}
		// add scanned count by transfer detail id
		err = s.db.Exec(fmt.Sprintf(`
		UPDATE 
			transfer_details
		SET
			%s = ?, updated_at = NOW()
		WHERE
			id = ? AND transfer_id = ?;`, updateField),
			req.ScannedPack, req.Id, req.TransferId).Error
		if err != nil {
			s.log.Errorf("could not update transfer_details: %v", err)
			return domain.InternalServerError
		}
	}

	// update scanned count with unit quantity
	if req.ScannedUnit != nil {
		quantity := float64(int(returnDetail.ScannedCount)) + float64(*req.ScannedUnit)/returnDetail.UnitPerPack
		if quantity > returnDetail.ReceivedCount {
			return errors.New("expected_count could not be greater current count")
		}
		updateField := "expected_count"
		switch req.Status {
		case "checking":
			transferLog.Stage = constants.TransferLogStageChecking
			updateField = "accepted_count"
		case "get":
			transferLog.Stage = constants.TransferLogStageSent
			updateField = "scanned_count"
		}
		// add scanned count by transfer detail id
		err = s.db.Exec(fmt.Sprintf(`
		UPDATE 
			transfer_details
		SET 
			%s = ?, updated_at = NOW()
		WHERE 
			id = ? AND transfer_id = ?;`, updateField),
			quantity, req.Id, req.TransferId).Error
		if err != nil {
			s.log.Errorf("could not update transfer detail unit: %v", err)
			return domain.InternalServerError
		}
	}

	// save transfer log
	go s.SaveTransferLog(&transferLog)

	return nil

}

func (s *Services) UpdateReturnByBarcode(ctx context.Context, req *domain.TransferBarcodeRequest, user *domain.EmployeeClaims) error {

	if req.Count == 0 {
		req.Count = 1
	}
	transferLog := domain.TransferLog{
		TransferId:   req.TransferId,
		UserId:       user.UserId,
		TransferType: constants.TransferTypeReturn,
		Quantity:     req.Count,
		Stage:        constants.TransferLogStageSent,
	}

	if req.Id != "" {
		var productId string
		err := s.db.WithContext(ctx).
			Raw(`
		UPDATE transfer_details
		SET scanned_count = scanned_count + ?
		WHERE id = ? AND received_count >= scanned_count + ?
		RETURNING product_id;`,
				req.Count,
				req.Id,
				req.Count).
			Scan(&productId).Error
		if err != nil {
			s.log.Errorf("could not update transfer_details(%s) scanned_count: %v", req.Id, err)
			return domain.InternalServerError
		}
		transferLog.TransferDetailId = req.Id
		transferLog.ProductId = productId

	} else if req.Barcode != "" {
		var barcodeResponse []domain.TransferBarcodeResponse
		err := s.db.WithContext(ctx).
			Raw(`
			SELECT 
				t.id,
				t.product_id,
				p.name
			FROM transfer_details t 
			JOIN products p ON p.id = t.product_id 
			WHERE p.barcode = ? AND t.transfer_id = ?`,
				req.Barcode,
				req.TransferId).
			Scan(&barcodeResponse).Error
		if err != nil {
			s.log.Errorf("could not get transfer_details by barcode(%s): %v", req.Barcode, err)
			return domain.InternalServerError
		}
		if len(barcodeResponse) > 1 {
			return domain.DuplicateError
		}
		transferLog.TransferDetailId = barcodeResponse[0].Id
		transferLog.ProductId = barcodeResponse[0].ProductId

		err = s.db.WithContext(ctx).Exec(`
		UPDATE transfer_details t 
		SET scanned_count = scanned_count + ? 
		FROM products p 
		WHERE 
			t.transfer_id = ? AND 
			p.id = t.product_id AND 
			p.barcode = ? AND 
			t.received_count >= t.scanned_count + ?;`,
			req.Count,
			req.TransferId,
			req.Barcode,
			req.Count).Error
		if err != nil {
			s.log.Errorf("could not update transfer_details by barcode(%s): %v", req.Barcode, err)
			return domain.InternalServerError
		}
	} else {
		return domain.InvalidRequestBodyError
	}

	go s.SaveTransferLog(&transferLog)

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
	if param.CompanyId != "" {
		query = query.Joins("LEFT JOIN stores st ON transfers.from_store_id = st.id").
			Where("st.company_id = ?", param.CompanyId)
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
		LEFT JOIN stores st ON transfers.from_store_id = st.id
		WHERE transfers.entry_type = 2
	`

	var args []any
	var filters []string

	if param.StoreId != "" {
		filters = append(filters, "transfers.from_store_id = ?")
		args = append(args, param.StoreId)
	}
	if param.CompanyId != "" {
		filters = append(filters, "st.company_id = ?")
		args = append(args, param.CompanyId)
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
			FLOOR(transfer_details.expected_count) AS expected_count,
			ROUND(MOD(transfer_details.expected_count * p.unit_per_pack, p.unit_per_pack), 0) AS expected_unit,
			FLOOR(transfer_details.scanned_count) AS scanned_count,
			ROUND(MOD(transfer_details.scanned_count * p.unit_per_pack, p.unit_per_pack), 0) AS scanned_unit,
			FLOOR(transfer_details.accepted_count) AS accepted_count,
			FLOOR(transfer_details.onec_count) AS onec_count,
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
func (s *Services) SendReturn(ctx context.Context, returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// update return
	query := `UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`
	err := tx.WithContext(ctx).Exec(query, constants.GeneralStatusSent, userId, returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update return %v", err)
		return domain.InternalServerError
	}

	query2 := `DELETE FROM transfer_details WHERE expected_count = 0 AND transfer_id = ?;`
	err = tx.WithContext(ctx).Exec(query2, returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not delete scanned 0 return details: %v", err)
		return domain.InternalServerError
	}
	var details []domain.ReturnDetail
	query3 := `
		SELECT 
			td.*, 
			p.unit_per_pack
        FROM transfer_details td
		JOIN products p ON td.product_id = p.id
		WHERE td.transfer_id = ? and td.expected_count > 0;
	`
	err = tx.WithContext(ctx).Raw(query3, returnId).Scan(&details).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get return details: %v", err)
		return domain.InternalServerError
	}

	for _, detail := range details {
		// update store product quantities
		// if scanned count is 0, skip the update
		err = tx.WithContext(ctx).Exec(`UPDATE store_products SET unit_quantity = unit_quantity - ?, updated_at = NOW() WHERE id = ?`,
			(detail.ExpectedCount * float64(detail.UnitPerPack)), detail.StoreProductId).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store product pack quantity: %v", err)
			return domain.InternalServerError
		}
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
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
	err = s.DoRequestOnec(context.Background(), returnData, "/vozvrat")
	if err != nil {
		s.log.Warn("ERROR on sending return to 1C: %v", err)
		return err
	}
	return nil
}

func (s *Services) EditStatusToCheckingReturn(ctx context.Context, Id string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// update transfer status
	err := tx.WithContext(ctx).Exec("UPDATE transfers SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?", constants.GeneralStatusChecking, userId, Id).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update transfer(%s) status: %v", Id, err)
		return domain.InternalServerError
	}

	var res []domain.TransferDetail
	err = tx.WithContext(ctx).
		Raw(`
	SELECT 
		t.id, 
		t.received_count,
		t.expected_count,
		t.scanned_count,
		t.store_product_id,
		p.unit_per_pack
	FROM 
		transfer_details t 
	JOIN 
		products p ON p.id = t.product_id 
	WHERE 
		t.transfer_id = ?`, Id).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not select transfer_details by transfer_id(%s): %v", Id, err)
		return domain.InternalServerError
	}
	for _, item := range res {
		err = tx.WithContext(ctx).
			Exec(`
		UPDATE store_products 
		SET 
			unit_quantity = unit_quantity + ?
		WHERE id = ?;`,
				(item.ExpectedCount-item.ScannedCount)*float64(item.UnitPerPack),
				item.StoreProductId).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store_products(%s) %v", item.StoreProductId, err)
			return domain.InternalServerError
		}
	}
	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Error("could not completed transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// confirm return
func (s *Services) ConfirmReturn(ctx context.Context, returnId, storeId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	var returnInfo domain.Return
	// update confirm return
	query := `UPDATE transfers SET status = ?, accepted_by = ?, accepted_at = NOW() WHERE id = ? RETURNING *`
	err := tx.WithContext(ctx).Raw(query, constants.GeneralStatusSentOnec, userId, returnId).Scan(&returnInfo).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update return %v", err)
		return domain.InternalServerError
	}

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
		WHERE td.transfer_id = ? AND tr.status = 'sent-to-1c';
	`
	// get return data
	err = tx.WithContext(ctx).Raw(query2, returnId).Scan(&returnData.Товары).Error
	if err != nil {
		s.log.Errorf("could not get return data %v", err)
		return domain.InternalServerError
	}

	if len(returnData.Товары) < 1 {
		_ = tx.Rollback()
		s.log.Errorf("No products found for return %s", returnId)
		return domain.NotEnoughProductError
	}

	for i := range returnData.Товары {
		err = tx.WithContext(ctx).Exec(`
		UPDATE store_products
		SET
			unit_quantity = unit_quantity + ?
		WHERE id = ?;`,
			(returnData.Товары[i].ScannedCount-returnData.Товары[i].AcceptedCount)*float64(returnData.Товары[i].UnitPerPack),
			returnData.Товары[i].StoreProductId).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store_product on return confirm: %v", err)
			return domain.InternalServerError
		}
	}

	// get store data
	err = tx.WithContext(ctx).First(&store, "id = ?", storeId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get store data: %v", err)
		return domain.InternalServerError
	}

	returnData.Dok.DocumentNumber = "NP-" + cast.ToString(returnInfo.PublicId)
	returnData.Dok.DocumentDate = returnInfo.UpdatedAt.Format(constants.DateTimeFormatRFC3339)
	returnData.Apteka.Name = store.Name
	returnData.Apteka.StoreCode = store.StoreCode

	// complete transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}
	if s.cfg.OnecApiUrl != "test" {
		// send return to 1C
		err = s.DoRequestOnec(context.Background(), returnData, constants.OnecPathVozvrat)
		if err != nil {
			s.log.Errorf("could not send return to 1C: %v", err)
			return domain.InternalServerError
		}
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
	err := tx.Exec(query, constants.GeneralStatusCanceled, userId, returnId).Error
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
