package service

import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)



/*
	ASYNC FLOW AFTER PAYMENT SUCCESS
*/
func (s *PaymentService) afterPaymentSuccess(txnID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 1️⃣ Transaction
	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return
	}

	// 2️⃣ Accepted Service
	serviceObjID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		return
	}

	var service domain.AcceptedService
	err = s.acceptedServiceRepo.
		Col().
		FindOne(ctx, bson.M{"_id": serviceObjID}).
		Decode(&service)

	if err != nil {
		return
	}

	// 3️⃣ OTP + payment update
	otp := GenerateOTP()

	_, _ = s.acceptedServiceRepo.
		Col().
		UpdateByID(ctx, serviceObjID, bson.M{
			"$set": bson.M{
				"paymentStatus": "paid",
				"otp.code":      otp,
				"otp.verified":  false,
			},
		})

	// 4️⃣ Fetch provider FCM token
	var provider struct {
		FCMToken string `bson:"fcmToken"`
	}

	_ = s.providerRepo.FindOne(
		ctx,
		bson.M{"_id": service.Provider},
		&provider,
	)

	if provider.FCMToken != "" {
		_ = s.notify.SendToProvider(
			ctx,
			service.Provider.Hex(),
			"Payment Completed",
			fmt.Sprintf("Payment for %s completed.", service.ServiceType),
			map[string]string{
				"serviceId": service.ID.Hex(),
				"status":    "paid",
			},
		)
	}

	// 5️⃣ Socket emit
	room := service.ServiceRequest.Hex() + service.Provider.Hex()

	s.socket.EmitWithRetry(
		room,
		"payment:confirmation",
		map[string]any{
			"serviceId": service.ID.Hex(),
			"status":    "paid",
			"otp":       otp,
		},
		3,
	)
}

/*
	ASYNC FLOW AFTER PAYMENT FAILURE
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

	_, _ = s.acceptedServiceRepo.
		Col().
		UpdateByID(ctx, serviceObjID, bson.M{
			"$set": bson.M{"paymentStatus": "failed"},
		})

	var service domain.AcceptedService
	_ = s.acceptedServiceRepo.
		Col().
		FindOne(ctx, bson.M{"_id": serviceObjID}).
		Decode(&service)

	room := service.ServiceRequest.Hex() + service.Provider.Hex()

	s.socket.EmitWithRetry(
		room,
		"payment:failed",
		map[string]any{
			"serviceId": service.ID.Hex(),
			"status":    "failed",
		},
		3,
	)
}
