package service
import (
	"context"
	"log"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"errors"
)


func (s *BiddingService) AcceptOffer(
	ctx context.Context,
	serviceID string,
	providerID string,
) error {

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	// ---------------- PARSE IDS ----------------
	if s.rdb.Exists(ctx, "service:stop:"+serviceID).Val() == 1 {
		return errors.New("service is no longer available")
	}

	if s.rdb.Exists(ctx, "service:locked:"+serviceID).Val() == 1 {
		return errors.New("service is no longer available")
	}

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return err
	}

	// ---------------- ATOMIC CONFIRM ----------------

	update := bson.M{
		"provider":               providerOID,
		"status":                 domain.StatusConfirmed,
		"timestamps.confirmedAt": time.Now(),
		"updatedAt":              time.Now(),
	}

	res, err := s.acceptedRepo.Col().UpdateOne(
		ctx,
		bson.M{
			"_id":    serviceOID,
			"status": "searching_after_cancel", // 👈 only 1 wins
		},
		bson.M{
			"$set": update,
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return errors.New("service already assigned to another provider")
	}

	// 🛑 winner freezes discovery + notifies others
	go s.stopAllDiscovery(ctx, serviceID)

	// ---------------- LOAD SERVICE ----------------

	svc, err := s.acceptedRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	// ---------------- LOAD USER ----------------

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return err
	}

	// ---------------- DISTANCE + ETA ----------------

	key := "service:dist:" + svc.ID.Hex() + ":" + providerID

	distance, _ := s.rdb.Get(ctx, key).Float64()

	eta := estimateETA(distance)

	// ---------------- SOCKET PAYLOAD ----------------

	payload := map[string]any{
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

	// 👉 PROVIDER ROOM
	s.socket.Emit(
		"provider:"+providerID,
		"payment:success",
		payload,
	)

	// 👉 USER ROOM
	s.socket.Emit(
		"user:"+svc.ID.Hex(),
		"payment:success",
		map[string]any{
			"serviceId": svc.ID.Hex(),
		},
	)

	log.Println("✅ AcceptOffer winner provider:", providerID)

	return nil
}

func (s *BiddingService) stopAllDiscovery(ctx context.Context, serviceID string) {

	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID
	providersKey := "service:providers:" + serviceID

	s.rdb.Set(ctx, stopKey, "1", 45*time.Minute)
	s.rdb.Set(ctx, lockKey, "1", 45*time.Minute)

	// ---------------- EMIT CANCEL TO ALL PROVIDERS ----------------

	providers, err := s.rdb.SMembers(ctx, providersKey).Result()
	if err == nil {

		for _, pid := range providers {

			s.socket.Emit(
				"provider:"+pid,
				"service:cancelled",
				map[string]any{
					"serviceId": serviceID,
					"reason":    "accepted_by_other_provider",
				},
			)
		}
	}

	s.rdb.Del(ctx, providersKey)

	cleanupServiceKeys(ctx, s.rdb, serviceID)

	log.Println("🛑 discovery stopped for service:", serviceID)
}


