package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/internal/controller/ws"
)

func (s *Services) NotifyOnlineOrder(StoreID string, orderDisplayId int) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Online buyurtma: %d", orderDisplayId),
		ContentRu: fmt.Sprintf("Онлайн заказ: %d", orderDisplayId),
		ContentEn: fmt.Sprintf("Online order: %d", orderDisplayId),
		HeaderUz:  "Yangi onlayn buyurtma",
		HeaderRu:  "Новый онлайн заказ",
		HeaderEn:  "New online order",
		StoreId:   StoreID,
		OrderId:   fmt.Sprintf("%d", orderDisplayId),
	}

	s.hub.SendMessage(ws.Message{
		StoreID: StoreID,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventNoorOrder,
			Data:  notification,
		},
	})
}
