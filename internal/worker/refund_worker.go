package worker

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
)

type RefundWorker struct {
	rdb *redis.Client
	processor ports.RefundProcessor
	refundRepo ports.RefundRepo
}

func NewRefundWorker(
	rdb *redis.Client,
	processor ports.RefundProcessor,
	refundRepo ports.RefundRepo,
) *RefundWorker {
	return &RefundWorker{
		rdb: rdb,
		processor: processor,
		refundRepo: refundRepo,
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
		log.Println("refund worker redis error:", err)
		return
	}

	var job domain.RefundJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return
	}

	// ===========================
	// 📌 Mark processing
	// ===========================
	_ = w.refundRepo.UpdateByMihPayID(ctx, job.MihPayID,
		bson.M{"$set": bson.M{
			"status":     "processing",
			"updatedAt": time.Now(),
		}},
	)

	// ===========================
	// 🚀 Call gateway
	// ===========================
	err = w.processor.ProcessRefund(ctx, job.MihPayID, job.Amount)

	if err == nil {

		log.Println("✅ refund success:", job.MihPayID)

		_ = w.refundRepo.UpdateByMihPayID(ctx, job.MihPayID,
			bson.M{"$set": bson.M{
				"status":     "success",
				"updatedAt": time.Now(),
			}},
		)

		return
	}

	log.Println("❌ refund failed:", err)

	// ===========================
	// 🔁 Retry / DLQ
	// ===========================
	if job.Retries < 5 {

		job.Retries++

		delay := time.Duration(math.Pow(2, float64(job.Retries))) * time.Second
		time.Sleep(delay)

		b, _ := json.Marshal(job)
		_ = w.rdb.RPush(ctx, "refund:queue", b).Err()

	} else {

		log.Println("💀 refund moved to DLQ:", job.MihPayID)

		b, _ := json.Marshal(job)
		_ = w.rdb.RPush(ctx, "refund:dlq", b).Err()

		_ = w.refundRepo.UpdateByMihPayID(ctx, job.MihPayID,
			bson.M{"$set": bson.M{
				"status":     "failed",
				"updatedAt": time.Now(),
			}},
		)
	}
}
