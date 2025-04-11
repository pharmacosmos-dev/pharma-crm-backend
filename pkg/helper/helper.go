package helper

import (
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
)

// StatusToRussian converts a status string to a Russian translation
func StatusToRussian(status string) string {
	switch status {
	case "completed":
		return "Завершения"
	case "canceled":
		return "Отменен"
	case "pending":
		return "Ожидание"
	default:
		return "Новый"
	}
}

// return amount by checking sale payment
func SalePaymentAmount(salePayments []*domain.SalePayment, payType string) float64 {
	var amount float64
	for _, payment := range salePayments {
		if payType == payment.PaymentType.Type {
			amount = payment.Amount
		}
	}
	return amount
}

// check user role is admin or superadmin
func IsAdmin(employee domain.Employee, cfg *config.Config) bool {
	role := employee.RoleType
	if role != config.ADMIN && role != config.SUPERADMIN && role != config.FOUNDER && role != config.ACCOUNTANT && role != config.AUTOZAKAZ {
		return false
	}
	return true
}
