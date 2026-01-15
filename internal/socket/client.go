package socket

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan any
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan any, 256),
	}
}

func (c *Client) ReadPump(hub *Hub, room string) {
	defer func() {
		hub.Leave(room, c)
		c.conn.Close()
	}()

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			log.Println("socket write error:", err)
			break
		}
	}
}
