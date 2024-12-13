package payment

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
)

type PaymentAction struct {
	Cfg *config.Config
	Log *logger.Logger
}

type PaymentService interface {
	ClickPass(ctx context.Context, req *domain.ClickPassRequest) (*domain.ClickPassResponse, error)
	ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}) (*domain.ClickPassResponse, error)
}

func (h *PaymentAction) ClickPass(ctx context.Context, req *domain.ClickPassRequest) (*domain.ClickPassResponse, error) {
	
	return nil, nil
}


