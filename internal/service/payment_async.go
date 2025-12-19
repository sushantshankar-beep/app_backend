package service

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
AFTER PAYMENT SUCCESS
- Assign provider
- Release Redis reservation
- Notify provider
- Emit socket event
*/
func (s *PaymentService) afterPaymentSuccess(txnID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return
	}

	serviceID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		return
	}

	var svc domain.AcceptedService
	if err := s.acceptedServiceRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceID}).
		Decode(&svc); err != nil {
		return
	}

	_, _ = s.acceptedServiceRepo.Col().UpdateByID(
		ctx,
		serviceID,
		bson.M{
			"$set": bson.M{
				"paymentStatus": "paid",
				"status":        "assigned",
				"updatedAt":     time.Now(),
			},
		},
	)

	// ✅ Redis reservation released
	_ = s.redis.Del(ctx, "reserve:"+svc.Provider.Hex()).Err()

	_ = s.notify.SendToProvider(
		ctx,
		svc.Provider.Hex(),
		"Job Assigned",
		"Payment completed successfully",
		map[string]string{
			"serviceId": svc.ID.Hex(),
		},
	)

	room := svc.ServiceRequest.Hex() + svc.Provider.Hex()
	s.socket.EmitWithRetry(
		room,
		"payment:success",
		svc,
		3,
	)
}

/*
AFTER PAYMENT FAILURE
- Unlock provider
- Notify both sides
*/
func (s *PaymentService) afterPaymentFailed(txnID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return
	}

	serviceObjID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		return
	}

	var svc domain.AcceptedService
	_ = s.acceptedServiceRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceObjID}).
		Decode(&svc)

	_, _ = s.acceptedServiceRepo.Col().
		UpdateByID(ctx, serviceObjID, bson.M{
			"$set": bson.M{"paymentStatus": "failed"},
		})

	// Release reservation on failure
	_ = s.redis.Del(ctx, "reserve:"+svc.Provider.Hex()).Err()

	room := svc.ServiceRequest.Hex() + svc.Provider.Hex()
	s.socket.EmitWithRetry(
		room,
		"payment:failed",
		map[string]any{
			"serviceId": svc.ID.Hex(),
			"status":    "failed",
		},
		3,
	)
}
