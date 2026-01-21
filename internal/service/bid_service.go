package service

import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"
"go.mongodb.org/mongo-driver/bson"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BiddingService struct {
	rdb          *redis.Client
	socket       *socket.Emitter
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	bidRepo      *repository.BidRepo
	counterRepo  *repository.CounterRepo
}

func NewBiddingService(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo     *repository.UserRepo,
	bidRepo *repository.BidRepo,
	counterRepo *repository.CounterRepo,
) *BiddingService {
	return &BiddingService{
		rdb:          rdb,
		socket:       socket,
		acceptedRepo: acceptedRepo,
		userRepo: userRepo,
		bidRepo: bidRepo,
		counterRepo: counterRepo,
	}
}

/* ================= START SEARCH ================= */

func (s *BiddingService) StartSearch(ctx context.Context,userID domain.UserID,vehicleType string,vehicleNumber string,brand string,modelYear int,fuelType string,serviceType string,issues []string,lat, lng float64) (string, error){
	userOID, _ := primitive.ObjectIDFromHex(string(userID))

	seq, _ := s.counterRepo.Next(ctx, "service")
	serviceNumber := fmt.Sprintf("VHBK%05d", seq)
	svc := &domain.AcceptedService{
		User:          userOID,
		Status:        domain.StatusSearching,
		VehicleType:   vehicleType,
		VehicleNumber: vehicleNumber,
		ServiceNumber: serviceNumber,
		Brand:         brand,
		ModelYear:     modelYear,
		FuelType:      fuelType,
		ServiceType:   serviceType,
		Issues:        issues,
		ServiceLocation: &domain.Location{
			Latitude:  lat,
			Longitude: lng,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	

	if err := s.acceptedRepo.Create(ctx, svc); err != nil {
		return "", err
	}
	fmt.Println("AFTER INSERT svc.ID =", svc.ID.Hex())

	// 🔥 Cache service meta (single source for sockets)
	s.rdb.HSet(ctx, "service:meta:"+svc.ID.Hex(), map[string]any{
		"userId":        userID,
		"vehicleType":   vehicleType,
		"vehicleNumber": vehicleNumber,
		"brand":         brand,
		"modelYear":     modelYear,
		"fuelType":      fuelType,
		"serviceType":   serviceType,
	})


	// 🔥 Start provider discovery
	go s.findProviders(
		svc.ID.Hex(),
		lat,
		lng,
		issues,
		vehicleType,
		vehicleNumber,
		brand,
		modelYear,
		fuelType,
		serviceType,
	)

	return svc.ID.Hex(), nil
}

/* ================= FIND PROVIDERS ================= */

func (s *BiddingService) findProviders(
	serviceID string,
	lat, lng float64,
	issues []string,
	vehicleType string,
	vehicleNumber string,
	brand string,
	modelYear int,
	fuelType string,
	serviceType string,
) {
	ctx := context.Background()

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return
	}

	// 🔹 Fetch service once
	svc, err := s.acceptedRepo.FindByID(ctx, serviceID)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(svc.User)
	// 🔹 Fetch user once
	user, err := s.userRepo.FindByID(ctx, svc.User)
	if err != nil {
		fmt.Println("❌ user not found:", err)
		return
	}
	rejectedKey := "service:rejected:" + serviceID
	attemptedKey := "service:attempted:" + serviceID

	// 🔍 Radius expansion steps
	radiusSteps := []float64{5, 10, 20, 50}

	for _, radius := range radiusSteps {

		// 🔒 Re-check service status
		var current domain.AcceptedService
		if err := s.acceptedRepo.Col().
			FindOne(ctx, bson.M{"_id": serviceOID}).
			Decode(&current); err != nil {
			return
		}

		if current.Status != domain.StatusSearching {
			return
		}

		// 🔍 Find online providers
		providers, err := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			lng,
			lat,
			&redis.GeoRadiusQuery{
				Radius:   radius,
				Unit:     "km",
				WithDist: true, // ✅ REQUIRED
			},
		).Result()

		if err != nil || len(providers) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}

		// 🕒 Allow socket joins
		time.Sleep(2 * time.Second)

		for _, p := range providers {

			providerID := p.Name

			// ❌ Skip rejected providers
			if rejected, _ := s.rdb.SIsMember(
				ctx,
				rejectedKey,
				providerID,
			).Result(); rejected {
				continue
			}

			// ❌ Skip already attempted providers
			if attempted, _ := s.rdb.SIsMember(
				ctx,
				attemptedKey,
				providerID,
			).Result(); attempted {
				continue
			}

			distance := p.Dist
			eta := estimateETA(distance)

			// 🔹 Cache distance
			s.rdb.Set(
				ctx,
				"service:dist:"+serviceID+":"+providerID,
				distance,
				15*time.Minute,
			)

			// 🔔 Emit bid request
			s.socket.EmitWithRetry(
				"provider:"+providerID,
				"bid:request",
				map[string]any{
					"serviceId": serviceID,
					"user": map[string]any{
						"id":   svc.User.Hex(),
						"name": user.Name,
					},
					"vehicle": map[string]any{
						"type":   vehicleType,
						"number": vehicleNumber,
						"brand":  brand,
						"year":   modelYear,
						"fuel":   fuelType,
					},
					"serviceType": serviceType,
					"issues":      issues,
					"distanceKm":  distance,
					"etaMin":      eta,
					"radiusKm":    radius,
					"expiresIn":   60,
				},
				1,
			)

			// 🔒 Mark provider as attempted
			s.rdb.SAdd(ctx, attemptedKey, providerID)
			s.rdb.Expire(ctx, attemptedKey, 30*time.Minute)
		}

		// ⏳ Wait before next radius
		time.Sleep(6 * time.Second)

		// 🔁 Increment retry count
		_, _ = s.acceptedRepo.Col().UpdateByID(
			ctx,
			serviceOID,
			bson.M{"$inc": bson.M{"retryCount": 1}},
		)
	}

	// ❌ No provider found
	_, _ = s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": bson.M{"status": "no_provider_foundx"}},
	)
}


/* ================= PLACE BID ================= */

func (s *BiddingService) PlaceBid(
	ctx context.Context,
	serviceID string,
	providerID string,
	price int,
) (string, error) {

	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
	providerOID, _ := primitive.ObjectIDFromHex(providerID)

	bid := &domain.BidLog{
		ServiceID:  serviceOID,
		ProviderID: providerOID,
		Price:      price,
		CreatedAt:  time.Now(),
	}

	bidOID, err := s.bidRepo.Insert(ctx, bid)
	if err != nil {
		return "", err
	}

	distance, _ := s.rdb.Get(
		ctx,
		"service:dist:"+serviceID+":"+providerID,
	).Float64()

	eta := estimateETA(distance)

	// 🔥 SEND BID TO USER
	s.socket.Emit(
		"user:"+serviceID,
		"bid:update",
		map[string]any{
			"bidId": bidOID.Hex(),
			"price": price,
			"provider": map[string]any{
				"id":         providerID,
				"distanceKm": distance,
				"etaMin":     eta,
			},
		},
	)

	return bidOID.Hex(), nil
}
func (s *BiddingService) AcceptBid(
	ctx context.Context,
	serviceID string,
	bidID string,
	providerID string,
	price float64,
) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	bidOID, err := primitive.ObjectIDFromHex(bidID)
	if err != nil {
		return err
	}

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return err
	}

	// 🔒 Assign provider & lock service
	if err := s.acceptedRepo.UpdateByID(
		ctx,
		serviceOID,
		map[string]any{
			"$set": map[string]any{
				"provider":      providerOID,
				"acceptedBid":   bidOID,
				"finalPrice":    price,
				"status":        domain.StatusProviderAssigned,
				"paymentStatus": domain.PaymentPending,
				"updatedAt":     time.Now(),
			},
		},
	); err != nil {
		return err
	}
	s.rdb.Set(ctx,
		"provider:busy:"+providerID,
		serviceID,
		2*time.Hour, // auto release safety
	)

	// ✅ REMOVE PROVIDER FROM GEO (VERY IMPORTANT)
	s.rdb.ZRem(ctx, "providers:geo", providerID)

	// 🔔 Notify PROVIDER
	s.socket.Emit(
		"provider:"+providerID,
		"bid:accepted",
		map[string]any{
			"serviceId": serviceID,
			"price":     price,
		},
	)

	// 🔔 Notify USER (recommended)
	s.socket.Emit(
		"user:"+serviceID,
		"service:assigned",
		map[string]any{
			"serviceId":  serviceID,
			"providerId": providerID,
			"price":      price,
		},
	)

	return nil
}
func (s *BiddingService) RejectBid(
	ctx context.Context,
	serviceID string,
	providerID string,
) error {

	key := "service:rejected:" + serviceID

	// 🔒 add provider to rejected set
	if err := s.rdb.SAdd(ctx, key, providerID).Err(); err != nil {
		return err
	}

	// ⏱ auto cleanup
	s.rdb.Expire(ctx, key, 30*time.Minute)

	// 🔔 notify provider (optional)
	s.socket.Emit(
		"provider:"+providerID,
		"bid:rejected",
		map[string]any{
			"serviceId": serviceID,
		},
	)

	return nil
}



/* ================= HELPERS ================= */

func estimateETA(distanceKm float64) int {
	if distanceKm <= 1 {
		return 5
	}
	return int(distanceKm*4 + 5)
}