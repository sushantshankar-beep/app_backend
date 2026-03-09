package service

import (
	"app_backend/internal/domain"
	"app_backend/internal/dto"
	"app_backend/internal/repository"
	"app_backend/internal/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	// "fmt"
	"math"
)

type BookingService struct {
	acceptedRepo    *repository.AcceptedServiceRepo
	userRepo        *repository.UserRepo
	providerRepo    *repository.ProviderRepo
	catalogRepo     *repository.ServiceCatalogRepo
	transactionRepo *repository.PaymentRepository
	settlementRepo  *repository.SettlementHistoryRepository
	complaintRepo   *repository.ComplaintRepo
	snapshotRepo *repository.SnapshotRepo
	couponSvc *CouponService
	refundRepo *repository.RefundRepo
}

func NewBookingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	catalogRepo *repository.ServiceCatalogRepo,
	transactionRepo *repository.PaymentRepository,
	settlementRepo *repository.SettlementHistoryRepository,
	complaintRepo *repository.ComplaintRepo,
	snapshotRepo *repository.SnapshotRepo,
	couponSvc *CouponService,
	refundRepo *repository.RefundRepo,


) *BookingService {
	return &BookingService{
		acceptedRepo:    acceptedRepo,
		userRepo:        userRepo,
		providerRepo:    providerRepo,
		catalogRepo:     catalogRepo,
		transactionRepo: transactionRepo,
		settlementRepo:  settlementRepo,
		complaintRepo:   complaintRepo,
		snapshotRepo:    snapshotRepo,
		couponSvc: couponSvc,
		refundRepo: refundRepo,
	}
}

func (s *BookingService) BuildBookingScreen(
	ctx context.Context,
	serviceID string,
	userID string,
	isNewUser bool,
) (map[string]any, error) {

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

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(svc.Provider.Hex()),
	)
	if err != nil {
		return nil, errors.New("provider not found")
	}

	var distanceKm float64
	var etaSeconds int64
	var etaMinutes int64

	if svc.UserLocation != nil &&
		svc.ProviderLocation != nil &&
		svc.UserLocation.Lat != 0 &&
		svc.UserLocation.Long != 0 &&
		svc.ProviderLocation.Lat != 0 &&
		svc.ProviderLocation.Long != 0 {

		distanceKm = distanceKmHaversine(
			svc.ProviderLocation.Lat,
			svc.ProviderLocation.Long,
			svc.UserLocation.Lat,
			svc.UserLocation.Long,
		)

		distanceKm = math.Round(distanceKm*100) / 100
		etaSeconds = estimateETASeconds(distanceKm)
		etaMinutes = etaSeconds / 60
	}

	fmt.Println("this is service type", svc.ServiceType)
	if len(svc.Issues) == 0 {
		return nil, errors.New("no issues attached to service")
	}

	pricing, err := s.couponSvc.ApplyAutoDiscount(ctx, serviceID, isNewUser, false)
	if err != nil {
		pricing = &domain.CouponApplyResult{
			ServiceAmount:    svc.FinalPrice,
			DiscountedAmount: svc.FinalPrice,
		}
	}

	source := pricing
	if svc.PaymentStatus == "pending" && svc.PendingCoupon != nil && svc.PendingCoupon.TotalDiscount > 0 {
		source = svc.PendingCoupon
	}

	gst := source.ServiceAmount * 18 / 100
	total := source.ServiceAmount + gst - source.TotalDiscount

	var appliedDiscount any
	if source.AppliedDiscount != nil {
		appliedDiscount = map[string]any{
			"code":                source.AppliedDiscount.Code,
			"discountAmt":         source.AppliedDiscount.DiscountAmt,
			"amountAfterDiscount": source.DiscountedAmount,
		}
	}

	var appliedPromo any
	if source.AppliedPromo != nil {
		appliedPromo = map[string]any{
			"code":                source.AppliedPromo.Code,
			"discountAmt":         source.AppliedPromo.DiscountAmt,
			"amountAfterDiscount": source.DiscountedAmount,
		}
	}

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
			"etaMinutes": etaMinutes,
			"distanceKm": distanceKm,
			"profileUrl": provider.ProfileURL,
		},

		"vehicle": map[string]any{
			"problem":       svc.ServiceType,
			"date":          time.Now().Format("2006-01-02 03:04 PM"),
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"fuelType":      svc.FuelType,
			"year":          svc.ModelYear,
			"model":         svc.Model,
		},

		"billing": map[string]any{
			"serviceAmount":   utils.RoundTo2(source.ServiceAmount),
			"discount":       utils.RoundTo2(source.TotalDiscount),
			"gst":             utils.RoundTo2(gst),
			"totalAmount":     utils.RoundTo2(total),
			"currency":        "INR",
			"appliedDiscount": appliedDiscount,
			"appliedPromo":    appliedPromo,
		},
	}, nil
}

func (s *BookingService) GetUserBookings(ctx context.Context, userID, status string, page, limit int) ([]dto.UserBookingDTO, int64,error) {
	sStatus, err := mapStatus(status)
	if err != nil {
		return nil,0, err
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil,0, err
	}

	user, err := s.userRepo.GetByID(ctx, userObjID)
	if err != nil {
		if err.Error() == "not found" {
			return []dto.UserBookingDTO{},0, nil
		}
		return nil,0, err
	}

	skip := int64((page - 1) * limit)

	raw,total, err := s.acceptedRepo.GetBookingsByUserAndStatus(ctx, userObjID, sStatus,skip,int64(limit))
	if err != nil {
		return nil,0, err
	}

	result := make([]dto.UserBookingDTO, 0, len(raw))

	for _, r := range raw {

		providerIDStr := r.Provider.Hex()
		if r.Provider.IsZero() && r.CancelledProviderID != "" {
			providerIDStr = r.CancelledProviderID
		}

		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(providerIDStr))

		if err != nil {
			continue
		}

		tx, err := s.transactionRepo.GetTransactionByServiceID(ctx, r.ID.Hex())
		if err != nil {
			continue
		}

		var cancelled *domain.CancelInfo
		var cancelledAt *time.Time

		serviceData := &r
		if r.Provider.IsZero() {
			snap, err := s.snapshotRepo.GetByServiceID(ctx, r.ID)
			if err == nil && snap != nil {
				serviceData = &snap.Service
			}
		}

		if serviceData.Cancelled != nil && serviceData.CancelledAt != nil {
			cancelled = serviceData.Cancelled
			cancelledAt = serviceData.CancelledAt
		}

		providerRaisedComplaint := false
		complaint, err := s.complaintRepo.FindByAcceptedServiceId(ctx, r.ID)
		if err == nil && complaint != nil && complaint.ProviderComplaint != nil {
			providerRaisedComplaint = true
		}

		dtoItem := dto.UserBookingDTO{
			ID:                 r.ID,
			UserID:             string(user.ID),
			ProfileURL:         user.ImageUrl,
			ServiceNumber:      r.ServiceNumber,
			VehicleNumber:      r.VehicleNumber,
			Brand:              r.Brand,
			Model:              r.Model,
			ModelYear:          r.ModelYear,
			Status:             string(r.Status),
			FinalPrice:         tx.Amount,
			UserName:           user.Name,
			ProviderName:       provider.Name,
			ProviderProfileUrl: provider.ProfileURL,
			VehicleType:        r.VehicleType,
			Rating:             provider.Rating,
			Issues:             r.Issues,
			CreatedAt:          r.CreatedAt,
			UpdatedAt:          r.UpdatedAt,
			Cancelled:          cancelled,
			CancelledAt:        cancelledAt,
			ProviderRaisedComplaint: providerRaisedComplaint,
		}

		result = append(result, dtoItem)

	}

	return result,total, nil
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
// func distanceKmHaversine(lat1, lon1, lat2, lon2 float64) float64 {
// 	const R = 6371

// 	dLat := (lat2 - lat1) * math.Pi / 180
// 	dLon := (lon2 - lon1) * math.Pi / 180

// 	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
// 		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
// 			math.Sin(dLon/2)*math.Sin(dLon/2)

// 	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

// 	return R * c
// }
// func estimateETASeconds(distanceKm float64) int64 {
// 	const avgSpeed = 30.0
// 	return int64((distanceKm / avgSpeed) * 3600)
// }

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
	var providerRating string

	var providerIDStr string

	if !r.Provider.IsZero() {
		providerIDStr = r.Provider.Hex()
	} else if r.CancelledProviderID != "" {
		providerIDStr = r.CancelledProviderID
	}

	if providerIDStr != "" {
		provider, err := s.providerRepo.FindByID(
			ctx,
			domain.ProviderID(providerIDStr),
		)
		if err == nil {
			providerName = provider.Name
			providerID = provider.ID
			providerProfileUrl = provider.ProfileURL
			providerRating = provider.Rating
		}
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

	var serviceCharge, totalDiscount, amountAfterDiscount, gstAmount, totalPayable float64

	hasDiscount := r.TotalDiscount > 0 || r.AppliedPromo != nil || r.AppliedDiscount != nil

	if r.FinalPrice > 0 && hasDiscount {
		serviceCharge = r.FinalPrice
		totalDiscount = r.TotalDiscount
		gstAmount = (serviceCharge * gstPercent) / 100
		amountAfterDiscount = serviceCharge - totalDiscount
		totalPayable = serviceCharge + gstAmount - totalDiscount
	} else {
		serviceCharge = r.FinalPrice
		gstAmount = (serviceCharge * gstPercent) / 100
		totalPayable = serviceCharge + gstAmount
	}

	var appliedPromo *dto.AppliedPromoDTO
	if r.AppliedPromo != nil {
		appliedPromo = &dto.AppliedPromoDTO{
			PromoID:     r.AppliedPromo.PromoID,
			Code:        r.AppliedPromo.Code,
			DiscountAmt: r.AppliedPromo.DiscountAmt,
		}
	}

	var appliedDiscount *dto.AppliedDiscountDTO
	if r.AppliedDiscount != nil {
		appliedDiscount = &dto.AppliedDiscountDTO{
			DiscountID:  r.AppliedDiscount.DiscountID,
			Code:        r.AppliedDiscount.Code,
			DiscountAmt: r.AppliedDiscount.DiscountAmt,
		}
	}

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

	serviceData := r
	if r.Provider.IsZero() {
		snap, err := s.snapshotRepo.GetByServiceID(ctx, r.ID)
		if err == nil && snap != nil {
			serviceData = &snap.Service
		}
	}

	if serviceData.Cancelled != nil && serviceData.CancelledAt != nil {
		cancelled = serviceData.Cancelled
		cancelledAt = serviceData.CancelledAt
	}
	var distanceKm float64
	var etaMinutes int64
	if r.UserLocation != nil &&
		r.ProviderLocation != nil &&
		r.UserLocation.Lat != 0 &&
		r.UserLocation.Long != 0 &&
		r.ProviderLocation.Lat != 0 &&
		r.ProviderLocation.Long != 0 {

		distanceKm = distanceKmHaversine(
			r.ProviderLocation.Lat,
			r.ProviderLocation.Long,
			r.UserLocation.Lat,
			r.UserLocation.Long,
		)

		distanceKm = math.Round(distanceKm*100) / 100

		etaMinutes = estimateETASeconds(distanceKm) / 60
	}

	var refundDTO *dto.RefundDTO

	if r.Status == domain.StatusCancelled && r.PaymentStatus == "paid" {
		refund, err := s.refundRepo.FindByServiceID(ctx, serviceID)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
		if refund != nil {
			refundDTO = &dto.RefundDTO{
				Amount:    refund.Amount,
				Status:    refund.Status,
				Reason:    refund.Reason,
				CreatedAt: refund.CreatedAt,
				UpdatedAt: refund.UpdatedAt,
			}
		}
	}

	timestamps := r.Timestamps
    if r.Status == domain.StatusCompleted && timestamps != nil {
      cleaned := *timestamps
      cleaned.CancelledAt = nil
      timestamps = &cleaned
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
		DistanceKm:         distanceKm,
		EtaMinutes:         etaMinutes,
		Timestamps:         timestamps,
		UserName:           user.Name,
		ProviderID:         providerID,
		ProviderName:       providerName,
		ProviderProfileUrl: providerProfileUrl,
		Rating:             providerRating,
		Billing: dto.BillingDetailsDTO{
			ServiceCharge: utils.RoundTo2(serviceCharge),
			GSTPercent:    gstPercent,
			GSTAmount:     utils.RoundTo2(gstAmount),
			TotalPayable:  utils.RoundTo2(totalPayable),
			PaymentStatus: string(r.PaymentStatus),
			TotalDiscount:       utils.RoundTo2(totalDiscount),
			AmountAfterDiscount: utils.RoundTo2(amountAfterDiscount),
			AppliedPromo:        appliedPromo,
			AppliedDiscount:     appliedDiscount,
		},
		Cancelled:        cancelled,
		CancelledAt:      cancelledAt,
		Complaint:        complaintDTO,
		UserLocation:     userLoc,
		ProviderLocation: providerLoc,
		Refund:           refundDTO,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}, nil
}


func (s *BookingService) GetProviderBookings(ctx context.Context, providerID, status string, page, limit int) (*dto.ProviderBookingResponse,int64, error) {
	sStatus, err := mapStatus(status)
	if err != nil {
        return nil,0, err
	}

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
        return nil, 0,err
	}

	skip := int64((page - 1) * limit)

    raw,total, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, sStatus,skip,int64(limit))
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.ProviderBookingDTO, 0, len(raw))

	for _, r := range raw {

		serviceData := &r

		if r.CancelledByProvider && r.CancelledProviderID == providerID {
			snap, err := s.snapshotRepo.GetByServiceID(ctx, r.ID)
			if err == nil && snap != nil {
				serviceData = &snap.Service
			}
		}

		userObjID, err := primitive.ObjectIDFromHex(serviceData.User.Hex())
		if err != nil {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, userObjID)
		if err != nil {
			continue
		}

		providerIDStr := serviceData.Provider.Hex()
if providerIDStr == "" {
			providerIDStr = providerID
}

		provider, err := s.providerRepo.FindByID(
			ctx,
			domain.ProviderID(providerIDStr),
		)
		if err != nil {
			continue
		}

		const gstPercent = 18.0

		serviceCharge := serviceData.FinalPrice
		gstAmount := (serviceCharge * gstPercent) / 100
		finalPrice := utils.RoundTo2(serviceCharge + gstAmount)

		var cancelled *domain.CancelInfo
		var cancelledAt *time.Time

		if serviceData.Cancelled != nil {
			cancelled = serviceData.Cancelled
			if serviceData.CancelledAt != nil {
				cancelledAt = serviceData.CancelledAt
			}
		}

		userRaisedComplaint := false
		complaint, err := s.complaintRepo.FindByAcceptedServiceId(ctx, serviceData.ID)
		if err == nil && complaint != nil && complaint.UserComplaint != nil {
			userRaisedComplaint = true
		}

		dtoItem := dto.ProviderBookingDTO{
            ID:             serviceData.ID,
            ProviderID:     providerIDStr,
            ProfileURL:     user.ImageUrl,
            ServiceNumber:  serviceData.ServiceNumber,
            Status:         string(serviceData.Status),
            FinalPrice:     finalPrice,
            ProviderName:   provider.Name,
            UserName:       user.Name,
            VehicleNumber:  serviceData.VehicleNumber,
            UserProfileUrl: user.ImageUrl,
            Brand:          serviceData.Brand,
            Model:          serviceData.Model,
            ModelYear:      serviceData.ModelYear,
            VehicleType:    serviceData.VehicleType,
            Rating:         user.Rating,
            Issues:         serviceData.Issues,
            Cancelled:      cancelled,
            CancelledAt:    cancelledAt,
            CreatedAt:      serviceData.CreatedAt,
            UpdatedAt:      serviceData.UpdatedAt,
			UserRaisedComplaint: userRaisedComplaint,
		}

		result = append(result, dtoItem)
	}

	response := &dto.ProviderBookingResponse{
		Bookings: result,
	}

	if containsStatus(sStatus, domain.StatusCompleted) || containsStatus(sStatus, domain.StatusCancelled) {
		response.Count = len(result)
	}

    return response,total, nil
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

	serviceObjID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	r, err := s.acceptedRepo.GetByID(ctx, serviceObjID)
	if err != nil {
		return nil, err
	}

	serviceData := r
	targetProviderID := providerID

	if r.CancelledByProvider && r.CancelledProviderID == providerID {
		snap, err := s.snapshotRepo.GetByServiceID(ctx, r.ID)
		if err == nil && snap != nil {
			serviceData = &snap.Service
		}
		targetProviderID = r.CancelledProviderID
	} else if r.Provider.Hex() == providerID {
		targetProviderID = r.Provider.Hex()
	}

	provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(targetProviderID),
	)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, serviceData.User)
	if err != nil {
		return nil, err
	}

	var complaintDTO *dto.ComplaintDTO

	var remark string
	var userRemark string

	complaint, err := s.complaintRepo.FindByAcceptedServiceId(ctx, serviceObjID )

	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	if complaint != nil {
		if complaint.Assessment != nil {
			remark = complaint.Assessment.RemarkForProvider
			userRemark = complaint.Assessment.RemarkForUser
		}

		if complaint.ProviderComplaint != nil {
			complaintDTO = &dto.ComplaintDTO{
				ID:                  complaint.ID.Hex(),
				ComplaintProviderID: complaint.ID.Hex(),
				ComplaintNumber:     complaint.ComplaintNumber,
				Status:              string(complaint.Status),
				Timeline:            complaint.Timeline,
				CreatedAt:           complaint.CreatedAt,
				UpdatedAt:           complaint.UpdatedAt,
				Remark:              remark,
				UserRemark:          userRemark,
				ProviderIssue: &dto.ProviderComplaintDTO{
					Problem:  complaint.ProviderComplaint.Problem,
					Photos:   complaint.ProviderComplaint.Photos,
					RaisedAt: complaint.ProviderComplaint.RaisedAt,
				},
			}
		}
	}

	const gstPercent = 18.0

	serviceCharge := serviceData.FinalPrice
	gstAmount := (serviceCharge * gstPercent) / 100
	totalPayable := serviceCharge + gstAmount

	var userLoc dto.UserLocation
	if serviceData.UserLocation != nil {
		userLoc = dto.UserLocation{
			Lat:  serviceData.UserLocation.Lat,
			Long: serviceData.UserLocation.Long,
		}
	}

	var providerLoc dto.ProviderLocation
	if serviceData.ProviderLocation != nil {
		providerLoc = dto.ProviderLocation{
			Lat:  serviceData.ProviderLocation.Lat,
			Long: serviceData.ProviderLocation.Long,
		}
	}

	var cancelled *domain.CancelInfo
	var cancelledAt *time.Time

	if serviceData.Cancelled != nil && serviceData.CancelledAt != nil {
		cancelled = serviceData.Cancelled
		cancelledAt = serviceData.CancelledAt
	}

	timestamps := serviceData.Timestamps
    if r.Status == domain.StatusCompleted && 
    !(r.CancelledByProvider && r.CancelledProviderID == providerID) &&
       timestamps != nil {
       cleaned := *timestamps
       cleaned.CancelledAt = nil
       timestamps = &cleaned
    }

	return &dto.ProviderBookingDetailDTO{
		ID:             serviceData.ID,
		ProviderID:     targetProviderID,
		ProfileURL:     user.ImageUrl,
		ServiceNumber:  serviceData.ServiceNumber,
		Status:         string(serviceData.Status),
		FinalPrice:     serviceData.FinalPrice,
		UserProfileUrl: user.ImageUrl,
		VehicleNumber:  serviceData.VehicleNumber,
		Brand:          serviceData.Brand,
		Model:          serviceData.Model,
		ModelYear:      serviceData.ModelYear,
		FuelType:       serviceData.FuelType,
		Rating:         user.Rating,
		VehicleType:    serviceData.VehicleType,
		Issues:         serviceData.Issues,
		Timestamps:     timestamps,
		ProviderName:   provider.Name,
		UserName:       user.Name,
		Complaint:      complaintDTO,
		Remark:     remark,
        UserRemark: userRemark, 
		Billing: dto.BillingDetailsDTO{
			ServiceCharge:  utils.RoundTo2(serviceCharge),
			GSTPercent:     gstPercent,
			GSTAmount:      utils.RoundTo2(gstAmount),
			TotalPayable:   serviceCharge,
			ProviderPayout: utils.RoundTo2(totalPayable),
			PaymentStatus:  string(serviceData.PaymentStatus),
		},
		Cancelled:        cancelled,
		CancelledAt:      cancelledAt,
		UserLocation:     userLoc,
		ProviderLocation: providerLoc,
		CreatedAt:        serviceData.CreatedAt,
		UpdatedAt:        serviceData.UpdatedAt,
	}, nil
}

func (s *BookingService) GetUserExpenses(ctx context.Context, userID string, page, limit int) ([]dto.UserExpenseDTO, float64, int64, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, 0, err
	}

	_, err = s.userRepo.GetByID(ctx, userObjID)
	if err != nil {
		return nil, 0, 0, err
	}

	skip := int64((page - 1) * limit)

	// 🔹 Paginated services
	services, total, err := s.acceptedRepo.GetCompletedServicesByUser(ctx, userID, skip, int64(limit))
	if err != nil {
		return nil, 0, 0, err
	}

	result := make([]dto.UserExpenseDTO, 0, len(services))

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
			return nil, 0, 0, err
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
	}

	// 🔹 Get ALL services (no pagination) for total expense
	allServices, _, err := s.acceptedRepo.GetCompletedServicesByUser(ctx, userID, 0, 1000000)
	if err != nil {
		return nil, 0, 0, err
	}

	var totalExpense float64

	for _, service := range allServices {
		transaction, err := s.transactionRepo.GetLatestPaidTransactionByServiceID(ctx, service.ID.Hex())
		if err != nil {
			continue
		}

		if transaction.Status == string(domain.PaymentPaid) {
			totalExpense += utils.RoundTo2(transaction.Amount)
		}
	}

	return result, totalExpense, total, nil
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
	completedBookings,_, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted},0,0)
	if err != nil {
		return nil, err
	}
	cancelledBookings, _, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCancelled},0,0)
	if err != nil {
		return nil, err
	}

	const gstPercent = 18.0

	allTimeEarning := 0.0
	todayEarning := 0.0
	today := time.Now().Truncate(24 * time.Hour)

	for _, booking := range completedBookings {
		serviceCharge := booking.FinalPrice
		gstAmount := (serviceCharge * gstPercent) / 100
		amount := utils.RoundTo2(serviceCharge + gstAmount)

		allTimeEarning += amount
		if booking.CreatedAt.Truncate(24 * time.Hour).Equal(today) {
			todayEarning += amount
		}
	}

	totalSettlement, err := s.settlementRepo.
		GetTotalSettlementByProvider(ctx, providerObjID)
	if err != nil {
		return nil, err
	}

	stats := &dto.DashboardStats{
		AllTimeEarning:    utils.RoundTo2(allTimeEarning),
		TodayEarning:      utils.RoundTo2(todayEarning),
		ServicesCompleted: len(completedBookings),
		PaymentSettlement: utils.RoundTo2(totalSettlement),
		CancelledServices: len(cancelledBookings),
	}
	return stats, nil
}

func (s *BookingService) GetProviderEarnings(ctx context.Context, providerID string, page, limit int) (*dto.EarningsResponse, int64, error) {
	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil,0, err
	}
	_, err = s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil,0, err
	}

	skip := int64((page - 1) * limit)

	completedBookings,totalCount, err := s.acceptedRepo.GetBookingsByProviderAndStatus(ctx, providerObjID, []domain.ServiceStatus{domain.StatusCompleted},skip, int64(limit))
	if err != nil {
		return nil,0, err
	}

	const gstPercent = 18.0

	earnings := make([]dto.EarningDetail, 0, len(completedBookings))
	for _, booking := range completedBookings {
		_, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err != nil {
			continue
		}

		serviceName := ""
		if len(booking.Issues) > 0 {
			serviceName = booking.Issues[0]
		}

		serviceCharge := booking.FinalPrice
		gstAmount := (serviceCharge * gstPercent) / 100
		finalPrice := utils.RoundTo2(serviceCharge + gstAmount)

		earnings = append(earnings, dto.EarningDetail{
			ID:          booking.ID.Hex(),
			ProviderId:  booking.Provider,
			ServiceName: serviceName,
			Amount:      finalPrice,
			CreatedAt:   booking.CreatedAt.Format(time.RFC3339),
		})
	}
	return &dto.EarningsResponse{Earnings: earnings},totalCount, nil
}
func (s *BookingService) GetProviderTodayEarnings(
	ctx context.Context,
	providerID string,
	page , limit int,
) (*dto.TodayEarningsResponse, int64, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil,0, err
	}

	_, err = s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))
	if err != nil {
		return nil,0, err
	}

	now := time.Now()
	startOfDay := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, now.Location(),
	)
	endOfDay := startOfDay.Add(24 * time.Hour)

	skip := int64((page - 1) * limit)

	completedBookings,totalCount, err := s.acceptedRepo.
		GetProviderCompletedBookingsByDate(
			ctx,
			providerObjID,
			startOfDay,
			endOfDay,
			skip,
			int64(limit),
		)
	if err != nil {
		return nil, 0, err
	}

	allTodayBookings, _, err := s.acceptedRepo.
		GetProviderCompletedBookingsByDate(
			ctx,
			providerObjID,
			startOfDay,
			endOfDay,
			0,
			0,
		)
	if err != nil {
		return nil, 0, err
	}

	const gstPercent = 18.0

	var grandTotal float64
	for _, booking := range allTodayBookings {
		_, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err != nil {
			continue
		}
		serviceCharge := booking.FinalPrice
		gstAmount := (serviceCharge * gstPercent) / 100
		grandTotal += utils.RoundTo2(serviceCharge + gstAmount)
	}

	earnings := make([]dto.EarningDetail, 0, len(completedBookings))
	for _, booking := range completedBookings {
		_, err := s.transactionRepo.GetTransactionByServiceID(ctx, booking.ID.Hex())
		if err != nil {
			continue
		}

		serviceCharge := booking.FinalPrice
		gstAmount := (serviceCharge * gstPercent) / 100
		finalPrice := utils.RoundTo2(serviceCharge + gstAmount)

		serviceName := ""
		if len(booking.Issues) > 0 {
			serviceName = booking.Issues[0]
		}

		earnings = append(earnings, dto.EarningDetail{
			ID:          booking.ID.Hex(),
			ProviderId:  booking.Provider,
			ServiceName: serviceName,
			Amount:      finalPrice,
			CreatedAt:   booking.CreatedAt.Format(time.RFC3339),
		})
	}

	return &dto.TodayEarningsResponse{
		Total:    utils.RoundTo2(grandTotal),
		Earnings: earnings,
	}, totalCount, nil
}

func (s *BookingService) GetProviderSettledEarnings(
	ctx context.Context,
	providerID string,
	page, limit int,
) (*dto.ProviderSettlementResponse, int64, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil,0, err
	}

	skip := int64((page - 1) * limit)

	records,totalCount, err := s.settlementRepo.GetProviderSettledRecords(ctx, providerObjID, skip , int64(limit))
	if err != nil {
		return nil, 0,err
	}

	allRecords, _, err := s.settlementRepo.GetProviderSettledRecords(ctx, providerObjID, 0, 0)
	if err != nil {
		return nil, 0, err
	}

	var grandTotal float64
	for _, rec := range allRecords {
		grandTotal += rec.NetAmount
	}

	settlements := make([]dto.ProviderSettlementItem, 0)
	for _, rec := range records {
		service, err := s.acceptedRepo.FindByID(ctx, rec.ServiceID.Hex())
		if err != nil {
			continue
		}

		settlements = append(settlements, dto.ProviderSettlementItem{
			ID:            rec.ServiceID.Hex(),
			ProviderID:    rec.ProviderID.Hex(),
			ServiceNumber: service.ServiceNumber,
			Amount:        utils.RoundTo2(rec.NetAmount),
			CreatedAt:     rec.CreatedAt.Format(time.RFC3339),
		})
	}

	return &dto.ProviderSettlementResponse{
		Total:       utils.RoundTo2(grandTotal),
		Settlements: settlements,
	}, totalCount, nil
}
