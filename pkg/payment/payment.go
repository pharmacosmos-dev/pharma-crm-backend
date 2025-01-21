package payment

import (
	"context"
	"net/http"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type PaymentAction struct {
	Cfg *config.Config
	Log *logger.Logger
}

type PaymentService interface {
	ClickPass(ctx context.Context, req *domain.ClickPassRequest) (*domain.ClickPassResponse, error)
	ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}) (*domain.ClickPassResponse, error)
}

func NewPaymentAction(cfg *config.Config, log *logger.Logger, db *gorm.DB) PaymentService {
	return &PaymentAction{
		Cfg: cfg,
		Log: log,
	}
}

func (h *PaymentAction) ClickPass(ctx context.Context, req *domain.ClickPassRequest) (*domain.ClickPassResponse, error) {
	
	return nil, nil
}

func (h *PaymentAction) ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}) (*domain.ClickPassResponse, error) {
	return nil, nil
}

func (h *PaymentAction) DoRequest(ctx context.Context, data map[string]interface{}) (*domain.ClickPassResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", "", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Auth", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}
