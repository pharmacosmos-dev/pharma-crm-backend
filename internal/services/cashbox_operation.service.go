package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

func (s *Services) GetCashboxOperationSummary(ctx context.Context, operationId string) (*domain.CashboxOperationSummary, error) {

	var saleStats domain.SaleStats
	query := `
	SELECT
		SUM(s.total_amount) AS amount,
		SUM(s.cash) AS total_cash,
		SUM(s.humo) AS total_humo,
		SUM(s.uzcard) AS total_uzcard,
		SUM(s.click) AS total_click,
		SUM(s.payme) AS total_payme,
		SUM(s.alif) AS total_alif
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
			Amount: saleStats.TotalCash,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Humo",
			Amount: saleStats.TotalHumo,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Uzcard",
			Amount: saleStats.TotalUzcard,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Click",
			Amount: saleStats.TotalClick,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Payme",
			Amount: saleStats.TotalPayme,
		},
		domain.SalePaymentCloseCashBox{
			Id:     "",
			Name:   "Alif",
			Amount: saleStats.TotalAlif,
		},
	)

	var res domain.CashboxOperationSummary
	res.PaymentTypeSum = typeSums
	res.TotalSum.TotalAmount = saleStats.TotalTransactionsSum

	return &res, nil
}
