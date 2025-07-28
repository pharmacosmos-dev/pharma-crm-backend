package ws

import (
	"encoding/json"
	"log"
)

type Hub struct {
	clients    map[string]map[*Client]bool // by StoreID
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.StoreID] == nil {
				h.clients[client.StoreID] = make(map[*Client]bool)
			}
			h.clients[client.StoreID][client] = true

		case client := <-h.unregister:
			if clients, ok := h.clients[client.StoreID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
				}
			}

		case message := <-h.broadcast:
			if clients, ok := h.clients[message.StoreID]; ok {
				jsonData, err := json.Marshal(message.Payload)
				if err != nil {
					log.Printf("error marshalling ws message: %v", err)
					continue
				}
				for client := range clients {
					client.Send <- jsonData
				}
			}
		}
	}
}

func (h *Hub) SendMessage(req Message) {
	h.broadcast <- req
}
