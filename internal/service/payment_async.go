package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/events"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"go.mongodb.org/mongo-driver/bson"
	"encoding/json"
	// "strconv"
	// "fmt"
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

	// ---------------- SOCKET PAYLOAD ----------------

	providerPayload := map[string]any{
		"serviceId": svc.ID.Hex(),
		"serviceNo": svc.ServiceNumber,

		"user": map[string]any{
			"id":   string(user.ID),
			"name": user.Name,
			"profileUrl":user.ImageUrl,
		},

		"vehicle": map[string]any{
			"type":   svc.VehicleType,
			"number": svc.VehicleNumber,
			"brand":  svc.Brand,
			"fuel":   svc.FuelType,
			"year":   svc.ModelYear,
			"model":  svc.Model,
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
	providerBytes,_ := json.Marshal(providerPayload)

	// ---------------- SOCKET + NOTIFICATION ----------------

	if svc.Provider != primitive.NilObjectID {

		providerID := svc.Provider.Hex()

		// 🔥 SOCKET
		s.socket.Emit(
			"provider:"+providerID,
			"payment:success",
			providerPayload,
		)

		// 🔔 PUSH NOTIFICATION

		

		go s.notify.SendToProvider(
			context.Background(),
			providerID,
			"Payment Successful 💰",
			"User payment completed. Open app.",
			map[string]string{
				"type":      "payment_success",
				"serviceId": svc.ID.Hex(),
				"payload":   string(providerBytes),
			},
		)
	}

	// ---------------- USER SOCKET ----------------

	s.socket.Emit(
		"user:"+svc.ID.Hex(),
		"payment:success",
		map[string]any{
			"serviceId": svc.ID.Hex(),
		},
	)

	// ---------------- SIDE EFFECTS ----------------

	go s.invoiceSvc.GenerateInvoice(
		context.Background(),
		txn.UserID,
		txn.ServiceID,
	)

	s.events.Publish("payment.success", events.PaymentEvent{
		TxnID:     txnID,
		ServiceID: svc.ID.Hex(),
		Status:    "paid",
	})

	log.Println("DONEEEEE payment success")
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
		go s.releaseProviderAfterGrace(
			txn.ServiceID,
			svc.Provider.Hex(),
		)
	}
	userPayload := map[string]any{
		"serviceId": svc.ID.Hex(),
		"ttl":       ttl,
		"message":   msg,
		"reason":    reason,
	}

	userBytes, _ := json.Marshal(userPayload)

	go s.notify.SendToUser(
		context.Background(),
		txn.UserID,
		"Payment Failed ❌",
		msg,
		map[string]string{
			"type":      "payment_failed",
			"serviceId": svc.ID.Hex(),
			"payload":   string(userBytes), // ✅ SAME AS SOCKET
		},
	)

}


/*
RELEASE PROVIDER AFTER GRACE
*/
func (s *PaymentService) releaseProviderAfterGrace(serviceID, providerID string) {

	log.Println("⏳ grace timer started for provider:", providerID)

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
		log.Println("⛔ not releasing provider:", err)
		return
	}

	log.Println("✅ releasing provider:", providerID)

	// ---------------- REDIS CLEANUP ----------------
	if err := s.redis.Del(ctx, "reserve:"+providerID).Err(); err != nil {
		log.Println("⚠ redis delete failed:", err)
	}
	if err := s.redis.Del(ctx,"provider:busy:"+providerID); err != nil{
		log.Println("⚠ redis delete failed:", err)
	}

	// ---------------- FINAL STATUS ----------------
	_ = s.acceptedServiceRepo.UpdateStatus(
		ctx,
		serviceOID,
		"cancelled",
		bson.M{},
	)

	// ---------------- EVENTS ----------------
	s.events.Publish("payment.failed.final", events.PaymentEvent{
		ServiceID: serviceID,
		Status:    "cancelled",
	})

	// ---------------- USER SOCKET ----------------
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