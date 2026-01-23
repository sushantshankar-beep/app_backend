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

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// ---------------- LOAD TXN ----------------
	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil || txn.InvoiceGenerated {
		return
	}

	serviceOID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		return
	}

	// ---------------- UPDATE FIRST ----------------
	_ = s.acceptedServiceRepo.UpdatePaymentStatus(
		ctx,
		serviceOID,
		domain.PaymentPaid,
	)

	// ---------------- LOAD SERVICE ----------------
	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return
	}

	// ---------------- PARALLEL FETCH USER + DIST ----------------
	var (
		user     *domain.User
		distance float64
	)

	errCh := make(chan error, 2)

	go func() {
		u, err := s.userRepo.GetByID(ctx, svc.User)
		if err == nil {
			user = u
		}
		errCh <- err
	}()

	go func() {
		if svc.Provider != primitive.NilObjectID {
			key := "service:dist:" + svc.ID.Hex() + ":" + svc.Provider.Hex()
			distance, _ = s.redis.Get(ctx, key).Float64()
		}
		errCh <- nil
	}()

	<-errCh
	<-errCh

	eta := estimateETA(distance)

	// ---------------- PROVIDER PAYLOAD ----------------
	providerPayload := map[string]any{
		"serviceId": svc.ID.Hex(),
		"serviceNo": svc.ServiceNumber,

		"user": map[string]any{
			"id":   user.ID,
			"name": user.Name,
		},

		"vehicle": map[string]any{
			"type":   svc.VehicleType,
			"number": svc.VehicleNumber,
			"brand":  svc.Brand,
			"fuel":   svc.FuelType,
			"year":   svc.ModelYear,
		},

		"issues": svc.Issues,

		"payment": map[string]any{
			"status": "paid",
			"amount": svc.FinalPrice,
		},

		"tracking": map[string]any{
			"distanceKm": distance,
			"etaMin":     eta,
		},
	}

	// ---------------- SOCKET EMITS ----------------

	// 👉 Provider full payload
	if svc.Provider != primitive.NilObjectID  {
		s.socket.Emit(
			"provider:"+svc.Provider.Hex(),
			"payment:success",
			providerPayload,
		)
	}

	// 👉 User lightweight
	s.socket.Emit(
		"user:"+svc.ID.Hex(),
		"payment:success",
		map[string]any{
			"serviceId": svc.ID.Hex(),
		},
	)

	// ---------------- ASYNC SIDE EFFECTS ----------------
	go s.invoiceSvc.GenerateInvoice(context.Background(), txn.UserID, txn.ServiceID, nil)

	s.events.Publish("payment.success", events.PaymentEvent{
		TxnID:     txnID,
		ServiceID: svc.ID.Hex(),
		Status:    "paid",
	})
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

	if err := s.acceptedServiceRepo.UpdatePaymentStatus(
		ctx,
		serviceOID,
		domain.PaymentFailed,
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
	if svc.Provider != primitive.NilObjectID  {
		s.socket.Emit(
			"provider:"+svc.Provider.Hex(),
			"payment:failed",
			map[string]any{
				"serviceId": svc.ID.Hex(),
			},
		)
	}
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
