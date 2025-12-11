package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/validation"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServiceRequestService struct {
	serviceRepo ports.ServiceRequestRepository
	vehicleRepo ports.SavedVehicleRepository
}

func NewServiceRequestService(
	serviceRepo ports.ServiceRequestRepository,
	vehicleRepo ports.SavedVehicleRepository,
) *ServiceRequestService {
	return &ServiceRequestService{
		serviceRepo: serviceRepo,
		vehicleRepo: vehicleRepo,
	}
}

func (s *ServiceRequestService) CreateServiceRequest(
	ctx context.Context,
	userID string,
	// userLocation []float64,
	req validation.ServiceRequestRequest,
) (*domain.ServiceRequest, error) {

	objUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	existingVehicle, _ := s.vehicleRepo.FindByUserAndVehicleNumber(ctx, userID, req.VehicleNumber)

	if existingVehicle == nil {
		vehicle := &domain.SavedVehicleData{
			UserID:        objUserID,
			VehicleNumber: req.VehicleNumber,
			Brand:         req.Brand,
			Model:         req.Model,
			Year:          req.Year,
			FuelType:      req.FuelType,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_ = s.vehicleRepo.Create(ctx, vehicle)
	}

	var serviceRequest *domain.ServiceRequest

	if req.RequestID != "" {
		serviceRequest, err = s.serviceRepo.FindByID(ctx, req.RequestID)
		if err != nil {
			return nil, err
		}

		expiresAt := time.Now().Add(60 * time.Second)
		serviceRequest.ExpiresAt = expiresAt
		serviceRequest.UpdatedAt = time.Now()

		if err := s.serviceRepo.Update(ctx, serviceRequest); err != nil {
			return nil, err
		}

		return serviceRequest, nil
	}

	now := time.Now()
	expiresAt := now.Add(60 * time.Second)
	basePrice := 200.0

	location := &domain.GeoJSONLocation{
		Type:        "Point",
		// Coordinates: userLocation,
	}

	serviceRequest = &domain.ServiceRequest{
		User:          objUserID,
		VehicleNumber: req.VehicleNumber,
		VehicleType:   req.VehicleType,
		Brand:         req.Brand,
		Model:         req.Model,
		Year:          req.Year,
		FuelType:      req.FuelType,
		ServiceType:   req.ServiceType,
		Problems:      req.Problems,
		Description:   req.Description,
		Location:      location,
		Address:       req.Address,
		ScheduledDate: req.ScheduledDate,
		BasePrice:     basePrice,
		Status:        "pending",
		TotalBids:     0,
		ExpiresAt:     expiresAt,
		Metadata: domain.ServiceMetadata{
			BroadcastedTo: 0,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.serviceRepo.Create(ctx, serviceRequest); err != nil {
		return nil, err
	}

	return serviceRequest, nil
}