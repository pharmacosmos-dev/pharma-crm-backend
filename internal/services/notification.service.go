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

func (s *Services) NotifyOnlineOrderAcceptCourier(StoreId string, orderDisplayId int) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Kuryer onlayn buyurtmani qabul qildi: %d", orderDisplayId),
		ContentRu: fmt.Sprintf("Курьер принял онлайн заказ: %d", orderDisplayId),
		ContentEn: fmt.Sprintf("Courier has accepted the online order: %d", orderDisplayId),

		HeaderUz: "Kuryer buyurtmani qabul qildi",
		HeaderRu: "Курьер принял заказ",
		HeaderEn: "Order accepted by courier",

		StoreId: StoreId,
		OrderId: fmt.Sprintf("%d", orderDisplayId),
	}

	s.hub.SendMessage(ws.Message{
		StoreId: StoreId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventNoorOrderAcceptCourier,
			Data:  notification,
		},
	})
}

func (s *Services) NotifyOnlineOrderUpdatedStatus(StoreId string, orderDisplayId int) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Onlayn buyurtma statusi yangilandi: %d", orderDisplayId),
		ContentRu: fmt.Sprintf("Статус онлайн заказа обновлен: %d", orderDisplayId),
		ContentEn: fmt.Sprintf("Online order status updated: %d", orderDisplayId),

		HeaderUz: "Onlayn buyurtma statusi yangilandi",
		HeaderRu: "Статус онлайн заказа обновлен",
		HeaderEn: "Online order status updated",

		StoreId: StoreId,
		OrderId: fmt.Sprintf("%d", orderDisplayId),
	}

	s.hub.SendMessage(ws.Message{
		StoreId: StoreId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventNoorOrderAcceptCourier,
			Data:  notification,
		},
	})
}
