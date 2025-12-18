package socket

type Message struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type Client struct {
	send chan Message
}

func NewClient() *Client {
	return &Client{
		send: make(chan Message, 16),
	}
}
