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

/* ============================================================
                     TRACKING SCREENS
============================================================ */

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

	return map[string]any{
		"screen": "SERVICE_TRACKING",
		"status": svc.Status,
		"otp":    user.ServiceOTP,

		"provider": provider,

		"locations": map[string]any{
			"user":     svc.ServiceLocation,
			"provider": svc.ProviderLocation,
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

	// Validate flow
	if !domain.CanTransition(svc.Status, newStatus) {
		return errors.New("invalid service state transition")
	}

	now := time.Now()

	update := bson.M{
		"status":    newStatus,
		"updatedAt": now,
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


/* ============================================================
                     OTP VERIFY
============================================================ */

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
				"status":"otp_verified",
			},
		},
	)

	return s.UpdateStatus(ctx, serviceID, domain.StatusOTPVerified)
}
