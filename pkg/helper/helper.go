package helper

import (
	"fmt"
	"strings"

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

func SaleTypeToRussian(saleType string, saleNumber int) string {
	if saleType == "RETURN" {
		return fmt.Sprintf("Возврат #%d", saleNumber)
	}
	return fmt.Sprintf("Продажа #%d", saleNumber)

}

// return amount by checking sale payment
func SalePaymentAmount(salePayments []*domain.SalePayment, payType string) float64 {
	var amount float64
	for _, payment := range salePayments {
		if payment.PaymentType != nil {
			if payType == payment.PaymentType.Name {
				amount = payment.Amount
			}
		}
	}
	return amount
}

// check user role is admin or superadmin
func IsAdmin(employee domain.Employee, cfg *config.Config) bool {
	role := employee.RoleType
	if role != config.ADMIN && role != config.SUPERADMIN && role != config.FOUNDER && role != config.ACCOUNTANT && role != config.AUTOZAKAZ && role != config.DIRECTOR {
		return false
	}
	return true
}

// divide float to integer and fractional section
func SplitFloatParts(number float64) (intPart int, fracPart int) {
	str := fmt.Sprintf("%f", number)  // convert to string with decimals
	str = strings.TrimRight(str, "0") // remove trailing zeros
	parts := strings.Split(str, ".")  // split into integer and fractional parts

	intPart = 0
	fracPart = 0

	if len(parts) > 0 {
		fmt.Sscanf(parts[0], "%d", &intPart)
	}
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &fracPart)
	}
	return
}