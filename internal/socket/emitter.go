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
	for i := 0; i < retries; i++ {
		log.Printf("📡 Socket emit [%d] → %s", i+1, event)
		e.hub.Emit(room, event, payload)
		time.Sleep(200 * time.Millisecond)
	}
}
