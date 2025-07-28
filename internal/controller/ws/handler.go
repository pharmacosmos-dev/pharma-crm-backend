package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func ServeWs(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		storeID := c.Query("store_id")
		if storeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "store_id is required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := &Client{
			Hub:     hub,
			Conn:    conn,
			Send:    make(chan []byte, 1024),
			StoreID: storeID,
		}

		hub.register <- client

		go client.Write()
		go client.Read()
		c.JSON(http.StatusSwitchingProtocols, gin.H{"message": "WebSocket connection established"})
	}
}
