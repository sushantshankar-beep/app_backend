package worker

import (
	"context"
	"log"
	"time"

	// "app_backend/internal/domain"
	"app_backend/internal/ports"
)

type SearchWatchdog struct {
	repo    ports.AcceptedServiceRepository
	cleaner ports.SearchingServiceCleaner
}

func NewSearchWatchdog(
	repo ports.AcceptedServiceRepository,
	cleaner ports.SearchingServiceCleaner,
) *SearchWatchdog {

	return &SearchWatchdog{
		repo:    repo,
		cleaner: cleaner,
	}
}

func (w *SearchWatchdog) Start() {

	go func() {

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {

			ctx := context.Background()

			// ⏱ find stale SEARCHING services
			stale, err := w.repo.FindStaleSearching(
				ctx,
				time.Now().Add(-10*time.Minute),
			)

			if err != nil {
				log.Println("watchdog find stale err:", err)
				continue
			}

			for _, svc := range stale {

				log.Println("🧹 watchdog cancelling service", svc.ID.Hex())

				err := w.cleaner.CancelSearchingService(
					ctx,
					svc.ID.Hex(),
					svc.User.Hex(),
				)

				if err != nil {
					log.Println("watchdog cancel err:", err)
				}
			}
		}
	}()
}
