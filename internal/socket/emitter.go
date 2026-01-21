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
	msg := map[string]any{
		"event": event,
		"data":  data,
	}

	log.Printf("📡 socket emit room=%s event=%s\n", room, event)
	e.hub.Broadcast(room, msg)
}

func (e *Emitter) EmitWithRetry(
	room string,
	event string,
	data any,
	retries int,
) {
	for i := 0; i < retries; i++ {
		log.Printf("📡 socket emit room=%s event=%s try=%d\n", room, event, i+1)
		e.Emit(room, event, data)
		time.Sleep(150 * time.Millisecond)
	}
}
