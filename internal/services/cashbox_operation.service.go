package services

import (
	"context"
	"errors"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
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

func (s *Services) GetOpenCashboxOperationByStoreId(ctx context.Context, storeId string) (*domain.CashboxOperationDto, error) {
	query := `
	SELECT
		co.id,
		co.operation_id,
		co.cash_box_id,
		co.employee_id,
		co.current_employee_id,
		co.is_open,
		co.start_time,
		co.end_time,
		co.created_at,
		co.updated_at
	FROM cashbox_operations co
	JOIN cash_boxes cb ON co.cash_box_id = cb.id
	WHERE cb.store_id = ?
	 AND co.is_open = true
	 AND co.end_time IS NULL
	ORDER BY co.created_at DESC LIMIT 1;
	`
	var res domain.CashboxOperationDto
	err := s.db.WithContext(ctx).
		Raw(query, storeId).
		Take(&res).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NoOpenCashboxError
		}
		s.log.Errorf("could not get open cashbox_operation by store_id: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) GetOpenCashboxOperationByEmployeeId(ctx context.Context, employeeId string) (*domain.CashboxOperationDto, error) {
	query := `
	SELECT
		co.id,
		co.operation_id,
		co.cash_box_id,
		co.employee_id,
		co.current_employee_id,
		co.is_open,
		co.start_time,
		co.end_time,
		co.created_at,
		co.updated_at
	FROM cashbox_operations co
	JOIN cash_boxes cb ON co.cash_box_id = cb.id
	WHERE co.current_employee_id = ?
	AND co.is_open = TRUE
	AND co.end_time IS NULL
	ORDER BY co.created_at DESC LIMIT 1;
	`
	var res domain.CashboxOperationDto
	err := s.db.WithContext(ctx).
		Raw(query, employeeId).
		Take(&res).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NoOpenCashboxError
		}
		s.log.Errorf("could not get open cashbox_operation by employee_id: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}
