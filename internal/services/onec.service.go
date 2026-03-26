package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (s *Services) CreateImportFromOnec(ctx context.Context, req *domain.CreateOnecImportDto) error {
	company, err := s.getCompanyForCheckFranchise(ctx, req.Apteka.Franshiza)
	if err != nil {
		return err
	}

	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// get store info
	store, err := s.getOrCreateStoreByStoreCode(ctx, tx, &req.Apteka, company.ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// create new import
	importId, err := s.createNewImportOnImportingOnec(ctx, tx, &domain.ImportRequest{
		StoreID:        store.Id,
		DocumentNumber: req.Dok.DocumentNumber,
	})
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// create or get product(markings, barcodes) and create import_details
	err = s.createOrGetProductAndImportDetails(ctx, tx, req.Товары, importId, company.ID, store.Id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// check transaction completed
	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commited create onec import transaction: %v", err)
		return domain.InternalServerError
	}

	go s.updateImportTotalsAfterCreateNewImport(importId)

	return nil
}

func (s *Services) getCompanyForCheckFranchise(ctx context.Context, isFranchise bool) (*domain.Company, error) {
	var company domain.Company
	if isFranchise {
		err := s.db.WithContext(ctx).Take(&company, "name ilike ?", "%"+constants.PharmaCosmos+"%").Error // todo 1c given companyName
		if err != nil {
			s.log.Errorf("could not get company for check franchise: %v", err)
			return nil, domain.InternalServerError
		}
	} else {
		err := s.db.WithContext(ctx).Take(&company, "name ilike ?", "%"+constants.PharmaCosmos+"%").Error
		if err != nil {
			s.log.Errorf("could not get company for check franchise: %v", err)
			return nil, domain.InternalServerError
		}
	}

	return &company, nil
}

func (s *Services) getOrCreateStoreByStoreCode(ctx context.Context, tx *gorm.DB, req *domain.Apteka, companyId string) (*domain.Store, error) {
	// get store info
	var store domain.Store
	err := tx.WithContext(ctx).First(&store, "store_code = ?", req.StoreCode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		store, err = s.CreateStoreOnImport(ctx, tx, &domain.StoreRequest{Name: req.Name, StoreCode: req.StoreCode, CompanyId: companyId})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		s.log.Errorf("could not get store by store_code on importing: %v", err)
		return nil, domain.InternalServerError
	}

	return &store, nil
}

func (s *Services) createNewImportOnImportingOnec(ctx context.Context, tx *gorm.DB, req *domain.ImportRequest) (string, error) {
	var importId string
	// create new import
	query := `INSERT INTO imports(store_id, status, import_date, document_number) VALUES(?, ?, ?, ?) RETURNING id;`
	err := tx.WithContext(ctx).Raw(query, req.StoreID, constants.GeneralStatusNew, time.Now(), req.DocumentNumber).Scan(&importId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			s.log.Errorf("duplicate document_number: %v", err)
			return "", domain.AlreadyExistsError
		}
		s.log.Errorf("could not create new import dok on importing: %v", err)
		return "", domain.InternalServerError
	}

	return importId, nil
}

func (s *Services) createOrGetProductAndImportDetails(
	ctx context.Context,
	tx *gorm.DB,
	products []domain.ProductRequestOnecDto,
	importId string,
	companyId string,
	storeId string,
) error {
	for i := range products {
		// get producer by code
		producer, err := s.GetProducerByCode(ctx, tx, products[i].Manufacturer)
		if err != nil {
			s.log.Errorf("could not get producer by code on importing: %v", err)
			return domain.InternalServerError
		}
		// create product id
		productId := uuid.New().String()
		// create or update product
		err = tx.WithContext(ctx).Raw(`
		INSERT INTO products (
			material_code, 
			name, 
			barcode, 
			producer_id, 
			mxik, 
			is_marking,
			company_id
			)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (material_code) DO UPDATE
		SET
			producer_id = EXCLUDED.producer_id,
			is_marking = EXCLUDED.is_marking
		RETURNING id`,
			products[i].MaterialCode,
			products[i].Name,
			products[i].Barcode,
			producer.Id,
			products[i].Ikpu,
			products[i].Mar,
			companyId,
		).Scan(&productId).Error
		if err != nil {
			s.log.Errorf("could not creating new product on importing: %v", err)
			return domain.InternalServerError
		}
		err = tx.WithContext(ctx).Exec(`
    		INSERT INTO product_barcodes (
				product_id, 
				barcode, 
				status
				)
    		SELECT 
				?, ?, ?
    		WHERE NOT EXISTS (
    		    SELECT 1 FROM product_barcodes 
    		    WHERE product_id = ? AND barcode = ? AND status = ?
    		)
		`, productId,
			products[i].Barcode,
			constants.GeneralStatusCompleted,
			productId,
			products[i].Barcode,
			constants.GeneralStatusCompleted,
		).Error
		if err != nil {
			s.log.Errorf("could not create product barcode: %v", err)
			return domain.InternalServerError
		}
		// create import_detail
		var id string
		err = tx.WithContext(ctx).Raw(`
		INSERT INTO import_details(
			product_id, 
			import_id,
			received_count, 
			scanned_count, 
			supply_price, 
			supply_price_vat,
			retail_price, 
			retail_price_vat,
			vat, 
			vat_sum, 
			expire_date, 
			series_number, 
			sum_vat, 
			marking
			) 
			VALUES(
				?, ?, ?, 
				?, ?, ?, 
				?, ?, ?, 
				?, ?, ?, 
				?, ?) 
			RETURNING id`,
			productId,
			importId,
			products[i].Quantity,
			products[i].Quantity,
			products[i].SupplyPrice,
			products[i].SupplyPriceVat,
			products[i].RetailPrice,
			products[i].RetailPriceVat,
			cast.ToInt(products[i].Vat),
			products[i].VatSum,
			products[i].ExpireDate,
			products[i].ProductSeriesNumber,
			products[i].SumVat,
			utils.StringArray(products[i].Markirovka),
		).Scan(&id).Error
		if err != nil {
			s.log.Errorf("could not create import_details on importing: %v", err)
			return domain.InternalServerError
		}
		for _, marking := range products[i].Markirovka {
			err = tx.WithContext(ctx).Exec(`
				INSERT INTO product_markings (
					import_detail_id, 
					product_id, 
					marking, 
					store_id
					)
				VALUES(?, ?, ?, ?)`,
				id,
				productId,
				marking,
				storeId,
			).Error
			if err != nil {
				s.log.Errorf("could not insert marking on importing on importing: %v", err)
				return domain.InternalServerError
			}
		}
	}

	return nil

}

func (s *Services) updateImportTotalsAfterCreateNewImport(importId string) {
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

func (s *Services) CreateProductMaxPriceChanged(
	ctx context.Context,
	req *domain.ProductChangePriceRequest,
) (map[string]any, error) {

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, domain.InternalServerError
	}

	if len(req.Products) == 0 {
        return nil, domain.NewNotAdditionError(400, map[string]any{
            "message": "products list is empty",
        })
    }

	var notFoundCodes []int
	var updatedCount int

	for _, p := range req.Products {

		if p.MaxPrice <= 0 {
			tx.Rollback()
			return nil, domain.NewNotAdditionError(400, map[string]any{
				"message":       "max_price must be greater than 0",
				"material_code": p.MaterialCode,
			})
		}

		result := tx.WithContext(ctx).Exec(`
			UPDATE products
			SET max_price = ?
			WHERE material_code = ?
		`, p.MaxPrice, p.MaterialCode)

		if result.Error != nil {
			tx.Rollback()
			return nil, domain.InternalServerError
		}
	
		if result.RowsAffected == 0 {
			notFoundCodes = append(notFoundCodes, p.MaterialCode)
		} else {
			updatedCount++
		}
	}

	if updatedCount == 0 {
		tx.Rollback()
		return nil, domain.NewNotAdditionError(404, map[string]any{
			"message":                  "products not found",
			"not_found_material_codes": notFoundCodes,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return nil, domain.InternalServerError
	}

	return map[string]any{
		"total_requested": len(req.Products),
		"updated_count": updatedCount,
		"not_found_material_codes": notFoundCodes,
	}, nil
}


// func (s *Services) CreateOrUpdateBarcodes(ctx context.Context, req *domain.CreateOrUpdateBarcodesRequest) error {
// 	tx := s.db.Begin()
// 	defer func() {
// 		if r := recover(); r != nil {
// 			tx.Rollback()
// 		}
// 	}()

// 	// 1. material_code orqali product topish
// 	var productEntry struct {
// 		Id       string `gorm:"column:id"`
// 		UnitCode string `gorm:"column:unit_code"`
// 	}

// 	err := tx.WithContext(ctx).Table("products").
// 		Select("id, unit_code").
// 		Where("material_code = ?", req.MaterialCode).
// 		Limit(1).
// 		Scan(&productEntry).Error
// 	if err != nil {
// 		tx.Rollback()
// 		s.log.Errorf("failed to find product by material_code: %v", err)
// 		return domain.InternalServerError
// 	}

// 	if productEntry.Id == "" {
// 		tx.Rollback()
// 		s.log.Errorf("product not found for material_code: %d", req.MaterialCode)
// 		return domain.NotFoundError
// 	}

// 	for _, item := range req.ProductBarCodeItem {
// 		var barcodeEntry struct {
// 			Barcode  string `gorm:"column:barcode"`
// 			Mxik     string `gorm:"column:mxik"`
// 			UnitCode string `gorm:"column:unit_code"`
// 		}

// 		// 2. Barcode product_barcodes da bormi?
// 		err = tx.WithContext(ctx).Table("product_barcodes").
// 			Select("barcode, mxik, unit_code").
// 			Where("barcode = ? AND product_id = ?", item.Barcode, productEntry.Id).
// 			Limit(1).
// 			Scan(&barcodeEntry).Error
// 		if err != nil {
// 			tx.Rollback()
// 			s.log.Errorf("failed to check barcode: %v", err)
// 			return domain.InternalServerError
// 		}

// 		if barcodeEntry.Barcode != "" {
// 			// Barcode bor — nima o'zgardi?
// 			mxikChanged := barcodeEntry.Mxik != item.Ekpu
// 			unitCodeChanged := barcodeEntry.UnitCode != item.UnitCode

// 			barcodeUpdates := map[string]interface{}{}
// 			if mxikChanged {
// 				barcodeUpdates["mxik"] = item.Ekpu
// 			}
// 			if unitCodeChanged {
// 				barcodeUpdates["unit_code"] = item.UnitCode
// 			}

// 			// product_barcodes update
// 			if len(barcodeUpdates) > 0 {
// 				err = tx.WithContext(ctx).Table("product_barcodes").
// 					Where("barcode = ? AND product_id = ?", barcodeEntry.Barcode, productEntry.Id).
// 					Updates(barcodeUpdates).Error
// 				if err != nil {
// 					tx.Rollback()
// 					s.log.Errorf("failed to update barcode entry: %v", err)
// 					return domain.InternalServerError
// 				}
// 			}

// 			// products ni faqat mxik o'zgarganda update qil
// 			if mxikChanged {
// 				err = tx.WithContext(ctx).Table("products").
// 					Where("id = ?", productEntry.Id).
// 					Updates(map[string]interface{}{
// 						"mxik":      item.Ekpu,
// 						"unit_code": item.UnitCode,
// 					}).Error
// 				if err != nil {
// 					tx.Rollback()
// 					s.log.Errorf("failed to update product mxik and unit_code: %v", err)
// 					return domain.InternalServerError
// 				}
// 				productEntry.UnitCode = item.UnitCode
// 			}
// 		} else {
// 			// 3. Barcode yo'q — ekpu bilan tekshir
// 			// var ekpuEntry struct {
// 			// 	Mxik string `gorm:"column:mxik"`
// 			// }

// 			// err = tx.WithContext(ctx).Table("product_barcodes").
// 			// 	Select("mxik").
// 			// 	Where("mxik = ? AND product_id = ?", item.Ekpu, productEntry.Id).
// 			// 	Limit(1).
// 			// 	Scan(&ekpuEntry).Error
// 			// if err != nil {
// 			// 	tx.Rollback()
// 			// 	s.log.Errorf("failed to check ekpu: %v", err)
// 			// 	return domain.InternalServerError
// 			// }

// 			// Ekpu bor yoki yo'q — har ikki holda CREATE
// 			err = tx.WithContext(ctx).Exec(`
// 				INSERT INTO product_barcodes (product_id, barcode, mxik, unit_code, status)
// 				VALUES (?, ?, ?, ?, ?)
// 			`, productEntry.Id, item.Barcode, item.Ekpu, item.UnitCode, constants.GeneralStatusCompleted).Error
// 			if err != nil {
// 				tx.Rollback()
// 				s.log.Errorf("failed to create product barcode: %v", err)
// 				return domain.InternalServerError
// 			}
// 		}
// 	}

// 	if err := tx.Commit().Error; err != nil {
// 		s.log.Errorf("failed to commit CreateOrUpdateBarcodes: %v", err)
// 		return domain.InternalServerError
// 	}

// 	return nil
// }

type productEntryType struct {
	Id       string `gorm:"column:id"`
	UnitCode string `gorm:"column:unit_code"`
}

func (s *Services) CreateOrUpdateBarcodes(ctx context.Context, req *domain.CreateOrUpdateBarcodesRequest) error {
	productCache := make(map[int]productEntryType)

	for _, item := range req.ProductBarCodeItem {
		productEntry, ok := productCache[item.MaterialCode]
		if !ok {
			err := s.db.WithContext(ctx).Table("products").
				Select("id, unit_code").
				Where("material_code = ?", item.MaterialCode).
				Limit(1).
				Scan(&productEntry).Error
			if err != nil {
				s.log.Errorf("failed to find product by material_code %d: %v", item.MaterialCode, err)
				return domain.InternalServerError
			}
			if productEntry.Id == "" {
				s.log.Errorf("product not found for material_code: %d", item.MaterialCode)
				return domain.NotFoundError
			}
			productCache[item.MaterialCode] = productEntry
		}

		if err := s.processOneBarcode(ctx, item, productEntry); err != nil {
			return err
		}
	}

	return nil
}

func (s *Services) processOneBarcode(ctx context.Context, item domain.CreateOrUpdateBarcodeRequest, productEntry productEntryType) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var barcodeEntry struct {
		Barcode  string `gorm:"column:barcode"`
		Mxik     string `gorm:"column:mxik"`
		UnitCode string `gorm:"column:unit_code"`
	}

	err := tx.WithContext(ctx).Table("product_barcodes").
		Select("barcode, mxik, unit_code").
		Where("barcode = ? AND product_id = ?", item.Barcode, productEntry.Id).
		Limit(1).
		Scan(&barcodeEntry).Error
	if err != nil {
		tx.Rollback()
		s.log.Errorf("failed to check barcode: %v", err)
		return domain.InternalServerError
	}

	if barcodeEntry.Barcode != "" {
		mxikChanged := barcodeEntry.Mxik != item.Ekpu
		unitCodeChanged := barcodeEntry.UnitCode != item.UnitCode

		barcodeUpdates := map[string]interface{}{}
		if mxikChanged {
			barcodeUpdates["mxik"] = item.Ekpu
		}
		if unitCodeChanged {
			barcodeUpdates["unit_code"] = item.UnitCode
		}

		if len(barcodeUpdates) > 0 {
			err = tx.WithContext(ctx).Table("product_barcodes").
				Where("barcode = ? AND product_id = ?", barcodeEntry.Barcode, productEntry.Id).
				Updates(barcodeUpdates).Error
			if err != nil {
				tx.Rollback()
				s.log.Errorf("failed to update barcode entry: %v", err)
				return domain.InternalServerError
			}
		}

		if mxikChanged {
			err = tx.WithContext(ctx).Table("products").
				Where("id = ?", productEntry.Id).
				Updates(map[string]interface{}{
					"mxik":      item.Ekpu,
					"unit_code": item.UnitCode,
				}).Error
			if err != nil {
				tx.Rollback()
				s.log.Errorf("failed to update product mxik and unit_code: %v", err)
				return domain.InternalServerError
			}
		}
	} else {
		err = tx.WithContext(ctx).Exec(`
			INSERT INTO product_barcodes (product_id, barcode, mxik, unit_code, status)
			VALUES (?, ?, ?, ?, ?)
		`, productEntry.Id, item.Barcode, item.Ekpu, item.UnitCode, constants.GeneralStatusCompleted).Error
		if err != nil {
			tx.Rollback()
			s.log.Errorf("failed to create product barcode: %v", err)
			return domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		s.log.Errorf("failed to commit processOneBarcode: %v", err)
		return domain.InternalServerError
	}

	return nil
}



