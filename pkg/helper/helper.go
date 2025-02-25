package helper


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
