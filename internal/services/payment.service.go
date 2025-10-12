package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

func (s *Services) Payment(
	ctx context.Context,
	tx *gorm.DB,
	sale *domain.Sale,
	fiscal *domain.FiscalData,
) error {

	if sale.Click > 0 {
		payService, err := s.GetPaymentServiceByStoreId(ctx, tx, sale.StoreId, constants.PaymentTypeClick)
		if err != nil {
			return err
		}
		_, err = s.ClickPass(ctx, payService, sale)
		if err != nil {
			return err
		}
		return nil
	}

	if sale.Payme > 0 {
		payService, err := s.GetPaymentServiceByStoreId(ctx, tx, sale.StoreId, constants.PaymentTypePayme)
		if err != nil {
			return err
		}
		token := s.generatePaymeToken(payService)

		_, err = s.CreateReceiptAndPay(ctx, sale, fiscal, token)
		if err != nil {
			return err
		}
		return nil
	}

	if sale.Alif > 0 {
		payService, err := s.GetPaymentServiceByStoreId(ctx, tx, sale.StoreId, constants.PaymentTypeAlif)
		if err != nil {
			return err
		}
		_, err = s.AlifPay(ctx, payService, sale)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

// Save payment request to database
func (h *Services) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	query := `
	INSERT INTO payment_requests (
		request_id, 
		method, 
		payload, 
		transaction_id, 
		payment_provider
		)
		VALUES (?, ?, ?, ?, ?)`
	err := h.db.WithContext(ctx).Exec(
		query,
		req.RequestId,
		req.Method,
		req.Payload,
		req.TransactionID,
		req.PaymentProvider).Error
	if err != nil {
		h.log.Warn("ERROR on saving payment request: %v", err)
		return err
	}
	return nil
}

// Save payment response to database
func (h *Services) SaveResponse(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.Exec(
		`UPDATE 
			payment_requests 
		SET 
			response = ? 
		WHERE 
			transaction_id = ? AND 
			method = ?`,
		req.Response,
		req.TransactionID,
		req.Method,
	).Error
	if err != nil {
		h.log.Warn("ERROR on saving payment response: %v", err)
		return err
	}
	return nil
}
