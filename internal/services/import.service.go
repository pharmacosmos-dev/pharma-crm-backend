package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// Update value by chosen field
func (s *Storage) UpdateImportByField(tx *gorm.DB, id string, field, value string) (*domain.Import, error) {
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

// accept import
func (s *Storage) AcceptImport(tx *gorm.DB, id string, userID string) (*domain.Import, error) {
	var res domain.Import
	err := tx.Raw(`UPDATE imports SET status = ?, accepted_by = ? WHERE id = ? RETURNING *`, config.COMPLETED_IMPORT, userID, id).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &res, nil
}

// Canceled import
func (s *Storage) CancelImport(tx *gorm.DB, id string, userID string) (*domain.Import, error) {
	var res domain.Import
	err := tx.Raw(`UPDATE imports SET status = ?, accepted_by = ? WHERE id = ? RETURNING *`, config.CANCELED_IMPORT, userID, id).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &res, nil
}

// Add some imported products to stores
func (s *Storage) AddSomeImportedProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var (
		reqFakt domain.AcceptImport1C
	)
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
	reqFakt.Dok.DocumentDate = importData.ImportDate.Format(config.DATE_1C_FORMAT)
	reqFakt.Apteka.StoreCode = store.StoreCode
	reqFakt.Apteka.Name = store.Name

	// add products to store
	storeProductQuery := `INSERT INTO store_products(store_id, product_id, pack_quantity, unit_quantity, supply_price, retail_price, vat, expire_date) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range importDetails {
		if item.AcceptedCount > 0 {
			// add imported products to store_products
			err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.AcceptedCount, item.UnitPerPack*item.AcceptedCount, item.SupplyPriceVat, item.RetailPriceVat, item.Vat, item.ExpireDate).Error
			if err != nil {
				s.log.Error(err)
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
	err = s.DoRequest(context.Background(), reqFakt, "/prihod")
	if err != nil {
		s.log.Error(err)
		tx.Rollback()
		return errors.New("failed to send fakt to 1C")
	}
	return nil
}

// add all imported products to store
func (s *Storage) AddAllProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var (
		importDetails []domain.ImportDetail
		reqFakt       domain.AcceptImport1C
		store         *domain.Store
	)
	// update imports detail accepted_count to received_count
	err := tx.Exec(`
	UPDATE import_details 
	SET
		accepted_count = received_count
	WHERE import_id = ?`, importData.Id).Error
	if err != nil {
		s.log.Error(err)
		return err
	}

	// get import_detail list by import_id
	importDetails, err = s.GetImportDetailsByImportId(importData.Id)
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
	reqFakt.Dok.DocumentDate = importData.ImportDate.Format(config.DATE_1C_FORMAT)
	reqFakt.Apteka.StoreCode = store.StoreCode
	reqFakt.Apteka.Name = store.Name

	// add products to store
	storeProductQuery := `INSERT INTO store_products(store_id, product_id, pack_quantity, unit_quantity, supply_price, retail_price, vat, expire_date) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range importDetails {
		if item.ReceivedCount > 0 {
			err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.ReceivedCount, item.UnitPerPack*item.ReceivedCount, item.SupplyPriceVat, item.RetailPriceVat, item.Vat, item.ExpireDate).Error
			if err != nil {
				s.log.Error(err)
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

	// // send fakt to 1C
	err = s.DoRequest(context.Background(), reqFakt, "/prihod")
	if err != nil {
		s.log.Error(err)
		return errors.New("failed to send fakt to 1C")
	}

	return nil
}

// update import details to cancel
func (s *Storage) UpdateImportDetailsToCancel(tx *gorm.DB, importID string) error {
	err := tx.Exec(`UPDATE import_details SET canceled_count = received_count WHERE import_id = ?`, importID).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}

// create import details
func (s *Storage) CreateImportDetail(tx *gorm.DB, req *domain.ImportDetailRequest) (string, error) {
	var (
		id    string
		query = `INSERT INTO import_details(
			import_id, product_id, received_count, accepted_count, supply_price, supply_price_vat, retail_price, retail_price_vat, expire_date, vat, vat_sum, series_number, sum_vat, marking)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	)
	req.Marking = utils.StringArray(req.Marking)
	// complete query
	err := tx.Debug().Raw(query, req.ImportID, req.ProductID, req.ReceivedCount, req.AcceptedCount, req.SupplyPrice, req.SupplyPriceVat, req.RetailPrice, req.RetailPriceVat, req.ExpireDate, req.Vat, req.VatSum, req.SeriesNumber, req.SumVat, req.Marking).Scan(&id).Error
	if err != nil {
		s.log.Error(err)
		return "", err
	}
	return id, nil
}

// create product marking
func (s *Storage) CreateProductMarking(tx *gorm.DB, req domain.ProductMarkingReq) error {
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
func (s *Storage) ListImport(c *gin.Context, limit, offset int) ([]domain.Import, int64, error) {
	var (
		imports          []domain.Import
		totalCount       int64
		search           = c.Query("search")
		storeID          = c.Query("store_id")
		startDate        = c.Query("start_date")
		endDate          = c.Query("end_date")
		status           = c.Query("status")
		receivePriceFrom = c.Query("receive_amount_from")
		receivePriceTo   = c.Query("receive_amount_to")
		err              error
	)
	// get user id from header
	userId, ok := c.Get("user_id")
	if !ok {
		err = errors.New("user not found in context")
		return nil, 0, err
	}
	var employee domain.Employee
	err = s.db.First(&employee, "id = ?", userId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			err = errors.New("employee not found")
		}
		s.log.Error(err)
		return nil, 0, err
	}
	// check if employee is not admin or superadmin
	if !helper.IsAdmin(employee, s.cfg) {
		storeID = employee.StoreId
	}

	// Fetch imports with detailed data
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*, 
			SUM(import_details.retail_price*import_details.received_count) as received_amount, 
			SUM(import_details.retail_price*import_details.accepted_count) as accepted_amount, 
			SUM(import_details.retail_price_vat*import_details.received_count) as received_amount_vat,
			SUM(import_details.retail_price_vat*import_details.accepted_count) as accepted_amount_vat, 
			SUM(import_details.received_count) as received_count, 
			SUM(import_details.accepted_count) as accepted_count
		`).Joins("LEFT JOIN import_details ON imports.id = import_details.import_id")

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		imports.document_number ILIKE ? OR 
		CAST(imports.public_id AS TEXT) LIKE ?`, search, search)
	}
	if storeID != "" {
		query = query.Where("imports.store_id = ?", storeID)
	}
	if startDate != "" {
		query = query.Where("imports.import_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("imports.import_date <= ?", endDate)
	}
	if status != "" {
		query = query.Where("imports.status = ?", status)
	}
	if receivePriceFrom != "" {
		query = query.Where("received_amount >= ?", receivePriceFrom)
	}
	if receivePriceTo != "" {
		query = query.Where("received_amount <= ?", receivePriceTo)
	}
	err = query.Group("imports.id").
		Order("imports.created_at DESC").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Debug().
		Find(&imports).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return imports, totalCount, nil
}

// list import detail
func (s *Storage) ListImportDetail(c *gin.Context, limit, offset int) ([]domain.ImportDetail, int64, error) {
	var (
		importDetails      []domain.ImportDetail
		totalCount         int64
		importId           = c.Query("import_id")
		search             = c.Query("search")
		receivedAmountFrom = c.Query("received_amount_from")
		receivedAmountTo   = c.Query("received_amount_to")
	)

	// Parse no_barcode as boolean
	noBarcode, _ := strconv.ParseBool(c.Query("no_barcode"))
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

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		products.barcode LIKE ? OR 
		products.name ILIKE ? OR
		CAST(products.material_code AS TEXT) LIKE ?`, search, search, search)
	}
	if receivedAmountFrom != "" {
		query = query.Where("import_details.received_amount >= ?", receivedAmountFrom)
	}
	if receivedAmountTo != "" {
		query = query.Where("import_details.received_amount <= ?", receivedAmountTo)
	}

	if noBarcode {
		query = query.Where("products.barcode IS NULL OR products.barcode = ''")
	}
	err := query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("products.name").
		Debug().
		Find(&importDetails).Error
	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return importDetails, totalCount, nil
}

// get import details by import id
func (s *Storage) GetImportDetailsByImportId(importId string) ([]domain.ImportDetail, error) {
	var importDetails []domain.ImportDetail
	err := s.db.Raw(`
		SELECT 
			import_details.*, 
			products.unit_per_pack, 
			products.barcode, 
			products.name as product_name, 
			products.material_code, 
			COALESCE(pr.name, '') as producer_name
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
func (s *Storage) ListImportDetailByLastUpdated(c *gin.Context, limit, offset int) ([]domain.ImportDetail, int64, error) {
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
		Debug().
		Find(&importDetails).Error

	if err != nil {
		s.log.Error(err)
		return nil, 0, err
	}
	return importDetails, totalCount, nil
}

// send request to 1C for answering import details
func (s *Storage) DoRequest(ctx context.Context, data any, url string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	request, _ := json.Marshal(data)
	fmt.Println("REQUEST: ", string(request))
	buf := bytes.Buffer{}
	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		s.log.Error("failed to encode request data: %v", err)
		return fmt.Errorf("failed to encode request data: %v", err)
	}
	// Construct request
	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.BaseUrl1C+url, &buf)
	if err != nil {
		s.log.Error("failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	// set basic auth username and password
	req.SetBasicAuth(s.cfg.BaseUsername1C, s.cfg.BasePassword1C)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	response, err := client.Do(req)
	if err != nil {
		s.log.Error("failed to execute HTTP request: %v", err)
		return fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	// close response body
	defer response.Body.Close()
	// t, _ := io.ReadAll(response.Body)
	// fmt.Println("RESPONSE: ", string(t))

	var info map[string]any
	// read response body
	err = json.NewDecoder(response.Body).Decode(&info)
	if err != nil {
		s.log.Error("ERROR on decoding response: %w", err)
		return err
	}
	if value, ok := info["ok"]; !ok || value != true {
		s.log.Error(info)
		return fmt.Errorf("failed to answer prihod response: %v", info)
	}

	return nil
}
