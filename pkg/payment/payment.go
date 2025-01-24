package payment

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type PaymentAction struct {
	cfg *config.Config
	log *logger.Logger
	db  *gorm.DB
}

type PaymentService interface {
	ClickPass(ctx context.Context, click *domain.PaymentService, data *domain.FinalSale) (*domain.ClickPassResponse, error)
	ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}, token string) (*domain.ClickPassResponse, error)
}

func NewPaymentAction(cfg *config.Config, log *logger.Logger, db *gorm.DB) PaymentService {
	return &PaymentAction{
		cfg, log, db,
	}
}

// ClickPass implements PaymentService
func (h *PaymentAction) ClickPass(ctx context.Context, click *domain.PaymentService, data *domain.FinalSale) (*domain.ClickPassResponse, error) {
	var ()
	tr := h.db.Begin()
	defer func() {
		if err := recover(); err != nil {
			tr.Rollback()
		}
	}()
	// transaction structure
	transaction := domain.Transaction{
		ID:               uuid.New().String(),
		SalePaymentID:    click.ID,
		PaymentServiceID: click.ID,
		TransactionID:    uuid.New().String(),
		Status:           "pending",
		ResponseData:     "",
	}

	clickData := domain.ClickPassRequest{
		ServiceID:     click.ServiceID,
		OtpData:       data.App.OtpData,
		CashboxCode:   data.CashBoxId,
		Amount:        data.App.Amount,
		TransactionID: transaction.ID,
	}
	// Marshal click pass request
	t, _ := json.Marshal(clickData)
	// Save request of one click pass data
	err := h.SaveRequest(ctx, &domain.PaymentRequest{
		Method:          "click_pass",
		Payload:         string(t),
		TransactionID:   transaction.ID,
		PaymentProvider: "click",
	})
	if err != nil {
		return nil, err
	}
	// generate click pass auth token
	token := h.generateClickAuthToken(click.SecretKey, click.MerchantUserID)
	// send request to click pass
	res, err := h.ClickPassDoRequest(ctx, "/click_pass/payment", clickData, token)
	if err != nil {
		return nil, err
	}
	// convert to json response of click pass
	t, _ = json.Marshal(res)
	// save response to database
	err = h.SaveResponse(ctx, &domain.PaymentRequest{
		TransactionID: transaction.ID,
		Response:      string(t),
	})
	if err != nil {
		return nil, err
	}

	if res.ErrorCode == 0 {
		res, err = h.ClickCheckPaymentStatus(ctx, map[string]interface{}{
			"service_id": click.ServiceID,
			"payment_id": res.PaymentID,
		}, token)
		if err != nil {
			return nil, err
		}
	}

	transaction.ResponseData = string(t)

	err = tr.Create(&transaction).Error
	if err != nil {
		tr.Rollback()
		return nil, err
	}
	if err = tr.Commit().Error; err != nil {
		tr.Rollback()
		return nil, err
	}
	return res, nil
}

// Check click pass payment status
func (h *PaymentAction) ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}, token string) (*domain.ClickPassResponse, error) {
	url := fmt.Sprintf("/payment/status/%v/%v", data["service_id"], data["payment_id"])
	res, err := h.ClickPassDoRequest(ctx, h.cfg.ClickEndpointUrl+url, data, token)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *PaymentAction) SaveRequest(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Create(&req).Error
	if err != nil {
		return err
	}
	return nil
}

func (h *PaymentAction) SaveResponse(ctx context.Context, req *domain.PaymentRequest) error {
	err := h.db.WithContext(ctx).Raw(
		`UPDATE payment_requests SET response = ? WHERE transaction_id = ?`,
		req.Response, req.TransactionID,
	).Error
	if err != nil {
		return err
	}
	return nil
}

func (h *PaymentAction) generateClickAuthToken(secretKey string, merchantUserId int) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	digest := sha1.Sum([]byte(timestamp + secretKey))
	digestStr := fmt.Sprintf("%x", digest)
	return fmt.Sprintf("%d:%s:%s", merchantUserId, digestStr, timestamp)
}
