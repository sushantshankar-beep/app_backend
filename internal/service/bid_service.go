package service

import (
	"context"
	// "errors"
	"time"
	"fmt"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BiddingService struct {
	rdb          *redis.Client
	socket       *socket.Emitter
	acceptedRepo *repository.AcceptedServiceRepo
	cancelRepo   *repository.CancellationRepo
	bidRepo      *repository.BidRepo
	counterRepo  *repository.CounterRepo
}

func NewBiddingService(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
	cancelRepo *repository.CancellationRepo,
	bidRepo *repository.BidRepo,
	counterRepo *repository.CounterRepo,
) *BiddingService {
	return &BiddingService{
		rdb:          rdb,
		socket:       socket,
		acceptedRepo: acceptedRepo,
		cancelRepo:   cancelRepo,
		bidRepo:      bidRepo,
		counterRepo:  counterRepo,
	}
}

// ================= START SEARCH =================
func (s *BiddingService) StartSearch(
	ctx context.Context,
	userID domain.UserID,
	serviceType string,
	issues []string,
	lat, lng float64,
) (string, error) {

	userOID, _ := primitive.ObjectIDFromHex(string(userID))

	seq, err := s.counterRepo.Next(ctx, "service")
	if err != nil {
		return "", err
	}

	serviceNumber := fmt.Sprintf("BK-%05d", seq)

	svc := &domain.AcceptedService{
		ServiceNumber: serviceNumber,
		User:          userOID,
		Status:        domain.StatusSearching,
		PaymentStatus: domain.PaymentPending,
		ServiceType:   serviceType,
		Issues:        issues,
		ServiceLocation: &domain.Location{
			Latitude:  lat,
			Longitude: lng,
		},
		RetryCount: 0,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.acceptedRepo.Create(ctx, svc); err != nil {
		return "", err
	}

	go s.findProviders(svc.ID.Hex(), lat, lng, serviceType)
	return svc.ID.Hex(), nil
}

// ================= PLACE BID =================
func (s *BiddingService) PlaceBid(ctx context.Context,serviceID string,providerID string,price float64) (string, error) {
	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return "", err
	}
	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return "", err
	}
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
	s.socket.EmitWithRetry(
		"user:"+serviceID,
		"bid:update",
		map[string]any{
			"bidId":      bidOID.Hex(),
			"providerId": providerID,
			"price":      price,
		},
		2,
	)

	return bidOID.Hex(), nil
}

// ================= ACCEPT BID =================
func (s *BiddingService) AcceptBid(ctx context.Context,serviceID string,bidID string,providerID string,price float64) error {

	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
	bidOID, _ := primitive.ObjectIDFromHex(bidID)
	providerOID, _ := primitive.ObjectIDFromHex(providerID)

	err := s.acceptedRepo.UpdateByID(
		ctx,
		serviceOID,
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
	s.socket.EmitWithRetry(
		"provider:"+providerID,
		"bid:accepted",
		map[string]any{
			"serviceId": serviceID,
			"price":     price,
		},
		2,
	)
	return nil
}
// ================= FIND PROVIDERS =================
func (s *BiddingService) findProviders(serviceID string,lat, lng float64,serviceType string) {
	ctx := context.Background()
	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
	radiusSteps := []float64{5, 10, 20, 50}
	for _, radius := range radiusSteps {
		var svc domain.AcceptedService
		if err := s.acceptedRepo.Col().FindOne(ctx, bson.M{"_id": serviceOID}).Decode(&svc); err != nil {
			return
		}
		if svc.Status != "searching" {
			return
		}
		providers, _ := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			lng, lat,
			&redis.GeoRadiusQuery{
				Radius: radius,
				Unit:   "km",
			},
		).Result()
		for _, p := range providers {
			s.socket.EmitWithRetry(
				"provider:"+p.Name,
				"bid:request",
				map[string]any{
					"serviceId":   serviceID,
					"serviceType": serviceType,
					"radius":      radius,
				},
				1,
			)
		}

		time.Sleep(6 * time.Second)

		_, _ = s.acceptedRepo.Col().UpdateByID(
			ctx,
			serviceOID,
			bson.M{"$inc": bson.M{"retryCount": 1}},
		)
	}
	_, _ = s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": bson.M{"status": "no_provider_found"}},
	)
}
