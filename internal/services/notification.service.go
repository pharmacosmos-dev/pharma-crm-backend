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

func (s *Services) NotifyImportCreated(storeId string, documentNumber string) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Yangi import yaratildi: %s", documentNumber),
		ContentRu: fmt.Sprintf("Создан новый импорт: %s", documentNumber),
		ContentEn: fmt.Sprintf("New import created: %s", documentNumber),
		HeaderUz:  "Yangi import",
		HeaderRu:  "Новый импорт",
		HeaderEn:  "New import",
		StoreId:   storeId,
	}

	s.hub.SendMessage(ws.Message{
		StoreId: storeId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventImportCreated,
			Data:  notification,
		},
	})
}

func (s *Services) NotifyTransferChecking(storeId string, transferName string) {
	notification := domain.CreateNotificationDto{
		ContentUz: fmt.Sprintf("Transfer tekshirish uchun keldi: %s", transferName),
		ContentRu: fmt.Sprintf("Трансфер пришёл на проверку: %s", transferName),
		ContentEn: fmt.Sprintf("Transfer arrived for checking: %s", transferName),
		HeaderUz:  "Transfer keldi",
		HeaderRu:  "Трансфер получен",
		HeaderEn:  "Transfer received",
		StoreId:   storeId,
	}

	s.hub.SendMessage(ws.Message{
		StoreId: storeId,
		Payload: ws.OutgoingMessage{
			Event: constants.WsEventTransferChecking,
			Data:  notification,
		},
	})
}

// NotifyReminderCreated - eslatma yaratilganda tanlangan har bir aptekaga
// real vaqtda websocket orqali xabar yuboradi. Frontend shu event orqali
// from_date - to_date oralig'ida ovozli eslatmani (masalan har 15 daqiqada) boshlashi mumkin.
func (s *Services) NotifyReminderCreated(reminder *domain.Reminder) {
	for _, storeId := range reminder.StoreIds {
		s.hub.SendMessage(ws.Message{
			StoreId: storeId,
			Payload: ws.OutgoingMessage{
				Event: constants.WsEventReminderCreated,
				Data:  reminder,
			},
		})
	}
}

func (s *Services) NotifyTransferSent(storeId string, transferName string) {
  notification := domain.CreateNotificationDto{
    ContentUz: fmt.Sprintf("Transfer jo'natildi: %s", transferName),
    ContentRu: fmt.Sprintf("Трансфер отправлен: %s", transferName),
    ContentEn: fmt.Sprintf("Transfer sent: %s", transferName),
    HeaderUz:  "Transfer jo'natildi",
    HeaderRu:  "Трансфер отправлен",
    HeaderEn:  "Transfer sent",
    StoreId:   storeId,
  }

  s.hub.SendMessage(ws.Message{
    StoreId: storeId,
    Payload: ws.OutgoingMessage{
      Event: constants.WsEventTransferSent,
      Data:  notification,
    },
  })
}

