package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"

	"github.com/redis/go-redis/v9"
)

type RefundWorker struct {
	rdb       *redis.Client
	processor ports.RefundProcessor
}

func NewRefundWorker(
	rdb *redis.Client,
	processor ports.RefundProcessor,
) *RefundWorker {
	return &RefundWorker{
		rdb:       rdb,
		processor: processor,
	}
}

func (w *RefundWorker) Start() {
	go func() {
		for {
			w.process()
		}
	}()
}

func (w *RefundWorker) process() {
	ctx := context.Background()

	res, err := w.rdb.BLPop(ctx, 0, "refund:queue").Result()
	if err != nil {
		log.Println("refund worker error:", err)
		return
	}

	var job domain.RefundJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return
	}

	err = w.processor.ProcessRefund(ctx, job.MihPayID, job.Amount)
	if err != nil {
		log.Println("refund failed:", err)

		if job.Retries < 3 {
			job.Retries++
			b, _ := json.Marshal(job)
			time.Sleep(2 * time.Second)
			_ = w.rdb.RPush(ctx, "refund:queue", b).Err()
		}
	}
}
