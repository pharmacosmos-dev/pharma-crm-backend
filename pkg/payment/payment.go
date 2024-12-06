package payment

import (
	"context"

	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/logger"
)

type PaymentAction struct {
	Cfg *config.Config
	Log *logger.Logger
}

type PaymentService interface {
	ClickPass(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
	ClickCheckPaymentStatus(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
}
