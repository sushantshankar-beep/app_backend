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

func (e *Emitter) Emit(room, event string, data any) {
	e.hub.Broadcast(room, map[string]any{
		"event": event,
		"data":  data,
	})
	log.Printf("📡 socket emit %s -> %s\n", room, event)
}
func (e *Emitter) EmitWithRetry(
	room string,
	event string,
	data any,
	retries int,
) {
	for i := 0; i < retries; i++ {
		e.Emit(room, event, data)

		// small delay to avoid burst duplicates
		time.Sleep(150 * time.Millisecond)
	}
}