package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"app_backend/internal/domain"
)

type InvoiceQueue struct {
	redis *redis.Client
}

func NewInvoiceQueue(r *redis.Client) *InvoiceQueue {
	return &InvoiceQueue{redis: r}
}

func (q *InvoiceQueue) Publish(ctx context.Context, job domain.InvoiceJob) error {

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	err = q.redis.LPush(ctx, "invoice:queue", data).Err()
	if err != nil {
		return err
	}

	log.Println("📦 invoice job pushed:", job.ServiceID)

	return nil
}
