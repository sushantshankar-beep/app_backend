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

func (w *InvoiceWorker) Start(ctx context.Context) {

	log.Println("🚀 Invoice Worker Started")

	// 🔥 Warmup Chrome (fix first booking issue)
	go w.warmup()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Invoice Worker Stopped")
			return
		default:
			w.consume(ctx)
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (w *InvoiceWorker) warmup() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, _ = w.invoiceGen.GenerateInvoice(ctx, "000000000000000000000000", "000000000000000000000000")
}

func (w *InvoiceWorker) consume(parentCtx context.Context) {

	result, err := w.redis.BRPop(
		parentCtx,
		5*time.Second,
		"invoice:queue",
	).Result()

	if err != nil {
		if err == redis.Nil || parentCtx.Err() != nil {
			return
		}
		log.Println("❌ redis error:", err)
		time.Sleep(1 * time.Second)
		return
	}

	var job domain.InvoiceJob

	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		log.Println("❌ invalid job:", err)
		return
	}

	log.Println("📥 received invoice job:", job.TxnID)

	w.process(job)
}

func (w *InvoiceWorker) process(job domain.InvoiceJob) {

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println("🔥 panic during invoice generation:", r)
			w.retry(ctx, job)
		}
	}()

	// 1️⃣ Fetch transaction directly (no race condition)
	txn, err := w.paymentRepo.GetByTxnID(ctx, job.TxnID)
	if err != nil {
		log.Println("❌ failed to fetch txn:", err)
		w.retry(ctx, job)
		return
	}

	if txn.InvoiceGenerated {
		log.Println("⛔ invoice already generated:", job.ServiceID)
		return
	}

	// 2️⃣ Generate invoice FIRST
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

	if invoice == nil {
		log.Println("❌ invoice returned nil")
		w.retry(ctx, job)
		return
	}

	// 3️⃣ Mark AFTER success
	ok, err := w.paymentRepo.MarkInvoiceGenerated(ctx, job.TxnID)
	if err != nil {
		log.Println("❌ mark invoice failed:", err)
		w.retry(ctx, job)
		return
	}

	if !ok {
		log.Println("⛔ invoice already marked by another worker")
		return
	}

	log.Println("✅ invoice generated successfully:", invoice.InvoiceNumber)
}

func (w *InvoiceWorker) retry(ctx context.Context, job domain.InvoiceJob) {

	if job.Retry >= 5 {
		log.Println("💀 max retry reached:", job.ServiceID)
		return
	}

	job.Retry++

	data, _ := json.Marshal(job)

	backoff := time.Duration(job.Retry*5) * time.Second
	time.Sleep(backoff)

	if err := w.redis.LPush(ctx, "invoice:queue", data).Err(); err != nil {
		log.Println("❌ failed to requeue job:", err)
		return
	}

	log.Println("🔁 requeued invoice job:", job.ServiceID, "retry:", job.Retry)
}
