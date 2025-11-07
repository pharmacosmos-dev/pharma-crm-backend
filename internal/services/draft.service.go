package services

import (
	"context"
	"errors"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

func (s *Services) CreateDraft(ctx context.Context, req *domain.DraftRequest) (*domain.Sale, error) {
	// check cart_items exists
	items, err := s.getCartItemsBySaleId(ctx, req.SaleId)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, domain.NotEnoughProductError
	}

	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	err = s.createNewDraft(ctx, tx, req)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	sale, err := s.updateSaleField(ctx, tx, "status", constants.GeneralStatusDrafted, req.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// create or get sale
	res, err := s.CreateSale(ctx, tx, &domain.SaleRequest{
		EmployeeId:         req.CreatedBy,
		CashBoxOperationId: sale.CashBoxOperationId,
		StoreId:            sale.StoreId,
		CashboxId:          sale.CashboxId,
	})
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) createNewDraft(ctx context.Context, tx *gorm.DB, req *domain.DraftRequest) error {
	err := tx.WithContext(ctx).Table("drafts").Create(&req).Error
	if err != nil {
		s.log.Errorf("could not create new draft: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) updateDraftField(ctx context.Context, tx *gorm.DB, field string, value string, id string) (*domain.Draft, error) {
	var res domain.Draft
	err := tx.WithContext(ctx).Raw(`UPDATE drafts SET `+field+` = ? WHERE id = ? RETURNING *`, value, id).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not update draft by id: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

func (s *Services) CompleteDraft(ctx context.Context, draftId string) (*domain.Draft, error) {
	// start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	draft, err := s.updateDraftField(ctx, tx, "status", constants.GeneralStatusCompleted, draftId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	_, err = s.updateSaleField(ctx, tx, "status", constants.GeneralStatusPending, draft.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return draft, nil
}

func (s *Services) GetDraftById(ctx context.Context, id string) (*domain.Draft, error) {
	var draft domain.Draft
	err := s.db.
		WithContext(ctx).
		Preload("Customer").
		Preload("Store").
		Preload("Employee").
		First(&draft, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get draft by id: %v", err)
		return nil, domain.InternalServerError
	}

	cartItems, err := s.getCartItemWithProducts(ctx, s.db, draft.SaleId)
	if err != nil {
		return nil, err
	}

	cartItemTotal, err := s.GetCartItemsTotalAmount(ctx, s.db, draft.SaleId)
	if err != nil {
		return nil, err
	}

	draft.CartItems = cartItems
	draft.TotalPrice = cartItemTotal.TotalAmount

	return &draft, nil
}

func (s *Services) GetDrafts(ctx context.Context, params *domain.DraftQueryParams) ([]domain.Draft, int64, error) {
	// Base query with joins and aggregate fields
	query := s.db.
		WithContext(ctx).
		Table("drafts d").
		Preload("Store").
		Preload("Customer").
		Preload("Employee")

	// Filters
	query = query.Where("d.status = ?", constants.GeneralStatusPending)

	if params.Search != "" {
		query = query.Where("CAST(d.draft_number AS TEXT) LIKE ? ", "%"+params.Search+"%")
	}
	if params.StoreId != "" {
		query = query.Where("d.store_id = ?", params.StoreId)
	}
	if params.DraftDate != "" {
		// Validate the date format
		if _, err := time.Parse("2006-01-02", params.DraftDate); err != nil {
			return nil, 0, domain.InvalidQueryError
		}
		query = query.Where("d.draft_time::date = ?", params.DraftDate)
	}
	if params.CustomerId != "" {
		query = query.Where("d.customer_id = ?", params.CustomerId)
	}

	var totalCount int64
	if err := query.Model(&domain.Draft{}).Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Draft
	// Execute the query
	err := query.
		Select(
			"d.id",
			"d.draft_number",
			"d.store_id",
			"d.sale_id",
			"d.customer_id",
			"d.created_by",
			"d.description",
			"d.draft_time",
			"d.status",
			"d.created_at",
			"d.updated_at",
			"COALESCE(SUM(ci.quantity), 0) AS quantity",
			"COALESCE(SUM(ci.total_price), 0) AS total_price",
		).
		Joins("LEFT JOIN cart_items ci ON d.sale_id = ci.sale_id").
		Group("d.id").
		Limit(params.Limit).
		Offset(params.Offset).
		Order("d.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get drafts: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

func (s *Services) DeleteDraft(ctx context.Context, draftId string) error {
	draft, err := s.getFetchDraftById(ctx, draftId)
	if err != nil {
		return err
	}

	cartItems, err := s.getCartItemsBySaleId(ctx, draft.SaleId)
	if err != nil {
		return err
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range cartItems {
		err = s.IncrementQuantity(tx, item.StoreProductId, item.UnitQuantity)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	err = s.deleteCartItemsBySaleId(ctx, tx, draft.SaleId)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = s.deleteDraftById(ctx, tx, draftId)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

func (s *Services) getFetchDraftById(ctx context.Context, draftId string) (*domain.Draft, error) {
	var draft domain.Draft
	err := s.db.WithContext(ctx).First(&draft, "id = ?", draftId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.InternalServerError
		}
		s.log.Errorf("could not get draft by id error: %v", err)
		return nil, domain.InternalServerError
	}
	return &draft, nil
}

func (s *Services) getCartItemsBySaleId(ctx context.Context, saleId string) ([]domain.CartItem, error) {
	var cartItems []domain.CartItem
	err := s.db.
		WithContext(ctx).
		Model(&domain.CartItem{}).
		Select(
			"id",
			"sale_id",
			"store_product_id",
			"unit_quantity",
			"unit_price",
			"total_price",
		).
		Where("sale_id = ?", saleId).
		Find(&cartItems).Error
	if err != nil {
		s.log.Errorf("could not get cart_items by sale(%s) error: %v", saleId, err)
		return nil, domain.InternalServerError
	}

	return cartItems, nil
}

func (s *Services) deleteCartItemsBySaleId(ctx context.Context, tx *gorm.DB, saleId string) error {
	err := tx.WithContext(ctx).Delete(&domain.CartItem{}, "sale_id = ?", saleId).Error
	if err != nil {
		s.log.Errorf("could not delete cart_items by sale(%s) error: %v", saleId, err)
		return domain.InternalServerError
	}
	return nil
}

func (s *Services) deleteDraftById(ctx context.Context, tx *gorm.DB, draftId string) error {
	err := tx.WithContext(ctx).Delete(&domain.Draft{}, "id = ?", draftId).Error
	if err != nil {
		s.log.Errorf("could not delete draft by id error: %v", err)
		return domain.InternalServerError
	}
	return nil
}
