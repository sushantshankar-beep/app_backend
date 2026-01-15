package socket

import "sync"

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Join(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][c] = true
}

func (h *Hub) Leave(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[room]; ok {
		delete(clients, c)
		close(c.send)
	}
}

func (h *Hub) Emit(room string, msg any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.rooms[room] {
		select {
		case c.send <- msg:
		default:
		}
	}
}
