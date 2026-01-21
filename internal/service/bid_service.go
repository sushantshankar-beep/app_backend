package service
import (
	"context"
	"fmt"
	"time"

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

/* ================= START SEARCH ================= */

func (s *BiddingService) StartSearch(
	ctx context.Context,
	userID domain.UserID,
	serviceType string,
	issues []string,
	lat, lng float64,
) (string, error) {


	userOID, _ := primitive.ObjectIDFromHex(string(userID))
	seq, _ := s.counterRepo.Next(ctx, "service")

	serviceNumber := fmt.Sprintf("VHBK%05d", seq)

	svc := &domain.AcceptedService{
		ServiceNumber: serviceNumber,
		User:          userOID,
		Status:        domain.StatusSearching,
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

	go s.findProviders(svc.ID.Hex(), lat, lng, serviceType, issues)
	return svc.ID.Hex(), nil
}

/* ================= PLACE BID ================= */

func (s *BiddingService) PlaceBid(
	ctx context.Context,
	serviceID, providerID string,
	price float64,
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

	// 🔹 provider meta
	meta, _ := s.rdb.HGetAll(ctx, "provider:meta:"+providerID).Result()
	distance, _ := s.rdb.Get(ctx, "service:dist:"+serviceID+":"+providerID).Float64()
	eta := estimateETA(distance)

	s.socket.Emit(
		"user:"+serviceID,
		"bid:update",
		map[string]any{
			"bidId": bidOID.Hex(),
			"price": price,
			"provider": map[string]any{
				"id":       providerID,
				"name":     meta["name"],
				"rating":   meta["rating"],
				"distance": distance,
				"etaMin":   eta,
			},
		},
	)

	return bidOID.Hex(), nil
}

/* ================= FIND PROVIDERS ================= */

func (s *BiddingService) findProviders(
	serviceID string,
	lat, lng float64,
	serviceType string,
	issues []string,
) {
	ctx := context.Background()

	providers, _ := s.rdb.GeoRadius(
		ctx,
		"providers:geo",
		lng, lat,
		&redis.GeoRadiusQuery{
			Radius: 10,
			Unit:   "km",
			WithDist: true,
		},
	).Result()

	for _, p := range providers {
		providerID := p.Name
		distance := p.Dist
		eta := estimateETA(distance)

		// cache distance
		s.rdb.Set(ctx,
			"service:dist:"+serviceID+":"+providerID,
			distance,
			15*time.Minute,
		)

		s.socket.Emit(
			"provider:"+providerID,
			"bid:request",
			map[string]any{
				"serviceId":   serviceID,
				"serviceType": serviceType,
				"issues":      issues,
				"distance":    distance,
				"etaMin":      eta,
			},
		)
	}
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

	// update accepted service
	if err := s.acceptedRepo.UpdateByID(
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
	); err != nil {
		return err
	}

	// notify provider
	s.socket.EmitWithRetry(
		"provider:"+providerID,
		"bid:accepted",
		map[string]any{
			"serviceId": serviceID,
			"price":     price,
		},
		1,
	)

	return nil
}

/* ================= HELPERS ================= */

func estimateETA(distanceKm float64) int {
	avgSpeed := 30.0 // km/h
	return int((distanceKm / avgSpeed) * 60)
}
