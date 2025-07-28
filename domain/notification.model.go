package domain

type CreateNotificationDto struct {
	ContentUz string `json:"content_uz"`
	ContentRu string `json:"content_ru"`
	ContentEn string `json:"content_en"`
	HeaderUz  string `json:"header_uz"`
	HeaderRu  string `json:"header_ru"`
	HeaderEn  string `json:"header_en"`
	UserId    string `json:"user_id,omitempty"`
	MessageId string `json:"message_id,omitempty"`
	OrderId   string `json:"order_id,omitempty"`
	StoreId   string `json:"store_id,omitempty"`
}
