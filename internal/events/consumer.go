package events

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const maxRetry = 3

func SubscribeWithRetry(
	nc *nats.Conn,
	subject string,
	handler func([]byte) error,
) {
	_, _ = nc.Subscribe(subject, func(m *nats.Msg) {
		for attempt := 1; attempt <= maxRetry; attempt++ {
			if err := handler(m.Data); err == nil {
				return
			}
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		log.Println("🚨 DLQ:", subject)
		_ = nc.Publish(subject+".DLQ", m.Data)
	})
}
