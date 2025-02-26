package services

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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

// Add some imported products to stores
func (s *Storage) AddImportedProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var (
		importDetails []domain.ImportDetail
	)
	// import_detail list by import_id
	err := tx.Raw(`SELECT import_details.*, products.unit_per_pack FROM import_details JOIN products ON products.id = import_details.product_id WHERE import_id = ?`, importData.Id).Scan(&importDetails).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	// add products to store
	storeProductQuery := `INSERT INTO store_products(store_id, product_id, pack_quantity, unit_quantity, supply_price, retail_price, vat, expire_date) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range importDetails {
		if item.AcceptedCount > 0 {
			err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.AcceptedCount, item.UnitPerPack*item.AcceptedCount, item.SupplyPriceVat, item.RetailPriceVat, item.Vat, item.ExpireDate).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// add all imported products to store
func (s *Storage) AddAllProductsToStore(tx *gorm.DB, importData *domain.Import) error {
	var importDetails []domain.ImportDetail
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
	err = tx.Raw(`SELECT import_details.*, products.unit_per_pack FROM import_details JOIN products ON products.id = import_details.product_id WHERE import_id = ?`, importData.Id).Scan(&importDetails).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	// add products to store
	storeProductQuery := `INSERT INTO store_products(store_id, product_id, pack_quantity, unit_quantity, supply_price, retail_price, vat, expire_date) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`
	for _, item := range importDetails {
		if item.ReceivedCount > 0 {
			err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.ReceivedCount, item.UnitPerPack*item.ReceivedCount, item.SupplyPriceVat, item.RetailPriceVat, item.Vat, item.ExpireDate).Error
			if err != nil {
				s.log.Error(err)
				return err
			}
		}
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
			import_id, product_id, received_count, accepted_count, supply_price, supply_price_vat, retail_price, retail_price_vat, expire_date, vat, vat_sum, series_number, sum_vat)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	)

	err := tx.Debug().Raw(query, req.ImportID, req.ProductID, req.ReceivedCount, req.AcceptedCount, req.SupplyPrice, req.SupplyPriceVat, req.RetailPrice, req.RetailPriceVat, req.ExpireDate, req.Vat, req.VatSum, req.SeriesNumber, req.SumVat).Scan(&id).Error
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

	// Fetch imports with detailed data
	query := s.db.Model(&domain.Import{}).
		Preload("Store").
		Preload("Sender").
		Preload("Receiver").
		Select(`
			imports.*, 
			SUM(import_details.retail_price*import_details.received_count) as received_amount, 
			SUM(import_details.retail_price*import_details.accepted_count) as accepted_amount, 
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
		Order("imports.import_date DESC").
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
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
	// Fetch import details with detailed data
	query := s.db.Model(&domain.ImportDetail{}).
		Preload("Product").
		Preload("Import").
		Select(`
		import_details.*, 
		(import_details.retail_price*received_count) as received_amount,
		(import_details.retail_price*accepted_count) as accepted_amount,
		(import_details.retail_price_vat*received_count) as received_amount_vat,
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
