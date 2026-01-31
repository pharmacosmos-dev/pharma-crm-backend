package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// Create inventory creates a new inventory
func (s *Services) CreateTransfer(ctx context.Context, req *domain.TransferRequest) error {
	var id string
	// start transaction
	tx := s.db.Begin()

	// insert inventory into inventories table
	err := tx.WithContext(ctx).
		Raw(`
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
		_ = tx.Rollback()
		s.log.Errorf("could not create transfer: %v", err)
		return domain.InternalServerError
	}

	// if no products provided, get all products from store_products
	// and insert them into inventory_details
	err = tx.WithContext(ctx).
		Exec(`
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
			WHERE sp.store_id = ? AND sp.unit_quantity/p.unit_per_pack > 0;
		`, id, req.FromStoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not create transfer details: %v", err)
		return domain.InternalServerError
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could commit create transfer transaction: %v", err)
		return domain.InternalServerError
	}
	return nil
}

// get return by id
func (s *Services) GetTransferById(ctx context.Context, transferId string) (*domain.Transfer, error) {
	var tmpTransfer struct {
		Id                string     `gorm:"id"`
		PublicId          string     `gorm:"public_id"`
		FromStoreId       string     `gorm:"from_store_id"`
		ToStoreId         string     `gorm:"to_store_id"`
		Name              string     `gorm:"name"`
		Status            string     `gorm:"status"`
		ReceivedCount     float64    `gorm:"received_count"`
		ExpectedCount     float64    `gorm:"expected_count"`
		ScannedCount      float64    `gorm:"scanned_count"`
		AcceptedCount     float64    `gorm:"accepted_count"`
		ReceivedSupplySum float64    `gorm:"received_supply_sum"`
		ReceivedRetailSum float64    `gorm:"received_retail_sum"`
		AcceptedSupplySum float64    `gorm:"accepted_supply_sum"`
		AcceptedRetailSum float64    `gorm:"accepted_retail_sum"`
		FromStoreName     string     `gorm:"from_store_name"`
		FromStoreAddress  string     `gorm:"from_store_address"`
		FromStorePhone    string     `gorm:"from_store_phone"`
		ToStoreName       string     `gorm:"to_store_name"`
		ToStoreAddress    string     `gorm:"to_store_address"`
		ToStorePhone      string     `gorm:"to_store_phone"`
		CreatedBy         string     `gorm:"created_by"`
		UpdatedBy         string     `gorm:"updated_by"`
		AcceptedBy        string     `gorm:"accepted_by"`
		CreatedByName     string     `gorm:"created_by_name"`
		UpdatedByName     string     `gorm:"updated_by_name"`
		AcceptedByName    string     `gorm:"accepted_by_name"`
		CreatedAt         *time.Time `gorm:"created_at"`
		UpdatedAt         *time.Time `gorm:"updated_at"`
		AcceptedAt        *time.Time `gorm:"accepted_at"`
	}

	err := s.db.WithContext(ctx).
		Select(
			"t.id",
			"t.public_id",
			"t.name",
			"t.from_store_id",
			"t.to_store_id",
			"t.status",
			"t.created_at",
			"t.updated_at",
			"t.created_by",
			"t.updated_by",
			"t.accepted_by",
			"t.accepted_at",
			"SUM(trd.received_count) AS received_count",
			"SUM(trd.expected_count) AS expected_count",
			"SUM(trd.scanned_count) AS scanned_count",
			"SUM(trd.accepted_count) AS accepted_count",
			"SUM(trd.received_count*trd.supply_price) AS received_supply_sum",
			"SUM(trd.received_count*trd.retail_price) AS received_retail_sum",
			"SUM(trd.accepted_count*trd.supply_price) AS accepted_supply_sum",
			"SUM(trd.accepted_count*trd.retail_price) AS accepted_retail_sum",
			"fs.name AS from_store_name",
			"fs.address AS from_store_address",
			"fs.phone AS from_store_phone",
			"ts.name AS to_store_name",
			"ts.address AS to_store_address",
			"ts.phone AS to_store_phone",
			"e.full_name AS created_by_name",
			"e2.full_name AS updated_by_by_name",
			"e3.full_name AS accepted_by_name",
		).
		Table("transfers t").
		Joins("LEFT JOIN transfer_details trd ON t.id = trd.transfer_id").
		Joins("LEFT JOIN stores fs ON t.from_store_id = fs.id").
		Joins("LEFT JOIN stores ts ON t.to_store_id = ts.id").
		Joins("LEFT JOIN employees e ON t.created_by = e.id").
		Joins("LEFT JOIN employees e2 ON t.accepted_by = e2.id").
		Joins("LEFT JOIN employees e3 ON t.accepted_by = e3.id").
		Where("t.id = ?", transferId).
		Group("t.id, fs.id, ts.id, e.id, e2.id, e3.id").
		Take(&tmpTransfer).Error
	if err != nil {
		s.log.Errorf("could not gett transfer by id: %v", err)
		return nil, domain.InternalServerError
	}
	res := domain.Transfer{
		Id:                tmpTransfer.Id,
		PublicId:          tmpTransfer.PublicId,
		Name:              tmpTransfer.Name,
		FromStoreId:       tmpTransfer.FromStoreId,
		ToStoreId:         tmpTransfer.ToStoreId,
		Status:            tmpTransfer.Status,
		ReceivedCount:     tmpTransfer.ReceivedCount,
		ExpectedCount:     tmpTransfer.ExpectedCount,
		ScannedCount:      tmpTransfer.ScannedCount,
		AcceptedCount:     tmpTransfer.AcceptedCount,
		ReceivedSupplySum: tmpTransfer.ReceivedSupplySum,
		ReceivedRetailSum: tmpTransfer.ReceivedRetailSum,
		AcceptedSupplySum: tmpTransfer.AcceptedSupplySum,
		AcceptedRetailSum: tmpTransfer.AcceptedRetailSum,
		CreatedAt:         tmpTransfer.CreatedAt,
		UpdatedAt:         tmpTransfer.UpdatedAt,
		AcceptedAt:        tmpTransfer.AcceptedAt,
		CreatedById:       tmpTransfer.CreatedBy,
		UpdatedById:       tmpTransfer.UpdatedBy,
		AcceptedById:      tmpTransfer.AcceptedBy,
		CreatedBy: domain.NewNullStruct(domain.TransferEmployee{
			Id:       tmpTransfer.CreatedBy,
			FullName: tmpTransfer.CreatedByName,
		}, tmpTransfer.CreatedBy != ""),
		UpdatedBy: domain.NewNullStruct(domain.TransferEmployee{
			Id:       tmpTransfer.UpdatedBy,
			FullName: tmpTransfer.UpdatedByName,
		}, tmpTransfer.UpdatedBy != ""),
		AcceptedBy: domain.NewNullStruct(domain.TransferEmployee{
			Id:       tmpTransfer.AcceptedBy,
			FullName: tmpTransfer.AcceptedByName,
		}, tmpTransfer.AcceptedBy != ""),
		FromStore: domain.NewNullStruct(domain.TransferStore{
			Id:      tmpTransfer.FromStoreId,
			Name:    tmpTransfer.FromStoreName,
			Address: tmpTransfer.FromStoreAddress,
			Phone:   tmpTransfer.FromStorePhone,
		}, tmpTransfer.FromStoreId != ""),
		ToStore: domain.NewNullStruct(domain.TransferStore{
			Id:      tmpTransfer.ToStoreId,
			Name:    tmpTransfer.ToStoreName,
			Address: tmpTransfer.ToStoreAddress,
			Phone:   tmpTransfer.ToStorePhone,
		}, tmpTransfer.ToStoreId != ""),
	}

	return &res, nil
}

// get inventory list
func (s *Services) TransferList(ctx context.Context, params *domain.ReturnParam) ([]domain.Transfer, int64, error) {
	var tmpTransfer []struct {
		Id                string     `gorm:"id"`
		PublicId          string     `gorm:"public_id"`
		FromStoreId       string     `gorm:"from_store_id"`
		ToStoreId         string     `gorm:"to_store_id"`
		Name              string     `gorm:"name"`
		Status            string     `gorm:"status"`
		ReceivedCount     float64    `gorm:"received_count"`
		ExpectedCount     float64    `gorm:"expected_count"`
		ScannedCount      float64    `gorm:"scanned_count"`
		AcceptedCount     float64    `gorm:"accepted_count"`
		ReceivedSupplySum float64    `gorm:"received_supply_sum"`
		ReceivedRetailSum float64    `gorm:"received_retail_sum"`
		AcceptedSupplySum float64    `gorm:"accepted_supply_sum"`
		AcceptedRetailSum float64    `gorm:"accepted_retail_sum"`
		FromStoreName     string     `gorm:"from_store_name"`
		ToStoreName       string     `gorm:"to_store_name"`
		CreatedBy         string     `gorm:"created_by"`
		UpdatedBy         string     `gorm:"updated_by"`
		AcceptedBy        string     `gorm:"accepted_by"`
		CreatedByName     string     `gorm:"created_by_name"`
		UpdatedByName     string     `gorm:"updated_by_name"`
		AcceptedByName    string     `gorm:"accepted_by_name"`
		CreatedAt         *time.Time `gorm:"created_at"`
		UpdatedAt         *time.Time `gorm:"updated_at"`
		AcceptedAt        *time.Time `gorm:"accepted_at"`
	}

	query := s.db.WithContext(ctx).
		Select(
			"t.id",
			"t.public_id",
			"t.name",
			"t.from_store_id",
			"t.to_store_id",
			"t.status",
			"t.created_at",
			"t.updated_at",
			"t.created_by",
			"t.updated_by",
			"t.accepted_by",
			"t.accepted_at",
			"SUM(trd.received_count) AS received_count",
			"SUM(trd.expected_count) AS expected_count",
			"SUM(trd.scanned_count) AS scanned_count",
			"SUM(trd.accepted_count) AS accepted_count",
			"SUM(trd.received_count*trd.supply_price) AS received_supply_sum",
			"SUM(trd.received_count*trd.retail_price) AS received_retail_sum",
			"SUM(trd.accepted_count*trd.supply_price) AS accepted_supply_sum",
			"SUM(trd.accepted_count*trd.retail_price) AS accepted_retail_sum",
			"fs.name AS from_store_name",
			"ts.name AS to_store_name",
			"e.full_name AS created_by_name",
			"e2.full_name AS updated_by_name",
			"e3.full_name AS accepted_by_name",
		).
		Table("transfers t").
		Joins("LEFT JOIN transfer_details trd ON t.id = trd.transfer_id").
		Joins("LEFT JOIN stores fs ON t.from_store_id = fs.id").
		Joins("LEFT JOIN stores ts ON t.to_store_id = ts.id").
		Joins("LEFT JOIN employees e ON t.created_by = e.id").
		Joins("LEFT JOIN employees e2 ON t.updated_by = e2.id").
		Joins("LEFT JOIN employees e3 ON t.accepted_by = e3.id").
		Where("t.entry_type = ?", 1)

	// filter by from store id
	if params.StoreId != "" {
		query = query.Where("t.from_store_id = ? OR t.to_store_id = ?", params.StoreId, params.StoreId)
	}
	if params.CompanyId != "" {
		query = query.Where("fs.company_id = ? OR ts.company_id = ?", params.CompanyId, params.CompanyId)
	}

	// filter by search keyword
	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where("t.public_id LIKE ? OR t.name ILIKE ?", params.Search, params.Search)
	}

	if params.Status != "" {
		query = query.Where("t.status = ?", params.Status)
	}

	if params.StartDate != "" {
		query = query.Where("t.created_at >= ?", params.StartDate)
	}

	if params.EndDate != "" {
		query = query.Where("t.created_at <= ?", params.EndDate)
	}

	var totalCount int64
	// complete query
	err := query.
		Group("t.id, fs.id, ts.id, e.id, e2.id, e3.id").
		Order("t.created_at DESC").
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmpTransfer).Error
	if err != nil {
		s.log.Errorf("could not get transfers: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Transfer
	for _, item := range tmpTransfer {
		res = append(res, domain.Transfer{
			Id:                item.Id,
			PublicId:          item.PublicId,
			Name:              item.Name,
			FromStoreId:       item.FromStoreId,
			ToStoreId:         item.ToStoreId,
			Status:            item.Status,
			ReceivedCount:     item.ReceivedCount,
			ExpectedCount:     item.ExpectedCount,
			ScannedCount:      item.ScannedCount,
			AcceptedCount:     item.AcceptedCount,
			ReceivedSupplySum: item.ReceivedSupplySum,
			ReceivedRetailSum: item.ReceivedRetailSum,
			AcceptedSupplySum: item.AcceptedSupplySum,
			AcceptedRetailSum: item.AcceptedRetailSum,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
			AcceptedAt:        item.AcceptedAt,
			CreatedById:       item.CreatedBy,
			UpdatedById:       item.UpdatedBy,
			AcceptedById:      item.AcceptedBy,
			CreatedBy: domain.NewNullStruct(domain.TransferEmployee{
				Id:       item.CreatedBy,
				FullName: item.CreatedByName,
			}, item.CreatedBy != ""),
			UpdatedBy: domain.NewNullStruct(domain.TransferEmployee{
				Id:       item.UpdatedBy,
				FullName: item.UpdatedByName,
			}, item.UpdatedBy != ""),
			AcceptedBy: domain.NewNullStruct(domain.TransferEmployee{
				Id:       item.AcceptedBy,
				FullName: item.AcceptedByName,
			}, item.AcceptedBy != ""),
			FromStore: domain.NewNullStruct(domain.TransferStore{
				Id:   item.FromStoreId,
				Name: item.FromStoreName,
			}, item.FromStoreId != ""),
			ToStore: domain.NewNullStruct(domain.TransferStore{
				Id:   item.ToStoreId,
				Name: item.ToStoreName,
			}, item.ToStoreId != ""),
		})
	}

	return res, totalCount, nil
}

func (s *Services) TransferStats(ctx context.Context, params *domain.ReturnParam) (*domain.TransferStatusSummary, error) {

	query := `
		SELECT
			COALESCE(SUM(trd.received_count), 0) AS received_count,
			COALESCE(SUM(trd.accepted_count), 0) AS accepted_count,
			COALESCE(SUM(trd.received_count * trd.retail_price), 0) AS received_retail_sum,
			COALESCE(SUM(trd.accepted_count * trd.retail_price), 0) AS accepted_retail_sum
		FROM transfers t
		LEFT JOIN transfer_details trd ON t.id = trd.transfer_id
		LEFT JOIN stores fs ON t.from_store_id = fs.id
		LEFT JOIN stores ts ON t.to_store_id = ts.id
		WHERE t.entry_type = 1
	`

	var args []any

	if params.StoreId != "" {
		query += " AND (t.from_store_id = ? OR t.to_store_id = ?)"
		args = append(args, params.StoreId, params.StoreId)
	}
	if params.CompanyId != "" {
		query += " AND (fs.company_id = ? OR ts.company_id = ?)"
		args = append(args, params.CompanyId, params.CompanyId)
	}
	if params.Search != "" {
		search := fmt.Sprintf("%%%s%%", params.Search)
		query += " AND (t.public_id LIKE ? OR t.name ILIKE ?)"
		args = append(args, search, search)
	}
	if params.Status != "" {
		query += " AND t.status = ?"
		args = append(args, params.Status)
	}

	var res domain.TransferStatusSummary
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get transfer stats summary: %v", err)
		return nil, domain.InternalServerError
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
			transfer_details.expected_count,
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
func (s *Services) SendTransfer(ctx context.Context, transferId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var transfer domain.Transfer
	err := tx.WithContext(ctx).First(&transfer, "id = ?", transferId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer: %v", err)
		return domain.InternalServerError
	}

	if transfer.Status == constants.GeneralStatusSent {
		_ = tx.Rollback()
		return domain.AlreadySentError
	}

	// update confirm inventory
	query := `UPDATE transfers SET status = ?, updated_by = ? WHERE id = ?`
	err = tx.WithContext(ctx).Exec(query, constants.GeneralStatusSent, userId, transferId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update transfer: %v", err)
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
	err = tx.WithContext(ctx).Raw(query3, transferId).Scan(&details).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer details: %v", err)
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
				detail.ExpectedCount*float64(detail.UnitPerPack), detail.StoreProductId).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store product pack quantity: %v", err)
			return domain.InternalServerError
		}
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit send tranfer details transaction: %v", err)
		return domain.InternalServerError
	}

	go s.deleteNotInsertedTransferDetails(transferId)

	return nil
}

func (s *Services) EditStatusToCheckingTransfer(ctx context.Context, Id string, userId string) error {
	// update transfer status
	err := s.db.WithContext(ctx).Exec("UPDATE transfers SET status = ?, updated_by = ?, updated_at = NOW() WHERE id = ?", constants.GeneralStatusChecking, userId, Id).Error
	if err != nil {
		s.log.Errorf("could not update transfer(%s) status: %v", Id, err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) deleteNotInsertedTransferDetails(transferId string) {
	err := s.db.Exec("DELETE FROM transfer_details WHERE expected_count = 0 AND transfer_id = ?", transferId).Error
	if err != nil {
		s.log.Errorf("could not delete expected 0 transfer details: %v", err)
	}

}

func (s *Services) SendTransferToOnec(ctx context.Context, transferId string) error {
	var transfer domain.Transfer
	err := s.db.WithContext(ctx).First(&transfer, "id = ?", transferId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NotFoundError
		}
		s.log.Errorf("could not get transfer: %v", err)
		return domain.InternalServerError
	}

	var details []domain.TransferDetail
	err = s.db.WithContext(ctx).Raw(`
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
	`, transferId).Scan(&details).Error
	if err != nil {
		s.log.Errorf("could not get transfer_detail list: %v", err)
		return err
	}

	// get store info
	var toStore, fromStore domain.Store
	err = s.db.WithContext(ctx).First(&toStore, "id = ?", transfer.ToStoreId).Error
	if err != nil {
		s.log.Errorf("could not get toStore info: %v", err)
		return err
	}

	err = s.db.WithContext(ctx).First(&fromStore, "id = ?", transfer.FromStoreId).Error
	if err != nil {
		s.log.Errorf("could not get fromStore info: %v", err)
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
			Quantity:            v.AcceptedCount,
			RetailPrice:         v.RetailPrice,
			RetailPriceVat:      v.RetailPriceVat,
			SupplyPrice:         v.SupplyPrice,
			SupplyPriceVat:      v.SupplyPriceVat,
			Sum:                 v.AcceptedCount * v.RetailPrice,
			SumVat:              v.AcceptedCount * v.RetailPriceVat,
		})
	}

	data1C.Dok.DocumentDate = transfer.UpdatedAt.Add(5 * time.Hour).Format(time.RFC3339)
	data1C.Dok.DocumentNumber = "NP-" + cast.ToString(transfer.PublicId)

	data1C.Apteka.Name = toStore.Name
	data1C.Apteka.StoreCode = toStore.StoreCode
	data1C.AptekaOtkud.Name = fromStore.Name
	data1C.AptekaOtkud.StoreCode = fromStore.StoreCode
	err = s.DoRequestOnec(context.Background(), data1C, "/perekit")
	if err != nil {
		s.log.Errorf("could not send transfer to Onec: %v", err)
		return err
	}

	return nil
}

// check accepted_count is not null
func (s *Services) CheckAcceptedCount(ctx context.Context, transferId string) error {
	var count int64
	err := s.db.WithContext(ctx).Table("transfer_details").
		Where("transfer_id = ? AND accepted_count IS NULL", transferId).
		Count(&count).Error

	if err != nil {
		s.log.Error("failed to check accepted_count nulls:", err)
		return domain.InternalServerError
	}

	if count > 0 {
		return domain.AcceptedCountError
	}
	return nil
}

// confirm inventory
func (s *Services) ConfirmTransfer(ctx context.Context, transferId string, userId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// update confirm inventory
	var transfer domain.Transfer
	err := tx.WithContext(ctx).First(&transfer, "id = ?", transferId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer %v", err)
		return domain.InternalServerError
	}

	if transfer.Status == constants.GeneralStatusCompleted {
		_ = tx.Rollback()
		return domain.AlreadyCompletedError
	}

	query := `UPDATE transfers SET status = ?, accepted_by = ?, accepted_at = NOW() WHERE id = ? RETURNING *`
	err = tx.WithContext(ctx).Raw(query, constants.GeneralStatusCompleted, userId, transferId).Scan(&transfer).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update transfer %v", err)
		return domain.InternalServerError
	}

	var res []domain.TransferDetail
	err = tx.WithContext(ctx).Raw(`
	SELECT
		td.id,
		td.transfer_id,
		td.product_id,
		td.store_product_id,
		td.received_count,
		td.expected_count,
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
		WHERE td.transfer_id = ?;
	`, transferId).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get transfer_detail list: %v", err)
		return domain.InternalServerError
	}

	// insert transfered products to store_product
	query2 := `
		INSERT INTO store_products(
				product_id, 
				store_id,
				unit_quantity,
				retail_price,
				supply_price,
				vat,
				expire_date,
				vat_price,
				serial_number,
				import_detail_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var dataOnec domain.TransferData1C
	for _, item := range res {
		// return unscanned product to store
		err = tx.WithContext(ctx).Exec(`
		UPDATE store_products 
		SET unit_quantity = unit_quantity + ?
		WHERE id = ?;`,
			(item.ExpectedCount-item.AcceptedCount)*float64(item.UnitPerPack),
			item.StoreProductId).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not update store_product on return confirm: %v", err)
			return domain.InternalServerError
		}

		// execute query
		err = tx.WithContext(ctx).Exec(query2,
			item.ProductId,
			transfer.ToStoreId,
			(item.AcceptedCount * float64(item.UnitPerPack)),
			item.RetailPriceVat,
			item.SupplyPriceVat,
			12,
			item.ExpireDate,
			(item.RetailPriceVat*12)/112,
			item.SerialNumber,
			item.Id,
		).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not insert store product: %v", err)
			return domain.InternalServerError
		}

		// collect inventory products to send 1C
		dataOnec.Товары = append(dataOnec.Товары, domain.TransferProduct1C{
			MaterialCode:        item.MaterialCode,
			Name:                item.ProductName,
			Barcode:             item.Barcode,
			Manufacturer:        item.ProducerCode,
			ProductSeriesNumber: item.SerialNumber,
			ExpireDate:          item.ExpireDate,
			Quantity:            item.AcceptedCount,
			RetailPrice:         item.RetailPrice,
			RetailPriceVat:      item.RetailPriceVat,
			SupplyPrice:         item.SupplyPrice,
			SupplyPriceVat:      item.SupplyPriceVat,
			Sum:                 item.AcceptedCount * item.RetailPrice,
			SumVat:              item.AcceptedCount * item.RetailPriceVat,
		})
	}
	// get store info
	var toStore domain.Store
	err = tx.WithContext(ctx).First(&toStore, "id = ?", transfer.ToStoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get store info: %v", err)
		return domain.InternalServerError
	}

	// get store info
	var fromStore domain.Store
	err = tx.WithContext(ctx).First(&fromStore, "id = ?", transfer.FromStoreId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not get store info: %v", err)
		return domain.InternalServerError
	}

	// complete transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	// get document data and number
	dataOnec.Dok.DocumentDate = transfer.UpdatedAt.Format(time.RFC3339)
	dataOnec.Dok.DocumentNumber = "NP-" + cast.ToString(transfer.PublicId)

	// get store info
	dataOnec.Apteka.Name = toStore.Name
	dataOnec.Apteka.StoreCode = toStore.StoreCode
	dataOnec.AptekaOtkud.Name = fromStore.Name
	dataOnec.AptekaOtkud.StoreCode = fromStore.StoreCode
	if s.cfg.OnecApiUrl != "test" {
		// send inventory products data to 1C
		go s.DoRequestOnec(context.Background(), dataOnec, constants.OnecPathPerekit)
		// if err != nil {
		// 	s.log.Errorf("could not send transfer to onec: %v", err)
		// }
	}
	return nil
}

// canceled inventory
func (s *Services) CancelTransfer(ctx context.Context, returnId string, userId string) error {
	query := `UPDATE transfers SET status = ?, accepted_by = ?, updated_at = NOW() WHERE id = ? AND status = ?`
	err := s.db.WithContext(ctx).Exec(query,
		constants.GeneralStatusCanceled,
		userId,
		returnId,
		constants.GeneralStatusNew,
	).Error
	if err != nil {
		s.log.Errorf("could not update inventory %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) UpdateTransferByBarcode(
	ctx context.Context,
	req *domain.TransferBarcodeRequest,
	user *domain.EmployeeClaims,
	transferType int,
) error {
	// default count is 1
	if req.Count == 0 {
		req.Count = 1
	}

	transferLog := domain.TransferLog{
		TransferId:   req.TransferId,
		UserId:       user.UserId,
		TransferType: transferType,
	}

	transferLog.Stage = constants.TransferLogStageSent
	// get default update field
	updatedField := "scanned_count"
	if req.Status == "checking" {
		transferLog.Stage = constants.TransferLogStageChecking
		updatedField = "accepted_count"
	}

	if req.Id != "" {
		var transferDetailId string
		err := s.db.WithContext(ctx).
			Raw(fmt.Sprintf(`
				UPDATE transfer_details 
				SET %s = COALESCE(%s, 0) + ? 
				WHERE id = ? AND 
				expected_count >= COALESCE(%s,0) + ?
				RETURNING id;`,
				updatedField,
				updatedField,
				updatedField),
				req.Count,
				req.Id,
				req.Count).
			Scan(&transferDetailId).Error
		if err != nil {
			s.log.Errorf("could not update transfer_details(%s) scanned_count: %v", req.Id, err)
			return domain.InternalServerError
		}

		transferLog.TransferDetailId = transferDetailId
		transferLog.Quantity = req.Count

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
				req.Barcode, req.TransferId).
			Scan(&barcodeResponse).Error
		if err != nil {
			s.log.Errorf("could not get transfer_details by barcode(%s): %v", req.Barcode, err)
			return domain.InternalServerError
		}

		if len(barcodeResponse) > 1 {
			return domain.NewNotAdditionError(http.StatusMultiStatus, barcodeResponse)
		}
		if len(barcodeResponse) > 0 {
			transferLog.TransferDetailId = barcodeResponse[0].Id
			transferLog.ProductId = barcodeResponse[0].ProductId
		}
		transferLog.Quantity = req.Count

		err = s.db.WithContext(ctx).
			Exec(fmt.Sprintf(`
		UPDATE transfer_details t
		SET %s = COALESCE(%s, 0) + ?
		FROM products p 
		WHERE
			t.transfer_id = ? AND 
			p.id = t.product_id AND 
			p.barcode = ? AND 
			t.expected_count >= COALESCE(t.%s,0) + ?;`,
				updatedField,
				updatedField,
				updatedField),
				req.Count,
				req.TransferId,
				req.Barcode,
				req.Count,
			).Error
		if err != nil {
			s.log.Error("could not update transfer_details by barcode(%s): %v", req.Barcode, err)
			return domain.InternalServerError
		}

	} else {
		return domain.InvalidRequestBodyError
	}

	go s.SaveTransferLog(&transferLog)

	return nil
}

func (s *Services) DeleteTransfer(transferId string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// update confirm inventory
	err := tx.Exec("DELETE FROM transfers WHERE id = ?", transferId).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("ERROR on deleting transfer %v", err)
		return err
	}
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) SaveTransferLog(req *domain.TransferLog) {
	err := s.db.Create(&req).Error
	if err != nil {
		s.log.Errorf("could not save transfer log: %v", err)
	}
}

func (s *Services) GetTransferLogs(ctx context.Context, params *domain.ReturnDetailParam) ([]domain.TransferLog, int64, error) {
	var tmpTransferLog []struct {
		Id               int64      `gorm:"id"`
		TransferId       string     `gorm:"transfer_id"`
		TransferDetailId string     `gorm:"transfer_detail_id"`
		ProductId        string     `gorm:"product_id"`
		UserId           string     `gorm:"user_id"`
		TransferType     int        `gorm:"transfer_type"`
		Stage            int        `gorm:"stage"`
		Quantity         int        `gorm:"quantity"`
		CreatedAt        *time.Time `gorm:"created_at"`
		UpdatedAt        *time.Time `gorm:"updated_at"`
		FullName         string     `gorm:"full_name"`
	}

	var res []domain.TransferLog
	var totalCount int64
	err := s.db.WithContext(ctx).
		Model(&domain.TransferLog{}).
		Select("transfer_logs.*, em.full_name").
		Where("transfer_logs.transfer_id = ?", params.TransferId).
		Joins("JOIN employees em ON transfer_logs.user_id = em.id").
		Order("transfer_logs.created_at DESC").
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmpTransferLog).Error
	if err != nil {
		s.log.Errorf("could not get transfer logs: %v", err)
		return nil, 0, err
	}

	for _, log := range tmpTransferLog {
		res = append(res, domain.TransferLog{
			Id:               log.Id,
			TransferId:       log.TransferId,
			TransferDetailId: log.TransferDetailId,
			ProductId:        log.ProductId,
			UserId:           log.UserId,
			TransferType:     log.TransferType,
			Stage:            log.Stage,
			Quantity:         log.Quantity,
			CreatedAt:        log.CreatedAt,
			UpdatedAt:        log.UpdatedAt,
			Employee: domain.NewNullStruct(domain.EmployeeTransferLog{
				Id:       log.UserId,
				FullName: log.FullName,
			}, log.UserId != ""),
		})
	}

	return res, totalCount, nil
}
