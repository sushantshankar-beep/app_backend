package socket

import (
	"sync"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
	}
}

/* ================= ROOM MANAGEMENT ================= */

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

/* ================= BROADCAST ================= */

// ✅ THIS IS WHAT WAS MISSING
func (h *Hub) Broadcast(room string, message any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[room]
	if !ok {
		return
	}

	for c := range clients {
		select {
		case c.send <- message:
		default:
			// drop dead clients
			go h.Leave(room, c)
		}
	}
}
