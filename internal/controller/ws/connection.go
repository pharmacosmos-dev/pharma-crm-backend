package ws

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	Send    chan []byte
	StoreID string
}

type OutgoingMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type Message struct {
	StoreId string
	Payload OutgoingMessage
}

func (c *Client) Read() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) Write() {
	defer c.Conn.Close()
	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
}
