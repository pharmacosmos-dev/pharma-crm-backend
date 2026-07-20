package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// customerBalanceState holds the balance and loyalty percent returned by the
// UPDATE ... RETURNING queries in createLoyaltyCardTransaction.
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

// loyaltyCardTransactionQuery is the single source of truth for filtering
// loyalty_card_transactions. GetLoyaltyCardTransactions and
// GetLoyaltyCardTransactionDashboard both build on top of it, so they always
// report numbers for the exact same set of rows.
func (s *Services) loyaltyCardTransactionQuery(ctx context.Context, req *domain.LoyaltyCardTransactionListRequest) *gorm.DB {
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

	if err := s.loyaltyCardTransactionQuery(ctx, req).Count(&count).Error; err != nil {
		s.log.Errorf("could not count loyalty_card_transactions: %v", err)
		return nil, 0, domain.InternalServerError
	}

	err := s.loyaltyCardTransactionQuery(ctx, req).
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

// loyaltyCardMovementStats aggregates per-row numbers: how many in/out
// movements happened and how much money moved in each direction.
type loyaltyCardMovementStats struct {
	TotalInCount        int64   `gorm:"column:total_in_count"`
	TotalOutCount       int64   `gorm:"column:total_out_count"`
	TotalBonusInAmount  float64 `gorm:"column:total_bonus_in_amount"`
	TotalBonusOutAmount float64 `gorm:"column:total_bonus_out_amount"`
}

// getLoyaltyCardMovementStats counts in/out rows and sums their amounts.
// No deduplication needed here: an "in" row and an "out" row of the same sale
// are two distinct movements.
func (s *Services) getLoyaltyCardMovementStats(ctx context.Context, req *domain.LoyaltyCardTransactionListRequest) (*loyaltyCardMovementStats, error) {
	var stats loyaltyCardMovementStats

	err := s.loyaltyCardTransactionQuery(ctx, req).
		Select(`
			COUNT(*) FILTER (WHERE t.type = 'in')  AS total_in_count,
			COUNT(*) FILTER (WHERE t.type = 'out') AS total_out_count,
			COALESCE(SUM(t.bonus_in_amount), 0)    AS total_bonus_in_amount,
			COALESCE(SUM(t.bonus_out_amount), 0)   AS total_bonus_out_amount
		`).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// loyaltyCardSaleStats aggregates per-sale numbers: a sale with both an "in"
// row (cashback) and an "out" row (paid from balance) is counted once.
type loyaltyCardSaleStats struct {
	TotalCount         int64   `gorm:"column:total_count"`
	TotalSaleAmountSum float64 `gorm:"column:total_sale_amount_sum"`
}

// getLoyaltyCardSaleStats groups transactions by sale_id first, so a sale
// with two rows (in + out) is not double-counted.
func (s *Services) getLoyaltyCardSaleStats(ctx context.Context, req *domain.LoyaltyCardTransactionListRequest) (*loyaltyCardSaleStats, error) {
	distinctSales := s.loyaltyCardTransactionQuery(ctx, req).
		Group("t.sale_id").
		Select("MAX(t.total_sale_amount) AS total_sale_amount")

	var stats loyaltyCardSaleStats
	err := s.db.WithContext(ctx).
		Table("(?) AS sales", distinctSales).
		Select("COUNT(*) AS total_count, COALESCE(SUM(total_sale_amount), 0) AS total_sale_amount_sum").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetLoyaltyCardTransactionDashboard returns aggregated in/out stats for the
// same filters as GetLoyaltyCardTransactions (loyaltyCardTransactionQuery).
func (s *Services) GetLoyaltyCardTransactionDashboard(
	ctx context.Context,
	req *domain.LoyaltyCardTransactionListRequest,
) (*domain.LoyaltyCardTransactionDashboard, error) {
	movement, err := s.getLoyaltyCardMovementStats(ctx, req)
	if err != nil {
		s.log.Errorf("could not get loyalty_card_transactions movement stats: %v", err)
		return nil, domain.InternalServerError
	}

	sales, err := s.getLoyaltyCardSaleStats(ctx, req)
	if err != nil {
		s.log.Errorf("could not get loyalty_card_transactions sale stats: %v", err)
		return nil, domain.InternalServerError
	}

	return &domain.LoyaltyCardTransactionDashboard{
		TotalCount:          sales.TotalCount,
		TotalInCount:        movement.TotalInCount,
		TotalOutCount:       movement.TotalOutCount,
		TotalSaleAmountSum:  sales.TotalSaleAmountSum,
		TotalBonusInAmount:  movement.TotalBonusInAmount,
		TotalBonusOutAmount: movement.TotalBonusOutAmount,
	}, nil
}
