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
func (s *Services) AcceptImport(importID string, userID string, acceptType string) error {
	// begin transactions
	tx := s.db.Begin()

	// Get import info
	var res domain.Import
	err := tx.First(&res, "id = ?", importID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NotFoundError
		}
		s.log.Error("could not get import(%s) info: %v", importID, err)
		return domain.InternalServerError
	}

	// check error and rollback
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// update import status
	err = s.UpdateImportCompletedStatus(tx, importID, userID)
	if err != nil {
		s.log.Error("could not update import(%s) status: %v", importID, err)
		return domain.InternalServerError
	}

	if acceptType == "all" {
		// update accepted_count and scanned_count to received_count
		err = s.UpdateImportDetailsAccepted(tx, importID)
		if err != nil {
			return domain.InternalServerError
		}

		// add all imported products to store_products and send 1C
		err = s.AddAllProductsToStore(tx, &res)
		if err != nil {
			s.log.Error("could not accept import products: %v", err)
			return domain.InternalServerError
		}
	} else {
		err = s.AddSomeImportedProductsToStore(tx, &res)
		if err != nil {
			s.log.Error("could not accept some import products: %v", err)
			return domain.InternalServerError
		}
	}

	// completed transaction
	err = tx.Commit().Error
	if err != nil {
		s.log.Error("could not completed transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// Canceled import
func (s *Services) CancelImport(tx *gorm.DB, id string, userID string) (*domain.Import, error) {
	var res domain.Import
	err := tx.Raw(`UPDATE imports SET status = ?, accepted_by = ? WHERE id = ? RETURNING *`, constants.GeneralStatusCanceled, userID, id).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &res, nil
}

// Add some imported products to stores
func (s *Services) AddSomeImportedProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var reqFakt domain.AcceptImport1C

	// import_detail list by import_id
	importDetails, err := s.GetImportDetailsByImportId(importData.Id)
	if err != nil {
		return err
	}

	// get store info by using import_id
	store, err := s.GetStoreByImportId(importData.Id)
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
			err = tx.Exec(storeProductQuery,
				importData.StoreID,
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
				s.log.Warn("ERROR on inserting import products to store_product: %v", err)
				return err
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
	err = s.DoRequestOnec(context.Background(), reqFakt, constants.OnecPathPrihod)
	if err != nil {
		s.log.Error("could not send prixod response: %v", err)
	}

	return nil
}

// add all imported products to store
func (s *Services) AddAllProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var (
		reqFakt domain.AcceptImport1C
		store   *domain.Store
	)

	// get import_detail list by import_id
	details, err := s.GetImportDetailsByImportId(importData.Id)
	if err != nil {
		return err
	}
	// get store info by import_id
	store, err = s.GetStoreByImportId(importData.Id)
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

			err = tx.Exec(storeProductQuery,
				importData.StoreID,
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
				s.log.Error("could not add import products to store_product: %v", err)
				return err
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
	err = s.DoRequestOnec(context.Background(), reqFakt, constants.OnecPathPrihod)
	if err != nil {
		s.log.Error("could not send request to 1C", err)
	}

	return nil
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
	var (
		imports    []domain.Import
		totalCount int64
	)

	// Fetch imports with detailed data
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*,
			ROUND(SUM(import_details.retail_price * import_details.received_count)::numeric, 2) AS received_amount,
			ROUND(SUM(import_details.retail_price * import_details.accepted_count)::numeric, 2) AS accepted_amount,
			ROUND(SUM(import_details.retail_price_vat * import_details.received_count)::numeric, 2) AS received_amount_vat,
			ROUND(SUM(import_details.retail_price_vat * import_details.accepted_count)::numeric, 2) AS accepted_amount_vat,
			ROUND(SUM(import_details.received_count)::numeric, 2) AS received_count,
			ROUND(SUM(import_details.accepted_count)::numeric, 2) AS accepted_count
		`).Joins("LEFT JOIN import_details ON imports.id = import_details.import_id").
		Where("imports.entry_type = ?", 1)

	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where(`
		imports.document_number ILIKE ? OR
		CAST(imports.public_id AS TEXT) LIKE ?`, params.Search, params.Search)
	}
	if params.StoreId != "" {
		query = query.Where("imports.store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		query = query.Where("stores.company_id = ?", params.CompanyId).
			Joins("LEFT JOIN stores ON imports.store_id = stores.id")
	}
	if params.StartDate != "" && params.EndDate != "" {
		query = query.Where("imports.created_at BETWEEN ? AND ?", params.StartDate, params.EndDate)
	}
	if params.Status != "" {
		query = query.Where("imports.status = ?", params.Status)
	}
	if params.ReceivedAmountFrom > 0 {
		query = query.Where("received_amount >= ?", params.ReceivedAmountFrom)
	}
	if params.ReceivedAmountTo > 0 {
		query = query.Where("received_amount <= ?", params.ReceivedAmountTo)
	}
	err := query.Group("imports.id").
		Order("imports.created_at DESC").
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&imports).Error
	if err != nil {
		s.log.Errorf("could not get imports: %v", err)
		return nil, 0, domain.InternalServerError
	}
	return imports, totalCount, nil
}

func (s *Services) GetImportsStats(ctx context.Context, params *domain.ImportQueryParams) (*domain.ImportStatusSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN imports.status = 'completed' THEN import_details.retail_price_vat * import_details.accepted_count ELSE 0 END), 0) AS completed_received_vat_amount,
			COALESCE(SUM(CASE WHEN imports.status = 'new' THEN import_details.retail_price_vat * import_details.received_count ELSE 0 END), 0) AS new_accepted_vat_amount,
			COALESCE(SUM(CASE WHEN imports.status = 'completed' THEN import_details.accepted_count ELSE 0 END), 0) AS completed_accepted_count,
			COALESCE(SUM(CASE WHEN imports.status = 'new' THEN import_details.received_count ELSE 0 END), 0) AS new_received_count
		FROM imports
		LEFT JOIN import_details ON imports.id = import_details.import_id
		LEFT JOIN stores ON imports.store_id = stores.id
		WHERE imports.entry_type = 1
	`

	var args []any

	if params.StoreId != "" {
		query += " AND imports.store_id = ?"
		args = append(args, params.StoreId)
	}
	if params.CompanyId != "" {
		query += " AND stores.company_id = ?"
		args = append(args, params.CompanyId)
	}

	if params.StartDate != "" && params.EndDate != "" {
		query += " AND imports.created_at BETWEEN ? AND ? "
		args = append(args, params.StartDate, params.EndDate)
	}
	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		query += " AND (imports.document_number ILIKE ? OR CAST(imports.public_id AS TEXT) ILIKE ?)"
		args = append(args, params.Search, params.Search)
	}
	if params.Status != "" {
		query += " AND imports.status = ?"
		args = append(args, params.Status)
	}
	if params.ReceivedAmountFrom > 0 {
		query += `
		AND (
			SELECT ROUND(SUM(d.retail_price * d.received_count)::numeric, 2)
			FROM import_details d
			WHERE d.import_id = imports.id
		) >= ?
		`
		args = append(args, params.ReceivedAmountFrom)
	}
	if params.ReceivedAmountTo > 0 {
		query += `
		AND (
			SELECT ROUND(SUM(d.retail_price * d.received_count)::numeric, 2)
			FROM import_details d
			WHERE d.import_id = imports.id
		) <= ?
		`
		args = append(args, params.ReceivedAmountTo)
	}

	var summary domain.ImportStatusSummary
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&summary).Error
	if err != nil {
		s.log.Errorf("could not get imports stats: %v", err)
		return nil, domain.InternalServerError
	}
	return &summary, nil
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
func (s *Services) GetImportDetailsByImportId(importId string) ([]domain.ImportDetail, error) {
	var importDetails []domain.ImportDetail
	err := s.db.Raw(`
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
		s.log.Error(err)
		return nil, err
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
		s.log.Error("failed to encode request data: %v", err)
		return fmt.Errorf("failed to encode request data: %v", err)
	}
	req := &http.Request{}

	// Construct request
	req, err = http.NewRequestWithContext(ctx, "POST", s.cfg.OnecApiUrl+url, &buf)
	if err != nil {
		s.log.Error("failed to create HTTP request: %v", err)
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
			s.log.Error("failed to execute HTTP request: %v", err)
			return fmt.Errorf("failed to execute HTTP request: %v", err)
		}
		// close response body
		defer response.Body.Close()

		// var info map[string]any
		res, err := io.ReadAll(response.Body)
		if err != nil {
			s.log.Error("could not decode response: %w", err)
			return err
		}

		s.log.Info("RASXOD RESPONSE: %v", string(res))
	}
	return nil
}
