package services

import (
	"context"
	"errors"
	"time"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Services) GetDiscountCardByBarcode(ctx context.Context, barcode string) (*domain.DiscountCard, error) {
	var res domain.DiscountCard
	err := s.db.First(&res, "barcode = ?", barcode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get discount_card: %v", err)
		return nil, domain.InternalServerError
	}
	return &res, nil
}

func (s *Services) UpdateDiscountCard(req *domain.UpdateDiscountCardRequest) error {
	var card = domain.UpdateDiscountCard{
		Percent:   req.Percent,
		UpdatedBy: req.UpdatedBy,
		UpdatedAt: time.Now(),
	}

	if req.CustomerID != nil {
		card.CustomerID = req.CustomerID
	}

	return s.db.
		Model(&domain.DiscountCard{}).
		Where("id = ? AND deleted_at IS NULL", req.ID).
		Updates(card).
		Error
}

func (s *Services) DeleteDiscountCard(id, deletedBy string) error {
	return s.db.
		Model(&domain.DiscountCard{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": deletedBy,
		}).
		Error
}

func (s *Services) CreateSaleCustomerDiscount(ctx context.Context, tx *gorm.DB, req *domain.AddDiscountCard, discountCard *domain.DiscountCard) (*domain.SaleCustomerDiscount, error) {
	var customerDiscount domain.SaleCustomerDiscount
	err := tx.WithContext(ctx).Raw(`
	INSERT INTO sale_customer_discounts(
		customer_id, 
		sale_id, 
		discount_card_id, 
		discount_percent
		) 
	VALUES(?, ?, ?, ?) RETURNING *`,
		req.CustomerId,
		req.SaleId,
		discountCard.ID,
		discountCard.Percent).Scan(&customerDiscount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, domain.DuplicateError
		}
		s.log.Errorf("could not creating sale customer discount: %v", err)
		return nil, domain.InternalServerError
	}

	return &customerDiscount, nil
}

func (s *Services) DeleteSaleCustomerDiscount(ctx context.Context, tx *gorm.DB, req *domain.AddDiscountCard) error {
	err := tx.WithContext(ctx).Exec(`DELETE FROM sale_customer_discounts WHERE sale_id = ?`, req.SaleId).Error
	if err != nil {
		s.log.Errorf("could not delete sale_customer_discount: %v", err)
		return domain.InternalServerError
	}
	return nil
}
