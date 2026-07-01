package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// Create return creates a new return
func (s *Services) CreateReturn(ctx context.Context, req *domain.ReturnRequest) error {
	var activeInventoryCount int64
	err := s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM imports
		WHERE store_id = ?
		AND entry_type = 2
		AND status NOT IN ('completed', 'canceled')
	`, req.StoreId).Scan(&activeInventoryCount).Error
	if err != nil {
		s.log.Errorf("could not check active inventory: %v", err)
		return domain.InternalServerError
	}
	if activeInventoryCount > 0 {
		return domain.ActiveInventoryError
	}

	var id string
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// insert return into inventories table
	err = tx.WithContext(ctx).Raw(`
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
		_ = tx.Rollback()
		s.log.Errorf("could not create return: %v", err)
		return domain.InternalServerError
	}

	// if no products provided, get all products from store_products
	// and insert them into return_details
	err = tx.WithContext(ctx).Exec(
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
		    sp.store_id = ? AND sp.unit_quantity::numeric/p.unit_per_pack > 0`,
		id, req.StoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create return details: %v", err)
		return domain.InternalServerError
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit create return transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// get return by id
func (s *Services) GetReturnById(ctx context.Context, returnId string) (*domain.Return, error) {
	var res domain.Return
	err := s.db.WithContext(ctx).Model(&domain.Transfer{}).
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
		s.log.Errorf("could not get return by id: %v", err)
		return nil, domain.InternalServerError
	}

	err = s.db.WithContext(ctx).First(&res.Store, "id = ?", res.FromStoreId).Error
	if err != nil {
		s.log.Errorf("could not get store: %v", err)
		return &res, domain.InternalServerError
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
		ExpectedCount float64 `gorm:"expected_count"`
		ScannedCount  float64 `gorm:"scanned_count"`
		AcceptedCount float64 `gorm:"accepted_count"`
		RejectedCount float64 `gorm:"column:rejection_count"`
	}
	err := s.db.WithContext(ctx).Raw(`
	SELECT
		td.received_count,
		td.expected_count,
		td.scanned_count,
		td.accepted_count,
		td.rejection_count,
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

	var scannedPackVal float64
	if req.ScannedPack != nil {
		scannedPackVal = *req.ScannedPack
	}

	// transfer log
	transferLog := domain.TransferLog{
		TransferId:       req.TransferId,
		UserId:           userId,
		TransferDetailId: req.Id,
		ProductId:        returnDetail.ProductId,
		TransferType:     transferType,
		Quantity:         int(scannedPackVal),
	}

	// Version 2: new transfers use explicit pack + unit fields.
	// Old transfers keep expected_pack/expected_unit = NULL and fall through to version 1.
	if req.Pack != nil || req.Unit != nil {
		pack := 0
		unit := 0
		if req.Pack != nil {
			pack = *req.Pack	
		}
		if req.Unit != nil {
			unit = *req.Unit
		}
		count := math.Round((float64(pack)+float64(unit)/returnDetail.UnitPerPack)*10000) / 10000

		switch req.Status {
		case "rejection":
			transferLog.Stage = constants.TransferLogStageChecking
			if count != returnDetail.RejectedCount {
				return errors.New("invalid.quantity")
			}
			err = s.db.WithContext(ctx).Exec(`
				UPDATE transfer_details
				SET rejection_pack = ?, rejection_unit = ?, updated_at = NOW()
				WHERE id = ? AND transfer_id = ?
			`, pack, unit, req.Id, req.TransferId).Error
		case "checking":
			transferLog.Stage = constants.TransferLogStageChecking
			if count > returnDetail.ScannedCount {
				return errors.New("invalid.quantity")
			}
			if transferType == constants.TransferTypeReturn {
				err = s.db.WithContext(ctx).Exec(`
					UPDATE transfer_details
					SET accepted_count = ?, accepted_pack = ?, accepted_unit = ?, updated_at = NOW()
					WHERE id = ? AND transfer_id = ?
				`, count, pack, unit, req.Id, req.TransferId).Error
			} else {
				err = s.db.WithContext(ctx).Exec(`
					UPDATE transfer_details
					SET accepted_count = scanned_count, updated_at = NOW()
					WHERE id = ? AND transfer_id = ?
				`, req.Id, req.TransferId).Error
			}
		case "get":
			transferLog.Stage = constants.TransferLogStageSent
			if count > returnDetail.ExpectedCount {
				return errors.New("invalid.quantity")
			}
			err = s.db.WithContext(ctx).Exec(`
				UPDATE transfer_details
				SET scanned_count = ?,
				    scanned_pack = ?,
				    scanned_unit = ?,
				    updated_at   = NOW()
				WHERE id = ? AND transfer_id = ?
			`, count, pack, unit, req.Id, req.TransferId).Error
		default:
			transferLog.Stage = constants.TransferLogStageSent
			if count > returnDetail.ReceivedCount {
				return errors.New("invalid.quantity")
			}
			err = s.db.WithContext(ctx).Exec(`
				UPDATE transfer_details
				SET expected_count = ?,
					expected_pack  = ?,
				    expected_unit  = ?,
				    updated_at     = NOW()
				WHERE id = ? AND transfer_id = ?
			`, count, pack, unit, req.Id, req.TransferId).Error
		}
		if err != nil {
			s.log.Errorf("could not update transfer_detail pack/unit(%s): %v", req.Id, err)
			return domain.InternalServerError
		}
		transferLog.Quantity = pack
		go s.SaveTransferLog(&transferLog)
		return nil
	}

	// update scanned count with pack quantity
	if req.ScannedPack != nil {
		if *req.ScannedPack > returnDetail.ReceivedCount {
			return errors.New("invalid.quantity")
		}
		updateField := "expected_count"

		switch req.Status {
		case "checking":
			updateField = "accepted_count"
			transferLog.Stage = constants.TransferLogStageChecking
			if *req.ScannedPack > returnDetail.ScannedCount {
				return errors.New("invalid.quantity")
			}
		case "get":
			updateField = "scanned_count"
			transferLog.Stage = constants.TransferLogStageSent
			if *req.ScannedPack > returnDetail.ExpectedCount {
				return errors.New("invalid.quantity")
			}
		}
		err = s.db.WithContext(ctx).Exec(fmt.Sprintf(`
		UPDATE transfer_details
		SET %s = ?, updated_at = NOW()
		WHERE id = ? AND transfer_id = ?;`, updateField),
			*req.ScannedPack, req.Id, req.TransferId).Error
		if err != nil {
			s.log.Errorf("could not update transfer_details: %v", err)
			return domain.InternalServerError
		}
	}

	// update scanned count with unit quantity
	if req.ScannedUnit != nil {
		unitQty := float64(*req.ScannedUnit) / returnDetail.UnitPerPack
		updateField := "expected_count"
		var quantity float64
		switch req.Status {
		case "checking":
			transferLog.Stage = constants.TransferLogStageChecking
			updateField = "accepted_count"
			quantity = float64(int(returnDetail.AcceptedCount)) + unitQty
			if quantity > returnDetail.ScannedCount {
				return errors.New("invalid.quantity")
			}
		case "get":
			transferLog.Stage = constants.TransferLogStageSent
			updateField = "scanned_count"
			quantity = float64(int(returnDetail.ScannedCount)) + unitQty
			if quantity > returnDetail.ExpectedCount {
				return errors.New("invalid.quantity")
			}
		default:
			quantity = float64(int(returnDetail.ExpectedCount)) + unitQty
			if quantity > returnDetail.ReceivedCount {
				return errors.New("invalid.quantity")
			}
		}
		err = s.db.WithContext(ctx).Exec(fmt.Sprintf(`
		UPDATE transfer_details
		SET %s = ?, updated_at = NOW()
		WHERE id = ? AND transfer_id = ?;`, updateField),
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

// UpdateTransferDetailByUnit updates transfer detail quantity using dona (unit) count
// with math.Round to handle fractional pack counts (e.g. 1.3333, 1.5, 1.6667)
func (s *Services) UpdateTransferDetailByUnit(ctx context.Context, req *domain.ReturnAddProduct, userId string) error {
	var detail struct {
		ProductId     string  `gorm:"product_id"`
		UnitPerPack   float64 `gorm:"unit_per_pack"`
		ReceivedCount float64 `gorm:"received_count"`
		ExpectedCount float64 `gorm:"expected_count"`
		ScannedCount  float64 `gorm:"scanned_count"`
	}
	err := s.db.WithContext(ctx).Raw(`
	SELECT
		td.received_count,
		td.expected_count,
		td.scanned_count,
		p.id AS product_id,
		p.unit_per_pack
	FROM transfer_details td
	JOIN products p ON td.product_id = p.id
	WHERE td.id = ?;
	`, req.Id).Scan(&detail).Error
	if err != nil {
		s.log.Errorf("could not get transfer detail(%s): %v", req.Id, err)
		return domain.InternalServerError
	}

	receivedDona := math.Round(detail.ReceivedCount * detail.UnitPerPack)

	// if scanned_unit not provided, use full received_count
	var newTotalDona float64
	var logQuantity int
	if req.ScannedUnit == nil {
		newTotalDona = receivedDona
		logQuantity = int(receivedDona)
	} else {
		scannedDona := math.Round(detail.ScannedCount * detail.UnitPerPack)
		newTotalDona = scannedDona + float64(*req.ScannedUnit)
		logQuantity = *req.ScannedUnit
		if newTotalDona > receivedDona {
			return errors.New("invalid.quantity")
		}
	}

	updateField := "expected_count"
	switch req.Status {
	case "checking":
		updateField = "accepted_count"
		expectedDona := math.Round(detail.ExpectedCount * detail.UnitPerPack)
		if req.ScannedUnit != nil && newTotalDona > expectedDona {
			return errors.New("invalid.quantity")
		}
	case "get":
		updateField = "scanned_count"
		expectedDona := math.Round(detail.ExpectedCount * detail.UnitPerPack)
		if req.ScannedUnit != nil && newTotalDona > expectedDona {
			return errors.New("invalid.quantity")
		}
	}

	quantity := newTotalDona / detail.UnitPerPack

	err = s.db.WithContext(ctx).Exec(fmt.Sprintf(`
	UPDATE transfer_details
	SET %s = ?, updated_at = NOW()
	WHERE id = ? AND transfer_id = ?;`, updateField),
		quantity, req.Id, req.TransferId).Error
	if err != nil {
		s.log.Errorf("could not update transfer detail by unit: %v", err)
		return domain.InternalServerError
	}

	go s.SaveTransferLog(&domain.TransferLog{
		TransferId:       req.TransferId,
		UserId:           userId,
		TransferDetailId: req.Id,
		ProductId:        detail.ProductId,
		TransferType:     constants.TransferTypeMove,
		Quantity:         logQuantity,
	})

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
		WHERE id = ? 
			AND received_count >= scanned_count + ?
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
			WHERE p.barcode = ? AND t.transfer_id = ?;`,
				req.Barcode,
				req.TransferId).
			Scan(&barcodeResponse).Error
		if err != nil {
			s.log.Errorf("could not get transfer_details by barcode(%s): %v", req.Barcode, err)
			return domain.InternalServerError
		}
		if len(barcodeResponse) > 1 {
			return domain.NewNotAdditionError(http.StatusMultiStatus, barcodeResponse)
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
			t.expected_count >= t.scanned_count + ?;`,
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
func (s *Services) ReturnList(ctx context.Context, param *domain.ReturnParam) ([]domain.Return, int64, error) {
	var res []domain.Return
	var totalCount int64
	query := s.db.Model(&domain.Transfer{}).
		Preload("Store").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Preload("AcceptedBy").
		Preload("RejectionBy").
		Preload("CommentBy").
		Select(`
			transfers.id,
			transfers.public_id,
			transfers.from_store_id,
			transfers.name,
			transfers.status,
			transfers.comment,
			transfers.comment_by,
			transfers.is_auto,
			transfers.driver_office,
			transfers.driver_store_a,
			transfers.driver_store_b,
			transfers.created_by,
			transfers.updated_by,
			transfers.accepted_by,
			transfers.rejection_by,
			transfers.driver_rejection,
			transfers.created_at,
			transfers.updated_at,
			transfers.accepted_at,
			COALESCE(SUM(trd.received_count), 0)                  AS received_count,
			COALESCE(SUM(trd.expected_count), 0)                  AS expected_count,
			COALESCE(SUM(trd.scanned_count), 0)                   AS scanned_count,
			COALESCE(SUM(trd.scanned_count), 0)                   AS return_count,
			COALESCE(SUM(trd.accepted_count), 0)                  AS accepted_count,
			COALESCE(SUM(trd.rejection_count), 0)                 AS rejection_count,
			COALESCE(SUM(trd.received_count*trd.supply_price), 0) AS received_supply_sum,
			COALESCE(SUM(trd.received_count*trd.retail_price), 0) AS received_retail_sum,
			COALESCE(SUM(trd.expected_count*trd.supply_price), 0) AS expected_supply_sum,
			COALESCE(SUM(trd.expected_count*trd.retail_price), 0) AS expected_retail_sum,
			COALESCE(SUM(trd.scanned_count*trd.supply_price), 0)  AS scanned_supply_sum,
			COALESCE(SUM(trd.scanned_count*trd.retail_price), 0)  AS scanned_retail_sum,
			COALESCE(SUM(trd.accepted_count*trd.supply_price), 0) AS accepted_supply_sum,
			COALESCE(SUM(trd.accepted_count*trd.retail_price), 0) AS accepted_retail_sum
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

	if param.IsAuto != nil {
		query = query.Where("transfers.is_auto = ?", *param.IsAuto)
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
func (s *Services) GetReturnStats(ctx context.Context, params *domain.ReturnParam) (*domain.ReturnStatusSummary, error) {
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

	if params.StoreId != "" {
		filters = append(filters, "transfers.from_store_id = ?")
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		filters = append(filters, "st.company_id = ?")
		args = append(args, params.CompanyId)
	}
	if params.Search != "" {
		search := "%" + params.Search + "%"
		filters = append(filters, "(transfers.public_id ILIKE ? OR transfers.name ILIKE ?)")
		args = append(args, search, search)
	}
	if params.Status != "" {
		filters = append(filters, "transfers.status = ?")
		args = append(args, params.Status)
	}

	if len(filters) > 0 {
		query += " AND " + strings.Join(filters, " AND ")
	}

	var res domain.ReturnStatusSummary
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error; err != nil {
		s.log.Errorf("could not get return status summary: %v", err)
		return nil, domain.InternalServerError
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
			transfer_details.supply_price, 
			transfer_details.retail_price,
			transfer_details.received_count,
			transfer_details.created_at, 
			transfer_details.updated_at,
			transfer_details.expected_count,
			transfer_details.expected_pack,
			transfer_details.expected_unit,
			transfer_details.scanned_count,
			transfer_details.accepted_count,
			transfer_details.onec_count,
			transfer_details.rejection_count,
			transfer_details.rejection_pack,
			transfer_details.rejection_unit,
			FLOOR(transfer_details.received_count)::integer AS received_pack,
			ROUND((transfer_details.received_count - FLOOR(transfer_details.received_count)) * p.unit_per_pack)::integer AS received_unit,
			FLOOR(transfer_details.scanned_count)::integer AS scanned_pack,
			ROUND((transfer_details.scanned_count - FLOOR(transfer_details.scanned_count)) * p.unit_per_pack)::integer AS scanned_unit,
			FLOOR(transfer_details.accepted_count)::integer AS accepted_pack,
			ROUND((transfer_details.accepted_count - FLOOR(transfer_details.accepted_count)) * p.unit_per_pack)::integer AS accepted_unit,
			ROUND(transfer_details.received_count*transfer_details.retail_price, 2) AS received_sum,
			ROUND(transfer_details.scanned_count*transfer_details.retail_price, 2) AS scanned_sum,
    		p.name,
			p.material_code,
			p.unit_per_pack,
			sp.expire_date,
			COALESCE(sp.barcode, p.barcode) AS barcode,
			ut.short_name,
			pr.name AS producer`).
		Joins("JOIN products p ON transfer_details.product_id = p.id").
		Joins("JOIN store_products sp ON transfer_details.store_product_id = sp.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Joins("LEFT JOIN unit_types ut ON p.unit_type_id = ut.id").
		Where("transfer_details.transfer_id = ?", param.ReturnId)

	if param.Search != "" {
		switch utils.DefineProductSearchQuery(param.Search) {
		case "barcode":
			query = query.Where("COALESCE(sp.barcode, p.barcode) = ?", param.Search)
		case "name/category":
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ?", param.Search)
		default:
			param.Search = fmt.Sprintf("%%%s%%", param.Search)
			query = query.Where("p.name ILIKE ? OR COALESCE(sp.barcode, p.barcode) LIKE ?", param.Search, param.Search)
		}
	}
	// filter with return stats
	if param.Type != "" {
		switch param.Type {
		// case "shortage":
		// 	query = query.Where("transfer_details.received_count > transfer_details.scanned_count")
		case "expected":
			query = query.Where("transfer_details.expected_count > 0")
		case "not_expected":
			query = query.Where("transfer_details.expected_count = 0")
		case "scanned":
			query = query.Where("transfer_details.scanned_count > 0")
		case "not_scanned":
			query = query.Where("transfer_details.scanned_count = 0")
		case "accepted":
			query = query.Where("transfer_details.accepted_count > 0")
		case "not_accepted":
			query = query.Where("transfer_details.accepted_count = 0")
		case "rejection":
			query = query.Where("transfer_details.rejection_count > 0")
		// case "surplus":
		// 	query = query.Where("transfer_details.scanned_count > transfer_details.received_count")

		}
	}

	err := query.
		Order("transfer_details.created_at DESC, transfer_details.id").
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
func (s *Services) SendReturn(ctx context.Context, returnId string, userId string, DriverName string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var transfer domain.Transfer
	err := tx.WithContext(ctx).First(&transfer, "id = ?", returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer: %v", err)
		return domain.InternalServerError
	}

	if transfer.Status == constants.GeneralStatusSent {
		_ = tx.Rollback()
		return domain.AlreadySentError
	}

	// update return
	query := `UPDATE transfers SET status = ?, updated_by = ?, driver_office = ? WHERE id = ?`
	err = tx.WithContext(ctx).Exec(query, constants.GeneralStatusSent, userId, DriverName, returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update return %v", err)
		return domain.InternalServerError
	}

	var details []struct {
		Id             string  `gorm:"id" json:"id"`
		TransferId     string  `gorm:"transfer_id" json:"transfer_id"`
		StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
		ProductId      string  `gorm:"product_id" json:"product_id"`
		Name           string  `gorm:"name" json:"name"`
		UnitPerPack    float64 `gorm:"unit_per_pack" json:"unit_per_pack"`
		ReceivedCount  float64 `gorm:"received_count" json:"received_count"`
		ExpectedCount  float64 `gorm:"expected_count" json:"expected_count"`
		UnitQuantity   float64 `gorm:"unit_quantity" json:"unit_quantity"`
	}

	query3 := `
		SELECT 
			td.id,
			td.transfer_id,
			td.store_product_id,
			td.product_id,
			td.received_count,
			td.expected_count,
			p.name,
			p.unit_per_pack,
			sp.unit_quantity
        FROM transfer_details td
		JOIN products p ON td.product_id = p.id
		JOIN store_products sp ON td.store_product_id = sp.id
		WHERE td.transfer_id = ? and td.expected_count > 0;
	`
	err = tx.WithContext(ctx).Raw(query3, returnId).Scan(&details).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get return details: %v", err)
		return domain.InternalServerError
	}

	for _, detail := range details {
		if (detail.ExpectedCount * detail.UnitPerPack) > detail.UnitQuantity {
			_ = tx.Rollback()
			return domain.NewNotAdditionError(http.StatusConflict, map[string]any{
				"available_quantity": (detail.UnitQuantity / detail.UnitPerPack),
				"name":               detail.Name,
			})
		}
		// update store product quantities
		// if scanned count is 0, skip the update
		err = tx.WithContext(ctx).
			Exec(`UPDATE store_products SET unit_quantity = unit_quantity - ?, updated_at = NOW() WHERE id = ?`,
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

	go s.deleteReturnDetailByReturnId(returnId)

	return nil
}

func (s *Services) deleteReturnDetailByReturnId(returnId string) error {
	err := s.db.Exec("DELETE FROM transfer_details WHERE expected_count = 0 AND transfer_id = ?;", returnId).Error
	if err != nil {
		s.log.Errorf("could not delete scanned 0 return details: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// confirm return
func (s *Services) ReSendReturnToOnec(ctx context.Context, returnId string) error {
	var returnInfo domain.Transfer
	err := s.db.WithContext(ctx).Model(&domain.Transfer{}).Take(&returnInfo, "id = ?", returnId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NotFoundError
		}
		s.log.Errorf("could not get return data: %v", err)
		return domain.InternalServerError
	}

	// get store data
	var store domain.Store
	err = s.db.WithContext(ctx).Take(&store, "id = ?", returnInfo.FromStoreId).Error
	if err != nil {
		s.log.Errorf("could not get store info: %v", err)
		return domain.InternalServerError
	}

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
		WHERE 
			td.transfer_id = ? AND tr.status IN(?,?,?) AND tr.from_store_id = ?;
	`
	var returnData domain.ReturnData1C
	err = s.db.WithContext(ctx).
		Raw(query, returnId,
			constants.GeneralStatusSentOnec,
			constants.GeneralStatusCompleted,
			constants.GeneralStatusFailedSentOnec,
			returnInfo.FromStoreId).
		Scan(&returnData.Товары).Error
	if err != nil {
		s.log.Errorf("could not get transfer details fo send onec: %v", err)
		return domain.InternalServerError
	}

	returnData.Dok.DocumentNumber = "NP-" + cast.ToString(returnInfo.PublicId)
	returnData.Dok.DocumentDate = returnInfo.UpdatedAt.Add(time.Hour * 5).Format("2006-01-02T15:04:05")
	returnData.Apteka.Name = store.Name
	returnData.Apteka.StoreCode = store.StoreCode

	if len(returnData.Товары) < 1 {
		s.log.Warnf("No products found for resend return %s", returnId)
		return domain.NotEnoughProductError
	}

	// send return to Onec
	err = s.DoRequestOnec(context.Background(), returnData, "/vozvrat")
	if err != nil {
		s.log.Errorf("could not resend return request: %v", err)
		return domain.InternalServerError
	}

	if returnInfo.Status == constants.GeneralStatusFailedSentOnec {
		if upErr := s.db.WithContext(ctx).Exec(`UPDATE transfers SET status = ? WHERE id = ?`,
			constants.GeneralStatusCompleted, returnId).Error; upErr != nil {
			s.log.Errorf("ReSendReturnToOnec: could not reset status for %s: %v", returnId, upErr)
		}
	}

	return nil
}

func (s *Services) EditStatusToCheckingReturn(ctx context.Context, Id string, userId string, req *domain.EditStatusToCheckingRequest) error {
	var transfer struct {
		IsAuto bool `gorm:"is_auto"`
	}
	if err := s.db.WithContext(ctx).Raw("SELECT is_auto FROM transfers WHERE id = ?", Id).Scan(&transfer).Error; err != nil {
		s.log.Errorf("could not get transfer(%s): %v", Id, err)
		return domain.InternalServerError
	}

	if !transfer.IsAuto {
		var notScannedCount int64
		if err := s.db.WithContext(ctx).Raw(`
			SELECT COUNT(*) FROM transfer_details
			WHERE transfer_id = ? AND expected_count > 0 AND scanned_count = 0
		`, Id).Scan(&notScannedCount).Error; err != nil {
			s.log.Errorf("could not check scanned_count for return(%s): %v", Id, err)
			return domain.InternalServerError
		}
		if notScannedCount > 0 {
			return domain.ScannedCountZeroError
		}
	}

	err := s.db.WithContext(ctx).Exec(`
		UPDATE transfers
		SET status         = ?,
		    updated_by     = ?,
			driver_store_a = ?,
		    updated_at     = NOW()
		WHERE id = ?`,
		constants.GeneralStatusChecking, userId, req.DriverName,
		Id,
	).Error
	if err != nil {
		s.log.Errorf("could not update transfer(%s) status: %v", Id, err)
		return domain.InternalServerError
	}

	return nil
}

// confirm return
func (s *Services) ConfirmReturn(ctx context.Context, returnId, userId string, driverName string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var transfer domain.Transfer
	// get return info
	err := tx.WithContext(ctx).Take(&transfer, "id = ?", returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get return: %v", err)
		return domain.InternalServerError
	}
	// check if return is already confirmed
	if transfer.Status == constants.GeneralStatusCompleted || transfer.Status == constants.GeneralStatusSentOnec || transfer.Status == constants.GeneralStatusFailedSentOnec || transfer.Status == constants.GeneralStatusRejection {
		_ = tx.Rollback()
		return domain.AlreadyCompletedError
	}

	var returnData domain.ReturnData1C
	// get return data
	query2 := `
	SELECT
		td.id, 
		td.transfer_id, 
		p.material_code, 
		p.name,
		p.barcode,
		sp.expire_date,
		p.unit_per_pack,
		COALESCE(pr.code, '') as manufacturer, 
		td.serial_number AS product_series_number,
		td.accepted_count as quantity,
		td.supply_price AS supply_price_vat,
		td.retail_price AS retail_price_vat,
		(td.retail_price*td.accepted_count) AS sum_vat,
		td.expected_count,
		td.scanned_count,
		td.accepted_count,
		td.store_product_id
	FROM transfer_details td
		JOIN transfers tr ON td.transfer_id = tr.id
		JOIN products p ON td.product_id = p.id
		LEFT JOIN store_products sp ON sp.id = td.store_product_id
		LEFT JOIN producers pr ON p.producer_id = pr.id
		WHERE td.transfer_id = ?;
	`
	err = tx.WithContext(ctx).Raw(query2, returnId).Scan(&returnData.Товары).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get return data %v", err)
		return domain.InternalServerError
	}

	if len(returnData.Товары) < 1 {
		_ = tx.Rollback()
		s.log.Errorf("No products found for return %s", returnId)
		return domain.NotEnoughProductError
	}

	// check if any row has rejection (scanned_count != accepted_count)
	hasRejection := false
	for _, p := range returnData.Товары {
		if p.ScannedCount != p.AcceptedCount {
			hasRejection = true
			break
		}
	}

	newStatus := constants.GeneralStatusSentOnec
	if hasRejection {
		newStatus = constants.GeneralStatusRejection
	}

	// update confirm return
	query := `UPDATE transfers SET status = ?, accepted_by = ?, driver_store_b = ?, accepted_at = NOW() WHERE id = ? RETURNING *`
	err = tx.WithContext(ctx).Raw(query, newStatus, userId, driverName, returnId).Scan(&transfer).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update return: %v", err)
		return domain.InternalServerError
	}

	for _, p := range returnData.Товары {
		// add back only items not even scanned (expected - scanned);
		// rejected items (scanned - accepted) stay out of stock until ConfirmRejection
		addBack := (p.ExpectedCount - p.ScannedCount) * float64(p.UnitPerPack)
		if addBack > 0 {
			err = tx.WithContext(ctx).
				Exec(`UPDATE store_products SET unit_quantity = unit_quantity + ? WHERE id = ?`,
					addBack, p.StoreProductId).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("could not update store_product on return confirm: %v", err)
				return domain.InternalServerError
			}
		}

		// write rejection_count per detail row
		if p.ScannedCount != p.AcceptedCount {
			rejectionCount := math.Round((p.ScannedCount-p.AcceptedCount)*10000) / 10000
			err = tx.WithContext(ctx).Exec(`
				UPDATE transfer_details
				SET rejection_count = ?, updated_at = NOW()
				WHERE id = ?`, rejectionCount, p.Id).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("could not update transfer_detail rejection_count: %v", err)
				return domain.InternalServerError
			}
		}
	}
	var store domain.Store
	// get store data
	err = tx.WithContext(ctx).First(&store, "id = ?", transfer.FromStoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get store data: %v", err)
		return domain.InternalServerError
	}

	returnData.Dok.DocumentNumber = "NP-" + cast.ToString(transfer.PublicId)
	returnData.Dok.DocumentDate = transfer.UpdatedAt.Format(constants.DateTimeFormatRFC3339)
	returnData.Apteka.Name = store.Name
	returnData.Apteka.StoreCode = store.StoreCode

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	go func() {
		if s.cfg.OnecApiUrl == "test" {
			return
		}
		if err := s.DoRequestOnec(context.Background(), returnData, constants.OnecPathVozvrat); err != nil {
			s.log.Errorf("ConfirmReturn: 1C error for return %s: %v", returnId, err)
			if upErr := s.db.Exec(`UPDATE transfers SET status = ? WHERE id = ?`,
				constants.GeneralStatusFailedSentOnec, returnId).Error; upErr != nil {
				s.log.Errorf("ConfirmReturn: could not mark return %s as failed_sent_to_1c: %v", returnId, upErr)
			}
		}
	}()

	go s.updateStocksAfterVozvratFinished(returnId, transfer.FromStoreId)

	return nil
}

func (s *Services) CheckReturnAcceptedCount(ctx context.Context, transferId string) error {
	var nullCount int64
	err := s.db.WithContext(ctx).Table("transfer_details").
		Where("transfer_id = ? AND accepted_count IS NULL", transferId).
		Count(&nullCount).Error
	if err != nil {
		s.log.Error("failed to check accepted_count nulls:", err)
		return domain.InternalServerError
	}
	if nullCount > 0 {
		return domain.AcceptedCountError
	}

	return nil
}

func (s *Services) ConfirmRejection(ctx context.Context, returnId, userId string, req *domain.ConfirmRejectionRequest) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var transfer domain.Transfer
	if err := tx.WithContext(ctx).Take(&transfer, "id = ?", returnId).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get return(%s): %v", returnId, err)
		return domain.InternalServerError
	}
	if transfer.Status != constants.GeneralStatusRejection {
		_ = tx.Rollback()
		return domain.ConflictError
	}

	// fetch all rejected details from DB and add back stock
	var rejectedDetails []struct {
		StoreProductId string  `gorm:"column:store_product_id"`
		UnitPerPack    float64 `gorm:"column:unit_per_pack"`
		RejectionCount float64 `gorm:"column:rejection_count"`
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT td.store_product_id, p.unit_per_pack, td.rejection_count
		FROM transfer_details td
		JOIN products p ON td.product_id = p.id
		WHERE td.transfer_id = ? AND td.rejection_count > 0
	`, returnId).Scan(&rejectedDetails).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get rejection details for return(%s): %v", returnId, err)
		return domain.InternalServerError
	}

	for _, d := range rejectedDetails {
		addBack := d.RejectionCount * d.UnitPerPack
		if addBack > 0 {
			if err := tx.WithContext(ctx).Exec(`
				UPDATE store_products SET unit_quantity = unit_quantity + ? WHERE id = ?`,
				addBack, d.StoreProductId).Error; err != nil {
				_ = tx.Rollback()
				s.log.Errorf("could not restore rejection stock: %v", err)
				return domain.InternalServerError
			}
		}
	}

	// sum up rejection_count for transfers
	if err := tx.WithContext(ctx).Exec(`
		UPDATE transfers
		SET rejection_count  = (SELECT COALESCE(SUM(rejection_count), 0) FROM transfer_details WHERE transfer_id = ?),
		    driver_rejection  = ?,
		    rejection_by      = ?,
		    status            = ?,
		    updated_at        = NOW()
		WHERE id = ?`,
		returnId,
		req.DriverName,
		userId,
		constants.GeneralStatusCompleted,
		returnId,
	).Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not complete rejection for return(%s): %v", returnId, err)
		return domain.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit ConfirmRejection transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// canceled return
func (s *Services) CancelReturn(returnId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// update confirm return
	query := `UPDATE transfers SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ?`
	err := tx.Exec(query, constants.GeneralStatusCanceled, userId, returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not cancel return %v", err)
		return domain.InternalServerError
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) UpdateReturnComment(returnId string, comment string, userId string) error {
	err := s.db.Exec(`UPDATE transfers SET comment = ?, comment_by = ?, updated_at = NOW() WHERE id = ? AND entry_type = 2`, comment, userId, returnId).Error
	if err != nil {
		s.log.Errorf("could not update return comment: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// CreateAndSendReturnForOnec creates a return from 1C in one step.
// For each requested product (material_code + count), finds matching store_products
// FIFO by expire_date, inserts transfer_details, deducts stock, and returns unfulfilled products.
func (s *Services) CreateAndSendReturnForOnec(ctx context.Context, req *domain.OnecReturnRequest, userId string) (*domain.OnecReturnResponse, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var createdBy interface{}
	if userId != "" {
		createdBy = userId
	}

	var storeId string
	err := tx.WithContext(ctx).Raw(`SELECT id FROM stores WHERE store_code = ? LIMIT 1`, req.StoreCode).Scan(&storeId).Error
	if err != nil || storeId == "" {
		_ = tx.Rollback()
		s.log.Errorf("onec return: store_code=%d not found: %v", req.StoreCode, err)
		return nil, domain.NotFoundError
	}

	var activeInventoryCount int64
	err = s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM imports
		WHERE store_id = ?
		AND entry_type = 2
		AND status NOT IN ('completed', 'canceled')
	`, storeId).Scan(&activeInventoryCount).Error
	if err != nil {
		s.log.Errorf("could not check active inventory: %v", err)
		return nil, domain.InternalServerError
	}
	if activeInventoryCount > 0 {
		return nil, domain.ActiveInventoryError
	}

	var returnId string
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO transfers (from_store_id, name, created_by, entry_type, is_auto)
		VALUES (?, ?, ?, 2, TRUE)
		RETURNING id`,
		storeId, req.Name, createdBy,
	).Scan(&returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("onec return: create return: %v", err)
		return nil, domain.InternalServerError
	}

	type stockRow struct {
		StoreProductId string     `gorm:"column:store_product_id"`
		ProductId      string     `gorm:"column:product_id"`
		Available      float64    `gorm:"column:available"`
		SupplyPrice    float64    `gorm:"column:supply_price"`
		RetailPrice    float64    `gorm:"column:retail_price"`
		ExpireDate     *time.Time `gorm:"column:expire_date"`
		SerialNumber   string     `gorm:"column:serial_number"`
		UnitPerPack    float64    `gorm:"column:unit_per_pack"`
		Name           string     `gorm:"column:name"`
	}

	stockQuery := `
		SELECT
			sp.id AS store_product_id,
			sp.product_id,
			sp.unit_quantity::numeric / p.unit_per_pack AS available,
			sp.supply_price,
			sp.retail_price,
			sp.expire_date,
			sp.serial_number,
			p.unit_per_pack,
			p.name
		FROM store_products sp
		JOIN products p ON sp.product_id = p.id
		WHERE sp.store_id = ? AND %s AND sp.unit_quantity > 0
		ORDER BY sp.expire_date ASC NULLS LAST`

	sumAvailable := func(rs []stockRow) float64 {
		var total float64
		for _, r := range rs {
			total += r.Available
		}
		return total
	}

	enoughUnits := func(rs []stockRow, count float64) bool {
		if len(rs) == 0 {
			return false
		}
		return math.Round(sumAvailable(rs)*rs[0].UnitPerPack) >= math.Round(count*rs[0].UnitPerPack)
	}

	unfulfilled := make([]domain.OnecTransferUnfulfilled, 0)

	for _, product := range req.Products {
		var codeRows []stockRow
		err = tx.WithContext(ctx).Raw(fmt.Sprintf(stockQuery, "p.material_code = ?"),
			storeId, product.MaterialCode).Scan(&codeRows).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("onec return: fetch stock material_code=%d: %v", product.MaterialCode, err)
			return nil, domain.InternalServerError
		}

		rows := codeRows
		if !enoughUnits(codeRows, product.Count) && product.ProductName != "" {
			var nameRows []stockRow
			err = tx.WithContext(ctx).Raw(fmt.Sprintf(stockQuery, "p.name ILIKE ?"),
				storeId, product.ProductName).Scan(&nameRows).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("onec return: fetch stock by name=%s: %v", product.ProductName, err)
				return nil, domain.InternalServerError
			}
			rows = nameRows
		}

		productName := product.ProductName
		if productName == "" {
			if len(rows) > 0 {
				productName = rows[0].Name
			} else {
				productName = fmt.Sprintf("%d", product.MaterialCode)
			}
		}

		if !enoughUnits(rows, product.Count) {
			s.log.Warnf("onec return: not enough stock material_code=%d name=%s: available=%.4f requested=%.4f",
				product.MaterialCode, productName, sumAvailable(rows), product.Count)
			unfulfilled = append(unfulfilled, domain.OnecTransferUnfulfilled{
				MaterialCode: product.MaterialCode,
				Name:         productName,
				Requested:    product.Count,
				Accepted:     sumAvailable(rows),
				Remaining:    product.Count - sumAvailable(rows),
			})
			continue
		}

		remaining := product.Count
		for _, stock := range rows {
			if remaining <= 0 {
				break
			}
			toSet := remaining
			if toSet > stock.Available {
				toSet = stock.Available
			}

			rowPack := int(math.Floor(toSet))
			rowUnit := int(math.Round((toSet - math.Floor(toSet)) * stock.UnitPerPack))

			err = tx.WithContext(ctx).Exec(`
				INSERT INTO transfer_details (
					transfer_id, store_product_id, product_id,
					received_count, expected_count,
					expected_pack, expected_unit,
					supply_price, retail_price, expire_date, serial_number
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				returnId, stock.StoreProductId, stock.ProductId,
				stock.Available, toSet,
				rowPack, rowUnit,
				stock.SupplyPrice, stock.RetailPrice, stock.ExpireDate, stock.SerialNumber,
			).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("onec return: insert detail material_code=%d: %v", product.MaterialCode, err)
				return nil, domain.InternalServerError
			}

			err = tx.WithContext(ctx).Exec(`
				UPDATE store_products SET unit_quantity = unit_quantity - ?, updated_at = NOW() WHERE id = ?`,
				math.Round(toSet*stock.UnitPerPack), stock.StoreProductId).Error
			if err != nil {
				_ = tx.Rollback()
				s.log.Errorf("onec return: deduct store_product %s: %v", stock.StoreProductId, err)
				return nil, domain.InternalServerError
			}

			remaining -= toSet
		}

		if remaining > 0 {
			unfulfilled = append(unfulfilled, domain.OnecTransferUnfulfilled{
				MaterialCode: product.MaterialCode,
				Name:         productName,
				Requested:    product.Count,
				Accepted:     product.Count - remaining,
				Remaining:    remaining,
			})
		}
	}

	err = tx.WithContext(ctx).Exec(`
		UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`,
		constants.GeneralStatusSent, createdBy, returnId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("onec return: update status: %v", err)
		return nil, domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("onec return: commit: %v", err)
		return nil, domain.InternalServerError
	}

	return &domain.OnecReturnResponse{
		ReturnId:    returnId,
		Unfulfilled: unfulfilled,
	}, nil
}
