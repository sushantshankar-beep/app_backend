package service

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"app_backend/internal/domain"
	"app_backend/internal/socket"
	"app_backend/internal/repository"
)

type BiddingService struct {
	rdb              *redis.Client
	socket           *socket.Emitter
	acceptedRepo     *repository.AcceptedServiceRepo
	cancelRepo       *repository.CancellationRepo
}

func NewBiddingService(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
	cancelRepo *repository.CancellationRepo,
) *BiddingService {
	return &BiddingService{rdb, socket, acceptedRepo, cancelRepo}
}

/* ---------------- FIND MECHANICS ---------------- */

func (s *BiddingService) FindMechanics(
	ctx context.Context,
	serviceID string,
	lat, lng float64,
) {

	lockKey := "lock:find:" + serviceID
	ok, _ := s.rdb.SetNX(ctx, lockKey, 1, 30*time.Second).Result()
	if !ok {
		return
	}

	radiusLevels := []float64{10, 20, 50}

	objID, _ := primitive.ObjectIDFromHex(serviceID)
	var svc domain.AcceptedService
	s.acceptedRepo.Col().FindOne(ctx, bson.M{"_id": objID}).Decode(&svc)

	excluded := map[string]bool{}
	for _, p := range svc.NotToSendProviders {
		excluded[p.Hex()] = true
	}

	for _, radius := range radiusLevels {
		providers, _ := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			lng, lat,
			&redis.GeoRadiusQuery{
				Radius: radius,
				Unit:   "km",
			},
		).Result()

		if len(providers) == 0 {
			continue
		}

		for _, p := range providers {
			if excluded[p.Name] {
				continue
			}

			s.socket.EmitWithRetry(
				"provider:"+p.Name,
				"bid:request",
				map[string]any{
					"serviceId": serviceID,
					"radius": radius,
				},
				1,
			)
		}
		return
	}

	// Retry logic
	if svc.RetryCount >= svc.MaxRetries {
		s.acceptedRepo.Col().UpdateByID(ctx, objID, bson.M{
			"$set": bson.M{"status": "no_provider_found"},
		})

		s.socket.EmitWithRetry(
			"user:"+svc.User.Hex(),
			"service:failed",
			map[string]any{"reason": "No mechanics available"},
			2,
		)
		return
	}

	s.acceptedRepo.Col().UpdateByID(ctx, objID, bson.M{
		"$inc": bson.M{"retryCount": 1},
	})

	go s.FindMechanics(ctx, serviceID, lat, lng)
}

/* ---------------- PLACE BID ---------------- */

func (s *BiddingService) PlaceBid(
	ctx context.Context,
	serviceID, providerID string,
	price float64,
) error {

	key := "bid:" + serviceID + ":" + providerID

	old, _ := s.rdb.Get(ctx, key).Float64()
	if old != 0 && price >= old {
		return errors.New("higher bid ignored")
	}

	s.rdb.Set(ctx, key, price, 10*time.Minute)

	s.socket.EmitWithRetry(
		"user:"+serviceID,
		"bid:update",
		map[string]any{
			"providerId": providerID,
			"price": price,
		},
		1,
	)

	return nil
}

/* ---------------- ACCEPT BID ---------------- */

func (s *BiddingService) AcceptBid(
	ctx context.Context,
	serviceID, providerID string,
) error {

	lock := "reserve:" + providerID
	ok, _ := s.rdb.SetNX(ctx, lock, serviceID, 10*time.Minute).Result()
	if !ok {
		return errors.New("provider already reserved")
	}

	s.socket.EmitWithRetry(
		"provider:"+providerID,
		"bid:accepted",
		map[string]any{
			"serviceId": serviceID,
			"ttl": 600,
		},
		2,
	)

	return nil
}

/* ---------------- PROVIDER CANCEL ---------------- */

func (s *BiddingService) ProviderCancel(
	ctx context.Context,
	serviceID, providerID string,
) {

	objID, _ := primitive.ObjectIDFromHex(serviceID)
	provID, _ := primitive.ObjectIDFromHex(providerID)

	s.acceptedRepo.Col().UpdateByID(ctx, objID, bson.M{
		"$addToSet": bson.M{
			"notToSendProviders": provID,
		},
	})

	s.cancelRepo.Insert(ctx, &domain.CancellationLog{
		ServiceID: objID,
		ProviderID: provID,
		CancelledBy: "provider",
		CreatedAt: time.Now(),
	})
}
