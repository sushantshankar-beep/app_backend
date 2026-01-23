package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/events"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
)

/*
AFTER PAYMENT SUCCESS
*/
func (s *PaymentService) afterPaymentSuccess(txnID string) {

	ctx := context.Background()

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil || txn.InvoiceGenerated {
		log.Println("❌ afterPaymentSuccess exit:", err)
		return
	}

	serviceOID, _ := primitive.ObjectIDFromHex(txn.ServiceID)

	err = s.acceptedServiceRepo.UpdatePaymentStatus(
		ctx,
		serviceOID,
		domain.PaymentPaid,
	)

	if err != nil {
		log.Println("❌ UpdatePaymentStatus:", err)
	}

	s.events.Publish("payment.success", events.PaymentEvent{
		TxnID:     txnID,
		ServiceID: txn.ServiceID,
		Status:    "paid",
	})

	go s.invoiceSvc.GenerateInvoice(ctx, txn.UserID, txn.ServiceID, nil)

	/* 🔥 SOCKET SUCCESS */

	s.socket.EmitWithRetry(
		"user:"+txn.ServiceID,
		"payment:success",
		map[string]any{
			"serviceId": txn.ServiceID,
			"amount":    txn.Amount,
		},
		1,
	)
}

/*
AFTER PAYMENT FAILURE
*/
func (s *PaymentService) afterPaymentFailed(txnID string) {

	ctx := context.Background()

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return
	}

	serviceOID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		log.Println("❌ invalid service id:", txn.ServiceID)
		return
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		log.Println("❌ accepted service:", err)
		return
	}

	reason := domain.PaymentFailReason(txn.FailReason)
	if reason == "" {
		reason = domain.FailUnknown
	}

	if err := domain.CanEnterPaymentGrace(svc); err != nil {
		log.Println("⚠️ cannot enter grace:", err)
		return
	}

	if err := s.acceptedServiceRepo.UpdatePaymentStatus(
		ctx,
		serviceOID,
		domain.PaymentFailedGrace,
	); err != nil {
		log.Println("❌ update grace:", err)
	}

	ttl := 300
	msg := userMessageForReason(reason)
	room := "user:" + txn.ServiceID

	s.socket.EmitWithRetry(
		room,
		"payment:failed",
		map[string]any{
			"serviceId": svc.ID.Hex(),
			"ttl":       ttl,
			"message":   msg,
			"reason": reason,
		},
		1,
	)
}


/*
RELEASE PROVIDER AFTER GRACE
*/
func (s *PaymentService) releaseProviderAfterGrace(serviceID, providerID string) {
	time.Sleep(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return
	}

	if err := domain.CanReleaseProviderAfterFailure(svc); err != nil {
		return
	}

	_ = s.redis.Del(ctx, "reserve:"+providerID).Err()

	_ = s.acceptedServiceRepo.UpdatePaymentStatus(
		ctx,
		serviceOID,
		domain.PaymentFailed,
	)

	s.events.Publish("payment.failed.final", events.PaymentEvent{
		ServiceID: serviceID,
		Status:    "cancelled",
	})

	s.socket.EmitWithRetry(
		"user:"+svc.User.Hex(),
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
			"reason":    "payment_timeout",
		},
		3,
	)
}
