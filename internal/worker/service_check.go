package worker

import (
	"context"
	"log"
	"time"

	"app_backend/internal/ports"
)

type ProviderTimeoutWorker struct {
	repo     ports.AcceptedServiceRepository
	handler ports.ProviderTimeoutHandler

	interval time.Duration
	timeout  time.Duration
}

func NewProviderTimeoutWorker(
	repo ports.AcceptedServiceRepository,
	handler ports.ProviderTimeoutHandler,
) *ProviderTimeoutWorker {

	return &ProviderTimeoutWorker{
		repo:     repo,
		handler: handler,
		timeout:  10 * time.Minute,
		interval: 1 * time.Minute,
	}
}

func (w *ProviderTimeoutWorker) Start() {

	go func() {

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for range ticker.C {
			w.scan()
		}

	}()
}

func (w *ProviderTimeoutWorker) scan() {

	ctx := context.Background()

	cutoff := time.Now().Add(-w.timeout)

	services, err := w.repo.FindStuckAssigned(ctx, cutoff)
	if err != nil {
		log.Println("⛔ timeout scan failed:", err)
		return
	}

	for _, svc := range services {

		log.Println("⏱ provider timeout:", svc.ID.Hex())

		go w.handler.HandleProviderTimeout(ctx, svc)
	}
}
