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

func (e *Emitter) EmitWithRetry(
	room string,
	event string,
	payload any,
	retries int,
) {
	msg := map[string]any{
		"event": event,
		"data":  payload,
	}

	for i := 0; i < retries; i++ {
		log.Printf("📡 socket emit %s (try %d)", event, i+1)
		e.hub.Emit(room, msg)
		time.Sleep(200 * time.Millisecond)
	}
}
