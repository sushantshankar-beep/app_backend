package socket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all (prod: restrict)
	},
}

func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		room := c.Query("room")
		if room == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := NewClient()
		hub.Join(room, client)

		// READ loop (just keep connection alive)
		go func() {
			defer func() {
				hub.Leave(room, client)
				conn.Close()
			}()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()

		// WRITE loop
		for msg := range client.send {
			if err := conn.WriteJSON(msg); err != nil {
				break
			}
		}
	}
}
