package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"math"
)

type ServiceTrackingService struct {
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	socket       *socket.Emitter
}

func NewServiceTrackingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	socket *socket.Emitter,
) *ServiceTrackingService {
	return &ServiceTrackingService{
		acceptedRepo: acceptedRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		socket:       socket,
	}
}

/* ============================================================
                     TRACKING SCREENS
============================================================ */
func distanceKmHaversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func estimateETASeconds(distanceKm float64) int64 {
	const avgSpeed = 30.0
	return int64((distanceKm / avgSpeed) * 3600)
}

func (s *ServiceTrackingService) UserTrackingScreen(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, _ := s.userRepo.GetByID(ctx, svc.User)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(svc.Provider.Hex()))

	// ----------------------------------
	// 🧮 Calculate Distance & ETA
	// ----------------------------------
	var distanceKm float64
	var etaMinutes int64

	if svc.ProviderLocation.Lat != 0 &&
		svc.ProviderLocation.Long != 0 &&
		svc.UserLocation.Lat != 0 &&
		svc.UserLocation.Long != 0 {

		distanceKm = distanceKmHaversine(
			svc.ProviderLocation.Lat,
			svc.ProviderLocation.Long,
			svc.UserLocation.Lat,
			svc.UserLocation.Long,
		)

		etaMinutes = estimateETASeconds(distanceKm) / 60
	}

	// ----------------------------------
	// 💰 Billing
	// ----------------------------------
	gstMain := "18%"
	gst := svc.FinalPrice * 18 / 100
	total := svc.FinalPrice + gst

	// ----------------------------------
	// 📦 Response
	// ----------------------------------
	return map[string]any{
		"screen": "SERVICE_TRACKING",
		"status": svc.Status,
		"otp":    user.ServiceOTP,

		"provider": map[string]any{
			"id":         provider.ID,
			"name":       provider.Name,
			"rating":     provider.Rating,
			"etaMinutes": etaMinutes,
			"distanceKm": distanceKm,
		},

		"vehicle": map[string]any{
			"problem":       svc.ServiceType,
			"date":          time.Now().Format("2006-01-02"),
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"fuelType":      svc.FuelType,
			"year":          svc.ModelYear,
		},

		"locations": map[string]any{
			"user": map[string]any{
				"lat":  svc.UserLocation.Lat,
				"long": svc.UserLocation.Long,
			},
			"provider": map[string]any{
				"lat":  svc.ProviderLocation.Lat,
				"long": svc.ProviderLocation.Long,
			},
		},

		"billing": map[string]any{
			"serviceAmount": svc.FinalPrice,
			"gst":           gstMain,
			"totalAmount":   total,
			"currency":      "INR",
		},

		"timestamps": svc.Timestamps,
	}, nil
}


func (s *ServiceTrackingService) ProviderTrackingScreen(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, _ := s.userRepo.GetByID(ctx, svc.User)

	return map[string]any{
		"screen": "PROVIDER_TRACKING",
		"user": map[string]any{
			"name":  user.Name,
			"phone": user.Phone,
			"otp":   user.ServiceOTP,
		},
		"service": svc.ServiceType,
		"status":  svc.Status,
		"timestamps": svc.Timestamps,
	}, nil
}

/* ============================================================
                     STATUS ENGINE
============================================================ */

func (s *ServiceTrackingService) UpdateStatus(
	ctx context.Context,
	serviceID string,
	newStatus domain.ServiceStatus,
	lat float64,long float64,
) error {

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}
	log.Printf(
		"STATE CHANGE REQUEST service=%s from=%s to=%s\n",
		serviceID,
		svc.Status,
		newStatus,
	)

	// Validate flow
	if !domain.CanTransition(svc.Status, newStatus) {
		return errors.New("invalid service state transition")
	}

	now := time.Now()

	update := bson.M{
		"status":    newStatus,
		"updatedAt": now,
	}
	if lat != 0 && long != 0 {
		update["providerLocation"] = bson.M{
			"lat":       lat,
			"long":      long,
			"updatedAt": now,
		}
	}

	ts := bson.M{}

	switch newStatus {
	case domain.StatusStarted:
		ts["jobStartedAt"] = now

	case domain.StatusReachedLocation:
		ts["reachedAt"] = now

	case domain.StatusOTPVerified:
		ts["otpVerifiedAt"] = now

	case domain.StatusInProgress:
		ts["startedAt"] = now

	case domain.StatusCompleted:
		ts["completedAt"] = now
	}

	if len(ts) > 0 {
		update["timestamps"] = ts
	}

	if _, err := s.acceptedRepo.Col().
		UpdateByID(ctx, objID, bson.M{"$set": update}); err != nil {
		return err
	}

	// SOCKET
	userRoom := "user:" + svc.ID.Hex()
	providerRoom := "provider:" + svc.Provider.Hex()

	payload := map[string]any{
		"serviceId": svc.ID.Hex(),
		"status":    newStatus,
		"timestamps": ts,
	}

	s.socket.EmitWithRetry(userRoom, "service:status_update", payload, 1)
	s.socket.EmitWithRetry(providerRoom, "service:status_update", payload, 1)

	return nil
}


func (s *ServiceTrackingService) VerifyOTP(
	ctx context.Context,
	serviceID string,
	inputOTP string,
	lat float64,
	long float64,
) error {

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return err
	}

	if user.ServiceOTP != inputOTP {
		return errors.New("invalid otp")
	}

	now := time.Now()

	// ✅ Only update OTP fields here
	if _, err := s.acceptedRepo.Col().UpdateByID(
		ctx,
		objID,
		bson.M{
			"$set": bson.M{
				"otp.verified":   true,
				"otp.verifiedAt": now,
			},
		},
	); err != nil {
		return err
	}

	// ✅ Status + location handled centrally
	return s.UpdateStatus(
		ctx,
		serviceID,
		domain.StatusOTPVerified,
		lat,
		long,
	)
}

