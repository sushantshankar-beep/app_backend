package socket

import (
	"log"
	"time"
)

type Emitter struct {
	hub *Hub
}

func NewEmitter(hub *Hub) *Emitter {
	return &Emitter{hub: hub}
}

// Emit sends immediately (non-blocking).
func (e *Emitter) Emit(room, event string, data any) {

	msg := map[string]any{
		"event": event,
		"data":  data,
	}

	// ⚠️ comment out in production if noisy
	log.Printf("📡 socket emit room=%s event=%s\n", room, event)

	e.hub.Broadcast(room, msg)
}

// CloseRoom forces all clients in room to disconnect.
func (e *Emitter) CloseRoom(room string) {
	log.Printf("🚪 socket closing room=%s\n", room)
	e.hub.CloseRoom(room)
}

// EmitWithRetry fires instantly and retries asynchronously.
// DOES NOT block bidding fan-out.
func (e *Emitter) EmitWithRetry(
	room string,
	event string,
	data any,
	retries int,
) {

	// 🚀 first emit immediately
	e.Emit(room, event, data)

	if retries <= 1 {
		return
	}

	// 🔁 async retries
	go func() {
		for i := 1; i < retries; i++ {
			time.Sleep(150 * time.Millisecond)

			log.Printf(
				"🔁 retry emit room=%s event=%s try=%d\n",
				room,
				event,
				i+1,
			)

			e.Emit(room, event, data)
		}
	}()
}
