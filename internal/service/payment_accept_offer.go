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

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// ---------------- PARSE IDS ----------------

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return err
	}
	s.stopAllDiscovery(ctx, serviceID)

	// ---------------- LOAD SERVICE ----------------

	svc, err := s.acceptedRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	// 🔐 Prevent double confirmation
	if svc.Status == domain.StatusConfirmed {
		return errors.New("service already confirmed")
	}

	// ---------------- UPDATE PROVIDER + STATUS ----------------

	update := bson.M{
		"provider":               providerOID,
		"status":                 domain.StatusConfirmed,
		"timestamps.confirmedAt": time.Now(),
		"updatedAt":              time.Now(),
	}

	if err := s.acceptedRepo.UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": update},
	); err != nil {
		return err
	}

	// reload fresh copy
	svc, err = s.acceptedRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	// ---------------- LOAD USER ----------------

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return err
	}

	// ---------------- DISTANCE + ETA ----------------

	var distance float64

	key := "service:dist:" + svc.ID.Hex() + ":" + providerID
	distance, _ = s.rdb.Get(ctx, key).Float64()

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

	log.Println("✅ AcceptOffer reassigned provider:", providerID)

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


