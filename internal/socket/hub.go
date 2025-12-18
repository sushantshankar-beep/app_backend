package socket

import "sync"

type Hub struct {
	rooms map[string]map[*Client]bool
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Join(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][c] = true
}

func (h *Hub) Leave(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[room]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) Emit(room string, event string, payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.rooms[room] {
		select {
		case c.send <- Message{Event: event, Data: payload}:
		default:
			// drop if client is slow (FAST)
		}
	}
}
