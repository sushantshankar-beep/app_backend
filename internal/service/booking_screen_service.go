package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"fmt"
)

type BookingService struct {
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	catalogRepo  *repository.ServiceCatalogRepo
}

func NewBookingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	catalogRepo *repository.ServiceCatalogRepo,
) *BookingService {
	return &BookingService{
		acceptedRepo: acceptedRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		catalogRepo:  catalogRepo,
	}
}

func (s *BookingService) BuildBookingScreen(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	/* ---------------- Load Accepted Service ---------------- */

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, errors.New("invalid service id")
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	/* ---------------- Load User ---------------- */

	/* ---------------- Load Provider ---------------- */

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(svc.Provider.Hex()),
	)
	if err != nil {
		return nil, errors.New("provider not found")
	}

	/* ---------------- Load Service Catalog ---------------- */
    fmt.Println("this is service type",svc.ServiceType)
	if len(svc.Issues) == 0 {
		return nil, errors.New("no issues attached to service")
	}

	/* ---------------- Price Calculation ---------------- */

	gst := svc.FinalPrice * 18 / 100
	total := svc.FinalPrice + gst

	/* ---------------- Build Screen Payload ---------------- */

	return map[string]any{
		"screen": "BOOKING_DETAILS",

		"primaryButton": map[string]any{
			"label":  "Proceed to Payment",
			"action": "REDIRECT",
			"url":    "/payment/initiate?serviceId=" + svc.ID.Hex(),
		},

		"secondaryButton": map[string]any{
			"label":  "Go Back",
			"action": "BACK",
		},

		"booking": map[string]any{
			"bookingId": svc.ServiceNumber,
			"status":    "BID_ACCEPTED",
		},

		"provider": map[string]any{
			"id":         provider.ID,
			"name":       provider.Name,
			"rating":     provider.Rating,
			"etaMinutes": 6,
		},

		"vehicle": map[string]any{
			"problem": svc.ServiceType,
			"date":    time.Now().Format("2006-01-02"),
			"vehicleNumber": svc.VehicleNumber,
			"brand" : svc.Brand,
			"fuelType": svc.FuelType,
			"year":svc.ModelYear,
		},

		"billing": map[string]any{
			"serviceAmount":    svc.FinalPrice,
			"gst":         gst,
			"totalAmount": total,
			"currency":    "INR",
		},
	}, nil
}


func (s *BookingService) GetUserBookings(ctx context.Context, userID, status string) ([]domain.AcceptedService, error) {
	sStatus, err := mapStatus(status)
	if err != nil {
		return nil, err
	}

	bookings, err := s.acceptedRepo.GetBookingsByUserAndStatus(ctx, userID, sStatus)
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func mapStatus(status string) (domain.ServiceStatus, error) {
	switch status {
	case "ongoing":
		return domain.StatusStarted, nil
	case "completed":
		return domain.StatusCompleted, nil
	case "cancelled":
		return domain.StatusCancelled, nil
	default:
		return "", errors.New("invalid status")
	}
}