package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// Update value by chosen field
func (s *Services) UpdateImportByField(tx *gorm.DB, id string, field, value string) (*domain.Import, error) {
	var res domain.Import
	// build query
	query := fmt.Sprintf("UPDATE imports SET %s = ? WHERE id = ? RETURNING *", field)
	err := tx.Raw(query, value, id).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &res, nil
}

func (s *Services) UpdateImportCompletedStatus(tx *gorm.DB, importID, userID string) error {
	var res domain.Import
	// update import status
	err := tx.Raw(`UPDATE imports SET status = ?, accepted_by = ? WHERE id = ? RETURNING *`, constants.GeneralStatusCompleted, userID, importID).Scan(&res).Error
	if err != nil {
		s.log.Error("could not update import(%s) status: %v", importID, err)
		return err
	}

	return nil
}

// Accept import
func (s *Services) AcceptImport(ctx context.Context, importId string, userId string, acceptType string) error {
	// begin transactions
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var res domain.Import
	err := tx.WithContext(ctx).First(&res, "id = ?", importId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NotFoundError
		}
		s.log.Errorf("could not get import: %v", err)
		return domain.InternalServerError
	}

	if res.Status == constants.GeneralStatusCompleted {
		return domain.AlreadyCompletedError
	}

	// update import status
	err = tx.WithContext(ctx).
		Raw("UPDATE imports SET status = ?, accepted_by = ? WHERE id = ? AND status != ? RETURNING *",
			constants.GeneralStatusCompleted,
			userId,
			importId,
			constants.GeneralStatusCompleted,
		).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not update import(%s) status: %v", importId, err)
		return err
	}

	if res.Id == "" {
		_ = tx.Rollback()
		return domain.AlreadyCompletedError
	}

	if acceptType == "all" {
		// update accepted_count and scanned_count to received_count
		err = s.UpdateImportDetailsAccepted(tx, importId)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		// add all imported products to store_products and send 1C
		err = s.AddAllProductsToStore(ctx, tx, &res)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	} else {
		err = s.UpdateImportDetailsScanToAccept(tx, importId)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		err = s.AddSomeImportedProductsToStore(ctx, tx, &res)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	// completed transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not completed confirm import transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// Canceled import
func (s *Services) CancelImport(ctx context.Context, id string, userID string) error {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.WithContext(ctx).
		Exec(`UPDATE imports SET status = ?, accepted_by = ? WHERE id = ?;`,
			constants.GeneralStatusCanceled,
			userID,
			id,
		).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("could not cancel import %v", err)
		return domain.InternalServerError
	}

	err = s.UpdateImportDetailsToCancel(tx, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// completed transaction
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit cancel import tranaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// Add some imported products to stores
func (s *Services) AddSomeImportedProductsToStore(ctx context.Context, tx *gorm.DB, importData *domain.Import) error {
	var reqFakt domain.AcceptImport1C

	// import_detail list by import_id
	importDetails, err := s.GetImportDetailsByImportId(ctx, tx, importData.Id)
	if err != nil {
		return err
	}

	// get store info by using import_id
	store, err := s.GetStoreByImportId(ctx, tx, importData.Id)
	if err != nil {
		return err
	}

	// collect send fakt data
	reqFakt.Dok.DocumentNumber = importData.DocumentNumber
	reqFakt.Dok.DocumentDate = importData.ImportDate.Format(constants.DateTimeFormatRFC3339)
	reqFakt.Apteka.StoreCode = store.StoreCode
	reqFakt.Apteka.Name = store.Name

	// add products to store
	storeProductQuery := `
	INSERT INTO store_products(
		store_id,
		product_id,
		pack_quantity,
		unit_quantity,
		supply_price,
		retail_price,
		vat,
		expire_date,
		vat_price,
		import_detail_id,
		serial_number,
		mxik,
		unit_code,
		unit_label,
		is_marking,
	    company_id
		)
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range importDetails {
		if item.ScannedCount > 0 {
			err = tx.WithContext(ctx).
				Exec(storeProductQuery,
					importData.StoreId,
					item.ProductID,
					int(item.ScannedCount),
					int(item.ScannedCount*float64(item.UnitPerPack)),
					item.SupplyPriceVat,
					item.RetailPriceVat,
					item.Vat, item.ExpireDate,
					item.RetailPriceVat*12/112,
					item.Id,
					item.SeriesNumber,
					item.Mxik,
					item.UnitCode,
					item.UnitLabel,
					item.IsMarking,
					store.CompanyId,
				).Error
			if err != nil {
				s.log.Errorf("could not inserting import products to store_product: %v", err)
				return domain.InternalServerError
			}
			// collect fakt data
			reqFakt.Товары = append(reqFakt.Товары, domain.AcceptImport1CResponse{
				MaterialCode:        item.MaterialCode,
				Name:                item.ProductName,
				Barcode:             item.Barcode,
				Manufacturer:        item.ProducerName,
				ProductSeriesNumber: item.SeriesNumber,
				Quantity:            item.ReceivedCount,
				QuantityFakt:        item.AcceptedCount,
			})
		}
	}
	if len(reqFakt.Товары) == 0 {
		return errors.New("accepted products are not available")
	}

	// send fakt to 1C
	go s.DoRequestOnec(context.Background(), reqFakt, constants.OnecPathPrihod)

	go s.updateImportTotalsAfterConfirm(importData.Id)

	return nil
}

// add all imported products to store
func (s *Services) AddAllProductsToStore(ctx context.Context, tx *gorm.DB, importData *domain.Import) error {
	var (
		reqFakt domain.AcceptImport1C
		store   *domain.Store
	)

	// get import_detail list by import_id
	details, err := s.GetImportDetailsByImportId(ctx, tx, importData.Id)
	if err != nil {
		return err
	}
	// get store info by import_id
	store, err = s.GetStoreByImportId(ctx, tx, importData.Id)
	if err != nil {
		return err
	}

	// collect send fakt data
	reqFakt.Dok.DocumentNumber = importData.DocumentNumber
	reqFakt.Dok.DocumentDate = importData.ImportDate.Format(constants.DateTimeFormatRFC3339)
	reqFakt.Apteka.StoreCode = store.StoreCode
	reqFakt.Apteka.Name = store.Name

	// add products to store
	storeProductQuery := `
	INSERT INTO store_products(
		store_id,
		product_id,
		pack_quantity,
		unit_quantity,
		supply_price,
		retail_price,
		vat,
		expire_date,
		vat_price,
		import_detail_id,
		serial_number,
		mxik,
		unit_code,
		unit_label,
		is_marking,
	    company_id
		)
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range details {
		if item.ReceivedCount > 0 {
			err = tx.WithContext(ctx).Exec(storeProductQuery,
				importData.StoreId,
				item.ProductID,
				int(item.ReceivedCount),
				int(item.ReceivedCount*float64(item.UnitPerPack)),
				item.SupplyPriceVat,
				item.RetailPriceVat,
				item.Vat,
				item.ExpireDate,
				item.RetailPriceVat*12/112,
				item.Id,
				item.SeriesNumber,
				item.Mxik,
				item.UnitCode,
				item.UnitLabel,
				item.IsMarking,
				store.CompanyId,
			).Error
			if err != nil {
				s.log.Errorf("could not add import products to store_product: %v", err)
				return domain.InternalServerError
			}

			// collect product fakt data
			reqFakt.Товары = append(reqFakt.Товары, domain.AcceptImport1CResponse{
				MaterialCode:        item.MaterialCode,
				Name:                item.ProductName,
				Barcode:             item.Barcode,
				Manufacturer:        item.ProducerName,
				ProductSeriesNumber: item.SeriesNumber,
				Quantity:            item.ReceivedCount,
				QuantityFakt:        item.AcceptedCount,
			})
		}
	}

	// send fakt to 1C
	go s.DoRequestOnec(context.Background(), reqFakt, constants.OnecPathPrihod)
	go s.updateImportTotalsAfterConfirm(importData.Id)
	return nil
}

func (s *Services) updateImportTotalsAfterConfirm(importId string) {
	// update import totals
	query := `
	UPDATE imports
            SET 
                scanned_count = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0))
                    FROM import_details d
                    WHERE d.import_id = ?
                ), 0),
                scanned_sum = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0) * d.retail_price_vat)
                    FROM import_details d
                    WHERE d.import_id = ?
                ), 0),
                updated_at = NOW()
            WHERE id = ?;`
	err := s.db.Exec(query, importId, importId, importId).Error
	if err != nil {
		s.log.Error("could not update import totals after confirm: %v", err)
		return
	}
}

// update import details to cancel
func (s *Services) UpdateImportDetailsToCancel(tx *gorm.DB, importID string) error {
	err := tx.Exec(`UPDATE import_details SET canceled_count = received_count WHERE import_id = ?`, importID).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}

func (s *Services) UpdateImportDetailsAccepted(tx *gorm.DB, importID string) error {
	err := tx.Exec("UPDATE import_details SET accepted_count = received_count, scanned_count = received_count, updated_at = NOW() WHERE import_id = ?", importID).Error
	if err != nil {
		s.log.Error("could not update import details counts: %v", err)
		return err
	}
	return nil
}

func (s *Services) UpdateImportDetailsScanToAccept(tx *gorm.DB, importId string) error {
	err := tx.Exec("UPDATE import_details SET accepted_count = scanned_count, updated_at = NOW() WHERE import_id = ?", importId).Error
	if err != nil {
		s.log.Error("could not update import details counts: %v", err)
		return err
	}
	return nil
}

// create import details
func (s *Services) CreateImportDetail(tx *gorm.DB, req *domain.ImportDetailRequest) (string, error) {
	var (
		id    string
		query = `INSERT INTO import_details(
			import_id, product_id, received_count, scanned_count, accepted_count, supply_price, supply_price_vat, retail_price, retail_price_vat, expire_date, vat, vat_sum, series_number, sum_vat, marking)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	)
	req.Marking = utils.StringArray(req.Marking)
	// complete query
	err := tx.Raw(query, req.ImportID, req.ProductID,
		req.ReceivedCount, req.ReceivedCount, req.AcceptedCount,
		req.SupplyPrice, req.SupplyPriceVat, req.RetailPrice,
		req.RetailPriceVat, req.ExpireDate, req.Vat,
		req.VatSum, req.SeriesNumber, req.SumVat, req.Marking).Scan(&id).Error
	if err != nil {
		s.log.Error(err)
		return "", err
	}
	return id, nil
}

// create product marking
func (s *Services) CreateProductMarking(tx *gorm.DB, req domain.ProductMarkingReq) error {
	for _, item := range req.Marking {
		err := tx.Exec(`INSERT INTO product_markings(import_detail_id, product_id, marking) VALUES(?, ?, ?)`, req.ImportDetailId, req.ProductID, item).Error
		if err != nil {
			s.log.Error(err)
			return err
		}
	}
	return nil
}

// list import
func (s *Services) GetImports(ctx context.Context, params *domain.ImportQueryParams) ([]domain.Import, int64, error) {
	var tmpImport []struct {
		Id                string     `gorm:"id"`
		PublicId          int        `gorm:"public_id"`
		StoreId           string     `gorm:"store_id"`
		CreatedBy         string     `gorm:"created_by"`
		AcceptedBy        string     `gorm:"accepted_by"`
		DocumentNumber    string     `gorm:"document_number"`
		DocumentYear      int        `gorm:"document_year"`
		Status            string     `gorm:"status"`
		ImportDate        *time.Time `gorm:"import_date"`
		AcceptedAmount    float64    `gorm:"accepted_amount"`
		ReceivedAmount    float64    `gorm:"received_amount"`
		ReceivedCount     float64    `gorm:"received_count"`
		AcceptedCount     float64    `gorm:"accepted_count"`
		AcceptedAmountVat float64    `gorm:"accepted_amount_vat"`
		ReceivedAmountVat float64    `gorm:"received_amount_vat"`
		CreatedAt         *time.Time `gorm:"created_at"`
		UpdatedAt         *time.Time `gorm:"updated_at"`
		StoreName         string     `gorm:"store_name"`
		CreatedByName     string     `gorm:"created_by_name"`
		AcceptedByName    string     `gorm:"accepted_by_name"`
	}

	// Fetch imports with detailed data
	qb := s.db.
		WithContext(ctx).
		Table("imports im").
		Joins("JOIN stores st ON st.id = im.store_id").
		Joins("LEFT JOIN employees em ON im.created_by = em.id").
		Joins("LEFT JOIN employees em2 ON im.accepted_by = em2.id").
		Where("im.entry_type = ?", 1)

	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.document_number ILIKE ? OR im.public_id::text LIKE ?", params.Search, params.Search)
	}
	if params.StoreId != "" {
		qb = qb.Where("im.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}
	if params.StartDate != "" {
		qb = qb.Where("(im.created_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(im.created_at + interval '5 hours') <= ?", params.EndDate)
	}
	if params.Status != "" {
		qb = qb.Where("im.status = ?", params.Status)
	}
	if params.ReceivedAmountFrom > 0 {
		qb = qb.Where("im.received_sum >= ?", params.ReceivedAmountFrom)
	}
	if params.ReceivedAmountTo > 0 {
		qb = qb.Where("im.received_sum <= ?", params.ReceivedAmountTo)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get imports total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	err := qb.
		Select(
			"im.id",
			"im.public_id",
			"im.store_id",
			"im.name",
			"im.document_number",
			"im.document_year",
			"im.status",
			"im.import_date",
			"im.received_count AS received_count",
			"im.received_sum AS received_amount_vat",
			"im.scanned_count AS accepted_count",
			"im.scanned_sum AS accepted_amount_vat",
			"im.created_by",
			"im.accepted_by",
			"im.created_at",
			"im.updated_at",

			"st.name AS store_name",
			"em.full_name AS created_by_name",
			"em2.full_name AS accepted_by_name",
		).
		Order("im.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmpImport).Error
	if err != nil {
		s.log.Errorf("could not get imports: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res = []domain.Import{}
	for _, item := range tmpImport {
		res = append(res, domain.Import{
			Id:                item.Id,
			PublicId:          item.PublicId,
			StoreId:           item.StoreId,
			DocumentNumber:    item.DocumentNumber,
			DocumentYear:      item.DocumentYear,
			Status:            item.Status,
			ReceivedCount:     item.ReceivedCount,
			ReceivedAmountVat: item.ReceivedAmountVat,
			AcceptedCount:     item.AcceptedCount,
			AcceptedAmountVat: item.AcceptedAmountVat,
			CreatedBy:         item.CreatedBy,
			AcceptedBy:        item.AcceptedBy,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
			ImportDate:        item.ImportDate,
			Store: domain.NewNullStruct(domain.ImportStore{
				Id:   item.StoreId,
				Name: item.StoreName,
			}, item.StoreId != ""),
			Sender: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.CreatedBy,
				FullName: item.CreatedByName,
			}, item.CreatedBy != ""),
			Receiver: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.AcceptedBy,
				FullName: item.AcceptedByName,
			}, item.AcceptedBy != ""),
		})
	}

	return res, totalCount, nil
}

func (s *Services) GetImportsStats(ctx context.Context, params *domain.ImportQueryParams) (*domain.ImportStatusSummary, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"SUM(im.received_count) AS new_received_count",
			"SUM(im.received_sum) AS new_accepted_vat_amount",
			"SUM(im.scanned_count) AS completed_accepted_count",
			"SUM(im.scanned_sum) AS completed_received_vat_amount",
		).
		Table("imports im").
		Joins("JOIN stores st ON im.store_id = st.id")

	if params.StoreId != "" {
		qb = qb.Where("im.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		qb = qb.Where("st.company_id = ?", params.CompanyId)
	}

	if params.StartDate != "" {
		qb = qb.Where("(im.created_at + interval '5 hours') >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		qb = qb.Where("(im.created_at + interval '5 hours') <= ?", params.EndDate)
	}
	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.document_number ILIKE ? OR im.public_id::text LIKE ?", params.Search, params.Search)
	}
	if params.Status != "" {
		qb = qb.Where("im.status = ?", params.Status)
	}
	if params.ReceivedAmountFrom > 0 {
		qb = qb.Where("im.received_sum >= ?", params.ReceivedAmountFrom)
	}
	if params.ReceivedAmountTo > 0 {
		qb = qb.Where("im.received_sum <= ?", params.ReceivedAmountTo)
	}

	var res domain.ImportStatusSummary
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get imports stats: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

// list import detail
func (s *Services) ListImportDetail(param *domain.ImportQueryParams) ([]domain.ImportDetail, int64, error) {
	var (
		importDetails []domain.ImportDetail
		totalCount    int64
	)

	// Fetch import details with detailed data
	query := s.db.Model(&domain.ImportDetail{}).
		Preload("Product").
		Preload("Import").
		Select(`
		import_details.*,
		ROUND((import_details.retail_price*received_count)::numeric, 2) as received_amount,
		ROUND((import_details.retail_price*accepted_count)::numeric, 2) as accepted_amount,
		ROUND(sum_vat, 2) as received_amount_vat,
		ROUND((import_details.retail_price_vat*accepted_count)::numeric, 2) as accepted_amount_vat,
		COALESCE(unit_types.short_name, '') as unit_name
		`).
		Joins("LEFT JOIN products ON import_details.product_id = products.id").
		Joins("LEFT JOIN unit_types ON products.unit_type_id = unit_types.id").
		Where("import_id = ?", param.ImportId)

	if param.Search != "" {
		param.Search = fmt.Sprintf("%%%s%%", param.Search)
		query = query.Where(`(
		products.barcode LIKE ? OR
		products.name ILIKE ? OR
		CAST(products.material_code AS TEXT) LIKE ?)`, param.Search, param.Search, param.Search)
	}
	if param.ReceivedAmountFrom > 0 {
		query = query.Where("import_details.received_amount >= ?", param.ReceivedAmountFrom)
	}
	if param.ReceivedAmountTo > 0 {
		query = query.Where("import_details.received_amount <= ?", param.ReceivedAmountTo)
	}

	if param.NoBarcode {
		query = query.Where("products.barcode IS NULL OR products.barcode = ''")
	}

	if param.NoMarking {
		query = query.Where("array_length(import_details.marking, 1) IS NULL OR array_length(import_details.marking, 1) = 0")
	}
	err := query.Debug().
		Count(&totalCount).
		Limit(param.Limit).
		Offset(param.Offset).
		Order("products.name").
		Find(&importDetails).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	fmt.Println(totalCount, "and", len(importDetails))
	return importDetails, totalCount, nil
}

// get import details by import id
func (s *Services) GetImportDetailsByImportId(ctx context.Context, tx *gorm.DB, importId string) ([]domain.ImportDetail, error) {
	var importDetails []domain.ImportDetail
	err := tx.WithContext(ctx).
		Raw(`
		SELECT
			import_details.*,
			products.unit_per_pack,
			products.barcode,
			products.name as product_name,
			products.material_code,
			COALESCE(pr.name, '') as producer_name,
			products.mxik,
			products.unit_code,
			products.unit_label,
			products.is_marking
		FROM import_details
		JOIN products ON products.id = import_details.product_id
		LEFT JOIN producers pr ON pr.id = products.producer_id
		 WHERE import_id = ?`,
			importId).Scan(&importDetails).Error
	if err != nil {
		s.log.Errorf("could not get import details by import id: %v", err)
		return nil, domain.InternalServerError
	}

	return importDetails, nil
}

// get import detail list order by updated_at
func (s *Services) ListImportDetailByLastUpdated(c *gin.Context, limit, offset int) ([]domain.ImportDetail, int64, error) {
	var (
		importDetails      []domain.ImportDetail
		totalCount         int64
		importId           = c.Query("import_id")
		search             = c.Query("search")
		receivedAmountFrom = c.Query("received_amount_from")
		receivedAmountTo   = c.Query("received_amount_to")
		valueType          = c.Query("type")
	)

	// Fetch import details with detailed data
	query := s.db.Model(&domain.ImportDetail{}).
		Preload("Product").
		Preload("Import").
		Select(`
		import_details.*,
		(import_details.retail_price*received_count) as received_amount,
		(import_details.retail_price*accepted_count) as accepted_amount,
		sum_vat as received_amount_vat,
		(import_details.retail_price_vat*accepted_count) as accepted_amount_vat,
		COALESCE(unit_types.short_name, '') as unit_name`).
		Joins("LEFT JOIN products ON import_details.product_id = products.id").
		Joins("LEFT JOIN unit_types ON products.unit_type_id = unit_types.id").
		Where("import_id = ?", importId)
	// get search
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		products.barcode LIKE ? OR
		products.name ILIKE ? OR
		CAST(products.material_code AS TEXT) LIKE ?`, search, search, search)
	}
	// get received amount
	if receivedAmountFrom != "" {
		query = query.Where("import_details.received_amount >= ?", receivedAmountFrom)
	}
	// get received amount to
	if receivedAmountTo != "" {
		query = query.Where("import_details.received_amount <= ?", receivedAmountTo)
	}

	// get value type
	if valueType != "" {
		switch valueType {
		case "shortage":
			query = query.Where("import_details.received_count > import_details.accepted_count")
		case "scanned":
			query = query.Where("import_details.accepted_count > 0")
		case "surplus":
			query = query.Where("import_details.accepted_count > import_details.received_count")
		}
	}

	err := query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("import_details.updated_at DESC").
		Find(&importDetails).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return importDetails, totalCount, nil
}

// send request to 1C for answering import details
func (s *Services) DoRequestOnec(ctx context.Context, data any, url string) error {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	buf := bytes.Buffer{}
	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		s.log.Errorf("failed to encode request data: %v", err)
		return fmt.Errorf("failed to encode request data: %v", err)
	}
	req := &http.Request{}

	// Construct request
	req, err = http.NewRequestWithContext(ctx, "POST", s.cfg.OnecApiUrl+url, &buf)
	if err != nil {
		s.log.Errorf("failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// set basic auth username and password
	req.SetBasicAuth(s.cfg.OnecApiUsername, s.cfg.OnecApiPassword)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.cfg.OnecApiUrl != "test" {
		// Execute request
		response, err := client.Do(req)
		if err != nil {
			s.log.Errorf("failed to execute HTTP request: %v", err)
			return fmt.Errorf("failed to execute HTTP request: %v", err)
		}
		// close response body
		defer response.Body.Close()

		// var info map[string]any
		_, err = io.ReadAll(response.Body)
		if err != nil {
			s.log.Errorf("could not decode response: %v", err)
			return err
		}

	}
	return nil
}

func (s *Services) performImportTotals(ctx context.Context) {
	newUpdateQuery := `
	UPDATE imports i
	SET
		received_count = t.received_count,
		received_sum   = t.received_sum
	FROM (
		SELECT
			import_id,
			COALESCE(SUM(received_count), 0) AS received_count,
			COALESCE(SUM(received_count * retail_price_vat), 0) AS received_sum
		FROM import_details
		GROUP BY import_id
	) AS t
	WHERE i.id = t.import_id AND i.status = 'new' AND (i.received_sum = 0 OR i.received_count = 0 )
	AND i.entry_type = 1;
	`
	err := s.db.WithContext(ctx).Raw(newUpdateQuery).Error
	if err != nil {
		s.log.Errorf("could not update new imports total: %v", err)
	}
}

func (s *Services) UpdateImportTotal(importId string) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	query := `
	UPDATE imports i
	SET
		received_count = t.received_count,
		received_sum   = t.received_sum
	FROM (
		SELECT
			import_id,
			COALESCE(SUM(received_count), 0) AS received_count,
			COALESCE(SUM(received_count * retail_price_vat), 0) AS received_sum
		FROM import_details
		GROUP BY import_id
	) AS t
	WHERE i.id = ? AND i.id = t.import_id  AND i.entry_type = 1;
	`
	err := s.db.WithContext(ctx).Exec(query, importId).Error
	if err != nil {
		s.log.Errorf("could not update import totals after create new import: %v", err)
		return
	}
}
