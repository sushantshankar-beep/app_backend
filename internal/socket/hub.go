
package socket

import (
	"sync"
	"strings"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
	providerConn  map[string]*Client
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
		providerConn: make(map[string]*Client),
	}
}
func (h *Hub) Emit(room string, msg any){
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.rooms[room] {
		select {
			case c.send <- msg:
			default:
				go h.Leave(room, c)
			}
		}
}
/* ================= ROOM MANAGEMENT ================= */

func (h *Hub) Join(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 🔥 Ensure single connection per provider
	if strings.HasPrefix(room, "provider:") {

		// kick old connection if exists
		if old, ok := h.providerConn[room]; ok {
			delete(h.rooms[room], old)
			close(old.send)
		}

		h.providerConn[room] = c
	}

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

		if h.providerConn[room] == c {
			delete(h.providerConn, room)
		}

		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
}


/* ================= BROADCAST ================= */





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
			// message delivered
		default:
			// client is dead or blocked → remove it
			go h.Leave(room, c)
		}
	}
}
