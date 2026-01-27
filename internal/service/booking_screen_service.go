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
	transactionRepo *repository.PaymentRepository
}

func NewBookingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	catalogRepo *repository.ServiceCatalogRepo,
	transactionRepo *repository.PaymentRepository,
) *BookingService {
	return &BookingService{
		acceptedRepo: acceptedRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		catalogRepo:  catalogRepo,
		transactionRepo: transactionRepo,
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
		if err.Error() == "not found" {
			return []dto.UserBookingDTO{}, nil
		}
		return nil, err
	}

	raw, err := s.acceptedRepo.GetBookingsByUserAndStatus(ctx, userObjID, sStatus)
	if err != nil {
		return nil, err
	}

	result := make([]dto.UserBookingDTO, 0, len(raw))

	for _, r := range raw {
		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(r.Provider.Hex()))
		
		if err != nil {
			continue
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
			UpdatedAt:     r.UpdatedAt,
		})
	}

	return result, nil
}
func mapStatus(status string) ([]domain.ServiceStatus, error) {
	switch status {

	case "ongoing":
		return []domain.ServiceStatus{
			domain.StatusStarted,
			domain.StatusReachedLocation,
			domain.StatusOTPVerified,
			domain.StatusInProgress,
		}, nil

	case "completed":
		return []domain.ServiceStatus{
			domain.StatusCompleted,
		}, nil

	case "cancelled":
		return []domain.ServiceStatus{
			domain.StatusCancelled,
		}, nil

	default:
		return nil, errors.New("invalid status")
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
		UpdatedAt: r.UpdatedAt,
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

	raw, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, sStatus)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ProviderBookingDTO, 0, len(raw))

	for _, r := range raw {
		userObjID, err := primitive.ObjectIDFromHex(r.User.Hex())
		if err != nil {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, userObjID)
		if err != nil {
			continue
		}

		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(r.Provider.Hex()))
		if err != nil {
			continue
		}

		result = append(result, dto.ProviderBookingDTO{
			ID:            r.ID,
			ProviderID:    providerID,
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
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		})
	}

	response := &dto.ProviderBookingResponse{
		Bookings: result,
	}

	if containsStatus(sStatus, domain.StatusCompleted) || containsStatus(sStatus, domain.StatusCancelled) {
		response.Count = len(result)
	}

	return response, nil
}

func containsStatus(statuses []domain.ServiceStatus, target domain.ServiceStatus) bool {
	for _, s := range statuses {
		if s == target {
			return true
		}
	}
	return false
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
		UpdatedAt: r.UpdatedAt,
	}, nil

}

func (s *BookingService) GetUserExpenses(ctx context.Context, userID string) ([]dto.UserExpenseDTO, float64, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, err
	}

	_, err = s.userRepo.GetByID(ctx, userObjID)
	if err != nil {
		return nil, 0, err
	}

	services, err := s.acceptedRepo.GetCompletedServicesByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.UserExpenseDTO, 0, len(services))
	var totalExpense float64

	for _, service := range services {
		transaction, err := s.transactionRepo.GetTransactionByServiceID(ctx, service.ID.Hex())
		if err != nil {
			continue
		}

		if transaction.Status != string(domain.PaymentPaid) {
			continue
		}

		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(service.Provider.Hex()))
		if err != nil {
			return nil, 0, err
		}

		expense := dto.UserExpenseDTO{
			ServiceID:     service.ID.Hex(),
			ServiceNumber: service.ServiceNumber,
			ServiceType:   service.ServiceType,
			ProviderName:  provider.Name,
			Amount:        transaction.Amount,
			PaymentMethod: transaction.Method,
			CreatedAt:     service.CreatedAt,
			VehicleType:   service.VehicleType,
			VehicleNumber: service.VehicleNumber,
			Issues: service.Issues,
		}

		result = append(result, expense)
		totalExpense += transaction.Amount
	}

	return result, totalExpense, nil
}

func (s *BookingService) GetProviderDashboard(ctx context.Context, providerID string) (*dto.DashboardStats, error) {
	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	_, err = s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil, err
	}

	completedBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted},)
	if err != nil {
		return nil, err
	}

	cancelledBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID,[]domain.ServiceStatus{domain.StatusCancelled},)
	if err != nil {
		return nil, err
	}

	allTimeEarning := 0.0
	todayEarning := 0.0
	today := time.Now().Truncate(24 * time.Hour)

	for _, booking := range completedBookings {
		allTimeEarning += booking.FinalPrice
		if booking.CreatedAt.Truncate(24 * time.Hour).Equal(today) {
			todayEarning += booking.FinalPrice
		}
	}

	stats := &dto.DashboardStats{
		AllTimeEarning:    allTimeEarning,
		TodayEarning:      todayEarning,
		ServicesCompleted: len(completedBookings),
		PaymentSettlement:  0,
		CancelledServices: len(cancelledBookings),
	}

	return stats, nil
}

func (s *BookingService) GetProviderEarnings(ctx context.Context, providerID string) (*dto.EarningsResponse, error) {
	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	_, err = s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil, err
	}

	completedBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted},)
	if err != nil {
		return nil, err
	}

	earnings := make([]dto.EarningDetail, 0, len(completedBookings))

	for _, booking := range completedBookings {
		serviceName := ""
		if len(booking.Issues) > 0 {
			serviceName = booking.Issues[0]
		}

		earnings = append(earnings, dto.EarningDetail{
			ID:          booking.ID.Hex(),
			ProviderId:  booking.Provider,
			ServiceName: serviceName,
			Amount:      booking.FinalPrice,
			CreatedAt:   booking.CreatedAt.Format(time.RFC3339),
		})
	}

	return &dto.EarningsResponse{Earnings: earnings}, nil
}