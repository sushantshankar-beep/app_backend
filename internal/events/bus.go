package events

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type Bus struct {
	nc *nats.Conn
}

func NewBus(url string) *Bus {
	nc, err := nats.Connect(url)
	if err != nil {
		log.Fatal("❌ NATS connection failed:", err)
	}
	return &Bus{nc: nc}
}

func (b *Bus) Publish(subject string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}
	_ = b.nc.Publish(subject, data)
}

func (b *Bus) Conn() *nats.Conn {
	return b.nc
}
