package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetCashboxOperationSummary(ctx context.Context, operationId string) (*domain.CashboxOperationSummary, error) {

	var saleStats domain.SaleStats
	query := `
	SELECT
		SUM(s.total_amount) AS total_transaction_sum,
		SUM(s.cash) AS total_cash_sum,
		SUM(s.humo) AS total_humo_sum,
		SUM(s.uzcard) AS total_uzcard_sum,
		SUM(s.click) AS total_click_sum,
		SUM(s.payme) AS total_payme_sum,
		SUM(s.alif) AS total_alif_sum
	FROM sales s
	WHERE s.cash_box_operation_id = ? ORDER BY s.total_amount DESC;
	`
	err := s.db.WithContext(ctx).Raw(query, operationId).Scan(&saleStats).Error
	if err != nil {
		s.log.Errorf("could not get cashbox_operation summary: %v", err)
		return nil, domain.InternalServerError
	}

	var typeSums []domain.SalePaymentCloseCashBox

	typeSums = append(typeSums,
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Naqd",
			Amount: saleStats.TotalCashSum,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Humo",
			Amount: saleStats.TotalHumoSum,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Uzcard",
			Amount: saleStats.TotalUzcardSum,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Click",
			Amount: saleStats.TotalClickSum,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Payme",
			Amount: saleStats.TotalPaymeSum,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Alif",
			Amount: saleStats.TotalAlifSum,
		},
	)

	var res domain.CashboxOperationSummary
	res.PaymentTypeSum = typeSums
	res.TotalSum.TotalAmount = saleStats.TotalTransactionSum

	return &res, nil
}
