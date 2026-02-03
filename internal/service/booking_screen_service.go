package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/dto"
	"app_backend/internal/repository"
	"app_backend/internal/utils"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "fmt"
)

type BookingService struct {
	acceptedRepo    *repository.AcceptedServiceRepo
	userRepo        *repository.UserRepo
	providerRepo    *repository.ProviderRepo
	catalogRepo     *repository.ServiceCatalogRepo
	transactionRepo *repository.PaymentRepository
	settlementRepo  *repository.SettlementHistoryRepository
	complaintRepo   *repository.ComplaintRepo
}

func NewBookingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	catalogRepo *repository.ServiceCatalogRepo,
	transactionRepo *repository.PaymentRepository,
	settlementRepo *repository.SettlementHistoryRepository,
	complaintRepo *repository.ComplaintRepo,
) *BookingService {
	return &BookingService{
		acceptedRepo:    acceptedRepo,
		userRepo:        userRepo,
		providerRepo:    providerRepo,
		catalogRepo:     catalogRepo,
		transactionRepo: transactionRepo,
		settlementRepo:  settlementRepo,
		complaintRepo:   complaintRepo,
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
			"profileUrl":provider.ProfileURL,
		},

		"vehicle": map[string]any{
			"problem":       svc.ServiceType,
			"date":          time.Now().Format("2006-01-02 3:04 PM"),
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"fuelType":      svc.FuelType,
			"year":          svc.ModelYear,
			"model":         svc.Model,
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

		tx, err := s.transactionRepo.GetTransactionByServiceID(ctx, r.ID.Hex())
		if err != nil {
			continue
		}

		var cancelled *domain.CancelInfo
		var cancelledAt *time.Time

		if r.Cancelled != nil && r.CancelledAt != nil {
			cancelled = r.Cancelled
			cancelledAt = r.CancelledAt
		}

		dtoItem := dto.UserBookingDTO{
			ID:            r.ID,
			UserID:        string(user.ID),
			ProfileURL:    user.ImageUrl,
			ServiceNumber: r.ServiceNumber,
			Status:        string(r.Status),
			FinalPrice:    tx.Amount,
			UserName:      user.Name,
			ProviderName:  provider.Name,
			ProviderProfileUrl: provider.ProfileURL,
			VehicleType:   r.VehicleType,
			Ratings:       "",
			Issues:        r.Issues,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			Cancelled:   cancelled,
			CancelledAt: cancelledAt,
		}
		
		result = append(result, dtoItem)
		
	}

	return result, nil
}
func mapStatus(status string) ([]domain.ServiceStatus, error) {
	switch status {

	case "ongoing":
		return []domain.ServiceStatus{
			domain.StatusConfirmed,
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

func (s *BookingService) GetUserBookingDetails(
	ctx context.Context,
	userID,
	serviceID string,
) (*dto.UserBookingDetailDTO, error) {

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

	var providerName string
	var providerID domain.ProviderID
	var providerProfileUrl string

	if r.Provider != primitive.NilObjectID {
		provider, err := s.providerRepo.FindByID(
			ctx,
			domain.ProviderID(r.Provider.Hex()),
		)
		if err != nil {
			return nil, err
		}
		providerName = provider.Name
		providerID = provider.ID
		providerProfileUrl = provider.ProfileURL
	}

	var complaintDTO *dto.ComplaintDTO

	if r.ComplaintUser != nil {
		complaint, err := s.complaintRepo.FindByAcceptedServiceId(
			ctx,
			serviceObjID,
		)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}

		if complaint != nil {
			var remark string
			if complaint.Assessment != nil {
				remark = complaint.Assessment.RemarkForUser
			}

			complaintDTO = &dto.ComplaintDTO{
				ID:              complaint.ID.Hex(),
				ComplaintUserID: complaint.ID.Hex(),
				ComplaintNumber: complaint.ComplaintNumber,
				Status:          string(complaint.Status),
				Timeline:        complaint.Timeline,
				Remark:          remark,
				CreatedAt:       complaint.CreatedAt,
				UpdatedAt:       complaint.UpdatedAt,
			}

			if complaint.UserComplaint != nil {
				complaintDTO.UserIssue = &dto.UserComplaintDTO{
					Problem:  complaint.UserComplaint.Problem,
					Photos:   complaint.UserComplaint.Photos,
					RaisedAt: complaint.UserComplaint.RaisedAt,
				}
			}
		}
	}

	const gstPercent = 18.0

	serviceCharge := r.FinalPrice
	gstAmount := (serviceCharge * gstPercent) / 100
	totalPayable := serviceCharge + gstAmount

	var userLoc dto.UserLocation
	if r.UserLocation != nil {
		userLoc = dto.UserLocation{
			Lat:  r.UserLocation.Lat,
			Long: r.UserLocation.Long,
		}
	}

	var providerLoc dto.ProviderLocation
	if r.ProviderLocation != nil {
		providerLoc = dto.ProviderLocation{
			Lat:  r.ProviderLocation.Lat,
			Long: r.ProviderLocation.Long,
		}
	}

	var cancelled *domain.CancelInfo
	var cancelledAt *time.Time

	if r.Cancelled != nil && r.CancelledAt != nil {
		cancelled = r.Cancelled
		cancelledAt = r.CancelledAt
	}

	return &dto.UserBookingDetailDTO{
		ID:                 r.ID,
		UserID:             string(user.ID),
		ServiceNumber:      r.ServiceNumber,
		Status:             string(r.Status),
		FinalPrice:         r.FinalPrice,
		ProfileURL:         user.ImageUrl,
		VehicleNumber:      r.VehicleNumber,
		Brand:              r.Brand,
		Model:              r.Model,
		ModelYear:          r.ModelYear,
		FuelType:           r.FuelType,
		VehicleType:        r.VehicleType,
		Issues:             r.Issues,
		Timestamps:         r.Timestamps,
		UserName:           user.Name,
		ProviderID:         providerID,
		ProviderName:       providerName,
		ProviderProfileUrl: providerProfileUrl,
		Billing: dto.BillingDetailsDTO{
			ServiceCharge: utils.RoundTo2(serviceCharge),
			GSTPercent:    gstPercent,
			GSTAmount:     utils.RoundTo2(gstAmount),
			TotalPayable:  utils.RoundTo2(totalPayable),
			PaymentStatus: string(r.PaymentStatus),
		},
		Cancelled:        cancelled,
		CancelledAt:      cancelledAt,
		Complaint:        complaintDTO,
		UserLocation:     userLoc,
		ProviderLocation: providerLoc,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
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

		tx, err := s.transactionRepo.GetTransactionByServiceID(ctx, r.ID.Hex())
		if err != nil {
			continue
		}


		var cancelled *domain.CancelInfo
		var cancelledAt *time.Time

		if r.Cancelled != nil && r.CancelledAt != nil {
			cancelled = r.Cancelled
			cancelledAt = r.CancelledAt
		}

		dtoItem := dto.ProviderBookingDTO{
			ID:            r.ID,
			ProviderID:    providerID,
			ProfileURL:     user.ImageUrl,
			ServiceNumber: r.ServiceNumber,
			Status:        string(r.Status),
			FinalPrice:    tx.Amount,
			ProviderName:  provider.Name,
			UserName:      user.Name,
			VehicleNumber: r.VehicleNumber,
			UserProfileUrl:  user.ImageUrl,
			Brand:         r.Brand,
			Model:         r.Model,
			ModelYear:     r.ModelYear,
			VehicleType:   r.VehicleType,
			Issues:        r.Issues,
			Cancelled:   cancelled,
			CancelledAt: cancelledAt,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,

		}
		
		
		result = append(result, dtoItem)
		
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

func (s *BookingService) GetProviderBookingDetails(
	ctx context.Context,
	providerID,
	serviceID string,
) (*dto.ProviderBookingDetailDTO, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	serviceObjID, err := primitive.ObjectIDFromHex(serviceID)
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

	r, err := s.acceptedRepo.GetByID(ctx, serviceObjID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, r.User)
	if err != nil {
		return nil, err
	}

	var complaintDTO *dto.ComplaintDTO

	if r.ComplaintProvider != nil {
		complaint, err := s.complaintRepo.FindByAcceptedServiceId(
			ctx,
			serviceObjID,
		)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}

		if complaint != nil {
			var remark string
			if complaint.Assessment != nil {
				remark = complaint.Assessment.RemarkForProvider
			}

			complaintDTO = &dto.ComplaintDTO{
				ID:                  complaint.ID.Hex(),
				ComplaintProviderID: complaint.ID.Hex(),
				ComplaintNumber:     complaint.ComplaintNumber,
				Status:              string(complaint.Status),
				Timeline:            complaint.Timeline,
				CreatedAt:           complaint.CreatedAt,
				UpdatedAt:           complaint.UpdatedAt,
				Remark:              remark,
			}

			if complaint.ProviderComplaint != nil {
				complaintDTO.ProviderIssue = &dto.ProviderComplaintDTO{
					Problem:  complaint.ProviderComplaint.Problem,
					Photos:   complaint.ProviderComplaint.Photos,
					RaisedAt: complaint.ProviderComplaint.RaisedAt,
				}
			}
		}
	}

	const gstPercent = 18.0

	serviceCharge := r.FinalPrice
	gstOnCommission := (serviceCharge * gstPercent) / 100
	providerPayout := serviceCharge + gstOnCommission

	var userLoc dto.UserLocation
	if r.UserLocation != nil {
		userLoc = dto.UserLocation{
			Lat:  r.UserLocation.Lat,
			Long: r.UserLocation.Long,
		}
	}

	var providerLoc dto.ProviderLocation
	if r.ProviderLocation != nil {
		providerLoc = dto.ProviderLocation{
			Lat:  r.ProviderLocation.Lat,
			Long: r.ProviderLocation.Long,
		}
	}

	var cancelled *domain.CancelInfo
	var cancelledAt *time.Time

	if r.Cancelled != nil && r.CancelledAt != nil {
		cancelled = r.Cancelled
		cancelledAt = r.CancelledAt
	}

	return &dto.ProviderBookingDetailDTO{
		ID:               r.ID,
		ProviderID:       string(provider.ID),
		ProfileURL:       user.ImageUrl,
		ServiceNumber:    r.ServiceNumber,
		Status:           string(r.Status),
		FinalPrice:       r.FinalPrice,
		UserProfileUrl:   user.ImageUrl,
		VehicleNumber:    r.VehicleNumber,
		Brand:            r.Brand,
		Model:            r.Model,
		ModelYear:        r.ModelYear,
		FuelType:         r.FuelType,
		VehicleType:      r.VehicleType,
		Issues:           r.Issues,
		Timestamps:       r.Timestamps,
		ProviderName:     provider.Name,
		UserName:         user.Name,
		Complaint:        complaintDTO,
		Billing: dto.BillingDetailsDTO{
			ServiceCharge:  utils.RoundTo2(serviceCharge),
			GSTPercent:     gstPercent,
			GSTAmount:      utils.RoundTo2(gstOnCommission),
			TotalPayable:   serviceCharge,
			ProviderPayout: utils.RoundTo2(providerPayout),
			PaymentStatus:  string(r.PaymentStatus),
		},
		Cancelled:        cancelled,
		CancelledAt:      cancelledAt,
		UserLocation:     userLoc,
		ProviderLocation: providerLoc,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
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
		transaction, err := s.transactionRepo.GetLatestPaidTransactionByServiceID(ctx, service.ID.Hex())
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
			Issues:        service.Issues,
		}

		result = append(result, expense)
		totalExpense += transaction.Amount
	}

	if err := s.userRepo.UpdateTotalExpense(ctx, userObjID, totalExpense); err != nil {
		return nil, 0, err
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
	completedBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted})
	if err != nil {
		return nil, err
	}
	cancelledBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCancelled})
	if err != nil {
		return nil, err
	}
	allTimeEarning := 0.0
	todayEarning := 0.0
	today := time.Now().Truncate(24 * time.Hour)
	for _, booking := range completedBookings {
		transaction, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err == nil && transaction != nil {
			allTimeEarning += transaction.Amount
			if booking.CreatedAt.Truncate(24 * time.Hour).Equal(today) {
				todayEarning += transaction.Amount
			}
		}
	}

	totalSettlement, err := s.settlementRepo.
		GetTotalSettlementByProvider(ctx, providerObjID)
	if err != nil {
		return nil, err
	}
	stats := &dto.DashboardStats{
		AllTimeEarning:    allTimeEarning,
		TodayEarning:      todayEarning,
		ServicesCompleted: len(completedBookings),
		PaymentSettlement: totalSettlement,
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

	completedBookings, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted})
	if err != nil {
		return nil, err
	}

	earnings := make([]dto.EarningDetail, 0, len(completedBookings))
	for _, booking := range completedBookings {
		transaction, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err != nil {
			continue
		}

		serviceName := ""
		if len(booking.Issues) > 0 {
			serviceName = booking.Issues[0]
		}

		earnings = append(earnings, dto.EarningDetail{
			ID:          booking.ID.Hex(),
			ProviderId:  booking.Provider,
			ServiceName: serviceName,
			Amount:      transaction.Amount,
			CreatedAt:   booking.CreatedAt.Format(time.RFC3339),
		})
	}
	return &dto.EarningsResponse{Earnings: earnings}, nil
}
func (s *BookingService) GetProviderTodayEarnings(
	ctx context.Context,
	providerID string,
) (*dto.TodayEarningsResponse, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	_, err = s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startOfDay := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, now.Location(),
	)
	endOfDay := startOfDay.Add(24 * time.Hour)

	completedBookings, err := s.acceptedRepo.
		GetProviderCompletedBookingsByDate(
			ctx,
			providerObjID,
			startOfDay,
			endOfDay,
		)
	if err != nil {
		return nil, err
	}

	var total float64
	earnings := make([]dto.EarningDetail, 0, len(completedBookings))

	for _, booking := range completedBookings {
		transaction, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err != nil {
			continue
		}

		total += transaction.Amount

		serviceName := ""
		if len(booking.Issues) > 0 {
			serviceName = booking.Issues[0]
		}

		earnings = append(earnings, dto.EarningDetail{
			ID:          booking.ID.Hex(),
			ProviderId:  booking.Provider,
			ServiceName: serviceName,
			Amount:      transaction.Amount,
			CreatedAt:   booking.CreatedAt.Format(time.RFC3339),
		})
	}

	return &dto.TodayEarningsResponse{
		Total:    utils.RoundTo2(total),
		Earnings: earnings,
	}, nil
}

func (s *BookingService) GetProviderSettledEarnings(
	ctx context.Context,
	providerID string,
) (*dto.ProviderSettlementResponse, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	records, err := s.settlementRepo.GetProviderSettledRecords(ctx, providerObjID)
	if err != nil {
		return nil, err
	}

	var total float64
	settlements := make([]dto.ProviderSettlementItem, 0)

	for _, rec := range records {
		service, err := s.acceptedRepo.FindByID(ctx, rec.ServiceID.Hex())
		if err != nil {
			continue
		}

		total += rec.NetAmount

		settlements = append(settlements, dto.ProviderSettlementItem{
			ID:            rec.ServiceID.Hex(),
			ProviderID:    rec.ProviderID.Hex(),
			ServiceNumber: service.ServiceNumber,
			Amount:        rec.NetAmount,
			CreatedAt:     rec.CreatedAt.Format(time.RFC3339),
		})
	}

	return &dto.ProviderSettlementResponse{
		Total:       total,
		Settlements: settlements,
	}, nil
}

