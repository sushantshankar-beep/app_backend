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
	"log"
	"errors"
	"sync"
)

type BiddingService struct {
	rdb          *redis.Client
	socket       *socket.Emitter
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	bidRepo      *repository.BidRepo
	providerRepo *repository.ProviderRepo
	counterRepo  *repository.CounterRepo
}

func NewBiddingService(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo     *repository.UserRepo,
	bidRepo *repository.BidRepo,
	providerRepo *repository.ProviderRepo,
	counterRepo *repository.CounterRepo,
) *BiddingService {
	return &BiddingService{
		rdb:          rdb,
		socket:       socket,
		acceptedRepo: acceptedRepo,
		userRepo: userRepo,
		bidRepo: bidRepo,
		providerRepo: providerRepo,
		counterRepo: counterRepo,
	}
}
var ErrServiceAlreadyAssigned = errors.New("service already assigned")

/* ================= START SEARCH ================= */

func (s *BiddingService) StartSearch(ctx context.Context,userID domain.UserID,vehicleType string,vehicleNumber string,brand string,modelYear int,fuelType string,serviceType string,issues []string,lat, lng float64,model string) (string, error){
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
		Model:         model,
		UserLocation: &domain.UserLocation{
			Lat:  lat,
			Long: lng,
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
		"model" :         model,
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
		model,
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
	model string,
) {

	ctx := context.Background()

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return
	}

	lockKey := "service:locked:" + serviceID

	// radius plan
	radiusSteps := []float64{5, 10, 20, 50}

	const (
		maxConcurrentEmits = 25
		maxAttemptsPerProv = 3
		cooldownSeconds    = 60
	)

	for _, radius := range radiusSteps {

		// 🛑 stop if assigned
		if s.rdb.Exists(ctx, lockKey).Val() == 1 {
			return
		}

		// 🔒 DB state
		var current domain.AcceptedService
		if err := s.acceptedRepo.Col().
			FindOne(ctx, bson.M{"_id": serviceOID}).
			Decode(&current); err != nil {
			return
		}

		if current.Status != domain.StatusSearching {
			return
		}

		providers, err := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			lng,
			lat,
			&redis.GeoRadiusQuery{
				Radius:   radius,
				Unit:     "km",
				WithDist: true,
			},
		).Result()

		if err != nil || len(providers) == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, maxConcurrentEmits)

		for _, p := range providers {

			pid := p.Name
			dist := p.Dist

			// 🔐 cooldown check
			cooldownKey := "service:cooldown:" + serviceID + ":" + pid

			if s.rdb.Exists(ctx, cooldownKey).Val() == 1 {
				continue
			}

			// 🔢 attempt counter
			attemptKey := "service:attempts:" + serviceID + ":" + pid
			attempts,_ := s.rdb.Get(ctx, attemptKey).Int64()

			if attempts >= maxAttemptsPerProv {
				continue
			}

			// reserve cooldown immediately (atomic)
			ok, _ := s.rdb.SetNX(
				ctx,
				cooldownKey,
				"1",
				time.Duration(cooldownSeconds)*time.Second,
			).Result()

			if !ok {
				continue
			}

			// increment attempts
			s.rdb.Incr(ctx, attemptKey)
			s.rdb.Expire(ctx, attemptKey, 2*time.Hour)

			wg.Add(1)

			go func(providerID string, distance float64) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				// stop mid-flight
				if s.rdb.Exists(ctx, lockKey).Val() == 1 {
					return
				}

				s.socket.EmitWithRetry(
					"provider:"+providerID,
					"bid:request",
					map[string]any{
						"serviceId": serviceID,
						"vehicle": map[string]any{
							"type":   vehicleType,
							"number": vehicleNumber,
							"brand":  brand,
							"year":   modelYear,
							"fuel":   fuelType,
							"model":  model,
						},
						"serviceType": serviceType,
						"issues":      issues,
						"distanceKm":  distance,
						"etaMin":      estimateETA(distance),
						"radiusKm":    radius,
						"expiresIn":   60,
					},
					1,
				)

			}(pid, dist)
		}

		wg.Wait()

		time.Sleep(2 * time.Second)
	}

	// ❌ nobody accepted
	s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": bson.M{"status": domain.StatusCancelled}},
	)
}



/* ================= PLACE BID ================= */

func (s *BiddingService) PlaceBid(
	ctx context.Context,
	serviceID string,
	providerID string,
	price int,
) (string, error) {
	locked, _ := s.rdb.Exists(
		ctx,
		"service:locked:"+serviceID,
	).Result()

	if locked == 1 {
		return "", ErrServiceAlreadyAssigned
	}

	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
	providerOID, _ := primitive.ObjectIDFromHex(providerID)

	bid := &domain.BidLog{
		ServiceID:  serviceOID,
		ProviderID: providerOID,
		Price:      price,
		CreatedAt:  time.Now(),
	}
	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(providerID),
	)
	if err != nil {
		return "", err
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
				"name":       provider.Name,
				"etaMin":     eta,
				"profileUrl": provider.ProfileURL,
				"rating":     provider.Rating,
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
	res, err := s.acceptedRepo.Col().UpdateOne(
		ctx,
		bson.M{
			"_id":    serviceOID,
		},
		bson.M{
			"$set": bson.M{
				"provider":      providerOID,
				"acceptedBid":   bidOID,
				"finalPrice":    price,
				"status":        domain.StatusProviderAssigned,
				"paymentStatus": domain.PaymentPending,
				"updatedAt":     time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrServiceAlreadyAssigned
	}
	s.rdb.Set(ctx,
		"service:locked:"+serviceID,
		"1",
		30*time.Minute,
	)
	pos, err := s.rdb.GeoPos(ctx, "providers:geo", providerID).Result()
	if err != nil {
		log.Println("redis geopos failed:", err)
	}
	var lat, long float64
	if len(pos) > 0 && pos[0] != nil {
		long = pos[0].Longitude
		lat = pos[0].Latitude
	}
	now := time.Now()
	set := bson.M{
		"provider":      providerOID,
		"acceptedBid":   bidOID,
		"finalPrice":    price,
		"status":        domain.StatusProviderAssigned,
		"paymentStatus": domain.PaymentPending,
		"updatedAt":     now,
	}
	if lat != 0 && long != 0 {
		set["providerLocation"] = bson.M{
			"lat":       lat,
			"long":      long,
			"updatedAt": now,
		}
	}
	// 🔒 Assign provider & lock service
	if err := s.acceptedRepo.UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": set},
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

	// add to rejected set
	if err := s.rdb.SAdd(ctx, key, providerID).Err(); err != nil {
		return err
	}

	s.rdb.Expire(ctx, key, 30*time.Minute)

	// 🔔 PROVIDER
	s.socket.Emit(
		"provider:"+providerID,
		"bid:rejected",
		map[string]any{
			"serviceId": serviceID,
		},
	)

	// 🔔 USER
	s.socket.Emit(
		"user:"+serviceID,
		"bid:rejected",
		map[string]any{
			"serviceId":  serviceID,
			"providerId": providerID,
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