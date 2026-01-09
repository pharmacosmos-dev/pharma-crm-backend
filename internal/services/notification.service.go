package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/internal/controller/ws"
)

func (s *Services) NotifyOnlineOrder(StoreId string, orderDisplayId int) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Online buyurtma: %d", orderDisplayId),
		ContentRu: fmt.Sprintf("Онлайн заказ: %d", orderDisplayId),
		ContentEn: fmt.Sprintf("Online order: %d", orderDisplayId),
		HeaderUz:  "Yangi onlayn buyurtma",
		HeaderRu:  "Новый онлайн заказ",
		HeaderEn:  "New online order",
		StoreId:   StoreId,
		OrderId:   fmt.Sprintf("%d", orderDisplayId),
	}

	s.hub.SendMessage(ws.Message{
		StoreId: StoreId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventNoorOrder,
			Data:  notification,
		},
	})
}

func (s *Services) NotifyOnlineOrderCancel(StoreId string, orderDisplayId int) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Onlayn buyurtma bekor qilindi: %d", orderDisplayId),
		ContentRu: fmt.Sprintf("Онлайн заказ отменён: %d", orderDisplayId),
		ContentEn: fmt.Sprintf("Online order cancelled: %d", orderDisplayId),

		HeaderUz: "Onlayn buyurtma bekor qilindi",
		HeaderRu: "Онлайн заказ отменён",
		HeaderEn: "Online order cancelled",

		StoreId: StoreId,
		OrderId: fmt.Sprintf("%d", orderDisplayId),
	}

	s.hub.SendMessage(ws.Message{
		StoreId: StoreId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventNoorOrderCancel,
			Data:  notification,
		},
	})
}
