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

/* ---------------- USER TRACKING SCREEN ---------------- */
func (s *ServiceTrackingService) UserTrackingScreen(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	objID, _ := primitive.ObjectIDFromHex(serviceID)

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, _ := s.userRepo.GetByID(ctx, svc.User)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(svc.Provider.Hex()))

	return map[string]any{
		"screen": "SERVICE_TRACKING",
		"otp":    user.ServiceOTP, // 🔥 SAME OTP ALWAYS
		"status": svc.Status,

		"provider": map[string]any{
			"id":    provider.ID,
			"name":  provider.Name,
			"phone": provider.Phone,
		},

		"locations": map[string]any{
			"user":     svc.ServiceLocation,
			"provider": svc.ProviderLocation,
		},
	}, nil
}

/* ---------------- PROVIDER TRACKING SCREEN ---------------- */
func (s *ServiceTrackingService) ProviderTrackingScreen(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	objID, _ := primitive.ObjectIDFromHex(serviceID)

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
	}, nil
}

/* ---------------- VERIFY OTP ---------------- */
func (s *ServiceTrackingService) VerifyOTP(
	ctx context.Context,
	serviceID string,
	inputOTP string,
) error {

	objID, _ := primitive.ObjectIDFromHex(serviceID)

	var svc domain.AcceptedService
	_ = s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc)

	user, _ := s.userRepo.GetByID(ctx, svc.User)

	if user.ServiceOTP != inputOTP {
		return errors.New("invalid otp")
	}

	now := time.Now()

	_, _ = s.acceptedRepo.Col().UpdateByID(
		ctx,
		objID,
		bson.M{
			"$set": bson.M{
				"otp.verified":   true,
				"otp.verifiedAt": now,
				"status":         "STARTED",
			},
		},
	)

	// 🔴 SOCKET UPDATE TO BOTH
	room := svc.ServiceRequest.Hex()
	s.socket.EmitWithRetry(room, "otp:verified", true, 3)

	return nil
}
