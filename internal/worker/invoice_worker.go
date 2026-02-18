package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
)

type InvoiceWorker struct {
	redis       *redis.Client
	invoiceGen  ports.InvoiceGenerator
	paymentRepo *repository.PaymentRepository
}

func NewInvoiceWorker(
	r *redis.Client,
	invoiceGen ports.InvoiceGenerator,
	paymentRepo *repository.PaymentRepository,
) *InvoiceWorker {
	return &InvoiceWorker{
		redis:       r,
		invoiceGen:  invoiceGen,
		paymentRepo: paymentRepo,
	}
}

//////////////////////////////////////////////////////////
// START WORKER
//////////////////////////////////////////////////////////

func (w *InvoiceWorker) Start(ctx context.Context) {

	log.Println("🚀 Invoice Worker Started")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Invoice Worker Stopped")
			return
		default:
			w.consume(ctx)
		}
	}
}

//////////////////////////////////////////////////////////
// CONSUME JOB
//////////////////////////////////////////////////////////

func (w *InvoiceWorker) consume(ctx context.Context) {

	result, err := w.redis.BRPop(
		ctx,
		5*time.Second,
		"invoice:queue",
	).Result()

	if err != nil {
		if err == redis.Nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
		log.Println("❌ redis error:", err)
		return
	}

	var job domain.InvoiceJob

	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		log.Println("❌ invalid job:", err)
		return
	}

	w.process(ctx, job)
}

//////////////////////////////////////////////////////////
// PROCESS JOB
//////////////////////////////////////////////////////////

func (w *InvoiceWorker) process(ctx context.Context, job domain.InvoiceJob) {

	// 🔒 Atomic Guard
	ok, err := w.paymentRepo.MarkInvoiceGenerated(ctx, job.TxnID)
	if err != nil {
		log.Println("❌ mark invoice failed:", err)
		return
	}

	if !ok {
		log.Println("⛔ invoice already generated:", job.ServiceID)
		return
	}

	// 🧾 Generate Invoice
	invoice, err := w.invoiceGen.GenerateInvoice(
		ctx,
		job.UserID,
		job.ServiceID,
	)

	if err != nil {
		log.Println("❌ invoice generation failed:", err)
		w.retry(ctx, job)
		return
	}

	log.Println("✅ invoice generated:", invoice.InvoiceNumber)
}


func (w *InvoiceWorker) retry(ctx context.Context, job domain.InvoiceJob) {

	data, _ := json.Marshal(job)

	if err := w.redis.LPush(ctx, "invoice:retry", data).Err(); err != nil {
		log.Println("❌ failed to push retry job:", err)
		return
	}

	log.Println("🔁 moved to retry queue:", job.ServiceID)
}
