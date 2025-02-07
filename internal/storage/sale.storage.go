package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// Create sale payment
func (s *Storage) CreateSalePayment(tx *gorm.DB, req domain.FinalSale, item domain.FinalPaymentType, paymentServiceId *string, status string) (*domain.SalePayment, error) {
	var now = time.Now()
	salePayment := domain.SalePayment{}
	// Insert sale payments
	err := tx.Raw(`
	INSERT INTO sale_payments(
		id, sale_id, cash_box_operation_id, 
		payment_service_id, payment_type_id, 
		amount, paid_at, status) 
	VALUES(?, ?, ?, ?, ?, ?, ?, ?) RETURNING *`,
		uuid.New().String(), req.SaleID, req.CashBoxOperationId,
		paymentServiceId, item.PaymentTypeID, item.Amount, now, status).
		Scan(&salePayment).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	return &salePayment, nil
}

// Get Payment service with store id and payment type  if status is active
func (s *Storage) GetPaymentServiceByStoreId(storeId string, paymentType string) (*domain.PaymentService, error) {
	var res domain.PaymentService
	err := s.db.Raw(`SELECT * FROM payment_services WHERE store_id = ? AND type = ? AND is_active = true`,
		storeId, paymentType).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// update sale payment status
func (s *Storage) UpdateSalePaymentStatus(tx *gorm.DB, salePaymentID string) error {
	err := tx.Exec(`UPDATE sale_payments SET status = 'paid' WHERE id = ?`, salePaymentID).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}

// Create or update sale payment summary with on conflict do update
func (s *Storage) CreateOrUpdateSalePaymentSummary(tx *gorm.DB, cashBoxOperationId string, paymentTypeId string, amount float64) error {
	err := tx.Exec(`
				INSERT INTO sale_payment_summary (
					cash_box_operation_id, payment_type_id, total_amount
					) 
				VALUES (?, ?, ?)
				ON CONFLICT (cash_box_operation_id, payment_type_id) 
				DO UPDATE SET total_amount = EXCLUDED.total_amount + ?`, cashBoxOperationId, paymentTypeId, amount, amount).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}
