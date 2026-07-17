package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// customerBalanceState holds customer balance and percent returned by UPDATE ... RETURNING
type customerBalanceState struct {
	Balance            float64 `gorm:"column:balance"`
	LoyaltyCardPercent int     `gorm:"column:loyalty_card_percent"`
}

// createLoyaltyCardTransaction stores one loyalty card balance movement (in|out) of a customer.
// total_sale_amount is calculated from cart_items the same way as sales.total_amount.
func (s *Services) createLoyaltyCardTransaction(ctx context.Context, req *domain.LoyaltyCardTransaction) {
	err := s.db.WithContext(ctx).Exec(`
	INSERT INTO loyalty_card_transactions (
		sale_id, customer_id, "type", percent, total_sale_amount,
		old_balance_amount, bonus_in_amount, bonus_out_amount, new_balance_amount
	) VALUES (
		?, ?, ?, ?,
		(SELECT COALESCE(SUM(total_price) - SUM(discount_amount), 0) FROM cart_items WHERE sale_id = ?),
		?, ?, ?, ?
	)`,
		req.SaleId,
		req.CustomerId,
		req.Type,
		req.Percent,
		req.SaleId,
		req.OldBalanceAmount,
		req.BonusInAmount,
		req.BonusOutAmount,
		req.NewBalanceAmount,
	).Error
	if err != nil {
		s.log.Errorf("could not create loyalty_card_transaction (sale: %s, type: %s): %v", req.SaleId, req.Type, err)
	}
}

func (s *Services) loyaltyCardTransactionListQuery(ctx context.Context, req *domain.LoyaltyCardTransactionListRequest) *gorm.DB {
	qb := s.db.WithContext(ctx).
		Table("loyalty_card_transactions t").
		Joins("JOIN customers c ON c.id = t.customer_id")

	if req.StartDate != nil && !req.StartDate.GetTime().IsZero() {
		qb = qb.Where("t.created_at >= ?", req.StartDate.UTC())
	}
	if req.EndDate != nil && !req.EndDate.GetTime().IsZero() {
		qb = qb.Where("t.created_at <= ?", req.EndDate.UTC())
	}
	if req.Type != "" {
		qb = qb.Where("t.type = ?", req.Type)
	}
	if req.CustomerId != "" {
		qb = qb.Where("t.customer_id = ?", req.CustomerId)
	}
	if req.SaleId != "" {
		qb = qb.Where("t.sale_id = ?", req.SaleId)
	}
	if req.Search != "" {
		search := "%" + req.Search + "%"
		qb = qb.Where("(c.full_name ILIKE ? OR c.phone ILIKE ? OR c.loyalty_card_barcode ILIKE ?)", search, search, search)
	}

	return qb
}

func (s *Services) GetLoyaltyCardTransactions(
	ctx context.Context,
	req *domain.LoyaltyCardTransactionListRequest,
) ([]domain.LoyaltyCardTransactionListItem, int64, error) {
	var (
		items []domain.LoyaltyCardTransactionListItem
		count int64
	)

	if err := s.loyaltyCardTransactionListQuery(ctx, req).Count(&count).Error; err != nil {
		s.log.Errorf("could not count loyalty_card_transactions: %v", err)
		return nil, 0, domain.InternalServerError
	}

	err := s.loyaltyCardTransactionListQuery(ctx, req).
		Joins("JOIN sales sl ON sl.id = t.sale_id").
		Select(
			"t.id",
			"t.sale_id",
			"sl.sale_number",
			"t.customer_id",
			"c.public_id AS customer_public_id",
			"c.full_name AS customer_name",
			"c.phone AS customer_phone",
			"c.loyalty_card_barcode",
			"t.type",
			"t.percent",
			"t.total_sale_amount",
			"t.old_balance_amount",
			"t.bonus_in_amount",
			"t.bonus_out_amount",
			"t.new_balance_amount",
			"t.created_at",
		).
		Order("t.created_at DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&items).Error
	if err != nil {
		s.log.Errorf("could not get loyalty_card_transactions: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return items, count, nil
}
