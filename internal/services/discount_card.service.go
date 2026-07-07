package services

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Services) GetDiscountCardByBarcode(ctx context.Context, barcode string) (int, error) {
	var percent int
	err := s.db.WithContext(ctx).Raw("SELECT discount_percent FROM customers WHERE discount_card = ?", barcode).Scan(&percent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, domain.NotFoundError
		}
		s.log.Errorf("could not get discount_card: %v", err)
		return 0, domain.InternalServerError
	}
	if percent == 0 {
		return 0, domain.NotFoundError
	}
	return percent, nil
}

func (s *Services) UpdateDiscountCard(ctx context.Context, req *domain.UpdateDiscountCardRequest) error {
	err := s.db.WithContext(ctx).Exec("UPDATE customers SET discount_percent = ? WHERE id = ?", req.Percent, req.Id).Error
	if err != nil {
		s.log.Errorf("could not update customer discount_percent: %v", err)
		return domain.InternalServerError
	}
	return nil
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

func (s *Services) CreateSaleCustomerDiscount(
	ctx context.Context,
	tx *gorm.DB,
	req *domain.AddDiscountCard,
) (*domain.SaleCustomerDiscount, error) {
	var customerDiscount domain.SaleCustomerDiscount
	err := tx.WithContext(ctx).Raw(`
	INSERT INTO sale_customer_discounts(
		customer_id, 
		sale_id,
		discount_percent
		) 
	VALUES(?, ?, ?) RETURNING *`,
		req.CustomerId,
		req.SaleId,
		req.Percent,
	).Scan(&customerDiscount).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				// This — UNIQUE constraint violation
				return nil, domain.DuplicateError
			}
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
