package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

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

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return nil, errors.New("user not found")
	}

	/* ---------------- Load Provider ---------------- */

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(svc.Provider.Hex()),
	)
	if err != nil {
		return nil, errors.New("provider not found")
	}

	/* ---------------- Load Service Catalog ---------------- */

	catalog, err := s.catalogRepo.FindByName(ctx, svc.ServiceType)
	if err != nil {
		return nil, errors.New("service catalog not found")
	}

	/* ---------------- Price Calculation ---------------- */

	gst := svc.FinalPrice * catalog.GSTPercent / 100
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
			"bookingId": svc.NumericID,
			"status":    "BID_ACCEPTED",
		},

		"user": map[string]any{
			"name":  user.Name,
			"phone": user.Phone,
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
		},

		"billing": map[string]any{
			"basePrice":   catalog.BasePrice,
			"discount":    catalog.Discount,
			"coupon":      catalog.CouponAmount,
			"subtotal":    svc.FinalPrice,
			"gst":         gst,
			"totalAmount": total,
			"currency":    "INR",
		},
	}, nil
}
