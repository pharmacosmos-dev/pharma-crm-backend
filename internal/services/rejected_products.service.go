package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) CreateRejectedProduct(req *domain.RejectedProductRequest) error {
	if err := s.db.Table("rejected_products").Create(req).Error; err != nil {
		s.log.Errorf("could not create rejected product: %v", err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) GetRejectedProductsSearch(ctx context.Context, params *domain.RejectedProductQueryParam) ([]domain.Product, error) {
	var products []domain.Product

	query := s.db.
		Select(`
			p.id, 
			p.name, 
			p.barcode, 
			p.unit_per_pack, 
			pr.id AS producer_id,
			pr.name AS manufacturer
		`).
		Table("products p").
		Joins("LEFT JOIN producers AS pr ON pr.id = p.producer_id").
		Where("p.deleted_at IS NULL")

	if params.Search != "" {
		query = query.Where("p.name ILIKE ?", fmt.Sprintf("%%%s%%", params.Search))
	}
	err := query.
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&products).Error
	if err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, domain.InternalServerError
	}

	for i := range products {
		products[i].Producer = domain.NewNullStruct(domain.Producer{
			Id:   &products[i].ProducerID,
			Name: products[i].Manufacturer,
		}, products[i].ProducerID != "")
	}

	return products, nil
}

func (s *Services) ListRejectedProducts(ctx context.Context, params *domain.RejectedProductQueryParam) ([]domain.RejectedProduct, int64, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"rp.id AS id",
			"rp.store_id AS store_id",
			"rp.product_id AS product_id",
			"COALESCE(p.name, rp.product_name) AS product_name",
			"s.name AS store_name",
			"rp.rejected_times AS rejected_times",
			"rp.count AS count",
			"e.full_name AS created_by",
			"rp.created_at AS created_at",
		).
		Table("rejected_products rp").
		Joins("LEFT JOIN products p ON rp.product_id = p.id").
		Joins("LEFT JOIN stores s ON rp.store_id = s.id").
		Joins("LEFT JOIN employees e ON rp.created_by = e.id")

	if params.Search != "" {
		qb = qb.Where("p.name ILIKE ? OR rp.product_name ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	if params.StoreId != "" {
		qb = qb.Where("rp.store_id = ?", params.StoreId)
	}

	if params.ProductId != "" {
		qb = qb.Where("rp.product_id = ?", params.ProductId)
	}

	order := " created_at DESC"

	switch params.Order {
	case "+count":
		order = " count DESC"
	case "-count":
		order = " count ASC"
	case "+created_at":
		order = " created_at DESC"
	case "-created_at":
		order = " created_at ASC"
	default:
		order = " created_at DESC"
	}

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, 0, domain.InternalServerError
	}

	qb = qb.Order(order)

	var res []domain.RejectedProduct
	err := qb.Limit(params.Limit).Offset(params.Offset).Debug().Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get rejected_products %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// CreateReservedDocument - creates a new reserved document
func (s *Services) CreateReservedDocument(ctx context.Context, storeId string, req *domain.CreateReservedDocumentRequest, createdBy string) (*domain.Reserved, error) {
	// Validate store_id
	if storeId == "" {
		s.log.Errorf("store_id is empty")
		return nil, domain.NewError(400, "store_id is required")
	}

	// Check if store exists
	var store domain.Store
	if err := s.db.WithContext(ctx).Where("id = ?", storeId).First(&store).Error; err != nil {
		s.log.Errorf("store not found: %v", err)
		return nil, domain.NewError(404, "store not found")
	}

	reserved := &domain.Reserved{
		StoreId:        storeId,
		DocumentNumber: req.DocumentNumber,
		Status:         "new",
		Comment:        req.Comment,
		CreatedBy:      &createdBy,
	}

	if err := s.db.WithContext(ctx).Create(reserved).Error; err != nil {
		s.log.Errorf("could not create reserved document: %v", err)
		return nil, domain.InternalServerError
	}

	return reserved, nil
}

// AddReservedDetails - adds or updates reserved details
// If product already exists in the reserved document, it updates the quantity by adding the new quantity
func (s *Services) AddReservedDetails(ctx context.Context, reservedId string, req *domain.AddReservedDetailsRequest, updatedBy string) (*domain.ReservedDetails, error) {
	var existingDetail domain.ReservedDetails

	// Check if detail already exists for this product in this reserved document
	err := s.db.WithContext(ctx).
		Where("reserved_id = ? AND product_id = ?", reservedId, req.ProductId).
		First(&existingDetail).Error

	if err == nil {
		// Detail exists, update quantity by adding new quantity to existing
		existingDetail.Quantity += req.Quantity
		existingDetail.UpdatedBy = &updatedBy

		if err := s.db.WithContext(ctx).Save(&existingDetail).Error; err != nil {
			s.log.Errorf("could not update reserved detail: %v", err)
			return nil, domain.InternalServerError
		}

		// Update reserved document totals
		if err := s.updateReservedTotals(ctx, reservedId); err != nil {
			return nil, err
		}

		return &existingDetail, nil
	}

	// Detail doesn't exist, create new one
	detail := &domain.ReservedDetails{
		ReservedId: reservedId,
		ProductId:  req.ProductId,
		Quantity:   req.Quantity,
		CreatedBy:  &updatedBy,
		UpdatedBy:  &updatedBy,
	}

	if err := s.db.WithContext(ctx).Create(detail).Error; err != nil {
		s.log.Errorf("could not create reserved detail: %v", err)
		return nil, domain.InternalServerError
	}

	// Update reserved document totals
	if err := s.updateReservedTotals(ctx, reservedId); err != nil {
		return nil, err
	}

	return detail, nil
}

// ListReservedDetails - lists reserved details and creates/updates reserved document if needed
// If reserved document doesn't exist for store_id, creates new one
// If reserved document exists with status='new', uses it
// If product already exists in reserved details, updates quantity
func (s *Services) ListReservedDetails(ctx context.Context, storeId string, req *domain.ListReservedDetailsRequest, createdBy string) ([]domain.ReservedDetailsWithProduct, error) {
	// Validate store_id
	if storeId == "" {
		s.log.Errorf("store_id is empty")
		return nil, domain.NewError(400, "store_id is required")
	}

	// Check if store exists
	var store domain.Store
	if err := s.db.WithContext(ctx).Where("id = ?", storeId).First(&store).Error; err != nil {
		s.log.Errorf("store not found: %v", err)
		return nil, domain.NewError(404, "store not found")
	}

	// Check if reserved document exists for this store with status='new'
	var reserved domain.Reserved

	err := s.db.WithContext(ctx).
		Where("store_id = ? AND status = ?", storeId, "new").
		First(&reserved).Error

	if err != nil {
		// Reserved document with status='new' doesn't exist
		// Check if there's any reserved document with status='done'
		var doneReserved domain.Reserved
		_ = s.db.WithContext(ctx).
			Where("store_id = ? AND status = ?", storeId, "done").
			First(&doneReserved).Error

		// If status='done' exists OR no reserved document exists at all, create new one
		docNum := fmt.Sprintf("RES-%d", time.Now().Unix())
		newReserved := &domain.Reserved{
			StoreId:        storeId,
			DocumentNumber: docNum,
			Status:         "new",
			CreatedBy:      &createdBy,
		}

		if err := s.db.WithContext(ctx).Create(newReserved).Error; err != nil {
			s.log.Errorf("could not create reserved document: %v", err)
			return nil, domain.InternalServerError
		}

		reserved = *newReserved
	}

	// Check if product already exists in reserved_details
	var existingDetail domain.ReservedDetails

	detailErr := s.db.WithContext(ctx).
		Where("reserved_id = ? AND product_id = ?", reserved.Id, req.ProductId).
		First(&existingDetail).Error

	if detailErr == nil {
		// Product exists, update quantity by adding
		existingDetail.Quantity += req.Quantity
		existingDetail.UpdatedBy = &createdBy

		if err := s.db.WithContext(ctx).Save(&existingDetail).Error; err != nil {
			s.log.Errorf("could not update reserved detail: %v", err)
			return nil, domain.InternalServerError
		}
	} else {
		// Product doesn't exist, create new detail
		detail := &domain.ReservedDetails{
			ReservedId: reserved.Id,
			ProductId:  req.ProductId,
			Quantity:   req.Quantity,
			CreatedBy:  &createdBy,
			UpdatedBy:  &createdBy,
		}

		if err := s.db.WithContext(ctx).Create(detail).Error; err != nil {
			s.log.Errorf("could not create reserved detail: %v", err)
			return nil, domain.InternalServerError
		}
	}

	// Update reserved document totals
	if err := s.updateReservedTotals(ctx, reserved.Id); err != nil {
		return nil, err
	}

	// Get all details with product info for this reserved document
	var details []domain.ReservedDetailsWithProduct

	err = s.db.WithContext(ctx).
		Select(`
			rd.id,
			rd.reserved_id,
			rd.product_id,
			rd.quantity,
			rd.created_by,
			rd.updated_by,
			rd.created_at,
			rd.updated_at,
			p.name AS product_name,
			p.barcode AS product_code
		`).
		Table("reserved_details rd").
		Joins("LEFT JOIN products p ON rd.product_id = p.id").
		Where("rd.reserved_id = ?", reserved.Id).
		Scan(&details).Error

	if err != nil {
		s.log.Errorf("could not get reserved details: %v", err)
		return nil, domain.InternalServerError
	}

	return details, nil
}

// updateReservedTotals - updates total_quantity and total_product_count in reserved document
func (s *Services) updateReservedTotals(ctx context.Context, reservedId string) error {
	var totalQuantity float64
	var totalCount int64

	err := s.db.WithContext(ctx).
		Table("reserved_details").
		Where("reserved_id = ?", reservedId).
		Select("COALESCE(SUM(quantity), 0)", "COUNT(*)").
		Row().
		Scan(&totalQuantity, &totalCount)

	if err != nil {
		s.log.Errorf("could not calculate reserved totals: %v", err)
		return domain.InternalServerError
	}

	if err := s.db.WithContext(ctx).
		Model(&domain.Reserved{}).
		Where("id = ?", reservedId).
		Updates(map[string]interface{}{
			"total_quantity":      totalQuantity,
			"total_product_count": totalCount,
		}).Error; err != nil {
		s.log.Errorf("could not update reserved totals: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// InsertReservedDetailsDirect - direct insert to reserved_details from frontend
// Gets latest reserved document for store, creates new if status='done', then inserts/updates detail
func (s *Services) InsertReservedDetailsDirect(ctx context.Context, req *domain.InsertReservedDetailsDirectRequest, createdBy string) (*domain.ReservedDetailsWithProduct, error) {
	// Get latest reserved document for this store ordered by created_at DESC
	var reserved domain.Reserved

	err := s.db.WithContext(ctx).
		Where("store_id = ?", req.StoreId).
		Order("created_at DESC").
		First(&reserved).Error

	// If no reserved document exists or status is 'done', create new one
	if err != nil || reserved.Status == "done" {
		docNum := fmt.Sprintf("RES-%d", time.Now().Unix())
		newReserved := &domain.Reserved{
			StoreId:        req.StoreId,
			DocumentNumber: docNum,
			Status:         "new",
			CreatedBy:      &createdBy,
		}

		if err := s.db.WithContext(ctx).Create(newReserved).Error; err != nil {
			s.log.Errorf("could not create reserved document: %v", err)
			return nil, domain.InternalServerError
		}

		reserved = *newReserved
	}

	// Check if product already exists in reserved_details
	var existingDetail domain.ReservedDetails

	detailErr := s.db.WithContext(ctx).
		Where("reserved_id = ? AND product_id = ?", reserved.Id, req.ProductId).
		First(&existingDetail).Error

	if detailErr == nil {
		// Product exists, update quantity by adding
		existingDetail.Quantity += req.Quantity
		existingDetail.UpdatedBy = &createdBy

		if err := s.db.WithContext(ctx).Save(&existingDetail).Error; err != nil {
			s.log.Errorf("could not update reserved detail: %v", err)
			return nil, domain.InternalServerError
		}
	} else {
		// Product doesn't exist, create new detail
		detail := &domain.ReservedDetails{
			ReservedId: reserved.Id,
			ProductId:  req.ProductId,
			Quantity:   req.Quantity,
			CreatedBy:  &createdBy,
			UpdatedBy:  &createdBy,
		}

		if err := s.db.WithContext(ctx).Create(detail).Error; err != nil {
			s.log.Errorf("could not create reserved detail: %v", err)
			return nil, domain.InternalServerError
		}
	}

	// Update reserved document totals
	if err := s.updateReservedTotals(ctx, reserved.Id); err != nil {
		return nil, err
	}

	// Get the detail with product info
	var detail domain.ReservedDetailsWithProduct

	err = s.db.WithContext(ctx).
		Select(`
			rd.id,
			rd.reserved_id,
			rd.product_id,
			rd.quantity,
			rd.created_by,
			rd.updated_by,
			rd.created_at,
			rd.updated_at,
			p.name AS product_name,
			p.barcode AS product_code
		`).
		Table("reserved_details rd").
		Joins("LEFT JOIN products p ON rd.product_id = p.id").
		Where("rd.product_id = ? AND rd.reserved_id = ?", req.ProductId, reserved.Id).
		Scan(&detail).Error

	if err != nil {
		s.log.Errorf("could not get reserved detail: %v", err)
		return nil, domain.InternalServerError
	}

	return &detail, nil
}
