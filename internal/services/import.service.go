package services

import (
	"fmt"

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
		err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.AcceptedCount, item.UnitPerPack*item.AcceptedCount, item.SupplyPrice, item.RetailPrice, item.Vat, item.ExpireDate).Error
		if err != nil {
			return err
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
		err = tx.Exec(storeProductQuery, importData.StoreID, item.ProductID, item.ReceivedCount, item.UnitPerPack*item.ReceivedCount, item.SupplyPrice, item.RetailPrice, item.Vat, item.ExpireDate).Error
		if err != nil {
			s.log.Error(err)
			return err
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
			import_id, product_id, received_count, accepted_count, supply_price, retail_price, expire_date, vat, vat_sum, series_number)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	)

	err := tx.Raw(query, req.ImportID, req.ProductID, req.ReceivedCount, req.AcceptedCount, req.SupplyPrice, req.RetailPrice, req.ExpireDate, req.Vat, req.VatSum, req.SeriesNumber).Scan(&id).Error
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
