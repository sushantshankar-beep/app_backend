package service

import (
	"context"
	"errors"

	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/utils"

	"app_backend/internal/dto"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	fmt.Println("this is service type", svc.ServiceType)
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
			"problem":       svc.ServiceType,
			"date":          time.Now().Format("2006-01-02"),
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"fuelType":      svc.FuelType,
			"year":          svc.ModelYear,
		},

		"billing": map[string]any{
			"serviceAmount": svc.FinalPrice,
			"gst":           gst,
			"totalAmount":   total,
			"currency":      "INR",
		},
	}, nil
}

func (s *BookingService) GetUserBookings(ctx context.Context, userID, status string) ([]dto.UserBookingDTO, error) {

	sStatus, err := mapStatus(status)
	if err != nil {
		return nil, err
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userObjID)
	if err != nil {
		return nil, err
	}

	raw, err := s.acceptedRepo.GetBookingsByUserAndStatus(ctx, userID, sStatus)

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	result := make([]dto.UserBookingDTO, 0, len(raw))

	for _, r := range raw {
		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(r.Provider.Hex()))
		if err != nil {
			return nil, err
		}
		result = append(result, dto.UserBookingDTO{
			ID:            r.ID,
			UserID:        string(user.ID),
			ServiceNumber: r.ServiceNumber,
			Status:        string(r.Status),
			FinalPrice:    r.FinalPrice,
			UserName:      user.Name,
			ProviderName:  provider.Name,
			VehicleType:   r.VehicleType,
			Ratings:       "",
			CreatedAt:     r.CreatedAt,
			Issues:        r.Issues,
		})
	}

	return result, nil
}

func mapStatus(status string) (domain.ServiceStatus, error) {
	switch status {
	case
		"created",
		"assigned",
		"started",
		"reached_location",
		"otp_verified",
		"in_progress",
		"ongoing":
		return domain.StatusStarted, nil

	case "completed":
		return domain.StatusCompleted, nil

	case "cancelled":
		return domain.StatusCancelled, nil

	default:
		return "", errors.New("invalid status")
	}
}

func (s *BookingService) GetUserBookingDetails(ctx context.Context, userID, serviceID string) (*dto.UserBookingDetailDTO, error) {

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	serviceObjID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userObjID)
	if err != nil {
		return nil, err
	}

	r, err := s.acceptedRepo.GetByID(ctx, serviceObjID)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(r.Provider.Hex()),
	)
	if err != nil {
		return nil, err
	}

	const gstPercent = 18.0

	serviceCharge := r.FinalPrice
	gstAmount := (serviceCharge * gstPercent) / 100
	totalPayable := serviceCharge + gstAmount

	return &dto.UserBookingDetailDTO{
		ID:            r.ID,
		UserID:        string(user.ID),
		ServiceNumber: r.ServiceNumber,
		Status:        string(r.Status),
		FinalPrice:    r.FinalPrice,
		VehicleNumber: r.VehicleNumber,
		Brand:         r.Brand,
		Model:         r.Model,
		ModelYear:     r.ModelYear,
		FuelType:      r.FuelType,
		VehicleType:   r.VehicleType,
		Issues:        r.Issues,
		Timestamps:    r.Timestamps,
		UserName:      user.Name,
		ProviderName:  provider.Name,
		Billing: dto.BillingDetailsDTO{
			ServiceCharge: utils.RoundTo2(serviceCharge),
			GSTPercent:    gstPercent,
			GSTAmount:     utils.RoundTo2(gstAmount),
			TotalPayable:  utils.RoundTo2(totalPayable),
			PaymentStatus: string(r.PaymentStatus),
		},
		UserLocation: dto.UserLocation{
			Lat:  r.UserLocation.Lat,
			Long: r.UserLocation.Long,
		},
		ProviderLocation: dto.ProviderLocation{
			Lat:  r.ProviderLocation.Lat,
			Long: r.ProviderLocation.Long,
		},
		CreatedAt: r.CreatedAt,
	}, nil

}

func (s *BookingService) GetProviderBookings(ctx context.Context, providerID, status string) (*dto.ProviderBookingResponse, error) {

	sStatus, err := mapStatus(status)
	if err != nil {
		return nil, err
	}

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(providerObjID.Hex()),
	)
	if err != nil {
		return nil, err
	}

	raw, err := s.acceptedRepo.
		GetBookingsByProviderAndStatus(ctx, providerID, sStatus)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ProviderBookingDTO, 0, len(raw))

	for _, r := range raw {

		user, err := s.userRepo.GetByID(ctx, r.User)
		if err != nil {
			return nil, err
		}

		result = append(result, dto.ProviderBookingDTO{
			ID:            r.ID,
			ProviderID:    string(provider.ID),
			ServiceNumber: r.ServiceNumber,
			Status:        string(r.Status),
			FinalPrice:    r.FinalPrice,
			ProviderName:  provider.Name,
			UserName:      user.Name,
			VehicleNumber: r.VehicleNumber,
			Brand:         r.Brand,
			Model:         r.Model,
			ModelYear:     r.ModelYear,
			VehicleType:   r.VehicleType,
			Issues:        r.Issues,
			Ratings:       "",
			CreatedAt:     r.CreatedAt,
		})
	}

	response := &dto.ProviderBookingResponse{
		Bookings: result,
	}

	if sStatus == domain.StatusCompleted ||
		sStatus == domain.StatusCancelled {
		response.Count = len(raw)
	}

	return response, nil
}

func (s *BookingService) GetProviderBookingDetails(ctx context.Context, providerID, serviceID string) (*dto.ProviderBookingDetailDTO, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	serviceObjID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil, err
	}

	r, err := s.acceptedRepo.GetByID(ctx, serviceObjID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, r.User)
	if err != nil {
		return nil, err
	}

	const (
		gstPercent        = 18.0
		commissionPercent = 20.0
	)

	serviceCharge := r.FinalPrice
	commissionAmount := (serviceCharge * commissionPercent) / 100
	gstOnCommission := (commissionAmount * gstPercent) / 100
	providerPayout := serviceCharge - commissionAmount - gstOnCommission

	return &dto.ProviderBookingDetailDTO{
		ID:            r.ID,
		ProviderID:    string(provider.ID),
		ServiceNumber: r.ServiceNumber,
		Status:        string(r.Status),
		FinalPrice:    r.FinalPrice,
		VehicleNumber: r.VehicleNumber,
		Brand:         r.Brand,
		Model:         r.Model,
		ModelYear:     r.ModelYear,
		FuelType:      r.FuelType,
		VehicleType:   r.VehicleType,
		Issues:        r.Issues,
		Timestamps:    r.Timestamps,
		ProviderName:  provider.Name,
		UserName:      user.Name,
		Billing: dto.BillingDetailsDTO{
			ServiceCharge:     utils.RoundTo2(serviceCharge),
			CommissionPercent: commissionPercent,
			CommissionAmount:  utils.RoundTo2(commissionAmount),
			GSTPercent:        gstPercent,
			GSTAmount:         utils.RoundTo2(gstOnCommission),
			TotalPayable:      serviceCharge,
			ProviderPayout:    utils.RoundTo2(providerPayout),
			PaymentStatus:     string(r.PaymentStatus),
		},
		UserLocation: dto.UserLocation{
			Lat:  r.UserLocation.Lat,
			Long: r.UserLocation.Long,
		},
		ProviderLocation: dto.ProviderLocation{
			Lat:  r.ProviderLocation.Lat,
			Long: r.ProviderLocation.Long,
		},
		CreatedAt: r.CreatedAt,
	}, nil

}
